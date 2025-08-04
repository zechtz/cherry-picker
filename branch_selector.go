package main

import (
	"fmt"
	"os/exec"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// BranchSelector handles interactive branch selection at startup
type BranchSelector struct {
	branches      []string
	cursor        int
	selected      map[string]string // "source" or "target" -> branch name
	currentStep   string            // "source" or "target"
	completed     bool
	cancelled     bool
}

type branchSelectedMsg struct {
	step   string
	branch string
}

func NewBranchSelector() *BranchSelector {
	return &BranchSelector{
		selected:    make(map[string]string),
		currentStep: "source",
	}
}

func (bs *BranchSelector) Init() tea.Cmd {
	return bs.loadBranches
}

func (bs *BranchSelector) loadBranches() tea.Msg {
	branches, err := getAvailableBranches()
	if err != nil {
		return fmt.Errorf("failed to load branches: %v", err)
	}
	
	bs.branches = branches
	return nil
}

func (bs *BranchSelector) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			bs.cancelled = true
			return bs, tea.Quit
		case "up", "k":
			if bs.cursor > 0 {
				bs.cursor--
			}
		case "down", "j":
			if bs.cursor < len(bs.branches)-1 {
				bs.cursor++
			}
		case "enter", " ":
			if bs.cursor < len(bs.branches) {
				selectedBranch := bs.branches[bs.cursor]
				bs.selected[bs.currentStep] = selectedBranch
				
				if bs.currentStep == "source" {
					bs.currentStep = "target"
					bs.cursor = 0 // Reset cursor for target selection
				} else {
					bs.completed = true
					return bs, tea.Quit
				}
			}
		}
	}
	return bs, nil
}

func (bs *BranchSelector) View() string {
	if bs.cancelled {
		return ""
	}
	
	var title string
	var instructions string
	
	if bs.currentStep == "source" {
		title = "🌿 Select Source Branch"
		instructions = "Choose the branch to compare against (commits unique to current branch vs this branch)"
	} else {
		title = "🎯 Select Target Branch"
		instructions = fmt.Sprintf("Choose the destination branch for cherry-picking\nSource: %s", bs.selected["source"])
	}
	
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("205")).
		MarginBottom(1)
		
	instructionStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("241")).
		MarginBottom(2)
	
	selectedStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("212")).
		Background(lipgloss.Color("57"))
		
	normalStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("255"))
	
	s := titleStyle.Render(title) + "\n"
	s += instructionStyle.Render(instructions) + "\n"
	
	for i, branch := range bs.branches {
		cursor := " "
		if bs.cursor == i {
			cursor = ">"
			s += selectedStyle.Render(fmt.Sprintf("%s %s", cursor, branch)) + "\n"
		} else {
			s += normalStyle.Render(fmt.Sprintf("%s %s", cursor, branch)) + "\n"
		}
	}
	
	s += "\n"
	s += lipgloss.NewStyle().
		Foreground(lipgloss.Color("241")).
		Render("Use ↑/↓ or j/k to navigate, Enter to select, q to quit")
	
	return s
}

func getAvailableBranches() ([]string, error) {
	// Get all branches (local and remote)
	cmd := exec.Command("git", "branch", "-a", "--format=%(refname:short)")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get branches: %v", err)
	}
	
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	var branches []string
	seen := make(map[string]bool)
	
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		
		// Skip remote tracking branches that have local equivalents
		if strings.HasPrefix(line, "origin/") {
			localName := strings.TrimPrefix(line, "origin/")
			if localName == "HEAD" {
				continue
			}
			// Only add remote branch if no local equivalent exists
			if !seen[localName] {
				branches = append(branches, localName)
				seen[localName] = true
			}
		} else {
			// Local branch
			if !seen[line] {
				branches = append(branches, line)
				seen[line] = true
			}
		}
	}
	
	return branches, nil
}

// RunBranchSelector runs the interactive branch selection and returns selected branches
func RunBranchSelector() (sourceBranch, targetBranch string, err error) {
	selector := NewBranchSelector()
	
	p := tea.NewProgram(selector)
	if _, err := p.Run(); err != nil {
		return "", "", fmt.Errorf("failed to run branch selector: %v", err)
	}
	
	if selector.cancelled {
		return "", "", fmt.Errorf("branch selection cancelled")
	}
	
	if !selector.completed {
		return "", "", fmt.Errorf("branch selection incomplete")
	}
	
	return selector.selected["source"], selector.selected["target"], nil
}