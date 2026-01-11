package policy

import (
	"testing"

	"github.com/adrianpk/watchman/internal/parser"
)

func TestConfineToWorkspaceEvaluate(t *testing.T) {
	tests := []struct {
		name        string
		cmd         string
		wantAllowed bool
	}{
		// Allowed cases
		{
			name:        "go test relative",
			cmd:         "go test ./...",
			wantAllowed: true,
		},
		{
			name:        "go test package",
			cmd:         "go test ./pkg/...",
			wantAllowed: true,
		},
		{
			name:        "make test",
			cmd:         "make test",
			wantAllowed: true,
		},
		{
			name:        "go build current dir",
			cmd:         "go build .",
			wantAllowed: true,
		},
		{
			name:        "simple command no args",
			cmd:         "ls",
			wantAllowed: true,
		},
		{
			name:        "flags without values",
			cmd:         "go test -race -v ./...",
			wantAllowed: true,
		},
		{
			name:        "empty command",
			cmd:         "",
			wantAllowed: true,
		},

		// Blocked cases - absolute paths
		{
			name:        "rm absolute path",
			cmd:         "rm -rf /",
			wantAllowed: false,
		},
		{
			name:        "cat absolute path",
			cmd:         "cat /etc/passwd",
			wantAllowed: false,
		},
		{
			name:        "cp to absolute",
			cmd:         "cp file.txt /tmp/file.txt",
			wantAllowed: false,
		},
		{
			name:        "flag with absolute path",
			cmd:         "go test -coverprofile=/tmp/cover.out ./...",
			wantAllowed: false,
		},

		// Blocked cases - traversal
		{
			name:        "parent dir only",
			cmd:         "cat ..",
			wantAllowed: false,
		},
		{
			name:        "parent dir traversal",
			cmd:         "cat ../secrets",
			wantAllowed: false,
		},
		{
			name:        "deep traversal",
			cmd:         "cp ../../other/file .",
			wantAllowed: false,
		},
		{
			name:        "traversal in flag value",
			cmd:         "go test -coverprofile=../cover.out ./...",
			wantAllowed: false,
		},

		// Blocked cases - env vars with absolute paths
		{
			name:        "env var absolute path",
			cmd:         "GOMODCACHE=/tmp/mod go test ./...",
			wantAllowed: false,
		},
		{
			name:        "multiple env vars one absolute",
			cmd:         "FOO=bar GOBIN=/usr/local/bin go install ./...",
			wantAllowed: false,
		},
	}

	rule := ConfineToWorkspace{}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := parser.Parse(tt.cmd)
			got := rule.Evaluate(cmd)

			if got.Allowed != tt.wantAllowed {
				t.Errorf("Evaluate(%q) Allowed = %v, want %v", tt.cmd, got.Allowed, tt.wantAllowed)
			}

			if !tt.wantAllowed && got.Reason == "" {
				t.Error("blocked decision should have a reason")
			}
		})
	}
}

func TestCollectPathCandidates(t *testing.T) {
	cmd := parser.Parse("GOBIN=/tmp go test -coverprofile=cover.out ./pkg ./internal")
	candidates := collectPathCandidates(cmd)

	// Should contain: args (./pkg, ./internal), flag values (cover.out), env values (/tmp)
	if len(candidates) < 3 {
		t.Errorf("expected at least 3 candidates, got %d: %v", len(candidates), candidates)
	}

	hasAbsolute := false
	for _, c := range candidates {
		if c == "/tmp" {
			hasAbsolute = true
			break
		}
	}
	if !hasAbsolute {
		t.Error("expected /tmp from env var in candidates")
	}
}

func TestViolatesWorkspaceBoundary(t *testing.T) {
	tests := []struct {
		path     string
		violates bool
	}{
		{"", false},
		{".", false},
		{"./foo", false},
		{"foo/bar", false},
		{"./...", false},

		{"/", true},
		{"/tmp", true},
		{"/etc/passwd", true},

		{"..", true},
		{"../foo", true},
		{"../../bar", true},
		{"foo/../..", true},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			got := ViolatesWorkspaceBoundary(tt.path)
			if got != tt.violates {
				t.Errorf("ViolatesWorkspaceBoundary(%q) = %v, want %v", tt.path, got, tt.violates)
			}
		})
	}
}
