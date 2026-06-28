# GiiS-c0d3r Quick Start

Get up and running in 2 minutes.

## 1. Install

```bash
# Copy binary to PATH
sudo cp ./c0d3r /usr/local/bin/
chmod +x /usr/local/bin/c0d3r

# Or run directly from project folder
./c0d3r
```

## 2. Run

```bash
c0d3r
```

That's it. The launcher script automatically starts:
- Bridge on `http://127.0.0.1:8787`
- Shim on `http://127.0.0.1:8765`
- Interactive TUI

## 3. Use It

In the TUI:
- Type your prompt at the bottom
- Press Enter
- Watch the model code and interact with your filesystem
- Press `Ctrl+S` to switch sessions
- Press `Ctrl+M` to switch models
- Press `Ctrl+C` to exit

## Non-Interactive Mode

```bash
# Single prompt
c0d3r run "Add type hints to this file"

# From stdin
cat myfile.py | c0d3r run "Format and optimize"

# To stdout
c0d3r run "Generate boilerplate" > output.ts
```

## Continue a Session

```bash
# Resume most recent session
c0d3r --continue

# Resume specific session by ID
c0d3r --session abc123
```

## View Logs

```bash
# Show logs
c0d3r logs

# Follow in real time
c0d3r logs --follow
```

## Configure (Optional)

Create `.giis-code.json` in your project:

```json
{
  "options": {
    "debug": false,
    "context_paths": ["./AGENTS.md"]
  },
  "lsp": {
    "gopls": {
      "command": "gopls"
    }
  }
}
```

## Key Shortcuts

| Shortcut | Action |
|----------|--------|
| `Ctrl+C` | Exit |
| `Ctrl+M` | Switch model |
| `Ctrl+S` | Switch session |
| `Ctrl+P` | Commands palette |

## Troubleshooting

**Bridge/shim won't start**
```bash
pkill -f "giis-code.*bridge"
pkill -f "giis-shim.py"
c0d3r
```

**Claude/Codex CLI not found**
```bash
which claude
# Install if missing: brew install anthropic/cli/claude-code
```

**Models not showing**
```bash
c0d3r models
c0d3r dirs  # Check config location
```

## Next

- Full docs: [README.md](./README.md)
- Shim details: [SHIM.md](./SHIM.md)
- Development: [../AGENTS.md](../AGENTS.md)
