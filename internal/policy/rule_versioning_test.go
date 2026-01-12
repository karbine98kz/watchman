package policy

import (
	"testing"

	"github.com/adrianpk/watchman/internal/config"
)

func TestNewVersioningRule(t *testing.T) {
	tests := []struct {
		name string
		cfg  *config.VersioningConfig
	}{
		{
			name: "nil config",
			cfg:  nil,
		},
		{
			name: "with config",
			cfg: &config.VersioningConfig{
				Commit: config.CommitConfig{
					MaxLength: 72,
				},
				Branches: config.BranchesConfig{
					Protected: []string{"main"},
				},
				Operations: config.OperationsConfig{
					Block: []string{"push --force"},
				},
				Tool: "jj",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rule := NewVersioningRule(tt.cfg)
			if rule == nil {
				t.Error("NewVersioningRule returned nil")
			}
		})
	}
}

func TestEvaluate(t *testing.T) {
	tests := []struct {
		name        string
		rule        *VersioningRule
		command     string
		wantAllowed bool
	}{
		{
			name:        "non-git command allowed",
			rule:        &VersioningRule{},
			command:     "ls -la",
			wantAllowed: true,
		},
		{
			name: "blocked operation",
			rule: &VersioningRule{
				Operations: config.OperationsConfig{
					Block: []string{"push --force", "rebase"},
				},
			},
			command:     "git push --force origin main",
			wantAllowed: false,
		},
		{
			name: "allowed push",
			rule: &VersioningRule{
				Operations: config.OperationsConfig{
					Block: []string{"push --force"},
				},
			},
			command:     "git push origin main",
			wantAllowed: true,
		},
		{
			name: "blocked rebase",
			rule: &VersioningRule{
				Operations: config.OperationsConfig{
					Block: []string{"rebase"},
				},
			},
			command:     "git rebase -i HEAD~3",
			wantAllowed: false,
		},
		{
			name:        "linear workflow blocks merge",
			rule:        &VersioningRule{Workflow: "linear"},
			command:     "git merge feature-branch",
			wantAllowed: false,
		},
		{
			name:        "linear workflow allows rebase",
			rule:        &VersioningRule{Workflow: "linear"},
			command:     "git rebase main",
			wantAllowed: true,
		},
		{
			name:        "merge workflow blocks rebase",
			rule:        &VersioningRule{Workflow: "merge"},
			command:     "git rebase main",
			wantAllowed: false,
		},
		{
			name:        "merge workflow allows merge",
			rule:        &VersioningRule{Workflow: "merge"},
			command:     "git merge feature-branch",
			wantAllowed: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.rule.Evaluate(tt.command)
			if got.Allowed != tt.wantAllowed {
				t.Errorf("Evaluate() = %v, want %v, reason: %s", got.Allowed, tt.wantAllowed, got.Reason)
			}
		})
	}
}

func TestEvaluateCommit(t *testing.T) {
	tests := []struct {
		name        string
		rule        *VersioningRule
		command     string
		wantAllowed bool
	}{
		{
			name:        "non-commit command allowed",
			rule:        &VersioningRule{},
			command:     "git status",
			wantAllowed: true,
		},
		{
			name:        "commit without rules allowed",
			rule:        &VersioningRule{},
			command:     `git commit -m "Add feature"`,
			wantAllowed: true,
		},
		{
			name: "prefer jj blocks git commit",
			rule: &VersioningRule{
				Tool: "jj",
			},
			command:     `git commit -m "Add feature"`,
			wantAllowed: false,
		},
		{
			name: "prefer jj allows jj commit",
			rule: &VersioningRule{
				Tool: "jj",
			},
			command:     `jj commit -m "Add feature"`,
			wantAllowed: true,
		},
		{
			name: "protected branch blocked",
			rule: &VersioningRule{
				Branches: config.BranchesConfig{
					Protected: []string{"main", "master"},
				},
			},
			command:     `git commit -m "Fix" -b main`,
			wantAllowed: false,
		},
		{
			name: "max length exceeded",
			rule: &VersioningRule{
				Commit: config.CommitConfig{
					MaxLength: 10,
				},
			},
			command:     `git commit -m "This message is way too long"`,
			wantAllowed: false,
		},
		{
			name: "max length ok",
			rule: &VersioningRule{
				Commit: config.CommitConfig{
					MaxLength: 50,
				},
			},
			command:     `git commit -m "Short message"`,
			wantAllowed: true,
		},
		{
			name: "require uppercase fails",
			rule: &VersioningRule{
				Commit: config.CommitConfig{
					RequireUppercase: true,
				},
			},
			command:     `git commit -m "lowercase start"`,
			wantAllowed: false,
		},
		{
			name: "require uppercase passes",
			rule: &VersioningRule{
				Commit: config.CommitConfig{
					RequireUppercase: true,
				},
			},
			command:     `git commit -m "Uppercase start"`,
			wantAllowed: true,
		},
		{
			name: "no period fails",
			rule: &VersioningRule{
				Commit: config.CommitConfig{
					NoPeriod: true,
				},
			},
			command:     `git commit -m "Message with period."`,
			wantAllowed: false,
		},
		{
			name: "no period passes",
			rule: &VersioningRule{
				Commit: config.CommitConfig{
					NoPeriod: true,
				},
			},
			command:     `git commit -m "Message without period"`,
			wantAllowed: true,
		},
		{
			name: "prefix pattern fails",
			rule: &VersioningRule{
				Commit: config.CommitConfig{
					PrefixPattern: `\[JIRA-\d+\]`,
				},
			},
			command:     `git commit -m "Add feature"`,
			wantAllowed: false,
		},
		{
			name: "prefix pattern passes",
			rule: &VersioningRule{
				Commit: config.CommitConfig{
					PrefixPattern: `\[JIRA-\d+\]`,
				},
			},
			command:     `git commit -m "[JIRA-123] Add feature"`,
			wantAllowed: true,
		},
		{
			name: "require period fails",
			rule: &VersioningRule{
				Commit: config.CommitConfig{
					RequirePeriod: true,
				},
			},
			command:     `git commit -m "Message without period"`,
			wantAllowed: false,
		},
		{
			name: "require period passes",
			rule: &VersioningRule{
				Commit: config.CommitConfig{
					RequirePeriod: true,
				},
			},
			command:     `git commit -m "Message with period."`,
			wantAllowed: true,
		},
		{
			name: "single line fails with newline",
			rule: &VersioningRule{
				Commit: config.CommitConfig{
					SingleLine: true,
				},
			},
			command:     "git commit -m \"Subject\n\nBody text\"",
			wantAllowed: false,
		},
		{
			name: "single line passes",
			rule: &VersioningRule{
				Commit: config.CommitConfig{
					SingleLine: true,
				},
			},
			command:     `git commit -m "Single line message"`,
			wantAllowed: true,
		},
		{
			name: "forbid colons fails",
			rule: &VersioningRule{
				Commit: config.CommitConfig{
					ForbidColons: true,
				},
			},
			command:     `git commit -m "fix: bug in parser"`,
			wantAllowed: false,
		},
		{
			name: "forbid colons passes",
			rule: &VersioningRule{
				Commit: config.CommitConfig{
					ForbidColons: true,
				},
			},
			command:     `git commit -m "Fix bug in parser"`,
			wantAllowed: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.rule.EvaluateCommit(tt.command)
			if got.Allowed != tt.wantAllowed {
				t.Errorf("EvaluateCommit() = %v, want %v, reason: %s", got.Allowed, tt.wantAllowed, got.Reason)
			}
		})
	}
}

func TestExtractCommitMessage(t *testing.T) {
	tests := []struct {
		name    string
		command string
		want    string
	}{
		{
			name:    "double quoted -m",
			command: `git commit -m "Add feature"`,
			want:    "Add feature",
		},
		{
			name:    "single quoted -m",
			command: `git commit -m 'Fix bug'`,
			want:    "Fix bug",
		},
		{
			name:    "message flag long form",
			command: `git commit --message "Update docs"`,
			want:    "Update docs",
		},
		{
			name:    "message flag with equals",
			command: `git commit --message="Refactor code"`,
			want:    "Refactor code",
		},
		{
			name:    "no message",
			command: `git commit`,
			want:    "",
		},
		{
			name:    "jj commit",
			command: `jj commit -m "JJ commit"`,
			want:    "JJ commit",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractCommitMessage(tt.command)
			if got != tt.want {
				t.Errorf("extractCommitMessage() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestIsCommitCommand(t *testing.T) {
	tests := []struct {
		command  string
		isCommit bool
	}{
		{"git commit -m 'test'", true},
		{"jj commit -m 'test'", true},
		{"git status", false},
		{"git push", false},
		{"ls -la", false},
	}

	for _, tt := range tests {
		t.Run(tt.command, func(t *testing.T) {
			got := isCommitCommand(tt.command)
			if got != tt.isCommit {
				t.Errorf("isCommitCommand(%q) = %v, want %v", tt.command, got, tt.isCommit)
			}
		})
	}
}

func TestIsProtectedBranch(t *testing.T) {
	rule := &VersioningRule{
		Branches: config.BranchesConfig{
			Protected: []string{"main", "master", "release/*"},
		},
	}

	tests := []struct {
		branch    string
		protected bool
	}{
		{"main", true},
		{"master", true},
		{"feature/test", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.branch, func(t *testing.T) {
			got := rule.isProtectedBranch(tt.branch)
			if got != tt.protected {
				t.Errorf("isProtectedBranch(%q) = %v, want %v", tt.branch, got, tt.protected)
			}
		})
	}
}

func TestItoa(t *testing.T) {
	tests := []struct {
		n    int
		want string
	}{
		{0, "0"},
		{1, "1"},
		{42, "42"},
		{100, "100"},
		{72, "72"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := itoa(tt.n)
			if got != tt.want {
				t.Errorf("itoa(%d) = %q, want %q", tt.n, got, tt.want)
			}
		})
	}
}
