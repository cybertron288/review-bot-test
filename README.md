# Agentflow

A production-grade TUI tool that orchestrates Claude Code CLI (for planning and review) and OpenCode CLI (for implementation) in a unified agentic coding workflow.

## Features

- **Unified TUI Experience**: Single interface for the entire workflow
- **Three-Phase Workflow**: Plan → Review → Build
- **Mandatory Staff Engineer Review**: Cannot skip the review step
- **State Persistence**: Resume interrupted workflows
- **Process Management**: Health monitoring, graceful shutdown
- **Audit Logging**: Track all operations and costs

## Installation

```bash
# Clone the repository
git clone https://github.com/ravi/agentflow.git
cd agentflow

# Build
make build

# Or install directly
make install
```

## Prerequisites

- Go 1.22 or later
- Claude Code CLI (`claude`)
- OpenCode CLI (`opencode`)

## Usage

### Launch the TUI

```bash
agentflow
```

### Inside the TUI

1. **Type your task** in the input box and press Enter
2. **Wait for plan generation** (Claude Code)
3. **Iterate** (optional): Type feedback to refine the plan
4. **Review** (Ctrl+R): Triggers staff engineer review
5. **Iterate** (optional): Type feedback to refine the review
6. **Build** (Ctrl+B): Implements the reviewed plan (OpenCode)
7. **Done!** View the summary of changes

### Keybindings

| Key | Action |
|-----|--------|
| `Ctrl+Q` | Quit (saves state) |
| `Ctrl+P` | Open plans browser |
| `Ctrl+R` | Start staff engineer review |
| `Ctrl+B` | Start building |
| `Ctrl+E` | Open in $EDITOR |
| `Ctrl+H` / `?` | Show help |
| `Esc` | Cancel / go back |

### Slash Commands

Type these in the input box:

| Command | Action |
|---------|--------|
| `/plans` | Open plans browser |
| `/review` | Start review |
| `/build` | Start building |
| `/edit` | Open in $EDITOR |
| `/audit` | Show audit summary |
| `/help` | Show help |
| `/quit` | Quit |

### Resume a Session

```bash
agentflow --resume 001
```

### Headless Mode (CI/CD)

```bash
agentflow --task "Add user authentication" --auto-approve --build
```

## Configuration

Configuration file: `.agentflow/config.yaml`

```yaml
cli:
  claude_code: claude       # Path to Claude Code CLI
  opencode: opencode        # Path to OpenCode CLI

process:
  timeout: 300s             # Max time per CLI operation
  health_check_interval: 30s

permissions:
  auto_approve: true        # Auto-approve file operations

paths:
  plans_dir: .agentflow/plans

tui:
  theme: default
  editor: $EDITOR
```

## Directory Structure

```
your-project/
├── .agentflow/
│   ├── config.yaml          # Project-specific config
│   └── plans/
│       ├── 001_feature_timestamp/
│       │   ├── 001_feature_timestamp.md           # Plan
│       │   ├── 001_feature_timestamp_reviewed.md  # Reviewed plan
│       │   ├── state.json                         # Workflow state
│       │   └── audit.jsonl                        # Audit log
│       └── ...
```

## Development

```bash
# Build
make build

# Run tests
make test

# Run with race detector
make test-race

# Format code
make fmt

# Run linter
make lint
```

## License

MIT

