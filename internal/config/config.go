// Package config handles loading and merging configuration files.
package config

import (
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"
)

// Config represents the watchman configuration.
type Config struct {
	Version     int               `yaml:"version"`
	Rules       RulesConfig       `yaml:"rules"`
	Workspace   WorkspaceConfig   `yaml:"workspace"`
	Scope       ScopeConfig       `yaml:"scope"`
	Versioning  VersioningConfig  `yaml:"versioning"`
	Incremental IncrementalConfig `yaml:"incremental"`
	Invariants  InvariantsConfig  `yaml:"invariants,omitempty"`
	Commands    CommandsConfig    `yaml:"commands"`
	Tools       ToolsConfig       `yaml:"tools"`
	Hooks       []HookConfig      `yaml:"hooks,omitempty"`
}

// RulesConfig enables/disables semantic rules.
type RulesConfig struct {
	Workspace   bool `yaml:"workspace"`
	Scope       bool `yaml:"scope"`
	Versioning  bool `yaml:"versioning"`
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

// ScopeConfig controls which files can be modified.
type ScopeConfig struct {
	Allow []string `yaml:"allow"`
	Block []string `yaml:"block"`
}

// VersioningConfig controls commit and branch rules.
type VersioningConfig struct {
	Commit     CommitConfig     `yaml:"commit"`
	Branches   BranchesConfig   `yaml:"branches"`
	Operations OperationsConfig `yaml:"operations"`
	Workflow   string           `yaml:"workflow"`
	Tool       string           `yaml:"tool"`
}

// CommitConfig controls commit message validation.
type CommitConfig struct {
	MaxLength        int    `yaml:"max_length"`
	MaxFiles         int    `yaml:"max_files"`
	RequireUppercase bool   `yaml:"require_uppercase"`
	NoPeriod         bool   `yaml:"no_period"`
	RequirePeriod    bool   `yaml:"require_period"`
	SingleLine       bool   `yaml:"single_line"`
	ForbidColons     bool   `yaml:"forbid_colons"`
	Conventional     bool   `yaml:"conventional"`
	PrefixPattern    string `yaml:"prefix_pattern"`
}

// OperationsConfig controls blocked git operations.
type OperationsConfig struct {
	Block []string `yaml:"block"`
}

// BranchesConfig controls branch protection.
type BranchesConfig struct {
	Protected []string `yaml:"protected"`
}

// IncrementalConfig controls change size limits.
type IncrementalConfig struct {
	MaxFiles  int     `yaml:"max_files"`
	WarnRatio float64 `yaml:"warn_ratio"`
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

// HookConfig defines an external hook executable.
type HookConfig struct {
	Name    string        `yaml:"name"`
	Command string        `yaml:"command"`
	Args    []string      `yaml:"args,omitempty"`
	Tools   []string      `yaml:"tools"`
	Paths   []string      `yaml:"paths,omitempty"`
	Timeout time.Duration `yaml:"timeout,omitempty"`
	OnError string        `yaml:"on_error,omitempty"`
}

// InvariantsConfig defines declarative structural checks.
type InvariantsConfig struct {
	Coexistence []CoexistenceCheck `yaml:"coexistence,omitempty"`
	Content     []ContentCheck     `yaml:"content,omitempty"`
	Imports     []ImportCheck      `yaml:"imports,omitempty"`
	Naming      []NamingCheck      `yaml:"naming,omitempty"`
	Required    []RequiredCheck    `yaml:"required,omitempty"`
}

// CoexistenceCheck ensures related files exist together.
type CoexistenceCheck struct {
	Name    string `yaml:"name"`
	If      string `yaml:"if"`      // Glob pattern that triggers the check
	Require string `yaml:"require"` // Pattern that must exist (supports ${base}, ${name}, ${ext})
	Message string `yaml:"message,omitempty"`
}

// ContentCheck validates file content against patterns.
type ContentCheck struct {
	Name    string   `yaml:"name"`
	Paths   []string `yaml:"paths"`             // Glob patterns (supports ! for exclusion)
	Require string   `yaml:"require,omitempty"` // Regex that must match
	Forbid  string   `yaml:"forbid,omitempty"`  // Regex that must not match
	Message string   `yaml:"message,omitempty"`
}

// ImportCheck validates import statements (regex-based, not AST).
type ImportCheck struct {
	Name    string   `yaml:"name"`
	Paths   []string `yaml:"paths"`  // Files to check
	Forbid  string   `yaml:"forbid"` // Regex pattern for forbidden imports
	Message string   `yaml:"message,omitempty"`
}

// NamingCheck validates file naming conventions.
type NamingCheck struct {
	Name    string   `yaml:"name"`
	Paths   []string `yaml:"paths"`   // Directories/patterns to check
	Pattern string   `yaml:"pattern"` // Regex pattern filenames must match
	Message string   `yaml:"message,omitempty"`
}

// RequiredCheck ensures certain files exist in directories.
type RequiredCheck struct {
	Name    string `yaml:"name"`
	Dirs    string `yaml:"dirs"`           // Glob for directories to check
	When    string `yaml:"when,omitempty"` // Only check when this pattern exists
	Require string `yaml:"require"`        // File that must exist
	Message string `yaml:"message,omitempty"`
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
	c.Scope.Allow = appendUnique(c.Scope.Allow, overlay.Scope.Allow)
	c.Scope.Block = appendUnique(c.Scope.Block, overlay.Scope.Block)
	c.Versioning = overlay.Versioning
	c.Versioning.Branches.Protected = appendUnique(c.Versioning.Branches.Protected, overlay.Versioning.Branches.Protected)
	c.Incremental = overlay.Incremental
	c.Invariants = mergeInvariants(c.Invariants, overlay.Invariants)
	c.Commands.Block = appendUnique(c.Commands.Block, overlay.Commands.Block)
	c.Tools.Allow = appendUnique(c.Tools.Allow, overlay.Tools.Allow)
	c.Tools.Block = appendUnique(c.Tools.Block, overlay.Tools.Block)
	c.Hooks = appendHooksUnique(c.Hooks, overlay.Hooks)
}

func mergeInvariants(base, overlay InvariantsConfig) InvariantsConfig {
	return InvariantsConfig{
		Coexistence: appendCoexistenceUnique(base.Coexistence, overlay.Coexistence),
		Content:     appendContentUnique(base.Content, overlay.Content),
		Imports:     appendImportsUnique(base.Imports, overlay.Imports),
		Naming:      appendNamingUnique(base.Naming, overlay.Naming),
		Required:    appendRequiredUnique(base.Required, overlay.Required),
	}
}

func appendCoexistenceUnique(base, items []CoexistenceCheck) []CoexistenceCheck {
	seen := make(map[string]bool)
	for _, c := range base {
		seen[c.Name] = true
	}
	result := base
	for _, c := range items {
		if !seen[c.Name] {
			result = append(result, c)
			seen[c.Name] = true
		}
	}
	return result
}

func appendContentUnique(base, items []ContentCheck) []ContentCheck {
	seen := make(map[string]bool)
	for _, c := range base {
		seen[c.Name] = true
	}
	result := base
	for _, c := range items {
		if !seen[c.Name] {
			result = append(result, c)
			seen[c.Name] = true
		}
	}
	return result
}

func appendImportsUnique(base, items []ImportCheck) []ImportCheck {
	seen := make(map[string]bool)
	for _, c := range base {
		seen[c.Name] = true
	}
	result := base
	for _, c := range items {
		if !seen[c.Name] {
			result = append(result, c)
			seen[c.Name] = true
		}
	}
	return result
}

func appendNamingUnique(base, items []NamingCheck) []NamingCheck {
	seen := make(map[string]bool)
	for _, c := range base {
		seen[c.Name] = true
	}
	result := base
	for _, c := range items {
		if !seen[c.Name] {
			result = append(result, c)
			seen[c.Name] = true
		}
	}
	return result
}

func appendRequiredUnique(base, items []RequiredCheck) []RequiredCheck {
	seen := make(map[string]bool)
	for _, c := range base {
		seen[c.Name] = true
	}
	result := base
	for _, c := range items {
		if !seen[c.Name] {
			result = append(result, c)
			seen[c.Name] = true
		}
	}
	return result
}

func appendHooksUnique(base, items []HookConfig) []HookConfig {
	seen := make(map[string]bool)
	for _, h := range base {
		seen[h.Name] = true
	}
	result := base
	for _, h := range items {
		if !seen[h.Name] {
			result = append(result, h)
			seen[h.Name] = true
		}
	}
	return result
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
