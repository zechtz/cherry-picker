package main

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Config represents the application configuration
type Config struct {
	// Git configuration
	Git GitConfig `yaml:"git"`

	// UI configuration
	UI UIConfig `yaml:"ui"`

	// Behavior configuration
	Behavior BehaviorConfig `yaml:"behavior"`
}

// GitConfig contains git-related configuration
type GitConfig struct {
	// Target branch for cherry-picking (default: "clean-staging")
	TargetBranch string `yaml:"target_branch"`

	// Source branch to compare against (default: "dev")
	SourceBranch string `yaml:"source_branch"`

	// Remote name (default: "origin")
	Remote string `yaml:"remote"`

	// Whether to fetch remote before operations (default: true)
	AutoFetch bool `yaml:"auto_fetch"`

	// Branches to exclude from running the tool on
	ExcludedBranches []string `yaml:"excluded_branches"`
}

// UIConfig contains user interface configuration
type UIConfig struct {
	// Cursor blink interval in milliseconds (default: 500)
	CursorBlinkInterval int `yaml:"cursor_blink_interval"`

	// Show commit date in list (default: false)
	ShowCommitDate bool `yaml:"show_commit_date"`

	// Show commit author in list (default: false)
	ShowCommitAuthor bool `yaml:"show_commit_author"`

	// Maximum commit message length to display (default: 80)
	MaxCommitMessageLength int `yaml:"max_commit_message_length"`
}

// BehaviorConfig contains behavior-related configuration
type BehaviorConfig struct {
	// Default to reverse order (default: false)
	DefaultReverse bool `yaml:"default_reverse"`

	// Require confirmation before cherry-picking (default: true)
	ConfirmBeforeAction bool `yaml:"confirm_before_action"`

	// Auto-push after successful cherry-pick (default: false)
	AutoPush bool `yaml:"auto_push"`

	// Exit after successful cherry-pick (default: true)
	ExitAfterAction bool `yaml:"exit_after_action"`
}

// DefaultConfig returns a configuration with sensible defaults
func DefaultConfig() *Config {
	return &Config{
		Git: GitConfig{
			TargetBranch:     "clean-staging",
			SourceBranch:     "dev",
			Remote:           "origin",
			AutoFetch:        true,
			ExcludedBranches: []string{"dev", "staging", "live", "main", "master"},
		},
		UI: UIConfig{
			CursorBlinkInterval:    500,
			ShowCommitDate:         false,
			ShowCommitAuthor:       false,
			MaxCommitMessageLength: 80,
		},
		Behavior: BehaviorConfig{
			DefaultReverse:      false,
			ConfirmBeforeAction: true,
			AutoPush:            false,
			ExitAfterAction:     true,
		},
	}
}

// LoadConfig loads configuration from file, falling back to defaults
func LoadConfig() (*Config, error) {
	config := DefaultConfig()

	configPath := getConfigPath()
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		// Config file doesn't exist, return defaults
		return config, nil
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %v", err)
	}

	if err := yaml.Unmarshal(data, config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %v", err)
	}

	return config, nil
}

// SaveConfig saves configuration to file
func SaveConfig(config *Config) error {
	configPath := getConfigPath()
	
	// Create config directory if it doesn't exist
	configDir := filepath.Dir(configPath)
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %v", err)
	}

	data, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %v", err)
	}

	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %v", err)
	}

	return nil
}

// getConfigPath returns the path to the configuration file
func getConfigPath() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		// Fallback to current directory
		return ".cherry-picker.yaml"
	}
	return filepath.Join(homeDir, ".cherry-picker.yaml")
}

// GenerateDefaultConfigFile creates a default configuration file
func GenerateDefaultConfigFile() error {
	config := DefaultConfig()
	configPath := getConfigPath()

	// Check if config already exists
	if _, err := os.Stat(configPath); err == nil {
		return fmt.Errorf("config file already exists at %s", configPath)
	}

	if err := SaveConfig(config); err != nil {
		return err
	}

	fmt.Printf("✅ Default configuration file created at: %s\n", configPath)
	return nil
}