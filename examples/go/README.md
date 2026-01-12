# Go Project Configuration

Comprehensive Watchman configuration for Go projects.

## Usage

```bash
cp dot_watchman.yml /path/to/your/project/.watchman.yml
```

Then customize for your project's needs.

## What's Included

- **Workspace**: Allows Go build cache, blocks secrets
- **Scope**: Allows Go/config files, blocks vendor and generated code
- **Versioning**: 72-char commits, uppercase, no period, linear workflow
- **Incremental**: 10 file limit with 70% warning threshold
- **Invariants**:
  - Tests require implementation files
  - No TODOs in production code
  - Copyright headers required
  - No hardcoded secrets
  - No fmt.Print (use logging)
  - No panic in library code
  - Import restrictions (no unsafe, no reflect in domain)
  - Snake_case naming in internal/
- **Commands**: Blocks sudo, rm -rf, chmod, etc.

## Customization

The config is heavily commented. Adjust to your project:

1. **Relax constraints** - Comment out or remove checks you don't need
2. **Add constraints** - Add more invariants for your patterns
3. **Add hooks** - For AST-based or complex validation
