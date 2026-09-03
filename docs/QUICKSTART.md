# GiiS-c0d3r Quick Start

Get up and running in 2 minutes.

## 1. Install or Launch

Hosted install:

```bash
curl -fsSL https://giis.ai/install.sh | sh
```

If you already have this repository locally, launch the same wrapper from the repo root:

```bash
./c0d3r
```

## 2. Run

```bash
c0d3r
```

If you launched from the repository instead of installing to PATH, use:

```bash
./c0d3r
```

The launcher automatically starts:
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

## First-Time Model Selection

On first launch (or `Ctrl+M`) you'll see two provider groups — use `↑`/`↓` to move,
`Enter` to select:

- **GiiS Bridge** — free, local models: `claude-code`, `codex`, and anything your
  local Ollama/LM Studio setup exposes. These proxy your own already-authenticated
  `claude`/`codex` CLI sessions through the local bridge (`127.0.0.1:8787`) — no
  external account needed.
- **GiiS Cloud** — hosted models (`gpt-4o`, `claude-3-5-sonnet`, `gemini-1.5-pro`,
  `llama-3.1-70b`) that require a real GiiS Cloud account. Log in first with
  `c0d3r login`, not through the picker.

**Selecting a GiiS Bridge model for the first time will prompt "Enter your GiiS
Bridge Key."** This looks like a real API key field but isn't one — GiiS Bridge is
just your own local CLI sessions being proxied, and the bridge server doesn't
validate this value against anything external. Type anything (e.g. `local`) and
press Enter; it will say "GiiS Bridge Key validated" and save it to
`giis-code.json`. This only happens once — after that, GiiS Bridge models are
selectable directly with no extra step. It's easy to miss that this is a *new*
screen (it looks similar to the picker), which can make it seem like `↑`/`↓`/Enter
"aren't working" when they actually are.

## Non-Interactive Mode

```bash
# Single prompt
c0d3r run "Add type hints to this file"

# From stdin
cat myfile.py | c0d3r run "Format and optimize"

# To stdout
c0d3r run "Generate boilerplate" > output.ts
```

For repo-local usage, replace `c0d3r` with `./c0d3r`.

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
./c0d3r
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

**Can't select a model / arrow keys and Enter seem to do nothing**
This is almost always the "Enter your GiiS Bridge Key" screen from
[First-Time Model Selection](#first-time-model-selection) appearing without you
noticing — press Enter after arrowing to a model, and if the screen changes to
an "Enter your ... Key" prompt, type anything and press Enter again.

## Next

- Full docs: [README.md](./README.md)
- Shim details: [SHIM.md](./SHIM.md)
- Development: [../AGENTS.md](../AGENTS.md)
