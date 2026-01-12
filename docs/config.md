# Configuration

Watchman uses a two-level configuration system: global defaults and local overrides.

## Configuration Files

| Location | Purpose |
|----------|---------|
| `~/.config/watchman/config.yml` | Global defaults for all projects |
| `.watchman.yml` | Local overrides in project directory |

Local configuration can relax or tighten global rules per project.

## Quick Setup

Configure Claude Code hook with the `setup` command:

```bash
# Add watchman hook to ~/.claude/settings.json
watchman setup
```

Create config files with the `init` command:

```bash
# Create global config (~/.config/watchman/config.yml)
watchman init

# Create local config in current project (.watchman.yml)
watchman init --local
```

Both commands are idempotent - they do nothing if already configured.

## Structure

```yaml
version: 1

rules:
  workspace: true
  scope: false
  versioning: false
  incremental: false
  invariants: false
  patterns: false
  boundaries: false

workspace:
  allow: []
  block: []

scope:
  allow: []
  block: []

versioning:
  commit:
    max_length: 0
    require_uppercase: false
    no_period: false
    single_line: false
    forbid_colons: false
    prefix_pattern: ""
  branches:
    protected: []
  operations:
    block: []
  workflow: ""
  tool: ""

incremental:
  max_files: 0
  warn_ratio: 0.7

invariants:
  coexistence: []
  content: []
  imports: []
  naming: []
  required: []

commands:
  block: []

tools:
  allow: []
  block: []

hooks: []
```

## Rules

| Rule | Status |
|------|--------|
| `workspace` | Implemented |
| `scope` | Implemented |
| `versioning` | Implemented |
| `incremental` | Implemented |
| `invariants` | Implemented |
| `patterns` | Via Hooks |
| `boundaries` | Via Hooks |

Semantic rules apply to ALL tools. Blocking a rule blocks the *intent*, regardless of which tool attempts it.

| Rule | Key | Description | Status |
|------|-----|-------------|--------|
| Confine to workspace | `workspace` | No access outside the project directory | Implemented |
| Scope to defined files | `scope` | Limit modifications to explicitly declared files | Implemented |
| Version control rules | `versioning` | Commit message format and branch protection | Implemented |
| Require incremental changes | `incremental` | Reject large-scale rewrites in favor of small diffs | Implemented |
| Preserve key invariants | `invariants` | Declarative structural checks (regex/glob) | Implemented |
| External hooks | `hooks` | Execute custom validation via external programs | Implemented |
| Match established patterns | `patterns` | Ensure new code follows existing conventions | Via Hooks |
| Enforce explicit boundaries | `boundaries` | Respect module boundaries and dependency rules | Via Hooks |

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

## Incremental Rule

Limits how many files can be modified before requiring a commit or review.

```yaml
incremental:
  # Maximum modified files (0 = unlimited)
  max_files: 10

  # Warn at this ratio of max (0.7 = warn at 70%)
  warn_ratio: 0.7
```

Uses `git status` to track modified files. Warnings give the agent runway to wrap up; blocking forces a decision.

## Invariants Rule

Declarative structural checks using regex and glob patterns. Language-agnostic, no AST parsing. See [Invariants](invariants.md) for full documentation.

```yaml
rules:
  invariants: true

invariants:
  # Ensure related files exist together
  coexistence:
    - name: "test-requires-impl"
      if: "**/*_test.go"
      require: "${base}.go"
      message: "Test requires implementation file"

  # Validate file content
  content:
    - name: "no-todos"
      paths: ["**/*.go", "!**/*_test.go"]
      forbid: "TODO|FIXME"

  # Check import statements (regex-based)
  imports:
    - name: "no-internal-in-adapters"
      paths: ["adapters/**/*.go"]
      forbid: '".*internal/core"'

  # Validate file naming
  naming:
    - name: "cmd-main-only"
      paths: ["cmd/**/*.go"]
      pattern: "main\\.go$"

  # Require files in directories
  required:
    - name: "doc-required"
      dirs: "internal/**"
      when: "*.go"
      require: "doc.go"
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

## Hooks

External hooks allow custom validation via external programs. See [Rules: Hooks](rules.md#hooks-external-hooks) for full documentation.

```yaml
hooks:
  - name: "vendor-readonly"
    command: "./hooks/vendor-readonly.sh"
    tools: ["Write", "Edit"]
    paths: ["vendor/**"]
    timeout: 5s
    on_error: allow
```

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `name` | string | Yes | - | Unique identifier |
| `command` | string | Yes | - | Path to executable |
| `args` | []string | No | [] | Arguments to pass |
| `tools` | []string | Yes | - | Tools that trigger hook |
| `paths` | []string | No | [] | Glob patterns (empty = all) |
| `timeout` | duration | No | 5s | Max execution time |
| `on_error` | string | No | allow | Failure behavior: allow, deny |

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
