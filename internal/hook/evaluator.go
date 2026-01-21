// Package hook provides the core hook evaluation logic.
package hook

import (
	"os"
	"strings"

	"github.com/adrianpk/watchman/internal/config"
	"github.com/adrianpk/watchman/internal/parser"
	"github.com/adrianpk/watchman/internal/policy"
	"github.com/adrianpk/watchman/internal/state"
)

// Input represents the hook input from Claude Code.
type Input struct {
	HookType  string
	ToolName  string
	ToolInput map[string]interface{}
}

// Result represents the evaluation result.
type Result struct {
	Allowed bool
	Reason  string
	Warning string
}

// Evaluator evaluates hook inputs against configured rules.
type Evaluator struct {
	cfg          *config.Config
	hookMatcher  *HookMatcher
	hookExec     *HookExecutor
	stateManager *state.Manager
}

// NewEvaluator creates a new hook evaluator.
func NewEvaluator(cfg *config.Config) *Evaluator {
	sm := state.NewManager()
	_ = sm.Load() // Ignore error, use fresh state if load fails

	return &Evaluator{
		cfg:          cfg,
		hookMatcher:  NewHookMatcher(),
		hookExec:     NewHookExecutor(),
		stateManager: sm,
	}
}

// Evaluate processes the hook input and returns a result.
func (e *Evaluator) Evaluate(input Input) Result {
	// Check tool blocklist
	if e.isToolBlocked(input.ToolName) {
		return Result{Allowed: false, Reason: "tool is blocked by configuration: " + input.ToolName}
	}

	// Check tool allowlist
	if !e.isToolAllowed(input.ToolName) {
		return Result{Allowed: false, Reason: "tool is not in allowed list: " + input.ToolName}
	}

	// Non-filesystem tools are always allowed (but still track reminders)
	if !isFilesystemTool(input.ToolName) {
		return e.withReminders(Result{Allowed: true})
	}

	// Check command blocklist for Bash
	if input.ToolName == "Bash" {
		if cmd, ok := input.ToolInput["command"].(string); ok {
			if blocked := e.isCommandBlocked(cmd); blocked != "" {
				return Result{Allowed: false, Reason: "command is blocked by configuration: " + blocked}
			}
		}
	}

	// Check protected paths
	paths := ExtractPaths(input.ToolName, input.ToolInput)
	for _, p := range paths {
		if policy.IsAlwaysProtected(p) {
			return Result{Allowed: false, Reason: "path is protected and cannot be accessed. User must perform this action manually."}
		}
	}

	// Apply workspace rule
	if e.cfg.Rules.Workspace {
		if result := e.evaluateWorkspace(input); !result.Allowed {
			return result
		}
	}

	// Apply scope rule
	if e.cfg.Rules.Scope {
		if result := e.evaluateScope(input); !result.Allowed {
			return result
		}
	}

	// Apply versioning rule
	if e.cfg.Rules.Versioning && input.ToolName == "Bash" {
		if result := e.evaluateVersioning(input); !result.Allowed {
			return result
		}
	}

	// Apply incremental rule
	if e.cfg.Rules.Incremental && isModificationTool(input.ToolName) {
		if result := e.evaluateIncremental(); !result.Allowed {
			return result
		} else if result.Warning != "" {
			return e.withReminders(result)
		}
	}

	// Apply invariants rule
	if e.cfg.Rules.Invariants && isModificationTool(input.ToolName) {
		if result := e.evaluateInvariants(input); !result.Allowed {
			return result
		}
	}

	// Apply external hooks
	if len(e.cfg.Hooks) > 0 {
		if result := e.evaluateHooks(input); !result.Allowed {
			return result
		} else if result.Warning != "" {
			return e.withReminders(result)
		}
	}

	// Check reminders (post-execution, always runs for allowed operations)
	return e.evaluateReminders()
}

func (e *Evaluator) evaluateWorkspace(input Input) Result {
	rule := policy.NewConfineToWorkspace(&e.cfg.Workspace)
	paths := ExtractPaths(input.ToolName, input.ToolInput)
	for _, p := range paths {
		parsed := parser.Command{Args: []string{p}}
		decision := rule.Evaluate(parsed)
		if !decision.Allowed {
			return Result{Allowed: false, Reason: decision.Reason}
		}
	}
	return Result{Allowed: true}
}

func (e *Evaluator) evaluateScope(input Input) Result {
	rule := policy.NewScopeToFiles(&e.cfg.Scope)
	paths := ExtractPaths(input.ToolName, input.ToolInput)
	for _, p := range paths {
		parsed := parser.Command{Args: []string{p}}
		decision := rule.Evaluate(input.ToolName, parsed)
		if !decision.Allowed {
			return Result{Allowed: false, Reason: decision.Reason}
		}
	}
	return Result{Allowed: true}
}

func (e *Evaluator) evaluateVersioning(input Input) Result {
	cmd, ok := input.ToolInput["command"].(string)
	if !ok {
		return Result{Allowed: true}
	}
	rule := policy.NewVersioningRule(&e.cfg.Versioning)
	decision := rule.Evaluate(cmd)
	return Result{Allowed: decision.Allowed, Reason: decision.Reason}
}

func (e *Evaluator) evaluateIncremental() Result {
	rule := policy.NewIncrementalRule(&e.cfg.Incremental)
	decision := rule.Evaluate()
	return Result{Allowed: decision.Allowed, Reason: decision.Reason, Warning: decision.Warning}
}

func (e *Evaluator) evaluateInvariants(input Input) Result {
	rule := policy.NewInvariantsRule(&e.cfg.Invariants)
	paths := ExtractPaths(input.ToolName, input.ToolInput)

	// Get content for content-based checks
	content := ""
	if c, ok := input.ToolInput["content"].(string); ok {
		content = c
	}

	for _, p := range paths {
		decision := rule.Evaluate(input.ToolName, p, content)
		if !decision.Allowed {
			return Result{Allowed: false, Reason: decision.Reason}
		}
	}
	return Result{Allowed: true}
}

func (e *Evaluator) evaluateHooks(input Input) Result {
	paths := ExtractPaths(input.ToolName, input.ToolInput)

	cwd, err := os.Getwd()
	if err != nil {
		cwd = ""
	}

	hookInput := HookInput{
		ToolName:   input.ToolName,
		ToolInput:  input.ToolInput,
		Paths:      paths,
		WorkingDir: cwd,
	}

	var warnings []string

	for i := range e.cfg.Hooks {
		hookCfg := &e.cfg.Hooks[i]

		if !e.hookMatcher.Matches(hookCfg, input.ToolName, paths) {
			continue
		}

		result := e.hookExec.Execute(hookCfg, hookInput)

		if !result.Allowed {
			return Result{
				Allowed: false,
				Reason:  hookCfg.Name + ": " + result.Reason,
			}
		}

		if result.Warning != "" {
			warnings = append(warnings, hookCfg.Name+": "+result.Warning)
		}
	}

	if len(warnings) > 0 {
		return Result{Allowed: true, Warning: strings.Join(warnings, "; ")}
	}

	return Result{Allowed: true}
}

func (e *Evaluator) evaluateReminders() Result {
	if len(e.cfg.Reminders) == 0 {
		return Result{Allowed: true}
	}

	// Increment task count
	e.stateManager.IncrementTaskCount()

	// Check if any reminders should trigger
	triggered := e.stateManager.CheckReminders(e.cfg.Reminders)

	// Save state (ignore errors, non-critical)
	_ = e.stateManager.Save()

	if len(triggered) > 0 {
		return Result{
			Allowed: true,
			Warning: strings.Join(triggered, "; "),
		}
	}

	return Result{Allowed: true}
}

// withReminders combines a result with any triggered reminders.
// Should be called for all allowed operations to ensure reminders are tracked.
func (e *Evaluator) withReminders(result Result) Result {
	if !result.Allowed {
		return result
	}

	reminderResult := e.evaluateReminders()
	if reminderResult.Warning != "" {
		if result.Warning != "" {
			result.Warning = result.Warning + "; " + reminderResult.Warning
		} else {
			result.Warning = reminderResult.Warning
		}
	}
	return result
}

func (e *Evaluator) isToolBlocked(tool string) bool {
	for _, t := range e.cfg.Tools.Block {
		if strings.EqualFold(t, tool) {
			return true
		}
	}
	return false
}

func (e *Evaluator) isToolAllowed(tool string) bool {
	if len(e.cfg.Tools.Allow) == 0 {
		return true
	}
	for _, t := range e.cfg.Tools.Allow {
		if strings.EqualFold(t, tool) {
			return true
		}
	}
	return false
}

func (e *Evaluator) isCommandBlocked(cmd string) string {
	for _, pattern := range e.cfg.Commands.Block {
		// Patterns with spaces (like "rm -rf /") use substring matching
		if strings.Contains(pattern, " ") {
			if strings.Contains(cmd, pattern) {
				return pattern
			}
			continue
		}

		// Single-word patterns match only in command position
		if isCommandInPosition(cmd, pattern) {
			return pattern
		}
	}
	return ""
}

// isCommandInPosition checks if pattern appears as an actual command
// (first token of a pipeline/chain segment), not as an argument.
func isCommandInPosition(cmd, pattern string) bool {
	// Split by command separators: |, &&, ||, ;
	// We iterate through segments to find command positions
	segments := splitCommandSegments(cmd)

	for _, seg := range segments {
		seg = strings.TrimSpace(seg)
		if seg == "" {
			continue
		}

		// Extract the command (first token)
		command := extractCommandName(seg)
		if command == pattern {
			return true
		}
	}
	return false
}

// splitCommandSegments splits a shell command by |, &&, ||, ;
func splitCommandSegments(cmd string) []string {
	var segments []string
	var current strings.Builder
	i := 0

	for i < len(cmd) {
		ch := cmd[i]

		switch ch {
		case '|':
			segments = append(segments, current.String())
			current.Reset()
			// Skip || (treat as single separator)
			if i+1 < len(cmd) && cmd[i+1] == '|' {
				i++
			}
		case '&':
			if i+1 < len(cmd) && cmd[i+1] == '&' {
				segments = append(segments, current.String())
				current.Reset()
				i++ // Skip second &
			} else {
				// Background &, still part of current segment
				current.WriteByte(ch)
			}
		case ';':
			segments = append(segments, current.String())
			current.Reset()
		case '\'', '"':
			// Skip quoted strings entirely
			quote := ch
			current.WriteByte(ch)
			i++
			for i < len(cmd) && cmd[i] != quote {
				if cmd[i] == '\\' && i+1 < len(cmd) {
					current.WriteByte(cmd[i])
					i++
				}
				if i < len(cmd) {
					current.WriteByte(cmd[i])
					i++
				}
			}
			if i < len(cmd) {
				current.WriteByte(cmd[i])
			}
		default:
			current.WriteByte(ch)
		}
		i++
	}

	if current.Len() > 0 {
		segments = append(segments, current.String())
	}

	return segments
}

// extractCommandName extracts the actual command from a segment.
// Handles: VAR=value cmd, env cmd, leading spaces, etc.
func extractCommandName(segment string) string {
	segment = strings.TrimSpace(segment)
	tokens := tokenize(segment)

	for _, tok := range tokens {
		// Skip environment variable assignments (VAR=value)
		if strings.Contains(tok, "=") && !strings.HasPrefix(tok, "-") {
			continue
		}
		// Return first non-assignment token as the command
		return tok
	}
	return ""
}

// tokenize splits a command segment into space-separated tokens,
// respecting quotes.
func tokenize(s string) []string {
	var tokens []string
	var current strings.Builder
	inQuote := byte(0)

	for i := 0; i < len(s); i++ {
		ch := s[i]

		if inQuote != 0 {
			if ch == inQuote {
				inQuote = 0
			} else {
				current.WriteByte(ch)
			}
			continue
		}

		switch ch {
		case '\'', '"':
			inQuote = ch
		case ' ', '\t':
			if current.Len() > 0 {
				tokens = append(tokens, current.String())
				current.Reset()
			}
		default:
			current.WriteByte(ch)
		}
	}

	if current.Len() > 0 {
		tokens = append(tokens, current.String())
	}

	return tokens
}

var filesystemTools = map[string]bool{
	"Bash":  true,
	"Read":  true,
	"Write": true,
	"Edit":  true,
	"Glob":  true,
	"Grep":  true,
}

func isFilesystemTool(tool string) bool {
	return filesystemTools[tool]
}

func isModificationTool(tool string) bool {
	switch tool {
	case "Write", "Edit", "NotebookEdit":
		return true
	}
	return false
}
