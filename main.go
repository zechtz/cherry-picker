package main

import (
	"flag"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	var reverse bool
	flag.BoolVar(&reverse, "reverse", false, "display commits in reverse order")
	flag.Parse()

	cp := &CherryPicker{
		selected:    make(map[string]bool),
		cursorBlink: true,
		reverse:     reverse,
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