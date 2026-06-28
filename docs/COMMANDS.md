# c0d3r Command Reference

Quick reference guide for all c0d3r commands and common usage patterns.

## Table of Contents

- [Main Commands](#main-commands)
- [Authentication](#authentication)
- [Session Management](#session-management)
- [Information Commands](#information-commands)
- [Service Management](#service-management)
- [Tool Reference](#tool-reference)

---

## Main Commands

### Start Interactive Session

```bash
c0d3r
```

Start the interactive terminal UI. You can then:
- Type prompts and have conversations with Claude/Codex
- Use tools like `/view`, `/edit`, `/bash` to manipulate files
- Switch models with `Ctrl+M`
- View history with `Ctrl+H`

### Run Single Prompt (Non-Interactive)

```bash
c0d3r run "Your prompt here"
```

**Examples:**

```bash
# Generate code
c0d3r run "Write a hello world in Python"

# With stdin
cat file.py | c0d3r run "Add type hints"

# Save to file
c0d3r run "Generate config.yaml" > config.yaml

# Specific model
c0d3r run "Generate code" --model gpt-4o

# Custom temperature (more creative)
c0d3r run "Write a funny version" --temperature 1.5

# Custom max tokens
c0d3r run "Generate docs" --max-tokens 10000
```

### Continue Previous Session

```bash
# Resume most recent
c0d3r --continue

# Resume specific session
c0d3r --session abc123def456
```

---

## Authentication

### Login to Claude

```bash
c0d3r login claude
```

Auto-detects your Claude CLI authentication from `~/.claude/.credentials.json`.

**Options:**
```bash
c0d3r login claude --force         # Force re-authentication
```

### Login to Codex/OpenAI

```bash
c0d3r login codex
```

Auto-detects your OpenAI/Codex authentication from `~/.codex/auth.json`.

**Options:**
```bash
c0d3r login codex --force          # Force re-authentication
```

### Login to GiiS Cloud

```bash
c0d3r login giis-cloud
```

Prompts for email and password.

### Login to GitHub Copilot

```bash
c0d3r login copilot
```

Opens browser for device authentication.

### Login to Charm Hyper

```bash
c0d3r login hyper
```

Device flow authentication.

### Logout

```bash
# Remove specific provider
c0d3r logout claude
c0d3r logout codex

# Interactive selection
c0d3r logout              # Shows menu of logged-in providers

# Force without confirmation
c0d3r logout claude --force
```

---

## Session Management

### List All Sessions

```bash
c0d3r session list
```

Shows all active and archived sessions with timestamps and project paths.

### Delete a Session

```bash
c0d3r session delete abc123def456
```

### Export Session

```bash
c0d3r session export abc123def456 > backup.json
```

Saves complete session history to JSON for backup or sharing.

### Clear All Sessions

```bash
c0d3r session clear
c0d3r session clear --force        # Skip confirmation
```

---

## Information Commands

### List All Projects

```bash
c0d3r projects
```

Shows directories tracked by c0d3r.

**With details:**
```bash
c0d3r projects --long
```

**Sort options:**
```bash
c0d3r projects --sort modified
c0d3r projects --sort size
```

### List Available Models

```bash
c0d3r models
```

Shows all available models from authenticated providers with pricing and context windows.

**Filter by provider:**
```bash
c0d3r models --provider claude
c0d3r models --provider codex
```

**JSON output:**
```bash
c0d3r models --json | jq '.models[] | select(.provider=="claude")'
```

### View Usage Statistics

```bash
c0d3r stats
```

Shows:
- Total sessions and conversations
- Token usage (input/output)
- Models used
- Time spent
- Weekly/monthly breakdown

**Options:**
```bash
c0d3r stats --json                 # JSON output
c0d3r stats --period week          # This week only
c0d3r stats --period month         # This month only
```

### View Logs

```bash
c0d3r logs                         # Show recent logs
c0d3r logs --tail 100              # Last 100 lines
c0d3r logs --follow                # Follow in real-time
```

**Filter options:**
```bash
c0d3r logs --level error           # Errors only
c0d3r logs --grep "clauder"        # Pattern matching
c0d3r logs --follow --level debug  # Debug logs in real-time
```

### Show Configuration Directories

```bash
c0d3r dirs
```

Shows paths for:
- Config directory
- Data directory
- Cache directory
- History
- Sessions

---

## Service Management

### Start Server (Background)

```bash
c0d3r server
```

Start c0d3r as a background service.

**Options:**
```bash
c0d3r server --port 9000           # Custom HTTP port
c0d3r server --socket /tmp/c0d3r.sock  # Unix socket
c0d3r server --host 0.0.0.0        # Bind to all interfaces
```

### Bridge Management

```bash
c0d3r bridge start                 # Start bridge
c0d3r bridge stop                  # Stop bridge
c0d3r bridge status                # Check status
c0d3r bridge logs                  # View bridge logs
c0d3r bridge config                # Show configuration
```

The bridge runs on `http://127.0.0.1:8787` and proxies requests to Claude Code and Codex CLIs.

### Update Providers

```bash
c0d3r update-providers             # Update from default source
c0d3r update-providers file.json   # Load from file
c0d3r update-providers https://... # Download from URL
```

---

## Tool Reference

Tools are commands you can use inside c0d3r interactive sessions.

### `/view` — Display file contents

```bash
/view src/main.py                  # Entire file
/view src/main.py:10:20            # Lines 10-20
/view src/main.py:100              # Around line 100
```

### `/edit` — Modify file

```bash
/edit src/main.py
# Provide old_code and new_code in context
```

### `/bash` — Run shell command

```bash
/bash ls -la                       # List files
/bash npm run build                # Run build
/bash git status                   # Check git status
/bash python -m pytest             # Run tests
/bash curl https://...             # Make HTTP requests
```

### `/find` — Search for files

```bash
/find *.py                         # All Python files
/find -name test_*.py              # Test files
/find -type d src/                 # Directories in src/
```

### `/grep` — Search file contents

```bash
/grep "TODO" src/                  # Find TODOs
/grep -r "function" .              # Recursive search
/grep "import" *.py                # Search pattern
```

### `/create` — Create new file

```bash
/create src/new_file.py
```

### `/delete` — Delete file

```bash
/delete src/old_file.py            # Delete file
/delete -r src/temp/               # Delete directory
```

### `/tree` — Show directory structure

```bash
/tree src/
/tree --depth 2
```

---

## Global Flags

These work with any c0d3r command:

```bash
-c, --cwd <path>                   # Set working directory
-D, --data-dir <path>              # Custom data directory
-d, --debug                        # Enable debug logging
-H, --host <address>               # Server address
-s, --session <id>                 # Continue session
-C, --continue                     # Resume last session
-y, --yolo                         # Auto-accept prompts (dangerous!)
-v, --version                      # Show version
-h, --help                         # Show help
```

---

## Common Workflows

### Generate and Test Code

```bash
# Generate code
c0d3r run "Write a sorting algorithm in Python" > sort.py

# Run tests
c0d3r run "Write tests for this" < sort.py > sort_test.py

# Execute
python sort.py
python -m pytest sort_test.py
```

### Refactor Multiple Files

```bash
for file in src/*.py; do
  c0d3r run "Refactor this for readability" < "$file" > "$file.refactored"
  mv "$file.refactored" "$file"
done
```

### Interactive Development Session

```bash
# Start interactive
c0d3r

# Inside:
# 1. Type: "Create a CLI tool with argparse"
# 2. Claude generates code
# 3. Use /view to see code
# 4. Use /bash to test
# 5. Use /edit to refine
# 6. Press Ctrl+Z to pause

# Later, resume
c0d3r --continue
```

### Model Comparison

```bash
# Generate with Claude
c0d3r run "Generate API endpoint" --model claude-3-5-sonnet-20241022 > claude_api.py

# Generate with Codex
c0d3r run "Generate API endpoint" --model gpt-4o > codex_api.py

# Compare
diff claude_api.py codex_api.py
```

### Batch Processing

```bash
# Process all files matching pattern
find . -name "*.py" -type f | while read file; do
  echo "Processing $file..."
  c0d3r run "Add docstrings to all functions" < "$file" > "$file.doc"
  mv "$file.doc" "$file"
done
```

### Logging and Monitoring

```bash
# Watch logs in real-time
c0d3r logs --follow

# Filter for errors
c0d3r logs --level error

# Debug a specific session
c0d3r logs --follow --grep "session-id"
```

---

## Exit Codes

```
0    — Success
1    — General error
2    — Command not found
3    — Invalid arguments
4    — Permission denied
10   — API connection failed
11   — Authentication failed
12   — Rate limited
13   — Model unavailable
```

**Example:**
```bash
if c0d3r run "test" > /dev/null 2>&1; then
  echo "Success"
else
  case $? in
    11) echo "Authentication failed" ;;
    12) echo "Rate limited" ;;
    *) echo "Error code $?" ;;
  esac
fi
```

---

## Tips & Tricks

### Use with `xargs`

```bash
find src -name "*.py" | xargs -I {} c0d3r run "Add type hints" < {}
```

### Use with `watch`

```bash
# Watch logs update every 2 seconds
watch "c0d3r logs --tail 20"
```

### Pipeline with other tools

```bash
# Generate → Format → Save
c0d3r run "Generate config" | jq . > config.json

# View → Process → Save
cat file.py | c0d3r run "Explain this code" | tee explanation.md
```

### Batch with GNU Parallel

```bash
find src -name "*.py" -type f | \
  parallel "c0d3r run 'Add error handling' < {} > {}.fixed"
```

### Create aliases

```bash
# In ~/.bashrc or ~/.zshrc
alias c0="c0d3r"
alias c0r="c0d3r run"
alias c0l="c0d3r logs --follow"
alias c0p="c0d3r projects"
```

---

## Getting Help

```bash
c0d3r --help                       # General help
c0d3r run --help                   # Help for run command
c0d3r login --help                 # Help for login command
c0d3r session --help               # Help for session command
```

---

## See Also

- [API.md](API.md) — Detailed API documentation
- [README.md](README.md) — Project overview
- [QUICKSTART.md](QUICKSTART.md) — Get started in 5 minutes
- [SHIM.md](SHIM.md) — Understand the OpenAI-compatible shim
