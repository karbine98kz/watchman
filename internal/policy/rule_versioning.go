package policy

import (
	"regexp"
	"strings"
	"unicode"

	"github.com/adrianpk/watchman/internal/config"
)

// VersioningRule validates commit messages and branch protection.
type VersioningRule struct {
	Commit     config.CommitConfig
	Branches   config.BranchesConfig
	Operations config.OperationsConfig
	Workflow   string
	Tool       string
}

// NewVersioningRule creates a versioning rule from config.
func NewVersioningRule(cfg *config.VersioningConfig) *VersioningRule {
	if cfg == nil {
		return &VersioningRule{}
	}
	return &VersioningRule{
		Commit:     cfg.Commit,
		Branches:   cfg.Branches,
		Operations: cfg.Operations,
		Workflow:   cfg.Workflow,
		Tool:       cfg.Tool,
	}
}

// Evaluate checks if a git/jj command is allowed.
func (r *VersioningRule) Evaluate(command string) Decision {
	if !isGitCommand(command) {
		return Decision{Allowed: true}
	}

	if blocked := r.isBlockedOperation(command); blocked != "" {
		return Decision{
			Allowed: false,
			Reason:  "operation blocked by configuration: " + blocked,
		}
	}

	if reason := r.violatesWorkflow(command); reason != "" {
		return Decision{
			Allowed: false,
			Reason:  reason,
		}
	}

	if isCommitCommand(command) {
		return r.EvaluateCommit(command)
	}

	return Decision{Allowed: true}
}

func (r *VersioningRule) violatesWorkflow(cmd string) string {
	switch r.Workflow {
	case "linear":
		if strings.Contains(cmd, "git merge") || strings.Contains(cmd, "jj merge") {
			return "workflow is linear: use rebase instead of merge"
		}
	case "merge":
		if strings.Contains(cmd, "git rebase") || strings.Contains(cmd, "jj rebase") {
			return "workflow is merge-based: use merge instead of rebase"
		}
	}
	return ""
}

func (r *VersioningRule) isBlockedOperation(cmd string) string {
	for _, op := range r.Operations.Block {
		if strings.Contains(cmd, op) {
			return op
		}
	}
	return ""
}

func isGitCommand(cmd string) bool {
	return strings.Contains(cmd, "git ") || strings.Contains(cmd, "jj ")
}

// EvaluateCommit checks if a commit command is allowed.
func (r *VersioningRule) EvaluateCommit(command string) Decision {
	if !isCommitCommand(command) {
		return Decision{Allowed: true}
	}

	if r.Tool == "jj" && strings.Contains(command, "git commit") {
		return Decision{
			Allowed: false,
			Reason:  "prefer jj over git: use 'jj commit' instead of 'git commit'",
		}
	}

	branch := extractBranchFromCommand(command)
	if r.isProtectedBranch(branch) {
		return Decision{
			Allowed: false,
			Reason:  "cannot commit directly to protected branch: " + branch,
		}
	}

	message := extractCommitMessage(command)
	if message == "" {
		return Decision{Allowed: true}
	}

	if r.Commit.MaxLength > 0 && len(message) > r.Commit.MaxLength {
		return Decision{
			Allowed: false,
			Reason:  "commit message exceeds max length of " + itoa(r.Commit.MaxLength),
		}
	}

	if r.Commit.RequireUppercase && len(message) > 0 {
		first := rune(message[0])
		if !unicode.IsUpper(first) && unicode.IsLetter(first) {
			return Decision{
				Allowed: false,
				Reason:  "commit message must start with uppercase letter",
			}
		}
	}

	if r.Commit.NoPeriod && strings.HasSuffix(message, ".") {
		return Decision{
			Allowed: false,
			Reason:  "commit message must not end with period",
		}
	}

	if r.Commit.RequirePeriod && !strings.HasSuffix(message, ".") {
		return Decision{
			Allowed: false,
			Reason:  "commit message must end with period",
		}
	}

	if r.Commit.SingleLine && strings.Contains(message, "\n") {
		return Decision{
			Allowed: false,
			Reason:  "commit message must be single line (no body)",
		}
	}

	if r.Commit.ForbidColons && strings.Contains(message, ":") {
		return Decision{
			Allowed: false,
			Reason:  "commit message must not contain colons (no conventional commit prefixes)",
		}
	}

	if r.Commit.PrefixPattern != "" {
		re, err := regexp.Compile("^" + r.Commit.PrefixPattern)
		if err == nil && !re.MatchString(message) {
			return Decision{
				Allowed: false,
				Reason:  "commit message must match prefix pattern: " + r.Commit.PrefixPattern,
			}
		}
	}

	return Decision{Allowed: true}
}

func (r *VersioningRule) isProtectedBranch(branch string) bool {
	for _, p := range r.Branches.Protected {
		if p == branch {
			return true
		}
	}
	return false
}

func isCommitCommand(cmd string) bool {
	return strings.Contains(cmd, "git commit") || strings.Contains(cmd, "jj commit")
}

func extractBranchFromCommand(cmd string) string {
	if strings.Contains(cmd, " -b ") {
		parts := strings.Split(cmd, " -b ")
		if len(parts) > 1 {
			fields := strings.Fields(parts[1])
			if len(fields) > 0 {
				return fields[0]
			}
		}
	}
	return ""
}

func extractCommitMessage(cmd string) string {
	patterns := []string{" -m ", " --message ", " --message=", " -m="}

	for _, p := range patterns {
		if idx := strings.Index(cmd, p); idx != -1 {
			rest := cmd[idx+len(p):]
			return extractQuotedOrWord(rest)
		}
	}

	if strings.Contains(cmd, "<<") {
		return extractHeredocMessage(cmd)
	}

	return ""
}

func extractQuotedOrWord(s string) string {
	s = strings.TrimSpace(s)
	if len(s) == 0 {
		return ""
	}

	if s[0] == '"' {
		end := findClosingQuote(s[1:], '"')
		if end > 0 {
			return s[1 : end+1]
		}
	}

	if s[0] == '\'' {
		end := findClosingQuote(s[1:], '\'')
		if end > 0 {
			return s[1 : end+1]
		}
	}

	if strings.HasPrefix(s, "\"$(cat <<") {
		return extractHeredocFromCat(s)
	}

	fields := strings.Fields(s)
	if len(fields) > 0 {
		return fields[0]
	}
	return ""
}

func findClosingQuote(s string, quote rune) int {
	escaped := false
	for i, c := range s {
		if escaped {
			escaped = false
			continue
		}
		if c == '\\' {
			escaped = true
			continue
		}
		if c == quote {
			return i
		}
	}
	return -1
}

func extractHeredocMessage(cmd string) string {
	idx := strings.Index(cmd, "<<")
	if idx == -1 {
		return ""
	}

	rest := strings.TrimSpace(cmd[idx+2:])

	delimiter := ""
	if strings.HasPrefix(rest, "'") {
		end := strings.Index(rest[1:], "'")
		if end > 0 {
			delimiter = rest[1 : end+1]
			rest = rest[end+2:]
		}
	} else {
		fields := strings.Fields(rest)
		if len(fields) > 0 {
			delimiter = fields[0]
			rest = strings.TrimPrefix(rest, delimiter)
		}
	}

	if delimiter == "" {
		return ""
	}

	rest = strings.TrimSpace(rest)
	endIdx := strings.Index(rest, delimiter)
	if endIdx > 0 {
		return strings.TrimSpace(rest[:endIdx])
	}

	return ""
}

func extractHeredocFromCat(s string) string {
	start := strings.Index(s, "<<")
	if start == -1 {
		return ""
	}

	rest := s[start+2:]
	if strings.HasPrefix(rest, "'") {
		rest = rest[1:]
	}

	delimEnd := strings.IndexAny(rest, "'\n")
	if delimEnd == -1 {
		return ""
	}

	delimiter := rest[:delimEnd]
	rest = rest[delimEnd+1:]

	endIdx := strings.Index(rest, delimiter)
	if endIdx > 0 {
		return strings.TrimSpace(rest[:endIdx])
	}

	return ""
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var digits []byte
	for n > 0 {
		digits = append([]byte{byte('0' + n%10)}, digits...)
		n /= 10
	}
	return string(digits)
}
