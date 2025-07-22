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
		switch msg.String() {
		case "ctrl+c", "q":
			cp.quitting = true
			return cp, tea.Batch(tea.ExitAltScreen, tea.Quit)
		case "enter", " ":
			if len(cp.commits) > 0 {
				sha := cp.commits[cp.currentIndex].SHA
				cp.selected[sha] = !cp.selected[sha]
			}
		case "down", "j", "n":
			if cp.currentIndex < len(cp.commits)-1 {
				cp.currentIndex++
				cp.updateRangeEnd()
			}
		case "up", "k", "p":
			if cp.currentIndex > 0 {
				cp.currentIndex--
				cp.updateRangeEnd()
			}
		case "r":
			// Toggle range selection mode
			cp.toggleRangeSelection()
		case "d":
			// Toggle detail view
			cp.detailView = !cp.detailView
		case "a":
			// Select all commits
			for _, commit := range cp.commits {
				cp.selected[commit.SHA] = true
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

func (cp *CherryPicker) View() string {
	if cp.quitting {
		return ""
	}

	var s strings.Builder

	s.WriteString("📝 Cherry Pick Commits\n\n")
	s.WriteString("Available commits:\n")

	for i, commit := range cp.commits {
		cursor := "  "
		checkbox := "[ ]"
		commitText := commit.Full
		
		// Range selection highlighting
		if cp.isInRange(i) {
			cursor = "📍"
		}

		if cp.selected[commit.SHA] {
			checkbox = "[✓]"
			// Add strikethrough to selected commits
			commitText = "\033[9m" + commit.Full + "\033[0m"
		}

		if i == cp.currentIndex {
			cursor = "→ "
			// Add blinking cursor inside the checkbox
			if cp.cursorBlink {
				if cp.selected[commit.SHA] {
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

// getStatusLine returns current status information
func (cp *CherryPicker) getStatusLine() string {
	var status []string
	
	if cp.rangeSelection {
		status = append(status, "📍 Range Selection Mode")
	}
	
	if cp.detailView {
		status = append(status, "🔍 Detail View")
	}
	
	if cp.conflictMode {
		status = append(status, fmt.Sprintf("⚠️  Conflict in %s", cp.conflictCommit[:8]))
	}
	
	// Count merge commits
	mergeCount := 0
	for _, commit := range cp.commits {
		if commit.IsMerge {
			mergeCount++
		}
	}
	if mergeCount > 0 {
		status = append(status, fmt.Sprintf("🔀 %d merge commits", mergeCount))
	}
	
	if len(status) == 0 {
		return "Status: Ready"
	}
	
	return "Status: " + strings.Join(status, " | ")
}

// getControlsDisplay returns help text for available controls
func (cp *CherryPicker) getControlsDisplay() string {
	var controls []string
	
	// Basic controls
	controls = append(controls, "ENTER/SPACE=toggle")
	controls = append(controls, "↑↓/k j=navigate")
	
	// Action controls
	controls = append(controls, "e/x=execute cherry-pick")
	controls = append(controls, "i=interactive rebase")
	
	// Selection controls
	controls = append(controls, "r=range select")
	controls = append(controls, "a=select all")
	controls = append(controls, "c=clear all")
	
	// View controls
	controls = append(controls, "d=detail view")
	
	// System controls
	controls = append(controls, "q=quit without action")
	
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