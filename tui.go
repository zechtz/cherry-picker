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
	return tea.Tick(time.Millisecond*500, func(t time.Time) tea.Msg {
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
			}
		case "up", "k", "p":
			if cp.currentIndex > 0 {
				cp.currentIndex--
			}
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

		s.WriteString(fmt.Sprintf("%s%s %s\n", cursor, checkbox, commitText))
	}

	s.WriteString("\n")
	s.WriteString(cp.getSelectedCommitsDisplay())
	s.WriteString("\nControls: ENTER/SPACE=toggle, ↑↓/k j=navigate, q=quit\n")

	return s.String()
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