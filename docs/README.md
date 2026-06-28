# GiiS-c0d3r — Terminal AI Coding Assistant

GiiS-c0d3r is a terminal-first AI coding assistant built on the GiiS platform. It provides an agentic interface to Claude and Codex via the local bridge and shim system. No API credits needed—use your Claude Code or Codex CLI subscriptions.

The binary `c0d3r` is a rebranded version of [Crush](https://github.com/charmbracelet/crush) from Charmbracelet, adapted for GiiS.

## What It Does

- **Interactive terminal UI**: Run multi-turn conversations with Claude or Codex directly in your terminal
- **Non-interactive mode**: Pipe commands for automation and scripting
- **Session-based**: Maintain multiple work sessions with full context preservation
- **LSP-enhanced**: Uses Language Server Protocol for code intelligence (optional)
- **Local bridge**: Auto-starts background services (bridge on :8787, shim on :8765)
- **No API management**: Leverages your existing Claude Code and Codex CLI subscriptions

## Installation

### From Binary

The binary is pre-built and located in the project root:

```bash
# Copy to PATH
sudo cp ./c0d3r /usr/local/bin/
chmod +x /usr/local/bin/c0d3r

# Or run directly
./c0d3r
```

### From Source

Requires Go 1.21+:

```bash
go build .
./c0d3r
```

## Quick Start

Just run it:

```bash
./c0d3r
```

The first time you run it, the launcher script (`giis-code-launch.sh`) automatically:
1. Starts the bridge service on `http://127.0.0.1:8787`
2. Starts the shim service on `http://127.0.0.1:8765`
3. Launches the interactive UI

You're ready to code. No setup required.

## Commands

### Interactive Mode

```bash
c0d3r
```

Opens an interactive terminal UI where you can:
- Write prompts and send them to Claude/Codex
- Browse and edit files with the `view`, `edit`, `bash` tools
- Switch models mid-session with `Ctrl+M`
- View session history with `Ctrl+H`
- Access the session picker with `Ctrl+S`
- Close details panel with `Ctrl+D`

### Non-Interactive Mode

```bash
# Single prompt
c0d3r run "Add type hints to this Python file"

# With stdin
cat myfile.py | c0d3r run "Add type hints"

# With output redirection
c0d3r run "Generate boilerplate" > boilerplate.ts
```

### Session Management

```bash
# Continue most recent session
c0d3r --continue

# List projects (working directories)
c0d3r projects

# View logs
c0d3r logs
c0d3r logs --follow
c0d3r logs --tail 100

# Show model list
c0d3r models

# View usage stats
c0d3r stats
```

### Configuration & Login

```bash
# Login (one-time, stores token)
c0d3r login giis-cloud

# Logout
c0d3r logout giis-cloud

# Show config/data directories
c0d3r dirs
```

### Advanced

```bash
# Continue a specific session by ID
c0d3r --session abc123def456

# Run with custom working directory
c0d3r --cwd /path/to/project

# Run with custom data directory
c0d3r --data-dir ~/.my-giis-code

# Enable debug logging
c0d3r --debug

# Auto-accept all permissions (dangerous)
c0d3r --yolo

# Start only the server (for advanced users)
c0d3r server --listen 127.0.0.1:8787
```

## Architecture

GiiS-c0d3r uses a three-tier local architecture:

```
┌─────────────────────────────────────────┐
│          c0d3r (TUI/CLI)                │
│      Go-based terminal interface        │
└──────────────┬──────────────────────────┘
               │
               ├─► http://127.0.0.1:8787 (Bridge)
               │   Coordinates backend services
               │   Manages workspace state
               │   Handles authentication
               │
               └─► http://127.0.0.1:8765 (Shim)
                   OpenAI-compatible wrapper
                   Forwards to Claude/Codex CLI
                   Streaming & completion support
```

### Components

**c0d3r (Binary)**
- Terminal UI and CLI interface
- Session management (SQLite-backed)
- Tool execution (bash, file editing, grep, etc.)
- Configuration loading and validation

**Bridge (:8787)**
- Workspace coordination service
- Manages multiple clients connecting to the same project directory
- Handles state synchronization between clients
- Runs as background process (auto-started by launcher)

**Shim (:8765)**
- OpenAI-compatible API server written in Python
- Bridges Crush/c0d3r to local `claude` and `codex` CLIs
- Supports streaming and completion modes
- CORS-restricted to localhost only

## Configuration

Configuration files are checked in order (first match wins):

1. `.giis-code.json` (project-local)
2. `giis-code.json` (project-local)
3. `~/.config/giis-code/giis-code.json` (global)

Example configuration:

```json
{
  "$schema": "https://giis.ai/giis-code.json",
  "options": {
    "debug": false,
    "context_paths": ["./AGENTS.md"],
    "disabled_tools": [],
    "skills_paths": ["~/.config/giis-code/skills"]
  },
  "lsp": {
    "gopls": {
      "command": "gopls",
      "options": {
        "staticcheck": true,
        "semanticTokens": true
      }
    },
    "typescript": {
      "command": "typescript-language-server",
      "args": ["--stdio"]
    }
  },
  "permissions": {
    "allowed_tools": ["view", "ls", "grep", "edit", "bash"]
  }
}
```

### Context Files

GiiS-c0d3r reads project instructions from:

- `.giis-code.md` (GiiS-c0d3r-specific rules)
- `.agents.md` (generic agent instructions)
- `.claude.md` (Claude-specific rules)

These are optional but recommended for better results. They provide the model with project conventions, build commands, testing practices, etc.

### LSP Configuration

Add language servers for code intelligence:

```json
{
  "lsp": {
    "go": {
      "command": "gopls",
      "env": { "GOTOOLCHAIN": "go1.24.5" }
    },
    "rust": {
      "command": "rust-analyzer"
    },
    "python": {
      "command": "pyright",
      "args": ["--stdout"]
    }
  }
}
```

### Permissions

Allow tools without prompting:

```json
{
  "permissions": {
    "allowed_tools": ["view", "ls", "grep", "edit", "bash", "mcp_context7_get-library-doc"]
  }
}
```

Or use `--yolo` flag to auto-accept all permissions (dangerous).

### Disable Tools or Skills

Hide tools or skills from the agent:

```json
{
  "options": {
    "disabled_tools": ["bash", "sourcegraph"],
    "disabled_skills": ["crush-config"]
  }
}
```

## Data & Logs

### Directory Locations

```bash
# Show config and data locations
c0d3r dirs

# Unix
~/.config/giis-code/              # Config
~/.local/share/giis-code/         # Data (SQLite, ephemeral state)
./.giis-code/logs/                # Project logs

# Windows
%LOCALAPPDATA%\giis-code\         # Config
%LOCALAPPDATA%\giis-code\         # Data
```

### Logs

Project logs are written to `./.giis-code/logs/crush.log`:

```bash
# View logs
c0d3r logs

# Follow logs in real time
c0d3r logs --follow

# View last 500 lines
c0d3r logs --tail 500

# Enable debug logging in config
{
  "options": {
    "debug": true,
    "debug_lsp": true
  }
}
```

## The Launcher Script

The `giis-code-launch.sh` script is your entry point. It:

1. Checks if bridge is running (health endpoint at :8787/health)
2. Starts bridge in background if needed
3. Checks if shim is running (models endpoint at :8765/v1/models)
4. Starts shim in background if needed
5. Launches the c0d3r TUI with all arguments

```bash
# Run directly
bash giis-code-launch.sh

# Pass arguments through
bash giis-code-launch.sh run "Fix this bug"
bash giis-code-launch.sh --debug
bash giis-code-launch.sh --continue
```

## The Shim (giis-shim.py)

The Python shim is an OpenAI-compatible HTTP API server that wraps local `claude` and `codex` CLI commands. See [SHIM.md](./SHIM.md) for technical details.

## Sessions

Each session is a persistent conversation with full context history, stored in SQLite under `~/.local/share/giis-code/crush.db`.

- Sessions are keyed by working directory (cwd)
- Multiple sessions can exist per project
- Switch between sessions with the session picker (`Ctrl+S`)
- Sessions preserve all messages, tool calls, and state
- Create new sessions automatically on each fresh invocation (unless `--continue` or `--session` is used)

## Keyboard Shortcuts

In interactive mode:

| Shortcut | Action |
|----------|--------|
| `Ctrl+C` | Exit or cancel current operation |
| `Ctrl+M` | Switch models mid-session |
| `Ctrl+H` | Show message history |
| `Ctrl+S` | Open session picker |
| `Ctrl+D` | Toggle details panel |
| `Ctrl+P` | Open commands palette |
| `Tab` | Autocomplete |

## Tools

GiiS-c0d3r includes built-in tools:

- **`view`**: Read files (respects `.gitignore`)
- **`edit`**: Create, modify, and delete files
- **`bash`**: Execute shell commands
- **`ls`**: List directory contents
- **`grep`**: Search files with regex
- **`glob`**: Find files by pattern
- **`ask`**: Ask follow-up questions
- **`mcp`**: Model Context Protocol integration (extensible)

The model decides which tools to use. You can allow or deny each tool call in real time.

## Model Selection

Available models depend on your provider configuration. Check what's available:

```bash
c0d3r models
```

Models are loaded from:
1. `giis-code.json` (custom provider configs)
2. Embedded provider database (Catwalk)
3. Auto-discovery from local providers (Ollama, LM Studio, etc.)

### Custom Providers

Add OpenAI or Anthropic-compatible APIs:

```json
{
  "providers": {
    "deepseek": {
      "type": "openai-compat",
      "base_url": "https://api.deepseek.com/v1",
      "api_key": "$DEEPSEEK_API_KEY",
      "models": [
        {
          "id": "deepseek-chat",
          "name": "Deepseek V3",
          "context_window": 64000
        }
      ]
    }
  }
}
```

## Skills

GiiS-c0d3r supports agent skills for extending capabilities. Skills are folders with a `SKILL.md` file containing instructions.

Global skill paths:
- `~/.config/giis-code/skills/`
- `~/.agents/skills/`
- `~/.claude/skills/`

Project-local paths:
- `.giis-code/skills/`
- `.agents/skills/`
- `.claude/skills/`

User-invocable skills appear in the commands palette with a `user:` or `project:` prefix.

## Metrics & Telemetry

GiiS-c0d3r collects pseudonymous usage metrics (device hash tied, no prompts/responses) to inform development. Opt out:

```bash
export CRUSH_DISABLE_METRICS=1
```

Or in config:

```json
{
  "options": {
    "disable_metrics": true
  }
}
```

Respects `DO_NOT_TRACK=1` environment variable.

## Troubleshooting

### Bridge or Shim Won't Start

Check if processes are already running:

```bash
curl http://127.0.0.1:8787/health
curl http://127.0.0.1:8765/v1/models
```

Kill stale processes:

```bash
pkill -f "giis-code.*bridge"
pkill -f "giis-shim.py"
```

Then try again:

```bash
bash giis-code-launch.sh
```

### Claude or Codex CLI Not Found

Install Claude Code or Codex CLI:

```bash
# Claude Code
brew install anthropic/cli/claude-code

# Codex (if available)
# Check docs for installation
```

Verify they're in PATH:

```bash
which claude
which codex
```

### Models Not Appearing

Ensure you're logged in:

```bash
c0d3r models
```

Check configuration file location:

```bash
c0d3r dirs
```

Update providers:

```bash
c0d3r update-providers
```

### Clipboard Issues on Linux/BSD

Install clipboard tools:

```bash
# Wayland
sudo apt install wl-clipboard

# X11
sudo apt install xclip
# or
sudo apt install xsel
```

### Sessions Lost

Check if SQLite database exists:

```bash
ls ~/.local/share/giis-code/crush.db
```

If missing, sessions were reset. This is safe—start a new session.

### Slow Performance

Enable debug logging to identify bottlenecks:

```bash
c0d3r --debug
c0d3r logs --follow
```

Check for LSP errors that might slow down the agent. Disable LSPs if needed:

```json
{
  "lsp": {}
}
```

## Development

For development, see [AGENTS.md](../AGENTS.md) for architectural details and [SHIM.md](./SHIM.md) for shim documentation.

## Security

- **CORS restricted**: Bridge and shim only accept requests from `127.0.0.1`
- **No API keys in binary**: Uses local `claude` and `codex` CLI commands
- **Config is trusted code**: Crush reads and executes shell expansions in config (`$(...)` syntax)—don't launch Crush in untrusted directories
- **Permissions**: Tool execution requires explicit permission unless `--yolo` is used

## Attribution

GiiS-c0d3r is a rebranded version of [Crush](https://github.com/charmbracelet/crush) by [Charmbracelet](https://charm.land).

- Original: [github.com/charmbracelet/crush](https://github.com/charmbracelet/crush)
- License: FSL-1.1-MIT (Charm's fair-source license)

## License

See [LICENSE.md](../LICENSE.md) in the project root.

## Support

For issues or questions:
- Check the logs: `c0d3r logs`
- Review configuration: `c0d3r dirs` and inspect the config file
- Run with debug: `c0d3r --debug`

## Next Steps

1. **Get started**: Run `c0d3r` or `bash giis-code-launch.sh`
2. **Configure**: Create `.giis-code.json` or `.agents.md` in your project
3. **Add LSPs**: Configure language servers for code intelligence
4. **Extend**: Build custom skills or use MCP servers
