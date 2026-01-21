package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/adrianpk/watchman/internal/cli"
	"github.com/adrianpk/watchman/internal/config"
	"github.com/adrianpk/watchman/internal/hook"
)

func main() {
	// Handle CLI commands
	if len(os.Args) > 1 {
		if err := runCommand(os.Args[1]); err != nil {
			fatal("%v", err)
		}
		return
	}

	// Run hook evaluation
	if err := runHook(); err != nil {
		fatal("%v", err)
	}
}

func runCommand(cmd string) error {
	switch cmd {
	case "init":
		local := len(os.Args) > 2 && os.Args[2] == "--local"
		return cli.RunInit(local)
	case "setup":
		return cli.RunSetup()
	default:
		return fmt.Errorf("unknown command: %s", cmd)
	}
}

func runHook() error {
	cfg, err := config.Load()
	if err != nil {
		deny("watchman config error: " + err.Error())
		return nil
	}

	evaluator := hook.NewEvaluator(cfg)

	var input hookInput
	if err := json.NewDecoder(os.Stdin).Decode(&input); err != nil {
		deny("watchman input error: " + err.Error())
		return nil
	}

	result := evaluator.Evaluate(hook.Input{
		HookType:  input.HookType,
		ToolName:  input.ToolName,
		ToolInput: input.ToolInput,
	})

	if !result.Allowed {
		deny(result.Reason)
		return nil
	}

	allow(result.Warning)
	return nil
}

type hookInput struct {
	HookType  string                 `json:"hook_type"`
	ToolName  string                 `json:"tool_name"`
	ToolInput map[string]interface{} `json:"tool_input"`
}

type hookOutput struct {
	HookSpecificOutput *hookSpecificOutput `json:"hookSpecificOutput,omitempty"`
}

type hookSpecificOutput struct {
	HookEventName      string `json:"hookEventName"`
	PermissionDecision string `json:"permissionDecision"`
	AdditionalContext  string `json:"additionalContext,omitempty"`
	Reason             string `json:"reason,omitempty"`
}

func allow(additionalContext string) {
	out := hookOutput{
		HookSpecificOutput: &hookSpecificOutput{
			HookEventName:      "PreToolUse",
			PermissionDecision: "allow",
			AdditionalContext:  additionalContext,
		},
	}
	json.NewEncoder(os.Stdout).Encode(out)
	os.Exit(0)
}

func deny(reason string) {
	out := hookOutput{
		HookSpecificOutput: &hookSpecificOutput{
			HookEventName:      "PreToolUse",
			PermissionDecision: "deny",
			Reason:             reason,
		},
	}
	json.NewEncoder(os.Stdout).Encode(out)
	fmt.Fprintln(os.Stderr, reason)
	os.Exit(2)
}

func fatal(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
