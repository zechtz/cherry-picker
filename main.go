package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

type Commit struct {
	SHA     string
	Message string
	Full    string
}

type CherryPicker struct {
	currentBranch string
	authorName    string
	commits       []Commit
	selected      map[string]bool
	currentIndex  int
	quitting      bool
	cursorBlink   bool
}

type tickMsg time.Time

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

		if cp.selected[commit.SHA] {
			checkbox = "[✓]"
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

		s.WriteString(fmt.Sprintf("%s%s %s\n", cursor, checkbox, commit.Full))
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

func main() {
	cp := &CherryPicker{
		selected:    make(map[string]bool),
		cursorBlink: true,
	}

	if err := cp.setup(); err != nil {
		fmt.Printf("❌ Error: %v\n", err)
		os.Exit(1)
	}

	if len(cp.commits) == 0 {
		fmt.Println("✅ No unique commits found. Your branch is fully merged into dev.")
		return
	}

	// Run the TUI
	p := tea.NewProgram(cp, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error running TUI: %v\n", err)
		os.Exit(1)
	}

	// Handle selected commits
	if !cp.quitting {
		selectedSHAs := cp.getSelectedSHAs()
		if len(selectedSHAs) == 0 {
			fmt.Println("No commits selected. Exiting.")
			return
		}

		if err := cp.cherryPick(selectedSHAs); err != nil {
			fmt.Printf("❌ Error: %v\n", err)
			os.Exit(1)
		}
	}
}

func (cp *CherryPicker) setup() error {
	if err := cp.validateBranch(); err != nil {
		return err
	}

	if err := cp.fetchOrigin(); err != nil {
		return err
	}

	if err := cp.getUniqueCommits(); err != nil {
		return err
	}

	return nil
}

func (cp *CherryPicker) validateBranch() error {
	output, err := exec.Command("git", "branch", "--show-current").Output()
	if err != nil {
		return fmt.Errorf("not on a valid Git branch")
	}

	cp.currentBranch = strings.TrimSpace(string(output))
	if cp.currentBranch == "" {
		return fmt.Errorf("not on a valid Git branch")
	}

	if cp.currentBranch == "dev" || cp.currentBranch == "staging" || cp.currentBranch == "live" {
		return fmt.Errorf("don't run this script on dev/staging/live directly")
	}

	output, err = exec.Command("git", "config", "user.name").Output()
	if err != nil {
		return fmt.Errorf("could not get git user name")
	}
	cp.authorName = strings.TrimSpace(string(output))

	return nil
}

func (cp *CherryPicker) fetchOrigin() error {
	fmt.Printf("🔍 Detecting unique commits in %s that are not in dev...\n", cp.currentBranch)

	// Check if origin remote exists
	output, err := exec.Command("git", "remote").Output()
	if err != nil {
		fmt.Println("⚠️  No git remotes configured, working with local branches only")
		return nil
	}

	remotes := strings.TrimSpace(string(output))
	if !strings.Contains(remotes, "origin") {
		fmt.Println("⚠️  No 'origin' remote configured, working with local branches only")
		return nil
	}

	// Try to fetch, but don't fail if it doesn't work
	if err := exec.Command("git", "fetch", "origin").Run(); err != nil {
		fmt.Println("⚠️  Could not fetch from origin, working with local branches only")
	}

	return nil
}

func (cp *CherryPicker) getUniqueCommits() error {
	// Try origin/dev first, then fall back to dev, then compare with initial commit
	var cmd *exec.Cmd

	// Check if origin/dev exists
	if err := exec.Command("git", "rev-parse", "--verify", "origin/dev").Run(); err == nil {
		cmd = exec.Command("git", "log", "origin/dev..HEAD", "--author="+cp.authorName, "--oneline")
	} else if err := exec.Command("git", "rev-parse", "--verify", "dev").Run(); err == nil {
		cmd = exec.Command("git", "log", "dev..HEAD", "--author="+cp.authorName, "--oneline")
	} else {
		// No dev branch exists, show all commits by author
		cmd = exec.Command("git", "log", "--author="+cp.authorName, "--oneline")
	}

	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to get unique commits: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, " ", 2)
		if len(parts) >= 2 {
			cp.commits = append(cp.commits, Commit{
				SHA:     parts[0],
				Message: parts[1],
				Full:    line,
			})
		}
	}

	return nil
}

func (cp *CherryPicker) getSelectedCommits() []Commit {
	var selected []Commit
	for _, commit := range cp.commits {
		if cp.selected[commit.SHA] {
			selected = append(selected, commit)
		}
	}
	return selected
}

func (cp *CherryPicker) getSelectedSHAs() []string {
	var shas []string
	for _, commit := range cp.commits {
		if cp.selected[commit.SHA] {
			shas = append(shas, commit.SHA)
		}
	}
	return shas
}

func (cp *CherryPicker) cherryPick(shas []string) error {
	fmt.Println("🔀 Switching to clean-staging...")
	if err := exec.Command("git", "checkout", "clean-staging").Run(); err != nil {
		return fmt.Errorf("failed to checkout clean-staging: %v", err)
	}

	if err := exec.Command("git", "pull", "origin", "clean-staging").Run(); err != nil {
		return fmt.Errorf("failed to pull clean-staging: %v", err)
	}

	fmt.Println("🍒 Cherry-picking selected commits...")
	args := append([]string{"cherry-pick"}, shas...)
	if err := exec.Command("git", args...).Run(); err != nil {
		return fmt.Errorf("cherry-pick failed: %v", err)
	}

	fmt.Println("✅ Cherry-pick successful.")
	fmt.Println("🛑 Cherry-picked to clean-staging but not pushed. Review and push manually.")
	fmt.Println()
	fmt.Println("📣 Now you can open a merge request to live when ready.")

	return nil
}
