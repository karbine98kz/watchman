// Package policy provides rule evaluation for command validation.
package policy

import "github.com/adrianpk/watchman/internal/parser"

// Decision represents the result of evaluating a command against rules.
type Decision struct {
	Allowed bool
	Reason  string
}

// Rule evaluates a command and returns a decision.
type Rule interface {
	Evaluate(cmd parser.Command) Decision
}

// Policy holds a set of rules and evaluates commands against them.
type Policy struct {
	Rules []Rule
}

// Evaluate runs all rules against the command. First rule that denies wins.
func (p *Policy) Evaluate(cmd parser.Command) Decision {
	for _, rule := range p.Rules {
		decision := rule.Evaluate(cmd)
		if !decision.Allowed {
			return decision
		}
	}
	return Decision{Allowed: true}
}
