# Configuration

Watchman uses a two-level configuration system: global defaults and local overrides.

## Configuration Files

| Location | Purpose |
|----------|---------|
| `~/.config/watchman/config.yml` | Global defaults for all projects |
| `.watchman.yml` | Local overrides in project directory |

Local configuration can relax or tighten global rules per project.

## Quick Setup

Create config files with the `init` command:

```bash
# Create global config (~/.config/watchman/config.yml)
watchman init

# Create local config in current project (.watchman.yml)
watchman init --local
```

If a config file already exists, the command does nothing.

## Structure

```yaml
version: 1

rules:
  workspace: true
  scope: false
  incremental: false
  invariants: false
  patterns: false
  boundaries: false

workspace:
  allow: []
  block: []

commands:
  block: []

tools:
  allow: []
  block: []
```

## Rules

Semantic rules apply to ALL tools. Blocking a rule blocks the *intent*, regardless of which tool attempts it.

| Rule | Key | Description | Status |
|------|-----|-------------|--------|
| Confine to workspace | `workspace` | No access outside the project directory | Implemented |
| Scope to defined files | `scope` | Limit modifications to explicitly declared files | Planned |
| Require incremental changes | `incremental` | Reject large-scale rewrites in favor of small diffs | Planned |
| Preserve key invariants | `invariants` | Block changes that violate structural rules | Planned |
| Match established patterns | `patterns` | Ensure new code follows existing conventions | Planned |
| Enforce explicit boundaries | `boundaries` | Respect module boundaries and dependency rules | Planned |

## Workspace Rule

Controls access to paths outside the project directory.

```yaml
workspace:
  # Paths allowed outside workspace
  allow:
    - /tmp
    - ~/.cache/go-build

  # Paths blocked even inside workspace
  block:
    - .env
    - secrets/
```

## Commands Control

Blocks destructive shell commands regardless of other rules. This is a safety layer for commands that are never legitimate. Applies to the `Bash` tool regardless of the underlying shell (bash, zsh, fish, PowerShell, etc.).

```yaml
commands:
  block:
    - sudo
    - "rm -rf /"
    - chmod
    - chown
    - mkfs
    - "> /dev/"
```

## Tools Control

Optional layer to restrict which tools the agent can use.

```yaml
tools:
  # If empty, all tools allowed
  # If set, ONLY these tools allowed
  allow: []

  # Always blocked
  block: []
```

Available tools: `Bash`, `Read`, `Write`, `Edit`, `Glob`, `Grep`

## Local Overrides

`.watchman.yml` in project root overrides global settings:

```yaml
# Relax workspace for this project
workspace:
  allow:
    - /usr/local/include

# Enable scope rule for this project
rules:
  scope: true

scope:
  files:
    - src/**/*.go
    - internal/**/*.go
```

## Precedence

**Simple rule**: If `.watchman.yml` exists in the project, it is used exclusively. Global config is ignored.

- No merging occurs
- No inheritance
- What you see in local is what you get

This keeps behavior predictable and easy to reason about.

## Examples

### Global: Strict defaults

```yaml
# ~/.config/watchman/config.yml
version: 1

rules:
  workspace: true

commands:
  block:
    - sudo
    - "rm -rf"
    - chmod
    - chown
```

### Local: Relax for specific project

```yaml
# /path/to/project/.watchman.yml
workspace:
  allow:
    - /tmp/test-fixtures
```

### Local: Maximum restriction

```yaml
# /path/to/project/.watchman.yml
rules:
  workspace: true
  scope: true

scope:
  files:
    - src/**/*.ts
    - package.json

tools:
  allow:
    - Read
    - Edit
    - Glob
  block:
    - Bash
```
