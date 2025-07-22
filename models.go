package main

import "time"

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
	reverse       bool
	config        *Config
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