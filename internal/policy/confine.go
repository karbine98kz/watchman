package policy

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/adrianpk/watchman/internal/config"
	"github.com/adrianpk/watchman/internal/parser"
)

// alwaysProtected contains paths that are NEVER accessible, regardless of config.
// This is a hardcoded security boundary that cannot be overridden.
var alwaysProtected = []string{
	"~/.claude/",          // Claude settings, hooks
	"~/.ssh/",             // SSH keys
	"~/.aws/",             // AWS credentials
	"~/.gnupg/",           // GPG keys
	"~/.gpg/",             // GPG keys (alt)
	"~/.config/gh/",       // GitHub CLI credentials
	"~/.config/watchman/", // Watchman global config
	"~/.netrc",            // Network credentials
	"~/.git-credentials",  // Git credentials
	"~/go/bin/watchman",   // Watchman binary
}

// protectedFilenames are filenames that are protected in any directory.
var protectedFilenames = []string{
	".watchman.yml", // Local watchman config
}

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

// IsAlwaysProtected checks if a path matches any hardcoded protected path.
// This check cannot be overridden by configuration.
// Exported so main.go can call it regardless of workspace rule.
func IsAlwaysProtected(p string) bool {
	if p == "" {
		return false
	}

	absPath := p
	if !filepath.IsAbs(p) {
		if cwd, err := os.Getwd(); err == nil {
			absPath = filepath.Clean(filepath.Join(cwd, p))
		}
	} else {
		absPath = filepath.Clean(p)
	}

	filename := filepath.Base(absPath)
	for _, protected := range protectedFilenames {
		if filename == protected {
			return true
		}
	}

	for _, pattern := range alwaysProtected {
		isDir := strings.HasSuffix(pattern, "/")

		expandedPattern := strings.TrimSuffix(pattern, "/")
		if strings.HasPrefix(expandedPattern, "~/") {
			if home, err := userHomeDir(); err == nil {
				expandedPattern = filepath.Join(home, expandedPattern[2:])
			}
		}

		if isDir {
			if absPath == expandedPattern || strings.HasPrefix(absPath, expandedPattern+string(filepath.Separator)) {
				return true
			}
		} else if absPath == expandedPattern {
			return true
		}
	}

	return false
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

// matchPath checks if a path matches a pattern.
// Supports exact match and prefix match (pattern ending with /).
func matchPath(path, pattern string) bool {
	if strings.HasPrefix(pattern, "~/") {
		if home, err := userHomeDir(); err == nil {
			pattern = filepath.Join(home, pattern[2:])
		}
	}

	if path == pattern {
		return true
	}

	// Prefix match for directories (pattern ends with /)
	if strings.HasSuffix(pattern, "/") {
		return strings.HasPrefix(path, pattern) || path == strings.TrimSuffix(pattern, "/")
	}

	// Prefix match (path starts with pattern)
	if strings.HasPrefix(path, pattern+"/") || strings.HasPrefix(path, pattern+string(filepath.Separator)) {
		return true
	}

	return false
}

func userHomeDir() (string, error) {
	return os.UserHomeDir()
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
