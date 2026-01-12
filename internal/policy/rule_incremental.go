package policy

import (
	"os/exec"
	"strings"

	"github.com/adrianpk/watchman/internal/config"
)

// IncrementalRule enforces limits on the number of modified files.
type IncrementalRule struct {
	MaxFiles  int
	WarnRatio float64
	countFunc func() int // injectable for testing
}

// NewIncrementalRule creates a new incremental change rule.
func NewIncrementalRule(cfg *config.IncrementalConfig) *IncrementalRule {
	if cfg == nil {
		return &IncrementalRule{countFunc: countGitModifiedFiles}
	}
	return &IncrementalRule{
		MaxFiles:  cfg.MaxFiles,
		WarnRatio: cfg.WarnRatio,
		countFunc: countGitModifiedFiles,
	}
}

// Evaluate checks if the current number of modified files exceeds limits.
func (r *IncrementalRule) Evaluate() Decision {
	if r.MaxFiles <= 0 {
		return Decision{Allowed: true}
	}

	count := r.countModifiedFiles()
	if count < 0 {
		// Could not determine, allow to proceed
		return Decision{Allowed: true}
	}

	// Check if at or over max limit
	if count >= r.MaxFiles {
		return Decision{
			Allowed: false,
			Reason:  "maximum modified files reached (" + itoa(count) + "/" + itoa(r.MaxFiles) + "), commit or review changes before continuing",
		}
	}

	// Check if in warning zone
	warnThreshold := r.warnThreshold()
	if count >= warnThreshold {
		return Decision{
			Allowed: true,
			Warning: "approaching file limit: " + itoa(count) + "/" + itoa(r.MaxFiles) + " files modified, consider committing soon",
		}
	}

	return Decision{Allowed: true}
}

// warnThreshold calculates when to start warning.
func (r *IncrementalRule) warnThreshold() int {
	if r.WarnRatio <= 0 || r.WarnRatio >= 1 {
		// Default to 70% if not set or invalid
		return int(float64(r.MaxFiles) * 0.7)
	}
	return int(float64(r.MaxFiles) * r.WarnRatio)
}

// countModifiedFiles uses git status to count modified files.
func (r *IncrementalRule) countModifiedFiles() int {
	if r.countFunc != nil {
		return r.countFunc()
	}
	return countGitModifiedFiles()
}

// countGitModifiedFiles runs git status and counts modified files.
func countGitModifiedFiles() int {
	cmd := exec.Command("git", "status", "--porcelain")
	output, err := cmd.Output()
	if err != nil {
		return -1
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	if len(lines) == 1 && lines[0] == "" {
		return 0
	}

	count := 0
	for _, line := range lines {
		if len(line) >= 2 {
			// Count files that are modified, added, or deleted
			// Status codes: M (modified), A (added), D (deleted), R (renamed), C (copied)
			// First char = staged status, second char = working tree status
			status := line[:2]
			if status != "??" && status != "!!" {
				count++
			}
		}
	}
	return count
}
