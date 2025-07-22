package main

import "time"

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

type CherryPicker struct {
	currentBranch    string
	authorName       string
	commits          []Commit
	selected         map[string]bool
	currentIndex     int
	quitting         bool
	cursorBlink      bool
	reverse          bool
	config           *Config
	detailView       bool
	rangeSelection   bool
	rangeStart       int
	rangeEnd         int
	conflictMode     bool
	conflictCommit   string
	rebaseRequested  bool
	executeRequested bool
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