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
	branches        []string
	filteredBranches []string
	cursor          int
	selected        map[string]string // "source" or "target" -> branch name
	currentStep     string            // "source" or "target"
	completed       bool
	cancelled       bool
	loading         bool
	searchMode      bool
	searchQuery     string
}

type branchSelectedMsg struct {
	step   string
	branch string
}

type branchesLoadedMsg struct {
	branches []string
	err      error
}

func NewBranchSelector() *BranchSelector {
	return &BranchSelector{
		selected:    make(map[string]string),
		currentStep: "source",
		loading:     true,
	}
}

func (bs *BranchSelector) Init() tea.Cmd {
	return bs.loadBranches
}

func (bs *BranchSelector) loadBranches() tea.Msg {
	branches, err := getAvailableBranches()
	return branchesLoadedMsg{
		branches: branches,
		err:      err,
	}
}

func (bs *BranchSelector) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case branchesLoadedMsg:
		bs.loading = false
		if msg.err != nil {
			// Handle error case - for now just show empty branches
			bs.branches = []string{}
			bs.filteredBranches = []string{}
		} else {
			bs.branches = msg.branches
			bs.updateFilteredBranches()
		}
		return bs, nil
	case tea.KeyMsg:
		// Handle search mode input
		if bs.searchMode {
			switch msg.String() {
			case "enter":
				// Exit search mode
				bs.searchMode = false
				return bs, nil
			case "esc":
				// Cancel search and clear query
				bs.searchMode = false
				bs.searchQuery = ""
				bs.updateFilteredBranches()
				bs.cursor = 0
				return bs, nil
			case "backspace":
				if len(bs.searchQuery) > 0 {
					bs.searchQuery = bs.searchQuery[:len(bs.searchQuery)-1]
					bs.updateFilteredBranches()
					bs.cursor = 0
				}
				return bs, nil
			default:
				// Add character to search query
				if len(msg.String()) == 1 {
					bs.searchQuery += msg.String()
					bs.updateFilteredBranches()
					bs.cursor = 0
				}
				return bs, nil
			}
		}
		
		// Handle normal navigation
		switch msg.String() {
		case "ctrl+c", "q":
			bs.cancelled = true
			return bs, tea.Quit
		case "/", "f":
			// Enter search mode
			bs.searchMode = true
			return bs, nil
		case "up", "k":
			if bs.cursor > 0 {
				bs.cursor--
			}
		case "down", "j":
			maxIndex := len(bs.filteredBranches) - 1
			if bs.cursor < maxIndex {
				bs.cursor++
			}
		case "enter", " ":
			if bs.cursor < len(bs.filteredBranches) {
				selectedBranch := bs.filteredBranches[bs.cursor]
				bs.selected[bs.currentStep] = selectedBranch
				
				if bs.currentStep == "source" {
					bs.currentStep = "target"
					bs.cursor = 0 // Reset cursor for target selection
					bs.searchMode = false // Reset search mode
					bs.searchQuery = ""
					bs.updateFilteredBranches()
				} else {
					bs.completed = true
					return bs, tea.Quit
				}
			}
		}
	}
	return bs, nil
}

// updateFilteredBranches filters branches based on search query
func (bs *BranchSelector) updateFilteredBranches() {
	if bs.searchQuery == "" {
		bs.filteredBranches = bs.branches
		return
	}
	
	query := strings.ToLower(bs.searchQuery)
	bs.filteredBranches = []string{}
	
	for _, branch := range bs.branches {
		if strings.Contains(strings.ToLower(branch), query) {
			bs.filteredBranches = append(bs.filteredBranches, branch)
		}
	}
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
	
	// Show search input if in search mode
	if bs.searchMode {
		searchStyle := lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("226")).
			Background(lipgloss.Color("235"))
		s += searchStyle.Render(fmt.Sprintf("Search: %s█", bs.searchQuery)) + "\n\n"
	} else if bs.searchQuery != "" {
		// Show current search query when not actively searching
		s += lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")).
			Render(fmt.Sprintf("Filtered by: %s (press / to search, Esc to clear)", bs.searchQuery)) + "\n\n"
	} else {
		s += "\n"
	}
	
	if bs.loading {
		s += lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")).
			Render("Loading branches...") + "\n"
	} else if len(bs.filteredBranches) == 0 {
		if bs.searchQuery != "" {
			s += lipgloss.NewStyle().
				Foreground(lipgloss.Color("196")).
				Render(fmt.Sprintf("No branches found matching '%s'", bs.searchQuery)) + "\n"
		} else {
			s += lipgloss.NewStyle().
				Foreground(lipgloss.Color("196")).
				Render("No branches found") + "\n"
		}
	} else {
		for i, branch := range bs.filteredBranches {
			cursor := " "
			if bs.cursor == i {
				cursor = ">"
				s += selectedStyle.Render(fmt.Sprintf("%s %s", cursor, branch)) + "\n"
			} else {
				s += normalStyle.Render(fmt.Sprintf("%s %s", cursor, branch)) + "\n"
			}
		}
	}
	
	s += "\n"
	if bs.searchMode {
		s += lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")).
			Render("Type to search, Enter to exit search, Esc to clear and exit")
	} else {
		s += lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")).
			Render("Use ↑/↓ or j/k to navigate, Enter to select, / to search, q to quit")
	}
	
	return s
}

func getAvailableBranches() ([]string, error) {
	// Get only local branches
	cmd := exec.Command("git", "branch", "--format=%(refname:short)")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get branches: %v", err)
	}
	
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	var branches []string
	
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		branches = append(branches, line)
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