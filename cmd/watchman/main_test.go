package main

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

var binaryPath string

func TestMain(m *testing.M) {
	dir, err := os.MkdirTemp("", "watchman-test")
	if err != nil {
		os.Exit(1)
	}
	defer os.RemoveAll(dir)

	binaryPath = filepath.Join(dir, "watchman")
	cmd := exec.Command("go", "build", "-o", binaryPath, ".")
	cmd.Dir = "."
	if err := cmd.Run(); err != nil {
		os.Exit(1)
	}

	os.Exit(m.Run())
}

func runWatchman(t *testing.T, input string) (stdout, stderr string, exitCode int) {
	t.Helper()

	cmd := exec.Command(binaryPath)
	cmd.Stdin = bytes.NewBufferString(input)

	var outBuf, errBuf bytes.Buffer
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf

	err := cmd.Run()
	exitCode = 0
	if exitErr, ok := err.(*exec.ExitError); ok {
		exitCode = exitErr.ExitCode()
	} else if err != nil {
		t.Fatalf("cannot run binary: %v", err)
	}

	return outBuf.String(), errBuf.String(), exitCode
}

func makeInput(command string) string {
	input := map[string]interface{}{
		"hook_type": "PreToolUse",
		"tool_name": "Bash",
		"tool_input": map[string]interface{}{
			"command": command,
		},
	}
	data, _ := json.Marshal(input)
	return string(data)
}

func TestWatchmanAllowsRelativePaths(t *testing.T) {
	tests := []struct {
		name string
		cmd  string
	}{
		{"go test relative", "go test ./..."},
		{"go test package", "go test ./pkg/..."},
		{"make", "make test"},
		{"ls current", "ls ."},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stdout, stderr, exitCode := runWatchman(t, makeInput(tt.cmd))

			if exitCode != 0 {
				t.Errorf("expected exit 0, got %d (stderr: %s)", exitCode, stderr)
			}

			var output hookOutput
			if err := json.Unmarshal([]byte(stdout), &output); err != nil {
				t.Fatalf("cannot parse output: %v", err)
			}

			if output.Decision != "allow" {
				t.Errorf("expected allow, got %s", output.Decision)
			}
		})
	}
}

func TestWatchmanBlocksAbsolutePaths(t *testing.T) {
	tests := []struct {
		name string
		cmd  string
	}{
		{"rm root", "rm -rf /"},
		{"cat etc passwd", "cat /etc/passwd"},
		{"env absolute", "GOMODCACHE=/tmp/mod go test ./..."},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, stderr, exitCode := runWatchman(t, makeInput(tt.cmd))

			if exitCode != 2 {
				t.Errorf("expected exit 2, got %d", exitCode)
			}

			if stderr == "" {
				t.Error("expected error message in stderr")
			}
		})
	}
}

func TestWatchmanBlocksTraversal(t *testing.T) {
	tests := []struct {
		name string
		cmd  string
	}{
		{"parent dir", "cat .."},
		{"parent file", "cat ../secrets"},
		{"deep traversal", "cp ../../file ."},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, stderr, exitCode := runWatchman(t, makeInput(tt.cmd))

			if exitCode != 2 {
				t.Errorf("expected exit 2, got %d", exitCode)
			}

			if stderr == "" {
				t.Error("expected error message in stderr")
			}
		})
	}
}

func TestWatchmanAllowsNonFilesystemTools(t *testing.T) {
	input := `{"hook_type":"PreToolUse","tool_name":"WebSearch","tool_input":{"query":"test"}}`
	stdout, _, exitCode := runWatchman(t, input)

	if exitCode != 0 {
		t.Errorf("expected exit 0 for non-filesystem tool, got %d", exitCode)
	}

	var output hookOutput
	if err := json.Unmarshal([]byte(stdout), &output); err != nil {
		t.Fatalf("cannot parse output: %v", err)
	}

	if output.Decision != "allow" {
		t.Errorf("expected allow for non-filesystem tool, got %s", output.Decision)
	}
}

func TestWatchmanBlocksReadAbsolutePath(t *testing.T) {
	input := `{"hook_type":"PreToolUse","tool_name":"Read","tool_input":{"file_path":"/etc/passwd"}}`
	_, stderr, exitCode := runWatchman(t, input)

	if exitCode != 2 {
		t.Errorf("expected exit 2 for Read with absolute path, got %d", exitCode)
	}

	if stderr == "" {
		t.Error("expected error message in stderr")
	}
}

func TestWatchmanAllowsReadRelativePath(t *testing.T) {
	input := `{"hook_type":"PreToolUse","tool_name":"Read","tool_input":{"file_path":"./src/main.go"}}`
	stdout, _, exitCode := runWatchman(t, input)

	if exitCode != 0 {
		t.Errorf("expected exit 0 for Read with relative path, got %d", exitCode)
	}

	var output hookOutput
	if err := json.Unmarshal([]byte(stdout), &output); err != nil {
		t.Fatalf("cannot parse output: %v", err)
	}

	if output.Decision != "allow" {
		t.Errorf("expected allow, got %s", output.Decision)
	}
}

func TestWatchmanBlocksWriteAbsolutePath(t *testing.T) {
	input := `{"hook_type":"PreToolUse","tool_name":"Write","tool_input":{"file_path":"/tmp/evil.sh","content":"bad"}}`
	_, stderr, exitCode := runWatchman(t, input)

	if exitCode != 2 {
		t.Errorf("expected exit 2 for Write with absolute path, got %d", exitCode)
	}

	if stderr == "" {
		t.Error("expected error message in stderr")
	}
}

func TestWatchmanBlocksEditTraversal(t *testing.T) {
	input := `{"hook_type":"PreToolUse","tool_name":"Edit","tool_input":{"file_path":"../secret.txt"}}`
	_, stderr, exitCode := runWatchman(t, input)

	if exitCode != 2 {
		t.Errorf("expected exit 2 for Edit with traversal, got %d", exitCode)
	}

	if stderr == "" {
		t.Error("expected error message in stderr")
	}
}

func TestWatchmanBlocksGlobAbsolutePath(t *testing.T) {
	input := `{"hook_type":"PreToolUse","tool_name":"Glob","tool_input":{"pattern":"*.go","path":"/etc"}}`
	_, stderr, exitCode := runWatchman(t, input)

	if exitCode != 2 {
		t.Errorf("expected exit 2 for Glob with absolute path, got %d", exitCode)
	}

	if stderr == "" {
		t.Error("expected error message in stderr")
	}
}

func TestWatchmanBlocksGrepAbsolutePath(t *testing.T) {
	input := `{"hook_type":"PreToolUse","tool_name":"Grep","tool_input":{"pattern":"password","path":"/etc/passwd"}}`
	_, stderr, exitCode := runWatchman(t, input)

	if exitCode != 2 {
		t.Errorf("expected exit 2 for Grep with absolute path, got %d", exitCode)
	}

	if stderr == "" {
		t.Error("expected error message in stderr")
	}
}

func TestWatchmanInvalidJSON(t *testing.T) {
	_, stderr, exitCode := runWatchman(t, "not json")

	if exitCode != 1 {
		t.Errorf("expected exit 1 for invalid JSON, got %d", exitCode)
	}

	if stderr == "" {
		t.Error("expected error message for invalid JSON")
	}
}
