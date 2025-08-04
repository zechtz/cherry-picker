package main

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// Bubbletea model methods
func (cp *CherryPicker) Init() tea.Cmd {
	return cp.tickCmd()
}

func (cp *CherryPicker) tickCmd() tea.Cmd {
	interval := time.Duration(cp.config.UI.CursorBlinkInterval) * time.Millisecond
	return tea.Tick(interval, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func (cp *CherryPicker) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Handle search mode input differently
		if cp.searchMode {
			return cp.handleSearchInput(msg)
		}
		
		// Handle conflict mode input differently
		if cp.conflictMode {
			if cp.editorMode {
				return cp.handleEditorInput(msg)
			}
			return cp.handleConflictInput(msg)
		}
		
		// Handle branch mode input differently
		if cp.branchMode {
			return cp.handleBranchInput(msg)
		}
		
		// Handle author mode input differently
		if cp.authorMode {
			return cp.handleAuthorInput(msg)
		}
		
		switch msg.String() {
		case "ctrl+c", "q":
			cp.quitting = true
			return cp, tea.Batch(tea.ExitAltScreen, tea.Quit)
		case "enter", " ":
			commit := cp.getCurrentCommit()
			if commit != nil && !commit.AlreadyApplied {
				cp.selected[commit.SHA] = !cp.selected[commit.SHA]
			}
		case "down", "j", "n":
			maxIndex := cp.getMaxIndex()
			if cp.currentIndex < maxIndex && maxIndex >= 0 {
				cp.currentIndex++
				cp.updateRangeEnd()
				cp.updatePreview()
			}
		case "up", "k":
			if cp.currentIndex > 0 {
				cp.currentIndex--
				cp.updateRangeEnd()
				cp.updatePreview()
			}
		case "pagedown", "ctrl+f":
			// Jump down by page
			maxIndex := cp.getMaxIndex()
			if maxIndex >= 0 {
				cp.currentIndex += 25
				if cp.currentIndex > maxIndex {
					cp.currentIndex = maxIndex
				}
				cp.updateRangeEnd()
				cp.updatePreview()
			}
		case "pageup", "ctrl+b":
			// Jump up by page
			cp.currentIndex -= 25
			if cp.currentIndex < 0 {
				cp.currentIndex = 0
			}
			cp.updateRangeEnd()
			cp.updatePreview()
		case "/", "f":
			// Enter search mode
			cp.toggleSearchMode()
		case "p", "tab":
			// Toggle preview mode
			cp.togglePreviewMode()
		case "b":
			// Switch target branch
			cp.enterBranchMode("target")
		case "B":
			// Switch source branch
			cp.enterBranchMode("source")
		case "A":
			// Switch author
			cp.enterAuthorMode()
		case "r":
			// Toggle range selection mode
			cp.toggleRangeSelection()
		case "R":
			// Toggle reverse commit order
			cp.toggleCommitOrder()
		case "d":
			// Toggle detail view
			cp.detailView = !cp.detailView
		case "H":
			// Toggle hiding applied commits
			cp.hideApplied = !cp.hideApplied
		case "a":
			// Select all visible commits (except already applied ones)
			visibleCommits := cp.getVisibleCommits()
			for _, commit := range visibleCommits {
				if !commit.AlreadyApplied {
					cp.selected[commit.SHA] = true
				}
			}
		case "c":
			// Clear all selections
			cp.selected = make(map[string]bool)
		case "m":
			// Filter/highlight merge commits
			// This could be implemented as a filter mode
		case "i":
			// Interactive rebase selected commits
			if len(cp.getSelectedSHAs()) > 0 {
				cp.rebaseRequested = true
				cp.quitting = true
				return cp, tea.Batch(tea.ExitAltScreen, tea.Quit)
			}
		case "e", "x":
			// Execute cherry-pick for selected commits
			if len(cp.getSelectedSHAs()) > 0 {
				cp.executeRequested = true
				cp.quitting = true
				return cp, tea.Batch(tea.ExitAltScreen, tea.Quit)
			}
		case "?":
			// Show help (could be implemented as a help overlay)
		}
	case tickMsg:
		cp.cursorBlink = !cp.cursorBlink
		return cp, cp.tickCmd()
	}
	return cp, nil
}

// handleSearchInput handles keyboard input when in search mode
func (cp *CherryPicker) handleSearchInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Handle special keys first (control keys that shouldn't be added to search)
	switch msg.Type {
	case tea.KeyCtrlC:
		cp.quitting = true
		return cp, tea.Batch(tea.ExitAltScreen, tea.Quit)
	case tea.KeyEsc:
		// Exit search mode
		cp.toggleSearchMode()
		return cp, nil
	case tea.KeyEnter:
		// Exit search mode and keep current filter
		cp.searchMode = false
		if len(cp.filteredCommits) == 0 {
			// If no results, reset to show all commits
			cp.filteredCommits = nil
		}
		return cp, nil
	case tea.KeyBackspace:
		// Remove last character from search query
		if len(cp.searchQuery) > 0 {
			cp.searchQuery = cp.searchQuery[:len(cp.searchQuery)-1]
			cp.updateSearchResults()
		}
		return cp, nil
	case tea.KeyUp:
		// Navigate up in search results (only arrow keys, not 'k')
		if cp.currentIndex > 0 {
			cp.currentIndex--
			cp.updatePreview()
		}
		return cp, nil
	case tea.KeyDown:
		// Navigate down in search results (only arrow keys, not 'j')
		maxIndex := cp.getMaxIndex()
		if cp.currentIndex < maxIndex {
			cp.currentIndex++
			cp.updatePreview()
		}
		return cp, nil
	case tea.KeyTab:
		// Toggle selection of current commit in search mode (use TAB instead of SPACE)
		commit := cp.getCurrentCommit()
		if commit != nil && !commit.AlreadyApplied {
			cp.selected[commit.SHA] = !cp.selected[commit.SHA]
		}
		return cp, nil
	}
	
	// Handle regular character input - prioritize text input over everything else
	if len(msg.String()) == 1 && msg.String() >= " " && msg.String() <= "~" {
		// Add any printable ASCII character to search query
		cp.searchQuery += msg.String()
		cp.updateSearchResults()
	}
	
	return cp, nil
}

func (cp *CherryPicker) View() string {
	if cp.quitting {
		return ""
	}

	if cp.previewMode {
		return cp.renderPreviewView()
	}
	
	if cp.conflictMode {
		if cp.editorMode {
			return cp.renderEditorView()
		}
		return cp.renderConflictView()
	}
	
	if cp.branchMode {
		return cp.renderBranchView()
	}
	
	if cp.authorMode {
		return cp.renderAuthorView()
	}


	var s strings.Builder

	s.WriteString("📝 Cherry Pick Commits\n\n")
	
	// Show cherry-pick direction and author filter
	s.WriteString(fmt.Sprintf("🌿 Cherry-picking from %s → %s\n", 
		cp.config.Git.SourceBranch, 
		cp.config.Git.TargetBranch))
	s.WriteString(fmt.Sprintf("👤 Author Filter: %s\n\n", cp.selectedAuthor))
	
	// Show search interface if in search mode
	if cp.searchMode {
		s.WriteString("🔍 Search: " + cp.searchQuery + "█\n")
		s.WriteString("(ESC=exit search, ENTER=keep filter, ↑↓=navigate, TAB=toggle)\n\n")
		if len(cp.filteredCommits) == 0 && cp.searchQuery != "" {
			s.WriteString("No commits match your search.\n")
			return s.String()
		}
	}
	
	// Show appropriate title
	if cp.searchMode && cp.searchQuery != "" {
		s.WriteString(fmt.Sprintf("Filtered commits (%d results):\n", len(cp.filteredCommits)))
	} else {
		s.WriteString("Available commits:\n")
	}

	// Get commits to display (filtered or all)
	visibleCommits := cp.getVisibleCommits()
	
	// Pagination settings
	maxCommitsPerPage := 25
	startIndex := 0
	endIndex := len(visibleCommits)
	
	// Calculate pagination based on current cursor position
	if len(visibleCommits) > maxCommitsPerPage {
		// Ensure cursor is within bounds
		if cp.currentIndex >= len(visibleCommits) {
			cp.currentIndex = len(visibleCommits) - 1
		}
		if cp.currentIndex < 0 {
			cp.currentIndex = 0
		}
		
		// Calculate which page the cursor should be on
		cursorPage := cp.currentIndex / maxCommitsPerPage
		startIndex = cursorPage * maxCommitsPerPage
		endIndex = startIndex + maxCommitsPerPage
		if endIndex > len(visibleCommits) {
			endIndex = len(visibleCommits)
		}
		
		// Ensure the cursor is visible on the current page
		if cp.currentIndex < startIndex || cp.currentIndex >= endIndex {
			// Recalculate page to ensure cursor is visible
			cursorPage = cp.currentIndex / maxCommitsPerPage
			startIndex = cursorPage * maxCommitsPerPage
			endIndex = startIndex + maxCommitsPerPage
			if endIndex > len(visibleCommits) {
				endIndex = len(visibleCommits)
			}
		}
	}
	
	// Show pagination info if needed
	if len(visibleCommits) > maxCommitsPerPage {
		currentPage := (cp.currentIndex / maxCommitsPerPage) + 1
		totalPages := (len(visibleCommits) + maxCommitsPerPage - 1) / maxCommitsPerPage
		s.WriteString(fmt.Sprintf("Page %d of %d (showing %d-%d of %d commits)\n", 
			currentPage, totalPages, startIndex+1, endIndex, len(visibleCommits)))
	}
	
	// Display commits for current page
	for i := startIndex; i < endIndex; i++ {
		commit := visibleCommits[i]
		cursor := "  "
		checkbox := "[ ]"
		commitText := commit.Full
		
		// Range selection highlighting
		if cp.isInRange(i) {
			cursor = "📍"
		}

		// Handle already applied commits
		if commit.AlreadyApplied {
			checkbox = "[✗]"
			// Add strikethrough and dim styling for already applied commits
			commitText = "\033[9m\033[2m" + commit.Full + "\033[0m"
		} else if cp.selected[commit.SHA] {
			checkbox = "[✓]"
			// Add strikethrough to selected commits
			commitText = "\033[9m" + commit.Full + "\033[0m"
		}

		// Highlight current cursor position - use actual index not display index
		if i == cp.currentIndex {
			cursor = "→ "
			// Make the entire line highlighted for better visibility
			if cp.cursorBlink {
				if commit.AlreadyApplied {
					checkbox = "[✗]" // No blinking for already applied
				} else if cp.selected[commit.SHA] {
					checkbox = "[█]"
				} else {
					checkbox = "[█]"
				}
			}
			// Add background highlighting to the entire line
			commitText = "\033[7m" + commitText + "\033[0m"
		}

		// Add merge commit indicator
		mergeIndicator := ""
		if commit.IsMerge {
			mergeIndicator = " 🔀"
		}

		// Enhanced display with metadata if detail view is enabled
		if cp.detailView {
			dateStr := ""
			if !commit.Date.IsZero() {
				dateStr = commit.Date.Format("2006-01-02")
			}
			
			statsStr := ""
			if commit.Insertions > 0 || commit.Deletions > 0 {
				statsStr = fmt.Sprintf(" (+%d -%d)", commit.Insertions, commit.Deletions)
			}
			
			filesStr := ""
			if len(commit.FilesChanged) > 0 {
				if len(commit.FilesChanged) == 1 {
					filesStr = fmt.Sprintf(" [%s]", commit.FilesChanged[0])
				} else {
					filesStr = fmt.Sprintf(" [%d files]", len(commit.FilesChanged))
				}
			}
			
			s.WriteString(fmt.Sprintf("%s%s %s%s%s%s\n", cursor, checkbox, commitText, mergeIndicator, statsStr, filesStr))
			if dateStr != "" || commit.Author != "" {
				s.WriteString(fmt.Sprintf("    📅 %s 👤 %s\n", dateStr, commit.Author))
			}
		} else {
			s.WriteString(fmt.Sprintf("%s%s %s%s\n", cursor, checkbox, commitText, mergeIndicator))
		}
	}

	s.WriteString("\n")
	s.WriteString(cp.getSelectedCommitsDisplay())
	s.WriteString("\n")
	s.WriteString(cp.getStatusLine())
	s.WriteString("\n")
	s.WriteString(cp.getControlsDisplay())

	return s.String()
}

// renderPreviewView renders the commit preview interface
func (cp *CherryPicker) renderPreviewView() string {
	var s strings.Builder
	
	s.WriteString("📖 Commit Preview\n")
	s.WriteString("═══════════════════════════════════════════════════════════════════════════════\n\n")
	
	if cp.previewCommit == nil {
		s.WriteString("No commit selected for preview.\n")
		s.WriteString("\nPress 'p' or TAB to exit preview mode.")
		return s.String()
	}
	
	commit := cp.previewCommit
	
	// Header with commit info
	s.WriteString(fmt.Sprintf("🏷️  %s", commit.SHA))
	if commit.AlreadyApplied {
		s.WriteString(" ✗ ALREADY APPLIED")
	}
	if commit.IsMerge {
		s.WriteString(" 🔀 MERGE")
	}
	s.WriteString("\n\n")
	
	// Commit message
	s.WriteString("📝 Message:\n")
	s.WriteString(commit.Message + "\n\n")
	
	// Metadata
	if !commit.Date.IsZero() {
		s.WriteString(fmt.Sprintf("📅 Date: %s\n", commit.Date.Format("2006-01-02 15:04:05")))
	}
	if commit.Author != "" {
		s.WriteString(fmt.Sprintf("👤 Author: %s\n", commit.Author))
	}
	s.WriteString("\n")
	
	// Statistics
	if cp.previewStats != "" {
		s.WriteString(cp.previewStats)
		s.WriteString("\n")
	}
	
	// Diff preview (truncated)
	if cp.previewDiff != "" {
		s.WriteString("🔍 Diff Preview:\n")
		s.WriteString("───────────────────────────────────────────────────────────────────────────────\n")
		
		// Truncate diff if too long
		diffLines := strings.Split(cp.previewDiff, "\n")
		maxLines := 20 // Show first 20 lines of diff
		
		for i, line := range diffLines {
			if i >= maxLines {
				s.WriteString(fmt.Sprintf("... (%d more lines) ...\n", len(diffLines)-maxLines))
				break
			}
			
			// Add color coding for diff lines
			if strings.HasPrefix(line, "+") && !strings.HasPrefix(line, "+++") {
				s.WriteString("\033[32m" + line + "\033[0m\n") // Green for additions
			} else if strings.HasPrefix(line, "-") && !strings.HasPrefix(line, "---") {
				s.WriteString("\033[31m" + line + "\033[0m\n") // Red for deletions
			} else if strings.HasPrefix(line, "@@") {
				s.WriteString("\033[36m" + line + "\033[0m\n") // Cyan for hunk headers
			} else {
				s.WriteString(line + "\n")
			}
		}
		s.WriteString("───────────────────────────────────────────────────────────────────────────────\n\n")
	}
	
	// Controls
	s.WriteString("Controls: p/TAB=exit preview, ↑↓=navigate commits, SPACE=toggle selection, q=quit\n")
	
	return s.String()
}

// handleConflictInput handles keyboard input when in conflict resolution mode
func (cp *CherryPicker) handleConflictInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c", "q":
		cp.quitting = true
		return cp, tea.Batch(tea.ExitAltScreen, tea.Quit)
	case "esc":
		// Exit conflict mode without action
		cp.exitConflictMode()
	case "c":
		// Continue cherry-pick (if all conflicts resolved)
		if err := cp.continueConflictResolution(); err != nil {
			// Still have conflicts, stay in conflict mode
			cp.loadConflictFiles()
		} else {
			// Success, exit conflict mode
			cp.exitConflictMode()
		}
	case "a":
		// Abort cherry-pick
		if err := cp.abortConflictResolution(); err == nil {
			cp.exitConflictMode()
		}
	case "s":
		// Skip this commit
		if err := cp.skipConflictResolution(); err == nil {
			cp.exitConflictMode()
		}
	case "1":
		// Enter editor selection mode
		cp.enterEditorMode()
	case "2":
		// Skip this commit
		if err := cp.skipConflictResolution(); err == nil {
			cp.exitConflictMode()
		}
	case "3":
		// Abort cherry-pick
		if err := cp.abortConflictResolution(); err == nil {
			cp.exitConflictMode()
		}
	case "4":
		// Continue after manual resolution
		if err := cp.continueConflictResolution(); err != nil {
			// Still have conflicts, stay in conflict mode
			cp.loadConflictFiles()
		} else {
			// Success, exit conflict mode
			cp.exitConflictMode()
		}
	case "r":
		// Refresh conflict status
		cp.loadConflictFiles()
	}
	return cp, nil
}

// showFileResolutionOptions shows resolution options for a specific file
func (cp *CherryPicker) showFileResolutionOptions(fileIndex int) {
	// This would typically open a sub-menu or prompt
	// For now, we'll use a simple approach
	file := cp.conflictFiles[fileIndex]
	
	// For demonstration, let's auto-resolve with "ours" strategy
	// In a real implementation, this would show options
	if err := cp.resolveConflictWithStrategy(file.Path, "ours"); err == nil {
		cp.loadConflictFiles() // Refresh the conflict list
	}
}

// renderConflictView renders the conflict resolution interface
func (cp *CherryPicker) renderConflictView() string {
	var s strings.Builder
	
	s.WriteString("⚠️  Conflict Resolution\n")
	s.WriteString("═══════════════════════════════════════════════════════════════════════════════\n\n")
	
	if cp.conflictCommit != "" {
		s.WriteString(fmt.Sprintf("🔧 Resolving conflicts for commit: %s\n\n", cp.conflictCommit))
	}
	
	if len(cp.conflictFiles) == 0 {
		s.WriteString("✅ No conflicts detected. You can continue the cherry-pick.\n\n")
		s.WriteString("Press 'c' to continue, 'a' to abort, or 's' to skip this commit.\n")
		return s.String()
	}
	
	s.WriteString("📁 Conflicted Files:\n")
	s.WriteString("───────────────────────────────────────────────────────────────────────────────\n")
	
	for i, file := range cp.conflictFiles {
		status := "⚡"
		if !file.HasConflicts {
			status = "✅"
		}
		
		s.WriteString(fmt.Sprintf("%d. %s %s\n", i+1, status, file.Path))
		s.WriteString(fmt.Sprintf("   Status: %s - %s\n", file.Status, file.Description))
		
		if file.HasConflicts {
			s.WriteString("   Contains conflict markers (<<<<<<< ======= >>>>>>>)\n")
		} else {
			s.WriteString("   Resolved (no conflict markers)\n")
		}
		s.WriteString("\n")
	}
	
	s.WriteString("───────────────────────────────────────────────────────────────────────────────\n\n")
	
	// Resolution options
	s.WriteString("🔧 Resolution Options:\n")
	s.WriteString("• 1 = Choose editor to resolve conflicts\n")
	s.WriteString("• 2 = Skip this commit\n")
	s.WriteString("• 3 = Abort cherry-pick\n")
	s.WriteString("• 4 = Continue after manual resolution\n")
	s.WriteString("• r = Refresh conflict status\n")
	s.WriteString("• ESC = Exit conflict mode\n\n")
	
	s.WriteString("💡 Tip: Resolve conflicts manually in your editor, then press 'r' to refresh\n")
	s.WriteString("    and '4' to continue when all conflicts are resolved.\n")
	
	return s.String()
}

// handleBranchInput handles keyboard input when in branch selection mode
func (cp *CherryPicker) handleBranchInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c", "q":
		cp.quitting = true
		return cp, tea.Batch(tea.ExitAltScreen, tea.Quit)
	case "esc":
		// Exit branch mode without changes
		cp.exitBranchMode()
	case "enter":
		// Select the current branch and reload commits
		if err := cp.selectBranch(); err != nil {
			// Handle error, but for now just exit branch mode
			cp.exitBranchMode()
		}
	case "down", "j":
		// Navigate down in branch list
		if cp.branchIndex < len(cp.availableBranches)-1 {
			cp.branchIndex++
		}
	case "up", "k":
		// Navigate up in branch list
		if cp.branchIndex > 0 {
			cp.branchIndex--
		}
	case "r":
		// Refresh branch list
		cp.loadAvailableBranches()
	}
	return cp, nil
}

// handleAuthorInput handles keyboard input when in author selection mode
func (cp *CherryPicker) handleAuthorInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Handle search mode input differently
	if cp.authorSearchMode {
		return cp.handleAuthorSearchInput(msg)
	}
	
	switch msg.String() {
	case "ctrl+c", "q":
		cp.quitting = true
		return cp, tea.Batch(tea.ExitAltScreen, tea.Quit)
	case "esc":
		// Exit author mode without changes
		cp.exitAuthorMode()
	case " ":
		// Select the current author and reload commits
		if err := cp.selectAuthor(); err != nil {
			// Handle error, but for now just exit author mode
			cp.exitAuthorMode()
		}
	case "down", "j":
		// Navigate down in author list
		visibleAuthors := cp.getVisibleAuthors()
		if cp.authorIndex < len(visibleAuthors)-1 {
			cp.authorIndex++
		}
	case "up", "k":
		// Navigate up in author list
		if cp.authorIndex > 0 {
			cp.authorIndex--
		}
	case "pagedown", "ctrl+f":
		// Jump down by page in author list
		visibleAuthors := cp.getVisibleAuthors()
		maxIndex := len(visibleAuthors) - 1
		cp.authorIndex += 10
		if cp.authorIndex > maxIndex {
			cp.authorIndex = maxIndex
		}
	case "pageup", "ctrl+b":
		// Jump up by page in author list
		cp.authorIndex -= 10
		if cp.authorIndex < 0 {
			cp.authorIndex = 0
		}
	case "/", "f":
		// Enter author search mode
		cp.toggleAuthorSearchMode()
	case "r":
		// Refresh author list
		if err := cp.getAvailableAuthors(); err != nil {
			// Handle error - keep current list
		}
	}
	return cp, nil
}

// handleAuthorSearchInput handles keyboard input when in author search mode
func (cp *CherryPicker) handleAuthorSearchInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Handle special keys first (control keys that shouldn't be added to search)
	switch msg.Type {
	case tea.KeyCtrlC:
		cp.quitting = true
		return cp, tea.Batch(tea.ExitAltScreen, tea.Quit)
	case tea.KeyEsc:
		// Exit search mode
		cp.toggleAuthorSearchMode()
		return cp, nil
	case tea.KeyEnter:
		// Exit search mode and keep current filter
		cp.authorSearchMode = false
		if len(cp.filteredAuthors) == 0 {
			// If no results, reset to show all authors
			cp.filteredAuthors = nil
		}
		return cp, nil
	case tea.KeyBackspace:
		// Remove last character from search query
		if len(cp.authorSearchQuery) > 0 {
			cp.authorSearchQuery = cp.authorSearchQuery[:len(cp.authorSearchQuery)-1]
			cp.updateAuthorSearchResults()
		}
		return cp, nil
	case tea.KeyUp:
		// Navigate up in search results (only arrow keys, not 'k')
		if cp.authorIndex > 0 {
			cp.authorIndex--
		}
		return cp, nil
	case tea.KeyDown:
		// Navigate down in search results (only arrow keys, not 'j')
		visibleAuthors := cp.getVisibleAuthors()
		if cp.authorIndex < len(visibleAuthors)-1 {
			cp.authorIndex++
		}
		return cp, nil
	case tea.KeySpace:
		// Select current author in search mode
		if err := cp.selectAuthor(); err != nil {
			cp.exitAuthorMode()
		}
		return cp, nil
	}
	
	// Handle regular character input - prioritize text input over everything else
	if len(msg.String()) == 1 && msg.String() >= " " && msg.String() <= "~" {
		// Add any printable ASCII character to search query
		cp.authorSearchQuery += msg.String()
		cp.updateAuthorSearchResults()
	}
	
	return cp, nil
}

// renderBranchView renders the branch selection interface
func (cp *CherryPicker) renderBranchView() string {
	var s strings.Builder
	
	switchType := "Target"
	if cp.branchSwitchType == "source" {
		switchType = "Source"
	}
	
	s.WriteString(fmt.Sprintf("🌿 Switch %s Branch\n", switchType))
	s.WriteString("═══════════════════════════════════════════════════════════════════════════════\n\n")
	
	// Show current configuration
	s.WriteString("📋 Current Configuration:\n")
	s.WriteString(fmt.Sprintf("  Source Branch: %s (compare against)\n", cp.config.Git.SourceBranch))
	s.WriteString(fmt.Sprintf("  Target Branch: %s (cherry-pick to)\n", cp.config.Git.TargetBranch))
	s.WriteString(fmt.Sprintf("  Current Branch: %s\n", cp.currentBranch))
	s.WriteString("\n")
	
	if len(cp.availableBranches) == 0 {
		s.WriteString("❌ No available branches found.\n\n")
		s.WriteString("Press ESC to go back or 'r' to refresh.\n")
		return s.String()
	}
	
	s.WriteString(fmt.Sprintf("🌿 Select New %s Branch:\n", switchType))
	s.WriteString("───────────────────────────────────────────────────────────────────────────────\n")
	
	for i, branch := range cp.availableBranches {
		cursor := "  "
		if i == cp.branchIndex {
			cursor = "→ "
		}
		
		// Highlight current target/source branch
		current := ""
		if cp.branchSwitchType == "target" && branch == cp.config.Git.TargetBranch {
			current = " (current target)"
		} else if cp.branchSwitchType == "source" && branch == cp.config.Git.SourceBranch {
			current = " (current source)"
		}
		
		s.WriteString(fmt.Sprintf("%s%s%s\n", cursor, branch, current))
	}
	
	s.WriteString("───────────────────────────────────────────────────────────────────────────────\n\n")
	
	// Instructions
	s.WriteString("🔧 Controls:\n")
	s.WriteString("• ↑↓/k j = Navigate branches\n")
	s.WriteString("• ENTER = Select branch and reload commits\n")
	s.WriteString("• r = Refresh branch list\n")
	s.WriteString("• ESC = Cancel and go back\n\n")
	
	s.WriteString("💡 Tip: Selecting a new branch will reload the commit list and clear current selections.\n")
	
	return s.String()
}

// renderAuthorView renders the author selection interface
func (cp *CherryPicker) renderAuthorView() string {
	var s strings.Builder

	s.WriteString("📝 Cherry Pick Commits\n\n")
	
	// Show cherry-pick direction and current author filter
	s.WriteString(fmt.Sprintf("🌿 Cherry-picking from %s → %s\n", 
		cp.config.Git.SourceBranch, 
		cp.config.Git.TargetBranch))
	s.WriteString(fmt.Sprintf("👤 Current Author Filter: %s\n\n", cp.selectedAuthor))
	
	// Show search interface if in search mode
	if cp.authorSearchMode {
		s.WriteString("🔍 Search Authors: " + cp.authorSearchQuery + "█\n")
		s.WriteString("(ESC=exit search, ENTER=keep filter, ↑↓=navigate, SPACE=select)\n\n")
		if len(cp.filteredAuthors) == 0 && cp.authorSearchQuery != "" {
			s.WriteString("No authors match your search.\n")
			return s.String()
		}
	} 

	// Get authors to display (filtered or all)
	visibleAuthors := cp.getVisibleAuthors()
	if len(visibleAuthors) == 0 {
		s.WriteString("No authors found.\n")
		s.WriteString("\n")
		s.WriteString("Status: No authors available\n")
		s.WriteString("Controls: ESC=go back, r=refresh, q=quit\n")
		return s.String()
	}

	// Pagination settings for authors
	maxAuthorsPerPage := 10
	startIndex := 0
	endIndex := len(visibleAuthors)
	
	// Calculate pagination based on current cursor position
	if len(visibleAuthors) > maxAuthorsPerPage {
		page := cp.authorIndex / maxAuthorsPerPage
		startIndex = page * maxAuthorsPerPage
		endIndex = startIndex + maxAuthorsPerPage
		if endIndex > len(visibleAuthors) {
			endIndex = len(visibleAuthors)
		}
	}
	
	// Show title with pagination info
	if cp.authorSearchMode && cp.authorSearchQuery != "" {
		s.WriteString(fmt.Sprintf("Filtered authors (%d results):\n", len(visibleAuthors)))
	} else if len(visibleAuthors) > maxAuthorsPerPage {
		currentPage := (cp.authorIndex / maxAuthorsPerPage) + 1
		totalPages := (len(visibleAuthors) + maxAuthorsPerPage - 1) / maxAuthorsPerPage
		s.WriteString(fmt.Sprintf("Available authors (Page %d of %d, showing %d-%d of %d):\n", 
			currentPage, totalPages, startIndex+1, endIndex, len(visibleAuthors)))
	} else {
		s.WriteString("Available authors:\n")
	}
	
	// Display authors for current page
	for i := startIndex; i < endIndex; i++ {
		author := visibleAuthors[i]
		cursor := "  "
		authorText := author
		
		// Mark current selected author
		if author == cp.selectedAuthor {
			authorText = "✓ " + author + " (current)"
		}

		// Highlight current cursor position with background
		if i == cp.authorIndex {
			cursor = "→ "
			// Add background highlighting to the entire line for visibility
			authorText = "\033[7m" + authorText + "\033[0m"
		}

		s.WriteString(fmt.Sprintf("%s%s\n", cursor, authorText))
	}

	s.WriteString("\n")
	s.WriteString(fmt.Sprintf("Selected author: %s\n", cp.selectedAuthor))
	s.WriteString("\n")
	s.WriteString("Status: Ready\n")
	
	// Show appropriate controls based on pagination
	if len(cp.availableAuthors) > maxAuthorsPerPage {
		s.WriteString("Controls: ↑↓/j k=navigate, PgUp/PgDn=page, SPACE=select, /=search, ESC=cancel, q=quit\n")
	} else {
		s.WriteString("Controls: ↑↓/j k=navigate, SPACE=select, /=search, ESC=cancel, q=quit\n")
	}

	return s.String()
}

// getStatusLine returns current status information
func (cp *CherryPicker) getStatusLine() string {
	var status []string
	
	if cp.searchMode {
		status = append(status, "🔍 Search Mode")
	}
	
	if cp.previewMode {
		status = append(status, "📖 Preview Mode")
	}
	
	if cp.branchMode {
		status = append(status, fmt.Sprintf("🌿 Branch Selection (%s)", cp.branchSwitchType))
	}
	
	if cp.rangeSelection {
		status = append(status, "📍 Range Selection Mode")
	}
	
	if cp.detailView {
		status = append(status, "🔍 Detail View")
	}
	
	if cp.conflictMode {
		conflictCount := len(cp.conflictFiles)
		if conflictCount > 0 {
			status = append(status, fmt.Sprintf("⚠️  %d conflicts in %s", conflictCount, cp.conflictCommit[:8]))
		} else {
			status = append(status, fmt.Sprintf("⚠️  Conflict resolution for %s", cp.conflictCommit[:8]))
		}
	}
	
	// Count merge commits and already applied commits
	mergeCount := 0
	appliedCount := 0
	for _, commit := range cp.commits {
		if commit.IsMerge {
			mergeCount++
		}
		if commit.AlreadyApplied {
			appliedCount++
		}
	}
	if mergeCount > 0 {
		status = append(status, fmt.Sprintf("🔀 %d merge commits", mergeCount))
	}
	if appliedCount > 0 {
		if cp.hideApplied {
			status = append(status, fmt.Sprintf("👁️ %d applied commits hidden", appliedCount))
		} else {
			status = append(status, fmt.Sprintf("✗ %d already applied", appliedCount))
		}
	}
	
	// Show current sort order
	if cp.reverse {
		status = append(status, "📅 Reverse chronological")
	} else {
		status = append(status, "📅 Chronological")
	}
	
	if len(status) == 0 {
		return "Status: Ready"
	}
	
	return "Status: " + strings.Join(status, " | ")
}

// getControlsDisplay returns help text for available controls
func (cp *CherryPicker) getControlsDisplay() string {
	var controls []string
	
	if cp.searchMode {
		// Search mode controls
		controls = append(controls, "type=search")
		controls = append(controls, "ESC=exit search")
		controls = append(controls, "ENTER=keep filter")
		controls = append(controls, "↑↓=navigate")
		controls = append(controls, "TAB=toggle")
		controls = append(controls, "BACKSPACE=delete")
	} else {
		// Normal mode controls
		// Navigation & Selection
		controls = append(controls, "↑↓/k j=navigate")
		controls = append(controls, "PgUp/PgDn=page")
		controls = append(controls, "ENTER/SPACE=toggle")
		controls = append(controls, "r=range select")
		controls = append(controls, "a=select all")
		controls = append(controls, "c=clear all")
		
		// Search & View Options
		controls = append(controls, "/f=SEARCH")
		controls = append(controls, "p/TAB=PREVIEW")
		controls = append(controls, "b=TARGET BRANCH")
		controls = append(controls, "B=SOURCE BRANCH")
		controls = append(controls, "A=AUTHOR")
		controls = append(controls, "d=detail view")
		controls = append(controls, "H=HIDE APPLIED")
		controls = append(controls, "R=REVERSE ORDER")
		
		// Actions
		controls = append(controls, "e/x=execute cherry-pick")
		controls = append(controls, "i=interactive rebase")
		controls = append(controls, "q=quit")
	}
	
	return "Controls: " + strings.Join(controls, ", ")
}

func (cp *CherryPicker) getSelectedCommitsDisplay() string {
	selectedCommits := cp.getSelectedCommits()

	if len(selectedCommits) == 0 {
		return "Selected commits: (none)"
	}

	var s strings.Builder
	s.WriteString(fmt.Sprintf("Selected commits (%d):\n", len(selectedCommits)))
	for _, commit := range selectedCommits {
		s.WriteString(fmt.Sprintf("  ✓ %s\n", commit.Full))
	}

	return s.String()
}

// handleEditorInput handles keyboard input when in editor selection mode
func (cp *CherryPicker) handleEditorInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c", "q":
		cp.quitting = true
		return cp, tea.Batch(tea.ExitAltScreen, tea.Quit)
	case "esc":
		// Exit editor mode, back to conflict mode
		cp.exitEditorMode()
	case "enter", " ":
		// Select the current editor and open it
		if err := cp.selectEditor(); err != nil {
			// Handle error, but for now just exit editor mode
			cp.exitEditorMode()
		}
	case "down", "j":
		// Navigate down in editor list
		if cp.editorIndex < len(cp.availableEditors)-1 {
			cp.editorIndex++
		}
	case "up", "k":
		// Navigate up in editor list
		if cp.editorIndex > 0 {
			cp.editorIndex--
		}
	}
	return cp, nil
}

// renderEditorView renders the editor selection interface
func (cp *CherryPicker) renderEditorView() string {
	var s strings.Builder
	
	s.WriteString("✏️  Choose Editor for Conflict Resolution\n")
	s.WriteString("═══════════════════════════════════════════════════════════════════════════════\n\n")
	
	if len(cp.availableEditors) == 0 {
		s.WriteString("❌ No editors found on your system.\n\n")
		s.WriteString("Press ESC to go back to conflict resolution options.")
		return s.String()
	}
	
	s.WriteString("Available editors on your system:\n\n")
	
	for i, editor := range cp.availableEditors {
		cursor := "  "
		if i == cp.editorIndex {
			cursor = "→ "
			s.WriteString(fmt.Sprintf("\033[7m%s%d. %s\033[0m\n", cursor, i+1, editor.Description))
		} else {
			s.WriteString(fmt.Sprintf("%s%d. %s\n", cursor, i+1, editor.Description))
		}
		
		// Add additional info for different editor types
		if editor.Simple {
			s.WriteString("     Opens files with conflict markers (<<<<<<< ======= >>>>>>>)\n")
		} else if editor.Name == "mergetool" {
			s.WriteString("     Uses your configured git merge tool\n")
		} else {
			s.WriteString("     Advanced editor with conflict resolution features\n")
		}
		s.WriteString("\n")
	}
	
	s.WriteString("Controls:\n")
	s.WriteString("• ↑↓ or j/k = Navigate editors\n")
	s.WriteString("• Enter/Space = Select editor\n")
	s.WriteString("• ESC = Back to conflict options\n")
	s.WriteString("• q = Quit\n")
	
	return s.String()
}