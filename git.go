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

	// Check if current branch is in excluded branches
	for _, excluded := range cp.config.Git.ExcludedBranches {
		if cp.currentBranch == excluded {
			return fmt.Errorf("don't run this script on %s directly", excluded)
		}
	}

	output, err = exec.Command("git", "config", "user.name").Output()
	if err != nil {
		return fmt.Errorf("could not get git user name")
	}
	cp.authorName = strings.TrimSpace(string(output))

	return nil
}

func (cp *CherryPicker) fetchOrigin() error {
	fmt.Printf("🔍 Detecting unique commits in %s that are not in %s...\n", cp.currentBranch, cp.config.Git.SourceBranch)

	// Skip fetch if auto-fetch is disabled
	if !cp.config.Git.AutoFetch {
		return nil
	}

	// Check if remote exists
	output, err := exec.Command("git", "remote").Output()
	if err != nil {
		fmt.Println("⚠️  No git remotes configured, working with local branches only")
		return nil
	}

	remotes := strings.TrimSpace(string(output))
	if !strings.Contains(remotes, cp.config.Git.Remote) {
		fmt.Printf("⚠️  No '%s' remote configured, working with local branches only\n", cp.config.Git.Remote)
		return nil
	}

	// Try to fetch, but don't fail if it doesn't work
	if err := exec.Command("git", "fetch", cp.config.Git.Remote).Run(); err != nil {
		fmt.Printf("⚠️  Could not fetch from %s, working with local branches only\n", cp.config.Git.Remote)
	}

	return nil
}

func (cp *CherryPicker) getUniqueCommits() error {
	// Try remote/source branch first, then fall back to local source branch
	var cmd *exec.Cmd
	
	remoteBranch := cp.config.Git.Remote + "/" + cp.config.Git.SourceBranch
	localBranch := cp.config.Git.SourceBranch

	// Check if remote/source branch exists
	if err := exec.Command("git", "rev-parse", "--verify", remoteBranch).Run(); err == nil {
		cmd = exec.Command("git", "log", remoteBranch+"..HEAD", "--author="+cp.authorName, "--oneline")
	} else if err := exec.Command("git", "rev-parse", "--verify", localBranch).Run(); err == nil {
		cmd = exec.Command("git", "log", localBranch+"..HEAD", "--author="+cp.authorName, "--oneline")
	} else {
		// No source branch exists, show all commits by author
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
	targetBranch := cp.config.Git.TargetBranch
	remote := cp.config.Git.Remote
	
	fmt.Printf("🔀 Switching to %s...\n", targetBranch)
	if err := exec.Command("git", "checkout", targetBranch).Run(); err != nil {
		return fmt.Errorf("failed to checkout %s: %v", targetBranch, err)
	}

	if cp.config.Git.AutoFetch {
		if err := exec.Command("git", "pull", remote, targetBranch).Run(); err != nil {
			return fmt.Errorf("failed to pull %s: %v", targetBranch, err)
		}
	}

	fmt.Println("🍒 Cherry-picking selected commits...")
	args := append([]string{"cherry-pick"}, shas...)
	if err := exec.Command("git", args...).Run(); err != nil {
		return fmt.Errorf("cherry-pick failed: %v", err)
	}

	fmt.Println("✅ Cherry-pick successful.")
	
	if cp.config.Behavior.AutoPush {
		fmt.Printf("🚀 Pushing to %s...\n", remote)
		if err := exec.Command("git", "push", remote, targetBranch).Run(); err != nil {
			return fmt.Errorf("failed to push: %v", err)
		}
		fmt.Println("✅ Pushed successfully.")
	} else {
		fmt.Printf("🛑 Cherry-picked to %s but not pushed. Review and push manually.\n", targetBranch)
	}
	
	fmt.Println()
	fmt.Println("📣 Now you can open a merge request when ready.")

	return nil
}