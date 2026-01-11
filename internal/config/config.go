// Package config handles loading and merging configuration files.
package config

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Config represents the watchman configuration.
type Config struct {
	Version   int             `yaml:"version"`
	Rules     RulesConfig     `yaml:"rules"`
	Workspace WorkspaceConfig `yaml:"workspace"`
	Commands  CommandsConfig  `yaml:"commands"`
	Tools     ToolsConfig     `yaml:"tools"`
}

// RulesConfig enables/disables semantic rules.
type RulesConfig struct {
	Workspace   bool `yaml:"workspace"`
	Scope       bool `yaml:"scope"`
	Incremental bool `yaml:"incremental"`
	Invariants  bool `yaml:"invariants"`
	Patterns    bool `yaml:"patterns"`
	Boundaries  bool `yaml:"boundaries"`
}

// WorkspaceConfig controls the workspace confinement rule.
type WorkspaceConfig struct {
	Allow []string `yaml:"allow"`
	Block []string `yaml:"block"`
}

// CommandsConfig controls shell command filtering.
type CommandsConfig struct {
	Block []string `yaml:"block"`
}

// ToolsConfig controls which tools are available.
type ToolsConfig struct {
	Allow []string `yaml:"allow"`
	Block []string `yaml:"block"`
}

// Default returns the default configuration.
func Default() *Config {
	return &Config{
		Version: 1,
		Rules: RulesConfig{
			Workspace: true,
		},
	}
}

// Load loads configuration. If local config exists, it is used exclusively.
// Otherwise, global config is used. No merging occurs.
func Load() (*Config, error) {
	cfg := Default()

	// Check for local config first - if exists, use only local
	localPath := localConfigPath()
	if localPath != "" {
		if _, err := os.Stat(localPath); err == nil {
			if err := cfg.loadFrom(localPath); err != nil {
				return nil, err
			}
			return cfg, nil
		}
	}

	// No local config - use global
	globalPath := globalConfigPath()
	if globalPath != "" {
		if err := cfg.loadFrom(globalPath); err != nil && !os.IsNotExist(err) {
			return nil, err
		}
	}

	return cfg, nil
}

// loadFrom loads and merges a config file into the current config.
func (c *Config) loadFrom(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	var overlay Config
	if err := yaml.Unmarshal(data, &overlay); err != nil {
		return err
	}

	c.merge(&overlay)
	return nil
}

// merge applies overlay config onto the current config.
// Local values override global values.
// Block lists are appended, not replaced.
func (c *Config) merge(overlay *Config) {
	if overlay.Version > 0 {
		c.Version = overlay.Version
	}
	c.Rules = overlay.Rules
	c.Workspace.Allow = appendUnique(c.Workspace.Allow, overlay.Workspace.Allow)
	c.Workspace.Block = appendUnique(c.Workspace.Block, overlay.Workspace.Block)
	c.Commands.Block = appendUnique(c.Commands.Block, overlay.Commands.Block)
	c.Tools.Allow = appendUnique(c.Tools.Allow, overlay.Tools.Allow)
	c.Tools.Block = appendUnique(c.Tools.Block, overlay.Tools.Block)
}

func appendUnique(base, items []string) []string {
	seen := make(map[string]bool)
	for _, s := range base {
		seen[s] = true
	}
	result := base
	for _, s := range items {
		if !seen[s] {
			result = append(result, s)
			seen[s] = true
		}
	}
	return result
}

func globalConfigPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".config", "watchman", "config.yml")
}

// GlobalConfigPath returns the path to the global config file.
func GlobalConfigPath() string {
	return globalConfigPath()
}

func localConfigPath() string {
	cwd, err := os.Getwd()
	if err != nil {
		return ""
	}
	return filepath.Join(cwd, ".watchman.yml")
}
