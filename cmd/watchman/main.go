package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/adrianpk/watchman/internal/config"
	"github.com/adrianpk/watchman/internal/parser"
	"github.com/adrianpk/watchman/internal/policy"
)

type hookInput struct {
	HookType  string                 `json:"hook_type"`
	ToolName  string                 `json:"tool_name"`
	ToolInput map[string]interface{} `json:"tool_input"`
}

type hookOutput struct {
	Decision string `json:"decision"`
}

var filesystemTools = map[string]bool{
	"Bash":  true,
	"Read":  true,
	"Write": true,
	"Edit":  true,
	"Glob":  true,
	"Grep":  true,
}

func main() {
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "init":
			runInit()
			return
		case "setup":
			runSetup()
			return
		}
	}

	cfg, err := config.Load()
	if err != nil {
		fatal("cannot load config: %v", err)
	}

	var input hookInput
	if err := json.NewDecoder(os.Stdin).Decode(&input); err != nil {
		fatal("cannot decode input: %v", err)
	}

	if isToolBlocked(cfg, input.ToolName) {
		deny("tool is blocked by configuration: " + input.ToolName)
		return
	}

	if !isToolAllowed(cfg, input.ToolName) {
		deny("tool is not in allowed list: " + input.ToolName)
		return
	}

	if !filesystemTools[input.ToolName] {
		allow()
		return
	}

	if input.ToolName == "Bash" {
		if cmd, ok := input.ToolInput["command"].(string); ok {
			if blocked := isCommandBlocked(cfg, cmd); blocked != "" {
				deny("command is blocked by configuration: " + blocked)
				return
			}
		}
	}

	paths := extractPaths(input.ToolName, input.ToolInput)
	for _, p := range paths {
		if policy.IsAlwaysProtected(p) {
			deny("path is protected and cannot be accessed. User must perform this action manually.")
			return
		}
	}

	if cfg.Rules.Workspace {
		rule := policy.NewConfineToWorkspace(&cfg.Workspace)
		paths := extractPaths(input.ToolName, input.ToolInput)
		for _, p := range paths {
			parsed := parser.Command{Args: []string{p}}
			decision := rule.Evaluate(parsed)
			if !decision.Allowed {
				deny(decision.Reason)
				return
			}
		}
	}

	if cfg.Rules.Scope {
		rule := policy.NewScopeToFiles(&cfg.Scope)
		paths := extractPaths(input.ToolName, input.ToolInput)
		for _, p := range paths {
			parsed := parser.Command{Args: []string{p}}
			decision := rule.Evaluate(input.ToolName, parsed)
			if !decision.Allowed {
				deny(decision.Reason)
				return
			}
		}
	}

	if cfg.Rules.Versioning && input.ToolName == "Bash" {
		if cmd, ok := input.ToolInput["command"].(string); ok {
			rule := policy.NewVersioningRule(&cfg.Versioning)
			decision := rule.Evaluate(cmd)
			if !decision.Allowed {
				deny(decision.Reason)
				return
			}
		}
	}

	if cfg.Rules.Incremental && isModificationTool(input.ToolName) {
		rule := policy.NewIncrementalRule(&cfg.Incremental)
		decision := rule.Evaluate()
		if !decision.Allowed {
			deny(decision.Reason)
			return
		}
		if decision.Warning != "" {
			warn(decision.Warning)
		}
	}

	allow()
}

func isModificationTool(tool string) bool {
	switch tool {
	case "Write", "Edit", "NotebookEdit":
		return true
	}
	return false
}

func isToolBlocked(cfg *config.Config, tool string) bool {
	for _, t := range cfg.Tools.Block {
		if strings.EqualFold(t, tool) {
			return true
		}
	}
	return false
}

func isToolAllowed(cfg *config.Config, tool string) bool {
	// If no allowlist, all tools are allowed
	if len(cfg.Tools.Allow) == 0 {
		return true
	}
	for _, t := range cfg.Tools.Allow {
		if strings.EqualFold(t, tool) {
			return true
		}
	}
	return false
}

func isCommandBlocked(cfg *config.Config, cmd string) string {
	for _, pattern := range cfg.Commands.Block {
		if strings.Contains(cmd, pattern) {
			return pattern
		}
	}
	return ""
}

func extractPaths(toolName string, toolInput map[string]interface{}) []string {
	switch toolName {
	case "Bash":
		return extractBashPaths(toolInput)
	case "Read", "Write", "Edit":
		return extractFilePath(toolInput)
	case "Glob":
		return extractGlobPaths(toolInput)
	case "Grep":
		return extractGrepPaths(toolInput)
	}
	return nil
}

func extractBashPaths(toolInput map[string]interface{}) []string {
	cmdStr, ok := toolInput["command"].(string)
	if !ok {
		return nil
	}
	cmd := parser.Parse(cmdStr)
	var paths []string
	paths = append(paths, cmd.Args...)
	for _, v := range cmd.Flags {
		if v != "" {
			paths = append(paths, v)
		}
	}
	for _, v := range cmd.Env {
		paths = append(paths, v)
	}
	return paths
}

func extractFilePath(toolInput map[string]interface{}) []string {
	if fp, ok := toolInput["file_path"].(string); ok {
		return []string{fp}
	}
	return nil
}

func extractGlobPaths(toolInput map[string]interface{}) []string {
	var paths []string
	if p, ok := toolInput["path"].(string); ok {
		paths = append(paths, p)
	}
	if pattern, ok := toolInput["pattern"].(string); ok {
		paths = append(paths, pattern)
	}
	return paths
}

func extractGrepPaths(toolInput map[string]interface{}) []string {
	if p, ok := toolInput["path"].(string); ok {
		return []string{p}
	}
	return nil
}

func runInit() {
	local := len(os.Args) > 2 && os.Args[2] == "--local"

	var configPath string
	var configDir string

	if local {
		cwd, err := os.Getwd()
		if err != nil {
			fatal("cannot get working directory: %v", err)
		}
		configPath = filepath.Join(cwd, ".watchman.yml")
	} else {
		home, err := os.UserHomeDir()
		if err != nil {
			fatal("cannot get home directory: %v", err)
		}
		configDir = filepath.Join(home, ".config", "watchman")
		configPath = filepath.Join(configDir, "config.yml")
	}

	if _, err := os.Stat(configPath); err == nil {
		fmt.Printf("Config already exists: %s\n", configPath)
		os.Exit(0)
	}

	if configDir != "" {
		if err := os.MkdirAll(configDir, 0755); err != nil {
			fatal("cannot create config directory: %v", err)
		}
	}

	content := `version: 1

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

	if err := os.WriteFile(configPath, []byte(content), 0644); err != nil {
		fatal("cannot write config: %v", err)
	}

	fmt.Printf("Created config: %s\n", configPath)
}

func runSetup() {
	home, err := os.UserHomeDir()
	if err != nil {
		fatal("cannot get home directory: %v", err)
	}

	claudeDir := filepath.Join(home, ".claude")
	settingsPath := filepath.Join(claudeDir, "settings.json")
	watchmanPath := filepath.Join(home, "go", "bin", "watchman")

	if err := os.MkdirAll(claudeDir, 0755); err != nil {
		fatal("cannot create .claude directory: %v", err)
	}

	settings := make(map[string]interface{})

	data, err := os.ReadFile(settingsPath)
	if err == nil && len(data) > 0 {
		if err := json.Unmarshal(data, &settings); err != nil {
			fatal("cannot parse settings.json: %v", err)
		}
	}

	hooks, ok := settings["hooks"].(map[string]interface{})
	if !ok {
		hooks = make(map[string]interface{})
		settings["hooks"] = hooks
	}

	preToolUse, ok := hooks["PreToolUse"].([]interface{})
	if !ok {
		preToolUse = []interface{}{}
	}

	if hasWatchmanHook(preToolUse, watchmanPath) {
		fmt.Println("Watchman hook already configured")
		return
	}

	watchmanHook := map[string]interface{}{
		"matcher": "*",
		"hooks": []interface{}{
			map[string]interface{}{
				"type":    "command",
				"command": watchmanPath,
			},
		},
	}

	hooks["PreToolUse"] = []interface{}{watchmanHook}

	output, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		fatal("cannot marshal settings: %v", err)
	}

	if err := os.WriteFile(settingsPath, output, 0644); err != nil {
		fatal("cannot write settings.json: %v", err)
	}

	fmt.Printf("Configured hook: %s\n", settingsPath)
	fmt.Println("Run 'watchman init' to create watchman config")
}

func hasWatchmanHook(preToolUse []interface{}, watchmanPath string) bool {
	for _, entry := range preToolUse {
		e, ok := entry.(map[string]interface{})
		if !ok {
			continue
		}
		hooksList, ok := e["hooks"].([]interface{})
		if !ok {
			continue
		}
		for _, h := range hooksList {
			if h == "watchman" {
				return true
			}
			if hm, ok := h.(map[string]interface{}); ok {
				if cmd, ok := hm["command"].(string); ok {
					if strings.Contains(cmd, "watchman") {
						return true
					}
				}
			}
		}
	}
	return false
}

func allow() {
	json.NewEncoder(os.Stdout).Encode(hookOutput{Decision: "allow"})
	os.Exit(0)
}

func deny(reason string) {
	fmt.Fprintln(os.Stderr, reason)
	os.Exit(2)
}

func warn(message string) {
	fmt.Fprintln(os.Stderr, "warning: "+message)
}

func fatal(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
