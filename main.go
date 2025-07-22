package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strings"
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
}

func main() {
	cp := &CherryPicker{
		selected: make(map[string]bool),
	}

	if err := cp.run(); err != nil {
		fmt.Printf("❌ Error: %v\n", err)
		os.Exit(1)
	}
}

func (cp *CherryPicker) run() error {
	if err := cp.validateBranch(); err != nil {
		return err
	}

	if err := cp.fetchOrigin(); err != nil {
		return err
	}

	if err := cp.getUniqueCommits(); err != nil {
		return err
	}

	if len(cp.commits) == 0 {
		fmt.Println("✅ No unique commits found. Your branch is fully merged into dev.")
		return nil
	}

	cp.displayCommits()
	cp.interactiveSelection()

	selectedSHAs := cp.getSelectedSHAs()
	if len(selectedSHAs) == 0 {
		fmt.Println("No commits selected. Exiting.")
		return nil
	}

	return cp.cherryPick(selectedSHAs)
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

func (cp *CherryPicker) displayCommits() {
	fmt.Println("📝 Unique commits found:")
	for _, commit := range cp.commits {
		fmt.Println(commit.Full)
	}
	fmt.Println()
}

func (cp *CherryPicker) interactiveSelection() {
	fmt.Println("📝 Use arrow keys to navigate, SPACE to select/deselect, ENTER to confirm:")
	fmt.Println()

	for {
		cp.clearScreen()
		cp.displaySelectionUI()

		key := cp.readKey()
		switch key {
		case "up":
			if cp.currentIndex > 0 {
				cp.currentIndex--
			}
		case "down":
			if cp.currentIndex < len(cp.commits)-1 {
				cp.currentIndex++
			}
		case "space":
			sha := cp.commits[cp.currentIndex].SHA
			cp.selected[sha] = !cp.selected[sha]
		case "enter":
			return
		}
	}
}

func (cp *CherryPicker) clearScreen() {
	fmt.Print("\033[2J\033[H")
}

func (cp *CherryPicker) displaySelectionUI() {
	fmt.Printf("🔍 Detecting unique commits in %s that are not in dev...\n", cp.currentBranch)
	fmt.Println()
	fmt.Println("📝 Use arrow keys to navigate, SPACE to select/deselect, ENTER to confirm:")
	fmt.Println()

	for i, commit := range cp.commits {
		prefix := "  "
		checkbox := "[ ]"

		if cp.selected[commit.SHA] {
			checkbox = "[✓]"
		}

		if i == cp.currentIndex {
			prefix = "→ "
		}

		fmt.Printf("%s%s %s\n", prefix, checkbox, commit.Full)
	}

	selectedCount := 0
	for _, selected := range cp.selected {
		if selected {
			selectedCount++
		}
	}

	fmt.Printf("\nSelected: %d commits\n", selectedCount)
}

func (cp *CherryPicker) readKey() string {
	exec.Command("stty", "-echo").Run()
	exec.Command("stty", "cbreak").Run()
	defer exec.Command("stty", "echo").Run()
	defer exec.Command("stty", "-cbreak").Run()

	reader := bufio.NewReader(os.Stdin)
	char, _, _ := reader.ReadRune()

	if char == 27 { // ESC sequence
		char, _, _ = reader.ReadRune()
		if char == 91 { // [
			char, _, _ = reader.ReadRune()
			switch char {
			case 65: // A
				return "up"
			case 66: // B
				return "down"
			}
		}
	} else if char == 32 { // Space
		return "space"
	} else if char == 13 { // Enter
		return "enter"
	}

	return ""
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
