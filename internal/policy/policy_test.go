package policy

import (
	"testing"

	"github.com/adrianpk/watchman/internal/parser"
)

type allowAllRule struct{}

func (r allowAllRule) Evaluate(cmd parser.Command) Decision {
	return Decision{Allowed: true}
}

type denyAllRule struct {
	reason string
}

func (r denyAllRule) Evaluate(cmd parser.Command) Decision {
	return Decision{Allowed: false, Reason: r.reason}
}

func TestPolicyEvaluate(t *testing.T) {
	tests := []struct {
		name        string
		rules       []Rule
		cmd         string
		wantAllowed bool
		wantReason  string
	}{
		{
			name:        "empty policy allows all",
			rules:       []Rule{},
			cmd:         "go test ./...",
			wantAllowed: true,
		},
		{
			name:        "single allow rule",
			rules:       []Rule{allowAllRule{}},
			cmd:         "go test ./...",
			wantAllowed: true,
		},
		{
			name:        "single deny rule",
			rules:       []Rule{denyAllRule{reason: "blocked"}},
			cmd:         "go test ./...",
			wantAllowed: false,
			wantReason:  "blocked",
		},
		{
			name:        "first deny wins",
			rules:       []Rule{denyAllRule{reason: "first"}, denyAllRule{reason: "second"}},
			cmd:         "go test ./...",
			wantAllowed: false,
			wantReason:  "first",
		},
		{
			name:        "allow then deny",
			rules:       []Rule{allowAllRule{}, denyAllRule{reason: "denied"}},
			cmd:         "go test ./...",
			wantAllowed: false,
			wantReason:  "denied",
		},
		{
			name:        "deny then allow",
			rules:       []Rule{denyAllRule{reason: "denied"}, allowAllRule{}},
			cmd:         "go test ./...",
			wantAllowed: false,
			wantReason:  "denied",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &Policy{Rules: tt.rules}
			cmd := parser.Parse(tt.cmd)
			got := p.Evaluate(cmd)

			if got.Allowed != tt.wantAllowed {
				t.Errorf("Evaluate() Allowed = %v, want %v", got.Allowed, tt.wantAllowed)
			}
			if got.Reason != tt.wantReason {
				t.Errorf("Evaluate() Reason = %q, want %q", got.Reason, tt.wantReason)
			}
		})
	}
}

func TestPolicyEvaluateNilPolicy(t *testing.T) {
	p := &Policy{}
	cmd := parser.Parse("go test ./...")
	got := p.Evaluate(cmd)

	if !got.Allowed {
		t.Error("nil rules should allow all")
	}
}
