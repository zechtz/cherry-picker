package main

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
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
			sha := parts[0]
			message := parts[1]
			
			// Get detailed commit information
			commit, err := cp.getCommitDetails(sha, message, line)
			if err != nil {
				// Fallback to basic commit info if detailed fetch fails
				commit = Commit{
					SHA:     sha,
					Message: message,
					Full:    line,
				}
			}
			
			// Check if commit already exists in target branch
			commit.AlreadyApplied = cp.isCommitInTargetBranch(sha)
			
			cp.commits = append(cp.commits, commit)
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

// getCommitDetails fetches detailed information about a commit
func (cp *CherryPicker) getCommitDetails(sha, message, full string) (Commit, error) {
	commit := Commit{
		SHA:     sha,
		Message: message,
		Full:    full,
	}

	// Get commit date and author
	output, err := exec.Command("git", "show", "--format=%ai|%an|%P", "--name-only", sha).Output()
	if err != nil {
		return commit, err
	}

	lines := strings.Split(string(output), "\n")
	if len(lines) < 1 {
		return commit, fmt.Errorf("invalid git show output")
	}

	// Parse the format line: date|author|parents
	formatLine := lines[0]
	parts := strings.Split(formatLine, "|")
	if len(parts) >= 3 {
		// Parse date
		if date, err := time.Parse("2006-01-02 15:04:05 -0700", parts[0]); err == nil {
			commit.Date = date
		}
		
		// Parse author
		commit.Author = parts[1]
		
		// Parse parents to detect merge commits
		parents := strings.Fields(parts[2])
		commit.ParentCount = len(parents)
		commit.IsMerge = len(parents) > 1
	}

	// Parse changed files (skip empty lines and the format line)
	for i := 1; i < len(lines); i++ {
		if strings.TrimSpace(lines[i]) != "" {
			commit.FilesChanged = append(commit.FilesChanged, lines[i])
		}
	}

	// Get stats (insertions/deletions)
	statsOutput, err := exec.Command("git", "show", "--stat", "--format=", sha).Output()
	if err == nil {
		commit.Insertions, commit.Deletions = cp.parseGitStats(string(statsOutput))
	}

	return commit, nil
}

// parseGitStats parses git show --stat output to extract insertions and deletions
func (cp *CherryPicker) parseGitStats(statsOutput string) (int, int) {
	lines := strings.Split(statsOutput, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.Contains(line, "insertion") || strings.Contains(line, "deletion") {
			var insertions, deletions int
			
			// Look for patterns like "5 insertions(+), 3 deletions(-)"
			if strings.Contains(line, "insertion") {
				parts := strings.Fields(line)
				for i, part := range parts {
					if strings.Contains(part, "insertion") && i > 0 {
						if val, err := strconv.Atoi(parts[i-1]); err == nil {
							insertions = val
						}
						break
					}
				}
			}
			
			if strings.Contains(line, "deletion") {
				parts := strings.Fields(line)
				for i, part := range parts {
					if strings.Contains(part, "deletion") && i > 0 {
						if val, err := strconv.Atoi(parts[i-1]); err == nil {
							deletions = val
						}
						break
					}
				}
			}
			
			return insertions, deletions
		}
	}
	return 0, 0
}

// cherryPickWithConflictHandling performs cherry-pick with conflict resolution
func (cp *CherryPicker) cherryPickWithConflictHandling(shas []string) error {
	targetBranch := cp.config.Git.TargetBranch
	remote := cp.config.Git.Remote
	
	fmt.Printf("🔀 Switching to %s...\n", targetBranch)
	if err := exec.Command("git", "checkout", targetBranch).Run(); err != nil {
		return fmt.Errorf("failed to checkout %s: %v", targetBranch, err)
	}

	if cp.config.Git.AutoFetch {
		// Check if remote exists before trying to pull
		output, err := exec.Command("git", "remote").Output()
		if err == nil && strings.Contains(strings.TrimSpace(string(output)), remote) {
			// Remote exists, try to pull
			if err := exec.Command("git", "pull", remote, targetBranch).Run(); err != nil {
				fmt.Printf("⚠️  Could not pull from %s, continuing with local branch\n", remote)
			}
		} else {
			fmt.Printf("⚠️  No '%s' remote configured, using local branch only\n", remote)
		}
	}

	fmt.Println("🍒 Cherry-picking selected commits...")
	
	// Cherry-pick commits one by one to handle conflicts individually
	for i, sha := range shas {
		shaDisplay := sha
		if len(sha) > 8 {
			shaDisplay = sha[:8]
		}
		fmt.Printf("Cherry-picking %s (%d/%d)...\n", shaDisplay, i+1, len(shas))
		
		err := exec.Command("git", "cherry-pick", sha).Run()
		if err != nil {
			// Check if it's a conflict
			if cp.hasConflicts() {
				fmt.Printf("⚠️  Conflict detected in commit %s\n", sha)
				cp.conflictMode = true
				cp.conflictCommit = sha
				return fmt.Errorf("conflict in commit %s - resolve conflicts and continue", sha)
			}
			return fmt.Errorf("cherry-pick failed for %s: %v", sha, err)
		}
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

// hasConflicts checks if there are merge conflicts
func (cp *CherryPicker) hasConflicts() bool {
	output, err := exec.Command("git", "status", "--porcelain").Output()
	if err != nil {
		return false
	}
	
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "UU ") || strings.HasPrefix(line, "AA ") {
			return true
		}
	}
	return false
}

// resolveConflicts provides options for conflict resolution
func (cp *CherryPicker) resolveConflicts() error {
	fmt.Println("🔧 Conflict resolution options:")
	fmt.Println("1. Open merge tool: git mergetool")
	fmt.Println("2. Skip this commit: git cherry-pick --skip")
	fmt.Println("3. Abort cherry-pick: git cherry-pick --abort")
	fmt.Println("4. Continue after manual resolution: git cherry-pick --continue")
	
	return nil
}

// interactiveRebase launches interactive rebase for selected commits
func (cp *CherryPicker) interactiveRebase(shas []string) error {
	if len(shas) == 0 {
		return fmt.Errorf("no commits selected for rebase")
	}
	
	// Get the parent of the first commit for rebase
	firstSHA := shas[len(shas)-1] // Oldest commit (assuming reverse chronological order)
	parentOutput, err := exec.Command("git", "rev-parse", firstSHA+"^").Output()
	if err != nil {
		return fmt.Errorf("failed to get parent commit: %v", err)
	}
	
	parentSHA := strings.TrimSpace(string(parentOutput))
	
	fmt.Printf("🔄 Starting interactive rebase from %s...\n", parentSHA[:8])
	fmt.Println("This will open your default editor for rebase instructions.")
	fmt.Println("Available rebase commands:")
	fmt.Println("  pick = use commit")
	fmt.Println("  reword = use commit, but edit the commit message")
	fmt.Println("  edit = use commit, but stop for amending")
	fmt.Println("  squash = use commit, but meld into previous commit")
	fmt.Println("  fixup = like squash, but discard this commit's log message")
	fmt.Println("  drop = remove commit")
	fmt.Println()
	
	// Launch interactive rebase
	cmd := exec.Command("git", "rebase", "-i", parentSHA)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	
	return cmd.Run()
}

// isCommitInTargetBranch checks if a commit already exists in the target branch
// This checks both for exact SHA matches and for cherry-picked commits with same content
func (cp *CherryPicker) isCommitInTargetBranch(sha string) bool {
	targetBranch := cp.config.Git.TargetBranch
	remote := cp.config.Git.Remote
	
	// Try remote/target branch first, then fall back to local target branch
	var remoteBranch, localBranch string
	
	// Check if remote exists
	if output, err := exec.Command("git", "remote").Output(); err == nil {
		remotes := strings.TrimSpace(string(output))
		if strings.Contains(remotes, remote) {
			remoteBranch = remote + "/" + targetBranch
		}
	}
	localBranch = targetBranch
	
	// First try to check against remote target branch
	if remoteBranch != "" {
		if err := exec.Command("git", "rev-parse", "--verify", remoteBranch).Run(); err == nil {
			// Check for exact SHA match (ancestor check)
			cmd := exec.Command("git", "merge-base", "--is-ancestor", sha, remoteBranch)
			if err := cmd.Run(); err == nil {
				return true
			}
			// Check for cherry-picked commit with same content
			if cp.hasEquivalentCommitInBranch(sha, remoteBranch) {
				return true
			}
		}
	}
	
	// Fall back to local target branch
	if err := exec.Command("git", "rev-parse", "--verify", localBranch).Run(); err == nil {
		// Check for exact SHA match (ancestor check)
		cmd := exec.Command("git", "merge-base", "--is-ancestor", sha, localBranch)
		if err := cmd.Run(); err == nil {
			return true
		}
		// Check for cherry-picked commit with same content
		if cp.hasEquivalentCommitInBranch(sha, localBranch) {
			return true
		}
	}
	
	return false
}

// hasEquivalentCommitInBranch checks if a commit with the same patch exists in the target branch
func (cp *CherryPicker) hasEquivalentCommitInBranch(sha, targetBranch string) bool {
	// Get the patch content of the source commit
	sourcePatch, err := exec.Command("git", "show", "--format=", sha).Output()
	if err != nil {
		return false
	}
	
	// Get commit message and author info for additional matching
	sourceInfo, err := exec.Command("git", "show", "--format=%s|%an|%ae", "--name-only", sha).Output()
	if err != nil {
		return false
	}
	
	sourceLines := strings.Split(string(sourceInfo), "\n")
	if len(sourceLines) < 1 {
		return false
	}
	
	sourceMeta := strings.Split(sourceLines[0], "|")
	if len(sourceMeta) < 3 {
		return false
	}
	
	sourceSubject := sourceMeta[0]
	sourceAuthorName := sourceMeta[1]
	sourceAuthorEmail := sourceMeta[2]
	
	// Get all commits in target branch
	targetCommits, err := exec.Command("git", "log", "--format=%H|%s|%an|%ae", targetBranch).Output()
	if err != nil {
		return false
	}
	
	// Check each commit in target branch
	for _, line := range strings.Split(string(targetCommits), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		
		parts := strings.Split(line, "|")
		if len(parts) < 4 {
			continue
		}
		
		targetSHA := parts[0]
		targetSubject := parts[1]
		targetAuthorName := parts[2]
		targetAuthorEmail := parts[3]
		
		// Skip if basic metadata doesn't match
		if targetSubject != sourceSubject || 
		   targetAuthorName != sourceAuthorName || 
		   targetAuthorEmail != sourceAuthorEmail {
			continue
		}
		
		// Get patch content of target commit
		targetPatch, err := exec.Command("git", "show", "--format=", targetSHA).Output()
		if err != nil {
			continue
		}
		
		// Compare patch content (ignoring whitespace differences)
		if cp.patchesAreEquivalent(string(sourcePatch), string(targetPatch)) {
			return true
		}
	}
	
	return false
}

// patchesAreEquivalent compares two git patches to see if they represent the same changes
func (cp *CherryPicker) patchesAreEquivalent(patch1, patch2 string) bool {
	// Normalize patches by removing commit-specific headers and comparing diff content
	patch1Normalized := cp.normalizePatch(patch1)
	patch2Normalized := cp.normalizePatch(patch2)
	
	return patch1Normalized == patch2Normalized
}

// normalizePatch removes commit-specific information and normalizes diff content
func (cp *CherryPicker) normalizePatch(patch string) string {
	lines := strings.Split(patch, "\n")
	var normalizedLines []string
	
	for _, line := range lines {
		// Skip diff headers that contain commit-specific info
		if strings.HasPrefix(line, "diff --git") ||
		   strings.HasPrefix(line, "index ") ||
		   strings.HasPrefix(line, "--- a/") ||
		   strings.HasPrefix(line, "+++ b/") {
			continue
		}
		
		// Keep actual diff content (additions, deletions, context)
		if strings.HasPrefix(line, "+") || 
		   strings.HasPrefix(line, "-") || 
		   strings.HasPrefix(line, " ") ||
		   strings.HasPrefix(line, "@@") {
			normalizedLines = append(normalizedLines, line)
		}
	}
	
	return strings.Join(normalizedLines, "\n")
}

func (cp *CherryPicker) cherryPick(shas []string) error {
	targetBranch := cp.config.Git.TargetBranch
	remote := cp.config.Git.Remote
	
	fmt.Printf("🔀 Switching to %s...\n", targetBranch)
	if err := exec.Command("git", "checkout", targetBranch).Run(); err != nil {
		return fmt.Errorf("failed to checkout %s: %v", targetBranch, err)
	}

	if cp.config.Git.AutoFetch {
		// Check if remote exists before trying to pull
		output, err := exec.Command("git", "remote").Output()
		if err == nil && strings.Contains(strings.TrimSpace(string(output)), remote) {
			// Remote exists, try to pull
			if err := exec.Command("git", "pull", remote, targetBranch).Run(); err != nil {
				fmt.Printf("⚠️  Could not pull from %s, continuing with local branch\n", remote)
			}
		} else {
			fmt.Printf("⚠️  No '%s' remote configured, using local branch only\n", remote)
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