package main

import (
	"fmt"
	"strings"
	"time"
)

type Commit struct {
	SHA           string
	Message       string
	Full          string
	Date          time.Time
	Author        string
	IsMerge       bool
	ParentCount   int
	FilesChanged  []string
	Insertions    int
	Deletions     int
	AlreadyApplied bool
}

type ConflictFile struct {
	Path         string
	Status       string // "UU", "AA", "DD", etc.
	Description  string // Human-readable conflict type
	HasConflicts bool   // Whether file has conflict markers
}

type CherryPicker struct {
	currentBranch     string
	authorName        string
	selectedAuthor    string   // Author to filter commits by (defaults to authorName)
	availableAuthors  []string
	commits           []Commit
	selected          map[string]bool
	currentIndex      int
	quitting          bool
	cursorBlink       bool
	reverse           bool
	config            *Config
	detailView        bool
	rangeSelection    bool
	rangeStart        int
	rangeEnd          int
	conflictMode      bool
	conflictCommit    string
	conflictFiles     []ConflictFile
	conflictResolved  bool
	rebaseRequested   bool
	executeRequested  bool
	searchMode        bool
	searchQuery       string
	filteredCommits   []int  // indices of commits that match search
	previewMode       bool
	previewCommit     *Commit
	previewDiff       string
	previewStats      string
	branchMode        bool
	branchSwitchType  string // "target" or "source"
	availableBranches []string
	authorMode        bool
	authorIndex       int
	branchIndex      int
}

type tickMsg time.Time

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

// toggleRangeSelection toggles range selection mode
func (cp *CherryPicker) toggleRangeSelection() {
	if !cp.rangeSelection {
		// Start range selection
		cp.rangeSelection = true
		cp.rangeStart = cp.currentIndex
		cp.rangeEnd = cp.currentIndex
	} else {
		// End range selection and select all commits in range
		cp.selectRange()
		cp.rangeSelection = false
	}
}

// selectRange selects all commits in the current range
func (cp *CherryPicker) selectRange() {
	start := cp.rangeStart
	end := cp.rangeEnd
	
	// Ensure start <= end
	if start > end {
		start, end = end, start
	}
	
	// Select all commits in range (except already applied ones)
	for i := start; i <= end && i < len(cp.commits); i++ {
		if !cp.commits[i].AlreadyApplied {
			cp.selected[cp.commits[i].SHA] = true
		}
	}
}

// updateRangeEnd updates the end of the range selection
func (cp *CherryPicker) updateRangeEnd() {
	if cp.rangeSelection {
		cp.rangeEnd = cp.currentIndex
	}
}

// isInRange checks if a commit index is within the current range selection
func (cp *CherryPicker) isInRange(index int) bool {
	if !cp.rangeSelection {
		return false
	}
	
	start := cp.rangeStart
	end := cp.rangeEnd
	
	if start > end {
		start, end = end, start
	}
	
	return index >= start && index <= end
}

// toggleCommitOrder reverses the order of commits and adjusts the current index
func (cp *CherryPicker) toggleCommitOrder() {
	// Reverse the commits slice
	for i, j := 0, len(cp.commits)-1; i < j; i, j = i+1, j-1 {
		cp.commits[i], cp.commits[j] = cp.commits[j], cp.commits[i]
	}
	
	// Adjust current index to maintain position relative to the currently selected commit
	cp.currentIndex = len(cp.commits) - 1 - cp.currentIndex
	
	// If we're in range selection mode, we need to adjust those indices too
	if cp.rangeSelection {
		cp.rangeStart = len(cp.commits) - 1 - cp.rangeStart
		cp.rangeEnd = len(cp.commits) - 1 - cp.rangeEnd
		
		// Ensure rangeStart is still <= rangeEnd after reversal
		if cp.rangeStart > cp.rangeEnd {
			cp.rangeStart, cp.rangeEnd = cp.rangeEnd, cp.rangeStart
		}
	}
	
	// Toggle the reverse flag to track current state
	cp.reverse = !cp.reverse
}

// toggleSearchMode enters or exits search mode
func (cp *CherryPicker) toggleSearchMode() {
	if !cp.searchMode {
		// Enter search mode
		cp.searchMode = true
		cp.searchQuery = ""
		cp.updateSearchResults()
	} else {
		// Exit search mode
		cp.searchMode = false
		cp.searchQuery = ""
		cp.filteredCommits = nil
		// Reset cursor to a valid position
		if cp.currentIndex >= len(cp.commits) {
			cp.currentIndex = len(cp.commits) - 1
		}
		if cp.currentIndex < 0 {
			cp.currentIndex = 0
		}
	}
}

// updateSearchResults filters commits based on search query
func (cp *CherryPicker) updateSearchResults() {
	cp.filteredCommits = nil
	
	if cp.searchQuery == "" {
		// Empty search shows all commits
		for i := range cp.commits {
			cp.filteredCommits = append(cp.filteredCommits, i)
		}
	} else {
		// Filter commits based on fuzzy matching
		query := strings.ToLower(cp.searchQuery)
		for i, commit := range cp.commits {
			if cp.commitMatchesSearch(commit, query) {
				cp.filteredCommits = append(cp.filteredCommits, i)
			}
		}
	}
	
	// Reset cursor to first filtered result
	cp.currentIndex = 0
}

// commitMatchesSearch checks if a commit matches the search query
func (cp *CherryPicker) commitMatchesSearch(commit Commit, query string) bool {
	// Search in commit message
	if strings.Contains(strings.ToLower(commit.Message), query) {
		return true
	}
	
	// Search in full commit line
	if strings.Contains(strings.ToLower(commit.Full), query) {
		return true
	}
	
	// Search in SHA
	if strings.Contains(strings.ToLower(commit.SHA), query) {
		return true
	}
	
	// Search in author name
	if strings.Contains(strings.ToLower(commit.Author), query) {
		return true
	}
	
	// Search in changed files
	for _, file := range commit.FilesChanged {
		if strings.Contains(strings.ToLower(file), query) {
			return true
		}
	}
	
	return false
}

// getVisibleCommits returns the commits that should be displayed (filtered or all)
func (cp *CherryPicker) getVisibleCommits() []Commit {
	if !cp.searchMode || len(cp.filteredCommits) == 0 {
		return cp.commits
	}
	
	var visible []Commit
	for _, index := range cp.filteredCommits {
		if index < len(cp.commits) {
			visible = append(visible, cp.commits[index])
		}
	}
	return visible
}

// getCurrentCommit returns the currently selected commit (accounting for search filter)
func (cp *CherryPicker) getCurrentCommit() *Commit {
	if cp.searchMode && len(cp.filteredCommits) > 0 {
		if cp.currentIndex < len(cp.filteredCommits) {
			realIndex := cp.filteredCommits[cp.currentIndex]
			if realIndex < len(cp.commits) {
				return &cp.commits[realIndex]
			}
		}
	} else {
		if cp.currentIndex < len(cp.commits) {
			return &cp.commits[cp.currentIndex]
		}
	}
	return nil
}

// getMaxIndex returns the maximum valid index for navigation
func (cp *CherryPicker) getMaxIndex() int {
	if cp.searchMode {
		return len(cp.filteredCommits) - 1
	}
	return len(cp.commits) - 1
}

// togglePreviewMode enters or exits preview mode for the current commit
func (cp *CherryPicker) togglePreviewMode() {
	if !cp.previewMode {
		// Enter preview mode
		cp.previewMode = true
		commit := cp.getCurrentCommit()
		if commit != nil {
			cp.loadPreviewData(commit)
		}
	} else {
		// Exit preview mode
		cp.previewMode = false
		cp.previewCommit = nil
		cp.previewDiff = ""
		cp.previewStats = ""
	}
}

// loadPreviewData fetches detailed information for the given commit
func (cp *CherryPicker) loadPreviewData(commit *Commit) {
	cp.previewCommit = commit
	
	// Get the full diff
	if diff, err := cp.getCommitDiff(commit.SHA); err == nil {
		cp.previewDiff = diff
	} else {
		cp.previewDiff = "Error loading diff: " + err.Error()
	}
	
	// Get detailed stats
	if stats, err := cp.getCommitStats(commit.SHA); err == nil {
		cp.previewStats = stats
	} else {
		cp.previewStats = "Error loading stats: " + err.Error()
	}
}

// updatePreview updates the preview when cursor moves to a different commit
func (cp *CherryPicker) updatePreview() {
	if cp.previewMode {
		commit := cp.getCurrentCommit()
		if commit != nil && (cp.previewCommit == nil || cp.previewCommit.SHA != commit.SHA) {
			cp.loadPreviewData(commit)
		}
	}
}

// enterConflictMode sets up conflict resolution state
func (cp *CherryPicker) enterConflictMode(commit string) {
	cp.conflictMode = true
	cp.conflictCommit = commit
	cp.conflictResolved = false
	cp.loadConflictFiles()
}

// exitConflictMode clears conflict resolution state
func (cp *CherryPicker) exitConflictMode() {
	cp.conflictMode = false
	cp.conflictCommit = ""
	cp.conflictFiles = nil
	cp.conflictResolved = false
}

// loadConflictFiles detects and loads information about conflicted files
func (cp *CherryPicker) loadConflictFiles() {
	cp.conflictFiles = nil
	
	// This will be implemented in git.go
	if conflicts, err := cp.getConflictedFiles(); err == nil {
		cp.conflictFiles = conflicts
	}
}

// toggleConflictResolution toggles the conflict resolution interface
func (cp *CherryPicker) toggleConflictResolution() {
	if cp.conflictMode {
		cp.exitConflictMode()
	}
}

// enterBranchMode enters branch selection mode
func (cp *CherryPicker) enterBranchMode(switchType string) {
	cp.branchMode = true
	cp.branchSwitchType = switchType
	cp.branchIndex = 0
	cp.loadAvailableBranches()
}

// exitBranchMode exits branch selection mode
func (cp *CherryPicker) exitBranchMode() {
	cp.branchMode = false
	cp.branchSwitchType = ""
	cp.availableBranches = nil
	cp.branchIndex = 0
}

// loadAvailableBranches loads the list of available branches
func (cp *CherryPicker) loadAvailableBranches() {
	cp.availableBranches = nil
	
	// This will be implemented in git.go
	if branches, err := cp.getAvailableBranches(); err == nil {
		cp.availableBranches = branches
		
		// Try to select current target/source branch as default
		currentBranch := ""
		if cp.branchSwitchType == "target" {
			currentBranch = cp.config.Git.TargetBranch
		} else {
			currentBranch = cp.config.Git.SourceBranch
		}
		
		// Find and select current branch
		for i, branch := range cp.availableBranches {
			if branch == currentBranch {
				cp.branchIndex = i
				break
			}
		}
	}
}

// enterAuthorMode enters author selection mode
func (cp *CherryPicker) enterAuthorMode() {
	cp.authorMode = true
	cp.authorIndex = 0
	
	// Load available authors if not already loaded
	if len(cp.availableAuthors) == 0 {
		if err := cp.getAvailableAuthors(); err != nil {
			// Handle error - for now just use current author
			cp.availableAuthors = []string{cp.authorName}
		}
	}
	
	// Find current selected author index
	for i, author := range cp.availableAuthors {
		if author == cp.selectedAuthor {
			cp.authorIndex = i
			break
		}
	}
}

// exitAuthorMode exits author selection mode
func (cp *CherryPicker) exitAuthorMode() {
	cp.authorMode = false
	cp.authorIndex = 0
}

// selectAuthor applies the selected author and reloads commits
func (cp *CherryPicker) selectAuthor() error {
	if cp.authorIndex >= len(cp.availableAuthors) {
		return fmt.Errorf("invalid author selection")
	}
	
	cp.selectedAuthor = cp.availableAuthors[cp.authorIndex]
	cp.exitAuthorMode()
	
	// Reload commits with new author filter
	return cp.reloadCommits()
}

// selectBranch applies the selected branch and reloads commits
func (cp *CherryPicker) selectBranch() error {
	if cp.branchIndex >= len(cp.availableBranches) {
		return fmt.Errorf("invalid branch selection")
	}
	
	selectedBranch := cp.availableBranches[cp.branchIndex]
	
	// Update configuration
	if cp.branchSwitchType == "target" {
		cp.config.Git.TargetBranch = selectedBranch
	} else {
		cp.config.Git.SourceBranch = selectedBranch
	}
	
	// Exit branch mode
	cp.exitBranchMode()
	
	// Reload commits with new branch configuration
	return cp.reloadCommits()
}

// reloadCommits reloads the commit list with current configuration
func (cp *CherryPicker) reloadCommits() error {
	// Clear current state
	cp.commits = nil
	cp.selected = make(map[string]bool)
	cp.currentIndex = 0
	cp.filteredCommits = nil
	cp.searchQuery = ""
	cp.searchMode = false
	cp.previewMode = false
	cp.previewCommit = nil
	
	// Reload commits using existing logic
	if err := cp.fetchOrigin(); err != nil {
		return err
	}
	
	return cp.getUniqueCommits()
}