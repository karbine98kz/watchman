# Watchman Usage Guide

This is an early proof of concept. The implementation is minimal and the feature set is limited.

## Vision

Watchman explores mechanical enforcement of constraints on LLM-driven code generation. The core idea: soft constraints erode over time, so enforcement must be external and non-negotiable.

Seven archetypal rules were identified during initial design:

| Rule | Status | Description |
|------|--------|-------------|
| **Confine to workspace** | Implemented | No access outside the project directory |
| **Scope to defined files** | Implemented | Limit modifications to explicitly declared files |
| **Version control rules** | Implemented | Commit message format and branch protection |
| **Require incremental changes** | Implemented | Reject large-scale rewrites in favor of small, reviewable diffs |
| **Preserve key invariants** | Planned | Block changes that violate structural rules (naming, architecture) |
| **Match established patterns** | Planned | Ensure new code follows existing conventions |
| **Enforce explicit boundaries** | Planned | Respect module boundaries and dependency rules |

Four rules are currently implemented. See [rules.md](rules.md) for detailed documentation.

## What It Does

Watchman acts as a [Claude Code hook](https://docs.anthropic.com/en/docs/claude-code/hooks) that intercepts filesystem operations before execution. It monitors:

- **Bash** - shell commands
- **Read** - file reading
- **Write** - file creation
- **Edit** - file modification
- **Glob** - file pattern matching
- **Grep** - content searching

Currently, it enforces a single rule:

**Confine to Workspace**: Block any command that references paths outside the current project directory.

This blocks:
- Absolute paths (`/etc/passwd`, `/tmp/file`)
- Parent traversal (`../secrets`, `../../other`)
- Environment variables with absolute paths (`GOMODCACHE=/tmp/mod`)

## Installation

Build the binary:

```bash
go build -o watchman ./cmd/watchman
```

Or install to `$GOBIN`:

```bash
go install ./cmd/watchman
```

## Configuration

Add Watchman as a PreToolUse hook in your Claude Code settings (`~/.claude/settings.json`):

```json
{
  "hooks": {
    "PreToolUse": [
      { "matcher": "Bash", "hooks": [{ "type": "command", "command": "/path/to/watchman" }] },
      { "matcher": "Read", "hooks": [{ "type": "command", "command": "/path/to/watchman" }] },
      { "matcher": "Write", "hooks": [{ "type": "command", "command": "/path/to/watchman" }] },
      { "matcher": "Edit", "hooks": [{ "type": "command", "command": "/path/to/watchman" }] },
      { "matcher": "Glob", "hooks": [{ "type": "command", "command": "/path/to/watchman" }] },
      { "matcher": "Grep", "hooks": [{ "type": "command", "command": "/path/to/watchman" }] }
    ]
  }
}
```

Replace `/path/to/watchman` with the actual binary location.

## Behavior

When Claude Code executes a Bash command, Watchman receives the command as JSON on stdin and responds with a decision.

**Allowed command** (exit 0):
```bash
echo '{"hook_type":"PreToolUse","tool_name":"Bash","tool_input":{"command":"go test ./..."}}' | watchman
# Output: {"decision":"allow"}
# Exit: 0
```

**Blocked command** (exit 2):
```bash
echo '{"hook_type":"PreToolUse","tool_name":"Bash","tool_input":{"command":"cat /etc/passwd"}}' | watchman
# Stderr: cannot access paths outside the project workspace
# Exit: 2
```

## Examples

| Command | Result | Reason |
|---------|--------|--------|
| `go test ./...` | Allowed | Relative path |
| `make build` | Allowed | No paths |
| `cat /etc/passwd` | Blocked | Absolute path |
| `rm -rf /` | Blocked | Absolute path |
| `cat ../secrets` | Blocked | Parent traversal |
| `GOBIN=/usr/local/bin go install` | Blocked | Env var with absolute path |

## Limitations

This is a proof of concept with significant limitations:

- Only the "confine to workspace" rule is implemented
- No configuration file support (rule is hardcoded)
- No allowlist/blocklist for specific paths
- No program-specific rules
- Command parsing handles common cases but may miss edge cases

## Next Steps

See the [README](../README.md) for context on the project direction.
