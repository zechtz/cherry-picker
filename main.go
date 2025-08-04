package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	var reverse bool
	var generateConfig bool
	flag.BoolVar(&reverse, "reverse", false, "display commits in reverse order")
	flag.BoolVar(&generateConfig, "generate-config", false, "generate default configuration file")
	flag.Parse()

	// Handle config generation
	if generateConfig {
		if err := GenerateDefaultConfigFile(); err != nil {
			fmt.Printf("❌ Error generating config: %v\n", err)
			os.Exit(1)
		}
		return
	}

	// Load configuration
	config, err := LoadConfig()
	if err != nil {
		fmt.Printf("❌ Error loading config: %v\n", err)
		os.Exit(1)
	}

	// CLI flags override config defaults
	if reverse {
		config.Behavior.DefaultReverse = true
	}

	// Interactive branch selection at startup
	fmt.Println("🍒 Cherry Picker - Interactive Git Cherry-Pick Tool")
	fmt.Println()
	
	sourceBranch, targetBranch, err := RunBranchSelector()
	if err != nil {
		if strings.Contains(err.Error(), "cancelled") {
			// User chose to quit - exit gracefully without error message
			os.Exit(0)
		}
		fmt.Printf("❌ Error selecting branches: %v\n", err)
		os.Exit(1)
	}
	
	// Override config with selected branches
	config.Git.SourceBranch = sourceBranch
	config.Git.TargetBranch = targetBranch
	
	fmt.Printf("✅ Selected: %s → %s\n", sourceBranch, targetBranch)
	fmt.Println()

	cp := &CherryPicker{
		selected:    make(map[string]bool),
		cursorBlink: true,
		reverse:     config.Behavior.DefaultReverse,
		config:      config,
	}

	if err := cp.setup(); err != nil {
		fmt.Printf("❌ Error: %v\n", err)
		os.Exit(1)
	}

	if len(cp.commits) == 0 {
		fmt.Printf("✅ No commits found. %s is up to date with %s.\n", sourceBranch, targetBranch)
		return
	}

	// Run the TUI
	p := tea.NewProgram(cp, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error running TUI: %v\n", err)
		os.Exit(1)
	}

	// Handle selected commits based on exit reason
	if cp.quitting {
		// User pressed 'q' or 'ctrl+c' - check if they want to execute
		if !cp.executeRequested && !cp.rebaseRequested {
			fmt.Println("Exited without executing. No actions performed.")
			return
		}
	}

	// Execute requested actions
	selectedSHAs := cp.getSelectedSHAs()
	if len(selectedSHAs) == 0 {
		fmt.Println("No commits selected. Exiting.")
		return
	}

	// Check if interactive rebase was requested
	if cp.rebaseRequested {
		fmt.Println("🔄 Starting interactive rebase for selected commits...")
		if err := cp.interactiveRebase(selectedSHAs); err != nil {
			fmt.Printf("❌ Interactive rebase failed: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("✅ Interactive rebase completed.")
		return
	}

	// Execute cherry-pick (either via e/x or old q behavior for backward compatibility)
	if cp.executeRequested || (!cp.quitting) {
		if err := cp.cherryPickWithConflictHandling(selectedSHAs); err != nil {
			if strings.Contains(err.Error(), "conflict") {
				// Handle conflicts gracefully
				fmt.Printf("⚠️  %v\n", err)
				cp.resolveConflicts()
				fmt.Println("\nRun the tool again after resolving conflicts to continue.")
			} else {
				fmt.Printf("❌ Error: %v\n", err)
				os.Exit(1)
			}
		}
	}
}