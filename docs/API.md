# GiiS-c0d3r API Documentation

Complete reference for all c0d3r CLI commands, flags, and tool calls.

## Table of Contents

1. [Global Flags](#global-flags)
2. [Commands](#commands)
3. [Tool Calls](#tool-calls)
4. [Examples](#examples)

---

## Global Flags

These flags work with any c0d3r command:

### `-c, --cwd <path>`
Set the current working directory for the session.

```bash
c0d3r --cwd /path/to/project
c0d3r run "fix this" --cwd /path/to/project
```

### `-D, --data-dir <path>`
Use a custom data directory instead of `~/.config/giis-code`.

```bash
c0d3r --data-dir /custom/path/.giis-code
```

### `-d, --debug`
Enable debug logging output.

```bash
c0d3r --debug
```

### `-H, --host <host>`
Connect to a specific c0d3r server (for advanced use).

```bash
c0d3r --host unix:///run/user/1000/giis-code-1000.sock
```

### `-s, --session <id>`
Continue a specific session by ID.

```bash
c0d3r --session abc123def456
```

### `-C, --continue`
Continue the most recent session.

```bash
c0d3r --continue
```

### `-y, --yolo`
Auto-accept all permission prompts (dangerous—use with caution).

```bash
c0d3r --yolo
```

### `-v, --version`
Show version information.

```bash
c0d3r --version
```

### `-h, --help`
Show help for any command.

```bash
c0d3r --help
c0d3r login --help
c0d3r run --help
```

---

## Commands

### `c0d3r` (Interactive Mode)

Start the interactive terminal UI.

```bash
c0d3r
```

**Features:**
- Multi-turn conversations with Claude or Codex
- File viewing/editing with syntax highlighting
- Real-time token/context tracking in header
- Session persistence across restarts
- Model switching (Ctrl+M)
- LSP integration for code intelligence

**Keyboard Shortcuts:**
- `Ctrl+M` — Switch models
- `Ctrl+H` — View session history
- `Ctrl+S` — Open session picker
- `Ctrl+D` — Close/open details panel
- `Ctrl+C` — Cancel current operation
- `Ctrl+Z` — Go to sleep (suspend session)
- `Ctrl+L` — Clear screen
- `Tab` — Auto-complete

---

### `c0d3r run [prompt...]`

Run a single non-interactive prompt and output the result.

```bash
c0d3r run "Add type hints to this file"
c0d3r run "Refactor this function" "Then add tests"
cat file.py | c0d3r run "Fix syntax errors"
c0d3r run "Generate boilerplate" > output.ts
```

**Flags:**
- `--model <model>` — Specify model (e.g., `claude-3-5-sonnet-20241022`)
- `--format <format>` — Output format: `text`, `json`, `markdown`
- `--temperature <0-2>` — Model temperature (default: 1.0)
- `--max-tokens <n>` — Max output tokens (default: 4096)

**Examples:**
```bash
# With stdin
cat README.md | c0d3r run "Summarize this"

# With output redirection
c0d3r run "Generate config.toml" > config.toml

# Specific model
c0d3r run "Generate Python code" --model claude-3-5-sonnet-20241022

# Custom temperature
c0d3r run "Be creative" --temperature 1.5
```

---

### `c0d3r login [platform]`

Authenticate with a platform (Claude, Codex, GiiS Cloud, etc.).

```bash
c0d3r login claude          # Auto-detect Claude CLI credentials
c0d3r login codex           # Auto-detect OpenAI/Codex credentials
c0d3r login giis-cloud      # Login with GiiS Cloud email/password
c0d3r login hyper           # Login with Charm Hyper
c0d3r login copilot         # Login with GitHub Copilot
```

**Flags:**
- `-f, --force` — Force re-authentication even if already logged in

**Platforms:**
- `claude` — Claude AI (Anthropic)
- `codex` — Codex/OpenAI
- `giis-cloud` — GiiS Cloud (email/password required)
- `hyper` — Charm Hyper (device auth flow)
- `copilot` — GitHub Copilot (device auth flow)

**Examples:**
```bash
# Auto-link existing Claude CLI auth
c0d3r login claude

# Re-authenticate with new credentials
c0d3r login claude --force

# Login to multiple platforms
c0d3r login claude
c0d3r login codex
```

---

### `c0d3r logout [platform]`

Remove stored credentials for a platform.

```bash
c0d3r logout claude
c0d3r logout codex
c0d3r logout giis-cloud
```

**Flags:**
- `-f, --force` — Skip confirmation prompt

**Interactive Selection:**
If no platform specified, shows a menu of logged-in platforms.

```bash
c0d3r logout
# Shows:
# Logged-in platforms:
#   1. claude
#   2. codex
# Select a platform to logout (1-2):
```

---

### `c0d3r session [command]`

Manage c0d3r sessions.

```bash
c0d3r session list          # List all sessions
c0d3r session delete <id>   # Delete a session
c0d3r session export <id>   # Export session to JSON
c0d3r session clear         # Clear all sessions
```

**Subcommands:**

#### `c0d3r session list`
List all active and historical sessions.

```bash
c0d3r session list
# Output:
# ID                    Created              Project
# abc123def456          2026-06-28 14:32     ~/my-project
# xyz789uvw012          2026-06-27 10:15     ~/other-project
```

#### `c0d3r session delete <id>`
Delete a specific session.

```bash
c0d3r session delete abc123def456
```

#### `c0d3r session export <id>`
Export session history to JSON (for archival/sharing).

```bash
c0d3r session export abc123def456 > session.json
```

#### `c0d3r session clear`
Clear all sessions (requires confirmation).

```bash
c0d3r session clear --force
```

---

### `c0d3r projects`

List project directories tracked by c0d3r.

```bash
c0d3r projects
```

**Flags:**
- `-l, --long` — Show detailed info (dates, line counts)
- `--sort <field>` — Sort by `name`, `created`, `modified`, `size`

**Output:**
```
Project                        Modified              Lines
~/my-app                       2026-06-28 14:32      12,450
~/cli-tool                     2026-06-27 09:15      3,821
~/library                      2026-06-25 16:42      8,923
```

---

### `c0d3r models`

List all available AI models from authenticated providers.

```bash
c0d3r models
```

**Output:**
```
Provider    Model ID                            Context  Input/Output
claude      claude-3-5-sonnet-20241022         200k     $3/$15 per 1M
claude      claude-3-opus-20250219             200k     $15/$60 per 1M
codex       gpt-4o                             128k     $5/$15 per 1M
codex       gpt-4-turbo                        128k     $10/$30 per 1M
```

**Flags:**
- `--provider <name>` — Filter by provider
- `--json` — Output as JSON

---

### `c0d3r logs`

View c0d3r server logs.

```bash
c0d3r logs                  # Show recent logs
c0d3r logs --follow         # Follow logs in real-time (like `tail -f`)
c0d3r logs --tail 50        # Show last 50 lines
c0d3r logs --grep error     # Filter logs by pattern
```

**Flags:**
- `-f, --follow` — Follow logs (Ctrl+C to exit)
- `--tail <n>` — Show last n lines
- `--grep <pattern>` — Filter by regex pattern
- `--level <level>` — Filter by log level: `debug`, `info`, `warn`, `error`

**Output:**
```
[2026-06-28T22:15:32] INFO  Session abc123 started in ~/my-project
[2026-06-28T22:16:01] DEBUG Request to claude-3-5-sonnet-20241022
[2026-06-28T22:16:05] INFO  Response: 1,234 tokens
```

---

### `c0d3r dirs`

Show configuration and data directories.

```bash
c0d3r dirs
```

**Output:**
```
Config:    ~/.config/giis-code
Data:      ~/.local/share/giis-code
Cache:     ~/.cache/giis-code
History:   ~/.local/share/giis-code/history
Sessions:  ~/.local/share/giis-code/sessions
```

---

### `c0d3r stats`

Show usage statistics and analytics.

```bash
c0d3r stats
```

**Output:**
```
Sessions:           42
Total Conversations: 284
Total Tokens:       2,345,678
  Input:            1,234,567
  Output:           1,111,111

Models Used:
  claude-3-5-sonnet: 156 uses
  gpt-4o:            84 uses
  claude-3-opus:     44 uses

Time Spent:
  This Week:         12h 34m
  This Month:        48h 12m
  All Time:          156h 48m
```

**Flags:**
- `--json` — Output as JSON
- `--period <period>` — Filter by period: `today`, `week`, `month`, `all`

---

### `c0d3r server [--flags]`

Start a background c0d3r server (advanced).

```bash
c0d3r server
c0d3r server --port 9000
c0d3r server --socket /tmp/c0d3r.sock
```

**Flags:**
- `--port <n>` — HTTP port for server
- `--socket <path>` — Unix socket path
- `--host <addr>` — Bind address (default: 127.0.0.1)

---

### `c0d3r bridge [command]`

Manage the local bridge service (for Claude/Codex connectivity).

```bash
c0d3r bridge start          # Start bridge service
c0d3r bridge stop           # Stop bridge service
c0d3r bridge status         # Check bridge status
c0d3r bridge logs           # View bridge logs
c0d3r bridge config         # Show bridge configuration
```

**Service Details:**
- Bridge runs on `http://127.0.0.1:8787`
- Auto-started by launcher script
- Proxies requests to Claude Code and Codex CLIs
- Handles authentication and rate limiting

---

### `c0d3r update-providers [path-or-url]`

Update provider definitions (models, capabilities, etc.).

```bash
c0d3r update-providers                    # Update from default source
c0d3r update-providers /path/to/file.json  # Load from file
c0d3r update-providers https://...        # Download from URL
```

**Flags:**
- `--force` — Force update even if already up-to-date
- `--verify` — Verify provider definitions after update

---

### `c0d3r completion <shell>`

Generate shell completion scripts.

```bash
# Bash
c0d3r completion bash | sudo tee /usr/share/bash-completion/completions/c0d3r

# Zsh
c0d3r completion zsh | sudo tee /usr/share/zsh/site-functions/_c0d3r

# Fish
c0d3r completion fish | sudo tee /usr/share/fish/vendor_completions.d/c0d3r.fish

# PowerShell
c0d3r completion powershell | Out-String | Out-File -FilePath $PROFILE -Append
```

**Supported Shells:**
- `bash`
- `zsh`
- `fish`
- `powershell`

---

## Tool Calls

Inside c0d3r sessions, you can call tools for file operations, bash execution, and more.

### `view` — View file contents

```
/view /path/to/file.py          # View entire file
/view /path/to/file.py:10:20    # View lines 10-20
/view /path/to/file.py:100      # View around line 100
```

**Response:**
```
File: src/main.py (215 lines)

1  import sys
2  from pathlib import Path
...
215 if __name__ == "__main__":
```

---

### `edit` — Edit file

```
/edit /path/to/file.py
# Then provide the edit in context
```

**Format:**
```
<old_code>
replace_with
<new_code>
```

---

### `bash` — Execute shell command

```
/bash ls -la src/
/bash npm run build
/bash git status
/bash python -m pytest tests/
```

**Response:**
Stdout and stderr from the command.

---

### `find` — Find files

```
/find *.py              # Find Python files
/find -name test_*.py   # Find test files
/find -type d src/      # Find directories in src/
```

**Response:**
```
src/main.py
src/utils.py
src/config.py
tests/test_main.py
```

---

### `grep` — Search file contents

```
/grep "TODO" src/       # Find TODOs in src/
/grep -r "function" .   # Recursive search
/grep "import" *.py     # Search Python files
```

**Response:**
```
src/main.py:42: TODO: refactor this function
src/utils.py:15: TODO: add error handling
tests/test_main.py:108: function test_xyz()
```

---

### `create` — Create new file

```
/create /path/to/new_file.py
```

---

### `delete` — Delete file

```
/delete /path/to/file.py        # Delete file
/delete -r /path/to/directory   # Delete directory
```

---

### `tree` — Show directory tree

```
/tree src/
/tree --depth 2
```

**Response:**
```
src/
├── main.py
├── utils.py
├── config.py
└── commands/
    ├── login.py
    └── logout.py
```

---

## Examples

### Example 1: Interactive Coding Session

```bash
c0d3r
# Opens interactive UI
# Type: "Create a Python CLI app with argparse"
# Model generates code
# Use /view to see generated file
# Use /bash to test the code
# Use /edit to refine it
```

### Example 2: Non-Interactive Code Generation

```bash
c0d3r run "Generate a TypeScript React component for user login" > LoginForm.tsx
cat LoginForm.tsx
```

### Example 3: File Processing Pipeline

```bash
# Fix a Python file
c0d3r run "Add type hints to this file" < myfile.py > myfile_typed.py

# Refactor multiple files
for f in src/*.py; do
  c0d3r run "Refactor this code for readability" < "$f" > "$f.refactored"
  mv "$f.refactored" "$f"
done
```

### Example 4: Session Persistence

```bash
# Start session
c0d3r                          # Interactive mode
# Chat with Claude, set up some code
# Press Ctrl+Z to pause

# Resume later
c0d3r --continue               # Resume most recent session

# Or continue specific session
c0d3r --session abc123def456
```

### Example 5: Multi-Platform Workflow

```bash
# Authenticate with both Claude and Codex
c0d3r login claude
c0d3r login codex

# Use Claude for design, Codex for implementation
c0d3r run "Design a TypeScript API" --model claude-3-5-sonnet-20241022
c0d3r run "Implement the API in TypeScript" --model gpt-4o
```

### Example 6: Project Analysis

```bash
# List all projects
c0d3r projects

# Show usage stats
c0d3r stats

# View available models
c0d3r models

# Check session history
c0d3r session list
```

---

## Environment Variables

These environment variables can configure c0d3r:

### `GIIS_CODE_DATA_DIR`
Override default data directory.

```bash
export GIIS_CODE_DATA_DIR=/custom/path/.giis-code
c0d3r
```

### `GIIS_CODE_DEBUG`
Enable debug logging.

```bash
export GIIS_CODE_DEBUG=1
c0d3r
```

### `GIIS_CODE_MODEL`
Set default model.

```bash
export GIIS_CODE_MODEL=claude-3-5-sonnet-20241022
c0d3r run "Generate code"
```

### `GIIS_CODE_TEMPERATURE`
Set default temperature.

```bash
export GIIS_CODE_TEMPERATURE=0.8
```

### `GIIS_CODE_MAX_TOKENS`
Set default max tokens.

```bash
export GIIS_CODE_MAX_TOKENS=8000
```

---

## Status Codes & Errors

### Success
- `0` — Command completed successfully

### Client Errors
- `1` — General error
- `2` — Command not found
- `3` — Invalid arguments
- `4` — Permission denied

### Server Errors
- `10` — API connection failed
- `11` — Authentication failed
- `12` — Rate limited
- `13` — Model unavailable

### Example Error Handling

```bash
c0d3r run "test"
if [ $? -eq 0 ]; then
  echo "Success"
else
  echo "Failed with code $?"
fi
```

---

## Configuration

Configuration stored in `~/.config/giis-code/`:

- `workspace.json` — Sessions, providers, settings
- `config.toml` — Global configuration
- `credentials/` — Stored authentication tokens

Edit manually (be careful):

```bash
cat ~/.config/giis-code/workspace.json | jq '.providers'
```

---

## Troubleshooting

### Bridge not starting
```bash
c0d3r bridge status
c0d3r bridge logs --follow
```

### Authentication issues
```bash
c0d3r login claude --force
c0d3r login codex --force
```

### View detailed logs
```bash
c0d3r logs --follow --level debug
```

### Reset everything
```bash
rm -rf ~/.config/giis-code
c0d3r  # Will reinitialize
```

---

## API Rate Limits

These limits apply per provider:

- **Claude**: 5 requests/sec (via bridge)
- **Codex/OpenAI**: 3 requests/sec (via shim)
- **GiiS Cloud**: 10 requests/sec

Sessions will queue requests if limits exceeded.

---

## Version History

- **1.0.0** (2026-06-28) — Initial release with Claude & Codex support
