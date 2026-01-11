package policy

import (
	"testing"

	"github.com/adrianpk/watchman/internal/config"
	"github.com/adrianpk/watchman/internal/parser"
)

func TestNewScopeToFiles(t *testing.T) {
	tests := []struct {
		name string
		cfg  *config.ScopeConfig
		want *ScopeToFiles
	}{
		{
			name: "nil config",
			cfg:  nil,
			want: &ScopeToFiles{},
		},
		{
			name: "with allow and block",
			cfg: &config.ScopeConfig{
				Allow: []string{"src/**/*.go"},
				Block: []string{"vendor/**"},
			},
			want: &ScopeToFiles{
				Allow: []string{"src/**/*.go"},
				Block: []string{"vendor/**"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NewScopeToFiles(tt.cfg)
			if tt.cfg == nil {
				if got.Allow != nil || got.Block != nil {
					t.Errorf("expected empty rule for nil config")
				}
				return
			}
			if len(got.Allow) != len(tt.want.Allow) {
				t.Errorf("Allow = %v, want %v", got.Allow, tt.want.Allow)
			}
			if len(got.Block) != len(tt.want.Block) {
				t.Errorf("Block = %v, want %v", got.Block, tt.want.Block)
			}
		})
	}
}

func TestScopeToFilesEvaluate(t *testing.T) {
	tests := []struct {
		name        string
		rule        *ScopeToFiles
		toolName    string
		cmd         parser.Command
		wantAllowed bool
	}{
		{
			name:        "read tool always allowed",
			rule:        &ScopeToFiles{Allow: []string{"src/**"}},
			toolName:    "Read",
			cmd:         parser.Command{Args: []string{"/etc/passwd"}},
			wantAllowed: true,
		},
		{
			name:        "grep tool always allowed",
			rule:        &ScopeToFiles{Block: []string{"**"}},
			toolName:    "Grep",
			cmd:         parser.Command{Args: []string{"vendor/lib.go"}},
			wantAllowed: true,
		},
		{
			name:        "write tool no scope allows all",
			rule:        &ScopeToFiles{},
			toolName:    "Write",
			cmd:         parser.Command{Args: []string{"any/file.go"}},
			wantAllowed: true,
		},
		{
			name:        "write tool in scope allowed",
			rule:        &ScopeToFiles{Allow: []string{"src/**/*.go"}},
			toolName:    "Write",
			cmd:         parser.Command{Args: []string{"src/main.go"}},
			wantAllowed: true,
		},
		{
			name:        "write tool out of scope blocked",
			rule:        &ScopeToFiles{Allow: []string{"src/**/*.go"}},
			toolName:    "Write",
			cmd:         parser.Command{Args: []string{"vendor/lib.go"}},
			wantAllowed: false,
		},
		{
			name:        "edit tool blocked by block list",
			rule:        &ScopeToFiles{Block: []string{"vendor/**"}},
			toolName:    "Edit",
			cmd:         parser.Command{Args: []string{"vendor/lib.go"}},
			wantAllowed: false,
		},
		{
			name:        "block takes precedence over allow",
			rule:        &ScopeToFiles{Allow: []string{"**/*.go"}, Block: []string{"vendor/**"}},
			toolName:    "Edit",
			cmd:         parser.Command{Args: []string{"vendor/lib.go"}},
			wantAllowed: false,
		},
		{
			name:        "notebook edit blocked",
			rule:        &ScopeToFiles{Allow: []string{"notebooks/*.ipynb"}},
			toolName:    "NotebookEdit",
			cmd:         parser.Command{Args: []string{"analysis/test.ipynb"}},
			wantAllowed: false,
		},
		{
			name:        "notebook edit allowed",
			rule:        &ScopeToFiles{Allow: []string{"notebooks/*.ipynb"}},
			toolName:    "NotebookEdit",
			cmd:         parser.Command{Args: []string{"notebooks/test.ipynb"}},
			wantAllowed: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.rule.Evaluate(tt.toolName, tt.cmd)
			if got.Allowed != tt.wantAllowed {
				t.Errorf("Evaluate() = %v, want %v, reason: %s", got.Allowed, tt.wantAllowed, got.Reason)
			}
		})
	}
}

func TestScopeIsBlocked(t *testing.T) {
	rule := &ScopeToFiles{
		Block: []string{"vendor/**", "**/*_generated.go", ".env"},
	}

	tests := []struct {
		path    string
		blocked bool
	}{
		{"vendor/lib/file.go", true},
		{"vendor/other.go", true},
		{"src/types_generated.go", true},
		{"internal/api_generated.go", true},
		{".env", true},
		{"src/main.go", false},
		{"internal/api.go", false},
		{"README.md", false},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			got := rule.isBlocked(tt.path)
			if got != tt.blocked {
				t.Errorf("isBlocked(%q) = %v, want %v", tt.path, got, tt.blocked)
			}
		})
	}
}

func TestScopeIsInScope(t *testing.T) {
	tests := []struct {
		name    string
		rule    *ScopeToFiles
		path    string
		inScope bool
	}{
		{
			name:    "empty allow list allows all",
			rule:    &ScopeToFiles{},
			path:    "any/path/file.go",
			inScope: true,
		},
		{
			name:    "path matches allow pattern",
			rule:    &ScopeToFiles{Allow: []string{"src/**/*.go"}},
			path:    "src/pkg/file.go",
			inScope: true,
		},
		{
			name:    "path does not match allow pattern",
			rule:    &ScopeToFiles{Allow: []string{"src/**/*.go"}},
			path:    "vendor/lib.go",
			inScope: false,
		},
		{
			name:    "multiple allow patterns",
			rule:    &ScopeToFiles{Allow: []string{"src/**", "internal/**"}},
			path:    "internal/pkg/file.go",
			inScope: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.rule.isInScope(tt.path)
			if got != tt.inScope {
				t.Errorf("isInScope(%q) = %v, want %v", tt.path, got, tt.inScope)
			}
		})
	}
}

func TestMatchGlob(t *testing.T) {
	tests := []struct {
		name    string
		path    string
		pattern string
		match   bool
	}{
		{
			name:    "exact match",
			path:    "main.go",
			pattern: "main.go",
			match:   true,
		},
		{
			name:    "no match",
			path:    "main.go",
			pattern: "other.go",
			match:   false,
		},
		{
			name:    "wildcard extension",
			path:    "src/main.go",
			pattern: "*.go",
			match:   true,
		},
		{
			name:    "single directory wildcard",
			path:    "src/main.go",
			pattern: "src/*.go",
			match:   true,
		},
		{
			name:    "doublestar recursive",
			path:    "src/pkg/internal/file.go",
			pattern: "src/**/*.go",
			match:   true,
		},
		{
			name:    "doublestar at end",
			path:    "vendor/lib/deep/file.go",
			pattern: "vendor/**",
			match:   true,
		},
		{
			name:    "doublestar prefix mismatch",
			path:    "src/file.go",
			pattern: "vendor/**",
			match:   false,
		},
		{
			name:    "doublestar suffix match",
			path:    "src/deep/nested/file_generated.go",
			pattern: "**/*_generated.go",
			match:   true,
		},
		{
			name:    "filename only pattern",
			path:    "deep/nested/.env",
			pattern: ".env",
			match:   true,
		},
		{
			name:    "question mark wildcard",
			path:    "file1.go",
			pattern: "file?.go",
			match:   true,
		},
		{
			name:    "character class",
			path:    "test_a.go",
			pattern: "test_[abc].go",
			match:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := matchGlob(tt.path, tt.pattern)
			if got != tt.match {
				t.Errorf("matchGlob(%q, %q) = %v, want %v", tt.path, tt.pattern, got, tt.match)
			}
		})
	}
}

func TestMatchDoublestar(t *testing.T) {
	tests := []struct {
		name    string
		path    string
		pattern string
		match   bool
	}{
		{
			name:    "prefix and suffix",
			path:    "src/pkg/file.go",
			pattern: "src/**/*.go",
			match:   true,
		},
		{
			name:    "prefix only",
			path:    "vendor/any/path/file",
			pattern: "vendor/**",
			match:   true,
		},
		{
			name:    "suffix only",
			path:    "any/path/file.go",
			pattern: "**/*.go",
			match:   true,
		},
		{
			name:    "root level with doublestar",
			path:    "file.go",
			pattern: "**/*.go",
			match:   true,
		},
		{
			name:    "no prefix match",
			path:    "other/pkg/file.go",
			pattern: "src/**/*.go",
			match:   false,
		},
		{
			name:    "invalid pattern multiple doublestar",
			path:    "a/b/c/d.go",
			pattern: "**/**/*.go",
			match:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := matchDoublestar(tt.path, tt.pattern)
			if got != tt.match {
				t.Errorf("matchDoublestar(%q, %q) = %v, want %v", tt.path, tt.pattern, got, tt.match)
			}
		})
	}
}
