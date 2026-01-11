package policy

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/adrianpk/watchman/internal/config"
	"github.com/adrianpk/watchman/internal/parser"
)

// ConfineToWorkspace blocks commands that attempt to access paths outside the project.
type ConfineToWorkspace struct {
	Allow []string
	Block []string
}

// NewConfineToWorkspace creates a workspace rule from config.
func NewConfineToWorkspace(cfg *config.WorkspaceConfig) *ConfineToWorkspace {
	if cfg == nil {
		return &ConfineToWorkspace{}
	}
	return &ConfineToWorkspace{
		Allow: cfg.Allow,
		Block: cfg.Block,
	}
}

// Evaluate checks if the command attempts to access paths outside the workspace.
func (r *ConfineToWorkspace) Evaluate(cmd parser.Command) Decision {
	candidates := collectPathCandidates(cmd)

	for _, p := range candidates {
		if IsAlwaysProtected(p) {
			return Decision{
				Allowed: false,
				Reason:  "path is protected and cannot be accessed. User must perform this action manually.",
			}
		}
		if r.isBlocked(p) {
			return Decision{
				Allowed: false,
				Reason:  "path is blocked by configuration: " + p,
			}
		}
		if r.violatesBoundary(p) {
			return Decision{
				Allowed: false,
				Reason:  "cannot access paths outside the project workspace",
			}
		}
	}

	return Decision{Allowed: true}
}

// isBlocked checks if a path matches any block pattern.
func (r *ConfineToWorkspace) isBlocked(p string) bool {
	for _, pattern := range r.Block {
		if matchPath(p, pattern) {
			return true
		}
	}
	return false
}

// isAllowed checks if a path matches any allow pattern.
func (r *ConfineToWorkspace) isAllowed(p string) bool {
	for _, pattern := range r.Allow {
		if matchPath(p, pattern) {
			return true
		}
	}
	return false
}

// violatesBoundary checks if a path escapes the workspace,
// considering allow list exceptions.
func (r *ConfineToWorkspace) violatesBoundary(p string) bool {
	if p == "" {
		return false
	}

	cwd, err := os.Getwd()
	if err != nil {
		return true // fail closed
	}

	var absPath string
	if filepath.IsAbs(p) {
		absPath = filepath.Clean(p)
	} else {
		absPath = filepath.Clean(filepath.Join(cwd, p))
	}

	cwdClean := filepath.Clean(cwd)
	isInside := absPath == cwdClean || strings.HasPrefix(absPath, cwdClean+string(filepath.Separator))

	if isInside {
		return false
	}

	if r.isAllowed(p) {
		return false
	}

	return true
}

func collectPathCandidates(cmd parser.Command) []string {
	var out []string

	out = append(out, cmd.Args...)

	for _, v := range cmd.Flags {
		if v != "" {
			out = append(out, v)
		}
	}

	for _, v := range cmd.Env {
		out = append(out, v)
	}

	return out
}

// ViolatesWorkspaceBoundary checks if a path escapes the workspace.
// This is the legacy function for backward compatibility.
func ViolatesWorkspaceBoundary(p string) bool {
	rule := &ConfineToWorkspace{}
	return rule.violatesBoundary(p)
}
