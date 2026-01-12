package policy

import (
	"testing"

	"github.com/adrianpk/watchman/internal/config"
)

func TestNewIncrementalRule(t *testing.T) {
	tests := []struct {
		name string
		cfg  *config.IncrementalConfig
	}{
		{
			name: "nil config",
			cfg:  nil,
		},
		{
			name: "with config",
			cfg: &config.IncrementalConfig{
				MaxFiles:  10,
				WarnRatio: 0.7,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rule := NewIncrementalRule(tt.cfg)
			if rule == nil {
				t.Error("NewIncrementalRule returned nil")
			}
			if rule.countFunc == nil {
				t.Error("NewIncrementalRule should set countFunc")
			}
		})
	}
}

func TestIncrementalRule_WarnThreshold(t *testing.T) {
	tests := []struct {
		name      string
		maxFiles  int
		warnRatio float64
		want      int
	}{
		{
			name:      "default ratio when zero",
			maxFiles:  10,
			warnRatio: 0,
			want:      7, // 70% of 10
		},
		{
			name:      "custom ratio 0.8",
			maxFiles:  10,
			warnRatio: 0.8,
			want:      8,
		},
		{
			name:      "custom ratio 0.5",
			maxFiles:  20,
			warnRatio: 0.5,
			want:      10,
		},
		{
			name:      "invalid ratio > 1 defaults to 0.7",
			maxFiles:  10,
			warnRatio: 1.5,
			want:      7,
		},
		{
			name:      "negative ratio defaults to 0.7",
			maxFiles:  10,
			warnRatio: -0.5,
			want:      7,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rule := &IncrementalRule{
				MaxFiles:  tt.maxFiles,
				WarnRatio: tt.warnRatio,
			}
			got := rule.warnThreshold()
			if got != tt.want {
				t.Errorf("warnThreshold() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestIncrementalRule_EvaluateDisabled(t *testing.T) {
	// When MaxFiles is 0, rule is effectively disabled
	rule := &IncrementalRule{MaxFiles: 0}
	decision := rule.Evaluate()
	if !decision.Allowed {
		t.Error("Evaluate() should allow when MaxFiles is 0")
	}
}

func TestIncrementalRule_Evaluate(t *testing.T) {
	tests := []struct {
		name        string
		maxFiles    int
		warnRatio   float64
		fileCount   int
		wantAllowed bool
		wantWarning bool
		wantReason  bool
	}{
		{
			name:        "under threshold",
			maxFiles:    10,
			warnRatio:   0.7,
			fileCount:   3,
			wantAllowed: true,
			wantWarning: false,
		},
		{
			name:        "at warn threshold",
			maxFiles:    10,
			warnRatio:   0.7,
			fileCount:   7,
			wantAllowed: true,
			wantWarning: true,
		},
		{
			name:        "above warn threshold",
			maxFiles:    10,
			warnRatio:   0.7,
			fileCount:   8,
			wantAllowed: true,
			wantWarning: true,
		},
		{
			name:        "at max threshold",
			maxFiles:    10,
			warnRatio:   0.7,
			fileCount:   10,
			wantAllowed: false,
			wantWarning: false,
			wantReason:  true,
		},
		{
			name:        "above max threshold",
			maxFiles:    10,
			warnRatio:   0.7,
			fileCount:   15,
			wantAllowed: false,
			wantWarning: false,
			wantReason:  true,
		},
		{
			name:        "git status fails",
			maxFiles:    10,
			warnRatio:   0.7,
			fileCount:   -1,
			wantAllowed: true,
			wantWarning: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rule := &IncrementalRule{
				MaxFiles:  tt.maxFiles,
				WarnRatio: tt.warnRatio,
				countFunc: func() int { return tt.fileCount },
			}
			decision := rule.Evaluate()
			if decision.Allowed != tt.wantAllowed {
				t.Errorf("Evaluate() allowed = %v, want %v", decision.Allowed, tt.wantAllowed)
			}
			hasWarning := decision.Warning != ""
			if hasWarning != tt.wantWarning {
				t.Errorf("Evaluate() has warning = %v, want %v", hasWarning, tt.wantWarning)
			}
			hasReason := decision.Reason != ""
			if hasReason != tt.wantReason {
				t.Errorf("Evaluate() has reason = %v, want %v", hasReason, tt.wantReason)
			}
		})
	}
}

func TestIncrementalRule_CountModifiedFiles(t *testing.T) {
	t.Run("uses countFunc when set", func(t *testing.T) {
		rule := &IncrementalRule{
			countFunc: func() int { return 42 },
		}
		got := rule.countModifiedFiles()
		if got != 42 {
			t.Errorf("countModifiedFiles() = %d, want 42", got)
		}
	})

	t.Run("falls back to git when countFunc is nil", func(t *testing.T) {
		rule := &IncrementalRule{}
		got := rule.countModifiedFiles()
		// Just verify it returns a valid count (not necessarily 0)
		if got < -1 {
			t.Errorf("countModifiedFiles() = %d, want >= -1", got)
		}
	})
}

func TestCountGitModifiedFiles(t *testing.T) {
	// This test actually runs git status, so it's more of an integration test.
	count := countGitModifiedFiles()
	// Just verify it doesn't return an unexpected error
	if count < -1 {
		t.Errorf("countGitModifiedFiles() = %d, want >= -1", count)
	}
	t.Logf("Current modified files: %d", count)
}

func TestParseGitStatusOutput(t *testing.T) {
	// Test the parsing logic by testing countGitModifiedFiles indirectly
	// Since we can't easily mock exec.Command, we test what we can
	count := countGitModifiedFiles()
	if count < 0 {
		t.Skip("git status failed, skipping")
	}
	// If we're in a git repo, count should be >= 0
	t.Logf("Git status returned count: %d", count)
}
