// Package cli provides CLI command implementations.
package cli

import (
	"fmt"
	"os"
	"path/filepath"
)

// RunInit creates a watchman configuration file.
func RunInit(local bool) error {
	var configPath string
	var configDir string

	if local {
		cwd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("cannot get working directory: %w", err)
		}
		configPath = filepath.Join(cwd, ".watchman.yml")
	} else {
		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("cannot get home directory: %w", err)
		}
		configDir = filepath.Join(home, ".config", "watchman")
		configPath = filepath.Join(configDir, "config.yml")
	}

	if _, err := os.Stat(configPath); err == nil {
		fmt.Printf("Config already exists: %s\n", configPath)
		return nil
	}

	if configDir != "" {
		if err := os.MkdirAll(configDir, 0755); err != nil {
			return fmt.Errorf("cannot create config directory: %w", err)
		}
	}

	if err := os.WriteFile(configPath, []byte(defaultConfig), 0644); err != nil {
		return fmt.Errorf("cannot write config: %w", err)
	}

	fmt.Printf("Created config: %s\n", configPath)
	return nil
}

const defaultConfig = `version: 1

rules:
  workspace: true
  scope: false
  versioning: false
  incremental: false

workspace:
  allow:
    - /tmp/
  block: []

scope:
  allow: []
  block: []

versioning:
  commit:
    max_length: 0
    require_uppercase: false
    no_period: false
    single_line: false
    forbid_colons: false
    prefix_pattern: ""
  branches:
    protected: []
  operations:
    block: []
  workflow: ""
  tool: ""

incremental:
  max_files: 0
  warn_ratio: 0.7

commands:
  block: []

tools:
  allow: []
  block: []
`
