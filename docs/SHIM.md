# GiiS-c0d3r Shim Technical Documentation

The shim (`giis-shim.py`) is an OpenAI-compatible HTTP API server that bridges Crush/c0d3r to local `claude` and `codex` CLI commands.

## Overview

GiiS-c0d3r uses a local shim to avoid direct API calls. Instead of managing API credentials, it:

1. Wraps the `claude` CLI (Claude Code)
2. Wraps the `codex` CLI (Codex agent)
3. Exposes both as OpenAI-compatible HTTP endpoints
4. Runs on `http://127.0.0.1:8765` by default

This approach allows c0d3r to leverage existing Claude Code and Codex CLI subscriptions without consuming API credits.

## Running the Shim

### Auto-Start (Recommended)

The launcher script (`giis-code-launch.sh`) starts the shim automatically:

```bash
bash giis-code-launch.sh
```

### Manual Start

```bash
# Default: auto-detect backend (claude for most models, codex for codex model)
python3 giis-shim.py

# Force Claude backend
python3 giis-shim.py --backend claude

# Force Codex backend
python3 giis-shim.py --backend codex

# Custom port (default 8765)
python3 giis-shim.py --port 9000

# Verbose output
python3 giis-shim.py --backend claude
# Logs print to stderr
```

### Requirements

- Python 3.7+
- `claude` CLI installed (for Claude models)
- `codex` CLI installed (for Codex model)

Verify CLIs are available:

```bash
which claude
which codex
```

Install if missing:

```bash
# Claude Code
brew install anthropic/cli/claude-code

# Codex (check documentation for installation)
```

## API Endpoints

### GET /v1/models

Returns list of available models.

**Request:**
```bash
curl http://127.0.0.1:8765/v1/models
```

**Response:**
```json
{
  "object": "list",
  "data": [
    {
      "id": "claude-code",
      "object": "model",
      "created": 0,
      "owned_by": "giis"
    },
    {
      "id": "codex",
      "object": "model",
      "created": 0,
      "owned_by": "giis"
    }
  ]
}
```

### POST /v1/chat/completions

Submit a chat completion request. Supports both streaming and non-streaming responses.

**Request:**
```bash
curl -X POST http://127.0.0.1:8765/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d '{
    "model": "claude-code",
    "messages": [
      {
        "role": "system",
        "content": "You are a helpful coding assistant."
      },
      {
        "role": "user",
        "content": "Write a Python function to calculate factorial."
      }
    ],
    "stream": false
  }'
```

**Response (Non-Streaming):**
```json
{
  "id": "chatcmpl-abc123",
  "object": "chat.completion",
  "created": 1234567890,
  "model": "claude-code",
  "choices": [
    {
      "index": 0,
      "message": {
        "role": "assistant",
        "content": "def factorial(n):\n    if n <= 1:\n        return 1\n    return n * factorial(n - 1)"
      },
      "finish_reason": "stop"
    }
  ],
  "usage": {
    "prompt_tokens": 0,
    "completion_tokens": 0,
    "total_tokens": 0
  }
}
```

**Streaming Request:**
```bash
curl -X POST http://127.0.0.1:8765/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d '{
    "model": "claude-code",
    "messages": [{"role": "user", "content": "Hello"}],
    "stream": true
  }'
```

**Response (Streaming):**
```
data: {"id":"chatcmpl-abc","object":"chat.completion.chunk","created":1234567890,"model":"claude-code","choices":[{"index":0,"delta":{"content":"Hello"},"finish_reason":null}]}

data: {"id":"chatcmpl-def","object":"chat.completion.chunk","created":1234567890,"model":"claude-code","choices":[{"index":0,"delta":{"content":" world"},"finish_reason":null}]}

data: {"id":"chatcmpl-ghi","object":"chat.completion.chunk","created":1234567890,"model":"claude-code","choices":[{"index":0,"delta":{},"finish_reason":"stop"}]}

data: [DONE]
```

## Backend Selection

The shim supports two backends:

### Claude Backend

Used for Claude models. Calls `claude` CLI with streaming support.

```python
def call_claude(prompt: str) -> str:
    result = subprocess.run(
        ["claude", "--print", "--output-format", "json"],
        input=prompt,
        capture_output=True,
        text=True,
        timeout=120,
    )
```

**Streaming:** Supported via `stream-json` format.

### Codex Backend

Used for the Codex model. Calls `codex` CLI.

```python
def call_codex(prompt: str) -> str:
    result = subprocess.run(
        ["codex", "exec", "--json"],
        input=prompt,
        capture_output=True,
        text=True,
        timeout=120,
    )
```

**Streaming:** Not supported (uses completion mode only).

### Auto-Detection

When backend is `"auto"`:

1. Parse model ID (last part after `/`)
2. If model ID is `"codex"`, use Codex backend
3. Otherwise, use Claude backend

```python
def _backend_for_model(self, model: str) -> str:
    if self.backend != "auto":
        return self.backend
    model_id = model.rsplit("/", 1)[-1]
    if model_id == "codex":
        return "codex"
    return "claude"
```

## Message Format

Messages are converted to a flat prompt string before sending to CLI.

**Input (OpenAI format):**
```json
{
  "messages": [
    {"role": "system", "content": "You are helpful."},
    {"role": "assistant", "content": "Hi!"},
    {"role": "user", "content": "How are you?"}
  ]
}
```

**Converted to:**
```
[System]: You are helpful.

[Assistant]: Hi!

How are you?
```

Message content can be text or a list of content blocks (only text is extracted).

## CORS & Security

The shim restricts CORS to localhost only:

```python
self.send_header("Access-Control-Allow-Origin", "http://127.0.0.1")
```

All requests must come from `127.0.0.1`. External requests are rejected.

Headers include:

```python
self.send_header("Content-Type", "application/json")
self.send_header("Cache-Control", "no-cache")
self.send_header("Connection", "keep-alive")
self.send_header("Access-Control-Allow-Origin", "http://127.0.0.1")
self.send_header("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
self.send_header("Access-Control-Allow-Headers", "Content-Type, Authorization")
```

## Error Handling

Errors are returned in OpenAI format:

```json
{
  "error": {
    "message": "claude exited 1: command not found",
    "type": "server_error"
  }
}
```

Common errors:

- **CLI not found**: `which claude` returns nothing
- **CLI timeout**: Request exceeds 120-second timeout
- **CLI exit code non-zero**: Command failed with error

## Process Timeout

CLI calls timeout after 120 seconds:

```python
timeout=120
```

Change by modifying `call_claude()` and `call_codex()` functions.

## Logging

Logs are printed to stderr (prefixed with `[shim]`):

```python
def log_message(self, fmt, *args):
    print(f"[shim] {fmt % args}", file=sys.stderr)
```

Run with stderr redirected to see logs:

```bash
python3 giis-shim.py 2>&1 | tee /tmp/shim.log
```

## Performance Tuning

### Request Handling

The shim uses Python's `BaseHTTPRequestHandler` (single-threaded). For parallel requests, wrap with a concurrent server:

```python
from http.server import HTTPServer
from socketserver import ThreadingMixIn

class ThreadingServer(ThreadingMixIn, HTTPServer):
    pass

server = ThreadingServer(("127.0.0.1", 8765), ShimHandler)
```

### Caching

The shim does not cache responses. Each request calls the CLI.

### Connection Pooling

Not applicable—each request is independent.

## Integration with c0d3r

c0d3r configures the shim as a custom provider:

```json
{
  "providers": {
    "giis-code": {
      "type": "openai-compat",
      "base_url": "http://127.0.0.1:8765/v1",
      "models": [
        {"id": "claude-code", "name": "Claude Code"},
        {"id": "codex", "name": "Codex"}
      ]
    }
  }
}
```

The bridge service on `:8787` is separate and handles workspace coordination.

## Testing

Test the shim manually:

```bash
# Start shim
python3 giis-shim.py --backend claude &

# Test health
curl http://127.0.0.1:8765/v1/models

# Test completion
curl -X POST http://127.0.0.1:8765/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d '{
    "model": "claude-code",
    "messages": [{"role": "user", "content": "Hello"}],
    "stream": false
  }'

# Kill process
pkill -f giis-shim.py
```

## Debugging

Enable verbose output in CLI calls (modify Python code):

```python
proc = subprocess.Popen(
    ["claude", "--print", "--output-format", "stream-json", "--verbose"],
    stdin=subprocess.PIPE,
    stdout=subprocess.PIPE,
    stderr=subprocess.PIPE,  # Capture stderr
    text=True,
)
```

Print stderr for debugging:

```python
print(proc.stderr.read(), file=sys.stderr)
```

## Limitations

1. **No token counting**: `usage` field is always `{0, 0, 0}`
2. **No streaming for Codex**: Codex backend only supports completion mode
3. **Single-threaded**: Blocks during CLI execution
4. **No request caching**: Each request calls the CLI
5. **127.0.0.1 only**: CORS restricts to localhost
6. **120-second timeout**: Requests exceeding this fail

## Future Improvements

- Async/threaded request handling
- Response caching
- Token counting integration
- Streaming support for Codex
- Optional remote endpoints (non-localhost)
