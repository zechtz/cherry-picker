package main

import (
	"fmt"
	"os/exec"
	"strings"
)

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

	// Reverse the commits if the reverse flag is set
	if cp.reverse {
		for i, j := 0, len(cp.commits)-1; i < j; i, j = i+1, j-1 {
			cp.commits[i], cp.commits[j] = cp.commits[j], cp.commits[i]
		}
	}

	return nil
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