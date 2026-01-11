package policy

import (
	"path/filepath"
	"strings"

	"github.com/adrianpk/watchman/internal/config"
	"github.com/adrianpk/watchman/internal/parser"
)

// writeTools are tools that modify files.
var writeTools = map[string]bool{
	"Write":        true,
	"Edit":         true,
	"NotebookEdit": true,
}

// ScopeToFiles restricts modifications to declared file patterns.
type ScopeToFiles struct {
	Allow []string
	Block []string
}

// NewScopeToFiles creates a scope rule from config.
func NewScopeToFiles(cfg *config.ScopeConfig) *ScopeToFiles {
	if cfg == nil {
		return &ScopeToFiles{}
	}
	return &ScopeToFiles{
		Allow: cfg.Allow,
		Block: cfg.Block,
	}
}

// Evaluate checks if the command modifies files within the defined scope.
func (r *ScopeToFiles) Evaluate(toolName string, cmd parser.Command) Decision {
	if !writeTools[toolName] {
		return Decision{Allowed: true}
	}

	paths := collectPathCandidates(cmd)
	for _, p := range paths {
		if r.isBlocked(p) {
			return Decision{
				Allowed: false,
				Reason:  "path is blocked by scope configuration: " + p,
			}
		}
		if !r.isInScope(p) {
			return Decision{
				Allowed: false,
				Reason:  "path is outside allowed scope: " + p,
			}
		}
	}

	return Decision{Allowed: true}
}

// isBlocked checks if a path matches any block pattern.
func (r *ScopeToFiles) isBlocked(p string) bool {
	for _, pattern := range r.Block {
		if matchGlob(p, pattern) {
			return true
		}
	}
	return false
}

// isInScope checks if a path is within the allowed scope.
// If no allow patterns are defined, all paths are in scope.
func (r *ScopeToFiles) isInScope(p string) bool {
	if len(r.Allow) == 0 {
		return true
	}
	for _, pattern := range r.Allow {
		if matchGlob(p, pattern) {
			return true
		}
	}
	return false
}

// matchGlob matches a path against a glob pattern.
// Supports ** for recursive directory matching.
func matchGlob(path, pattern string) bool {
	path = filepath.Clean(path)
	pattern = filepath.Clean(pattern)

	if strings.Contains(pattern, "**") {
		return matchDoublestar(path, pattern)
	}

	matched, _ := filepath.Match(pattern, path)
	if matched {
		return true
	}

	matched, _ = filepath.Match(pattern, filepath.Base(path))
	return matched
}

// matchDoublestar handles ** glob patterns.
func matchDoublestar(path, pattern string) bool {
	parts := strings.Split(pattern, "**")
	if len(parts) != 2 {
		return false
	}

	prefix := strings.TrimSuffix(parts[0], string(filepath.Separator))
	suffix := strings.TrimPrefix(parts[1], string(filepath.Separator))

	if prefix != "" && !strings.HasPrefix(path, prefix) {
		return false
	}

	if suffix == "" {
		return true
	}

	remaining := path
	if prefix != "" {
		remaining = strings.TrimPrefix(path, prefix)
		remaining = strings.TrimPrefix(remaining, string(filepath.Separator))
	}

	if suffix == "" {
		return true
	}

	pathParts := strings.Split(remaining, string(filepath.Separator))
	for i := range pathParts {
		candidate := strings.Join(pathParts[i:], string(filepath.Separator))
		matched, _ := filepath.Match(suffix, candidate)
		if matched {
			return true
		}
		if len(pathParts[i:]) == 1 {
			matched, _ = filepath.Match(suffix, pathParts[len(pathParts)-1])
			if matched {
				return true
			}
		}
	}

	return false
}
