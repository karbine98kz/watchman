package policy

import (
	"path/filepath"
	"strings"

	"github.com/adrianpk/watchman/internal/parser"
)

// ConfineToWorkspace blocks commands that attempt to access paths outside the project.
type ConfineToWorkspace struct{}

// Evaluate checks if the command attempts to access paths outside the workspace.
func (r ConfineToWorkspace) Evaluate(cmd parser.Command) Decision {
	candidates := collectPathCandidates(cmd)

	for _, p := range candidates {
		if ViolatesWorkspaceBoundary(p) {
			return Decision{
				Allowed: false,
				Reason:  "cannot access paths outside the project workspace",
			}
		}
	}

	return Decision{Allowed: true}
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
func ViolatesWorkspaceBoundary(p string) bool {
	if p == "" {
		return false
	}

	if filepath.IsAbs(p) {
		return true
	}

	clean := filepath.Clean(p)

	if clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return true
	}

	return false
}
