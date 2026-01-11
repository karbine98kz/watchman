# Rules

Watchman enforces semantic rules that apply to all tools. Rules block intent, not specific tools.

## Workspace

**Status**: Implemented

Confines the AI agent to the project directory. Blocks access to paths outside the workspace.

### Purpose

Prevents the agent from:
- Reading sensitive files outside the project (`/etc/passwd`, `~/.ssh/`)
- Writing to system directories
- Escaping the project boundary via path traversal (`../`)

### Configuration

```yaml
rules:
  workspace: true

workspace:
  allow:
    - /tmp/
    - ~/.cache/go-build/
  block:
    - .env
    - secrets/
```

### Behavior

| Path | Result |
|------|--------|
| `./src/main.go` | Allowed |
| `/etc/passwd` | Blocked |
| `../other-project/` | Blocked |
| `/tmp/test.txt` | Allowed (if in allow list) |
| `.env` | Blocked (if in block list) |

### Protected Paths

Some paths are always protected regardless of configuration:

- `~/.claude/` - Claude settings and hooks
- `~/.ssh/` - SSH keys
- `~/.aws/` - AWS credentials
- `~/.gnupg/` - GPG keys
- `~/.config/watchman/` - Watchman global config
- `.watchman.yml` - Local config (any directory)

These cannot be overridden.

---

## Scope

**Status**: Implemented

Limits modifications to explicitly declared files within the workspace.

### Purpose

Even within the workspace, restrict which files can be modified. Useful for:
- Focusing changes on specific files
- Protecting generated code
- Preventing accidental edits to unrelated files

### Configuration

```yaml
rules:
  scope: true

scope:
  allow:
    - src/**/*.go
    - go.mod
  block:
    - vendor/**
    - **/*_generated.go
```

### Behavior

| Tool | Scope Applied |
|------|--------------|
| `Read` | No (read-only) |
| `Glob` | No (read-only) |
| `Grep` | No (read-only) |
| `Bash` | No |
| `Write` | Yes |
| `Edit` | Yes |
| `NotebookEdit` | Yes |

### Pattern Matching

Supports glob patterns:
- `*` - matches any characters except path separator
- `**` - matches any characters including path separator (recursive)
- `?` - matches any single character
- `[abc]` - matches character class

| Path | Pattern | Match |
|------|---------|-------|
| `src/main.go` | `src/**/*.go` | Yes |
| `src/pkg/util.go` | `src/**/*.go` | Yes |
| `vendor/lib.go` | `src/**/*.go` | No |
| `types_generated.go` | `**/*_generated.go` | Yes |
| `.env` | `.env` | Yes |

### Rules

1. If no `allow` patterns defined, all paths are allowed
2. If `allow` patterns defined, path must match at least one
3. `block` patterns take precedence over `allow`

---

## Incremental

**Status**: Planned

Rejects large-scale rewrites in favor of small, incremental changes.

---

## Invariants

**Status**: Planned

Blocks changes that violate structural rules defined for the project.

---

## Patterns

**Status**: Planned

Ensures new code follows existing conventions and patterns.

---

## Boundaries

**Status**: Planned

Respects module boundaries and dependency rules.
