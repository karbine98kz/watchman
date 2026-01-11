package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefault(t *testing.T) {
	cfg := Default()

	if cfg.Version != 1 {
		t.Errorf("Version = %d, want 1", cfg.Version)
	}
	if !cfg.Rules.Workspace {
		t.Error("Rules.Workspace should be true by default")
	}
	if cfg.Rules.Scope {
		t.Error("Rules.Scope should be false by default")
	}
}

func TestLoadFromFile(t *testing.T) {
	// Create temp config file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yml")

	content := `
version: 1
rules:
  workspace: true
  scope: true
workspace:
  allow:
    - /tmp
  block:
    - .env
commands:
  block:
    - sudo
    - rm -rf
tools:
  block:
    - Bash
`
	if err := os.WriteFile(configPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	cfg := Default()
	if err := cfg.loadFrom(configPath); err != nil {
		t.Fatal(err)
	}

	if !cfg.Rules.Workspace {
		t.Error("Rules.Workspace should be true")
	}
	if !cfg.Rules.Scope {
		t.Error("Rules.Scope should be true")
	}
	if len(cfg.Workspace.Allow) != 1 || cfg.Workspace.Allow[0] != "/tmp" {
		t.Errorf("Workspace.Allow = %v, want [/tmp]", cfg.Workspace.Allow)
	}
	if len(cfg.Workspace.Block) != 1 || cfg.Workspace.Block[0] != ".env" {
		t.Errorf("Workspace.Block = %v, want [.env]", cfg.Workspace.Block)
	}
	if len(cfg.Commands.Block) != 2 {
		t.Errorf("Commands.Block = %v, want 2 items", cfg.Commands.Block)
	}
	if len(cfg.Tools.Block) != 1 || cfg.Tools.Block[0] != "Bash" {
		t.Errorf("Tools.Block = %v, want [Bash]", cfg.Tools.Block)
	}
}

func TestMerge(t *testing.T) {
	base := &Config{
		Version: 1,
		Rules:   RulesConfig{Workspace: true},
		Workspace: WorkspaceConfig{
			Allow: []string{"/tmp"},
			Block: []string{".env"},
		},
		Scope: ScopeConfig{
			Allow: []string{"src/**"},
		},
	}

	overlay := &Config{
		Rules: RulesConfig{Workspace: true, Scope: true},
		Workspace: WorkspaceConfig{
			Allow: []string{"/var"},
			Block: []string{"secrets/"},
		},
		Scope: ScopeConfig{
			Allow: []string{"internal/**"},
			Block: []string{"vendor/**"},
		},
		Commands: CommandsConfig{
			Block: []string{"sudo"},
		},
	}

	base.merge(overlay)

	if !base.Rules.Workspace {
		t.Error("Rules.Workspace should be true")
	}
	if !base.Rules.Scope {
		t.Error("Rules.Scope should be true after merge")
	}
	if len(base.Workspace.Allow) != 2 {
		t.Errorf("Workspace.Allow = %v, want 2 items", base.Workspace.Allow)
	}
	if len(base.Workspace.Block) != 2 {
		t.Errorf("Workspace.Block = %v, want 2 items", base.Workspace.Block)
	}
	if len(base.Scope.Allow) != 2 {
		t.Errorf("Scope.Allow = %v, want 2 items", base.Scope.Allow)
	}
	if len(base.Scope.Block) != 1 {
		t.Errorf("Scope.Block = %v, want 1 item", base.Scope.Block)
	}
	if len(base.Commands.Block) != 1 {
		t.Errorf("Commands.Block = %v, want 1 item", base.Commands.Block)
	}
}

func TestMergeOverridesRules(t *testing.T) {
	base := &Config{
		Rules: RulesConfig{Workspace: true, Scope: true},
	}

	overlay := &Config{
		Rules: RulesConfig{Workspace: false, Scope: false},
	}

	base.merge(overlay)

	if base.Rules.Workspace {
		t.Error("Rules.Workspace should be false after merge")
	}
	if base.Rules.Scope {
		t.Error("Rules.Scope should be false after merge")
	}
}

func TestAppendUnique(t *testing.T) {
	base := []string{"a", "b"}
	items := []string{"b", "c"}

	result := appendUnique(base, items)

	if len(result) != 3 {
		t.Errorf("len(result) = %d, want 3", len(result))
	}
}

func TestLoadWithLocalConfig(t *testing.T) {
	tmpDir := t.TempDir()
	originalWd, _ := os.Getwd()
	defer os.Chdir(originalWd)
	os.Chdir(tmpDir)

	content := `version: 1
rules:
  workspace: false
`
	os.WriteFile(filepath.Join(tmpDir, ".watchman.yml"), []byte(content), 0644)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.Rules.Workspace {
		t.Error("Rules.Workspace should be false from local config")
	}
}

func TestLoadWithoutConfig(t *testing.T) {
	tmpDir := t.TempDir()
	originalWd, _ := os.Getwd()
	defer os.Chdir(originalWd)
	os.Chdir(tmpDir)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if !cfg.Rules.Workspace {
		t.Error("Rules.Workspace should be true by default")
	}
}

func TestLoadFromInvalidYAML(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yml")

	os.WriteFile(configPath, []byte("invalid: yaml: content:"), 0644)

	cfg := Default()
	err := cfg.loadFrom(configPath)

	if err == nil {
		t.Error("loadFrom should return error for invalid YAML")
	}
}

func TestLoadFromNonexistentFile(t *testing.T) {
	cfg := Default()
	err := cfg.loadFrom("/nonexistent/path/config.yml")

	if err == nil {
		t.Error("loadFrom should return error for nonexistent file")
	}
}

func TestGlobalConfigPath(t *testing.T) {
	path := GlobalConfigPath()

	if path == "" {
		t.Error("GlobalConfigPath should return non-empty path")
	}

	home, _ := os.UserHomeDir()
	expected := filepath.Join(home, ".config", "watchman", "config.yml")
	if path != expected {
		t.Errorf("GlobalConfigPath = %s, want %s", path, expected)
	}
}

func TestLocalConfigPath(t *testing.T) {
	path := localConfigPath()

	if path == "" {
		t.Error("localConfigPath should return non-empty path")
	}

	cwd, _ := os.Getwd()
	expected := filepath.Join(cwd, ".watchman.yml")
	if path != expected {
		t.Errorf("localConfigPath = %s, want %s", path, expected)
	}
}
