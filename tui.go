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
			return cp.handleConflictInput(msg)
		}
		
		// Handle branch mode input differently
		if cp.branchMode {
			return cp.handleBranchInput(msg)
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
			if cp.currentIndex < maxIndex {
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
		case "r":
			// Toggle range selection mode
			cp.toggleRangeSelection()
		case "R":
			// Toggle reverse commit order
			cp.toggleCommitOrder()
		case "d":
			// Toggle detail view
			cp.detailView = !cp.detailView
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
		return cp.renderConflictView()
	}
	
	if cp.branchMode {
		return cp.renderBranchView()
	}

	var s strings.Builder

	s.WriteString("📝 Cherry Pick Commits\n\n")
	
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
	
	for i, commit := range visibleCommits {
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

		if i == cp.currentIndex {
			cursor = "→ "
			// Add blinking cursor inside the checkbox
			if cp.cursorBlink {
				if commit.AlreadyApplied {
					checkbox = "[✗]" // No blinking for already applied
				} else if cp.selected[commit.SHA] {
					checkbox = "[█]"
				} else {
					checkbox = "[█]"
				}
			}
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
	case "1", "2", "3", "4", "5", "6", "7", "8", "9":
		// Select file by number for resolution
		fileIndex := int(msg.String()[0] - '1')
		if fileIndex < len(cp.conflictFiles) {
			cp.showFileResolutionOptions(fileIndex)
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
	s.WriteString("• Press number (1-9) to resolve specific file with 'ours' strategy\n")
	s.WriteString("• c = Continue cherry-pick (if all resolved)\n")
	s.WriteString("• a = Abort cherry-pick\n")
	s.WriteString("• s = Skip this commit\n")
	s.WriteString("• r = Refresh conflict status\n")
	s.WriteString("• ESC = Exit conflict mode\n\n")
	
	s.WriteString("💡 Tip: Resolve conflicts manually in your editor, then press 'r' to refresh\n")
	s.WriteString("    and 'c' to continue when all conflicts are resolved.\n")
	
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
		status = append(status, fmt.Sprintf("✗ %d already applied", appliedCount))
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
		controls = append(controls, "ENTER/SPACE=toggle")
		controls = append(controls, "r=range select")
		controls = append(controls, "a=select all")
		controls = append(controls, "c=clear all")
		
		// Search & View Options
		controls = append(controls, "/f=SEARCH")
		controls = append(controls, "p/TAB=PREVIEW")
		controls = append(controls, "b=TARGET BRANCH")
		controls = append(controls, "B=SOURCE BRANCH")
		controls = append(controls, "d=detail view")
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