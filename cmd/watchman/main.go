package main

import (
	"encoding/json"
	"fmt"
	"os"

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
	var input hookInput
	if err := json.NewDecoder(os.Stdin).Decode(&input); err != nil {
		fatal("cannot decode input: %v", err)
	}

	if !filesystemTools[input.ToolName] {
		allow()
		return
	}

	paths := extractPaths(input.ToolName, input.ToolInput)
	for _, p := range paths {
		if policy.ViolatesWorkspaceBoundary(p) {
			deny("cannot access paths outside the project workspace")
			return
		}
	}

	allow()
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

func allow() {
	json.NewEncoder(os.Stdout).Encode(hookOutput{Decision: "allow"})
	os.Exit(0)
}

func deny(reason string) {
	fmt.Fprintln(os.Stderr, reason)
	os.Exit(2)
}

func fatal(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
