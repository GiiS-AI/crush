#!/usr/bin/env python3
"""
GiiS-Code CLI Shim
Wraps `claude` and `codex` CLIs as an OpenAI-compatible API server.
GiiS-c0d3r points at http://localhost:8765/v1 as its provider.

Usage:
  python giis-shim.py              # uses `claude` by default
  python giis-shim.py --backend codex
  python giis-shim.py --port 8765
"""

import argparse
import json
import os
import subprocess
import sys
import time
import uuid
from http.server import BaseHTTPRequestHandler, HTTPServer
from pathlib import Path
from threading import Thread
from typing import Optional
import urllib.request
import urllib.error


def get_api_key(provider: str) -> Optional[str]:
    """Read stored API key from giis-code workspace config."""
    config_dir = Path.home() / ".config" / "giis-code"
    config_file = config_dir / "workspace.json"

    if not config_file.exists():
        return None

    try:
        with open(config_file) as f:
            config = json.load(f)

        providers = config.get("providers", {})
        provider_config = providers.get(provider, {})

        # Try oauth token first (from login commands)
        oauth = provider_config.get("oauth", {})
        if oauth.get("access_token"):
            return oauth["access_token"]

        # Fall back to direct api_key field if it exists
        if provider_config.get("api_key"):
            return provider_config["api_key"]
    except Exception:
        pass

    return None


def build_prompt(messages: list[dict]) -> str:
    """Convert OpenAI messages array to a flat prompt string."""
    parts = []
    for msg in messages:
        role = msg.get("role", "user")
        content = msg.get("content", "")
        if isinstance(content, list):
            content = " ".join(
                c.get("text", "") for c in content if isinstance(c, dict)
            )
        if role == "system":
            parts.append(f"[System]: {content}")
        elif role == "assistant":
            parts.append(f"[Assistant]: {content}")
        else:
            parts.append(content)
    return "\n\n".join(parts)


def call_claude_api(prompt: str, api_key: str) -> str:
    """Call Anthropic Claude API directly and return the result text."""
    url = "https://api.anthropic.com/v1/messages"
    headers = {
        "x-api-key": api_key,
        "anthropic-version": "2023-06-01",
        "content-type": "application/json",
    }
    body = json.dumps({
        "model": "claude-3-5-sonnet-20241022",
        "max_tokens": 4096,
        "messages": [{"role": "user", "content": prompt}],
    }).encode()

    req = urllib.request.Request(url, data=body, headers=headers, method="POST")
    try:
        with urllib.request.urlopen(req, timeout=120) as resp:
            data = json.loads(resp.read().decode())
            content = data.get("content", [])
            if content and isinstance(content, list):
                return content[0].get("text", "")
            return ""
    except urllib.error.HTTPError as e:
        raise RuntimeError(f"Anthropic API error {e.code}: {e.read().decode()[:500]}")


def call_claude(prompt: str) -> str:
    """Call Claude (via API or CLI fallback) and return the result text."""
    api_key = get_api_key("claude")
    if api_key:
        return call_claude_api(prompt, api_key)

    # Fallback to CLI if no API key stored
    result = subprocess.run(
        ["claude", "--print", "--output-format", "json"],
        input=prompt,
        capture_output=True,
        text=True,
        timeout=120,
    )
    if result.returncode != 0:
        raise RuntimeError(f"claude exited {result.returncode}: {result.stderr[:500]}")
    data = json.loads(result.stdout)
    return data.get("result", "")


def stream_claude_api(prompt: str, api_key: str):
    """Stream Claude API response using server-sent events."""
    url = "https://api.anthropic.com/v1/messages"
    headers = {
        "x-api-key": api_key,
        "anthropic-version": "2023-06-01",
        "content-type": "application/json",
    }
    body = json.dumps({
        "model": "claude-3-5-sonnet-20241022",
        "max_tokens": 4096,
        "messages": [{"role": "user", "content": prompt}],
        "stream": True,
    }).encode()

    req = urllib.request.Request(url, data=body, headers=headers, method="POST")
    try:
        with urllib.request.urlopen(req, timeout=120) as resp:
            for line in resp:
                line = line.decode().strip()
                if not line or line.startswith(":"):
                    continue
                if line.startswith("data: "):
                    try:
                        event = json.loads(line[6:])
                        if event.get("type") == "content_block_delta":
                            delta = event.get("delta", {})
                            if delta.get("type") == "text_delta":
                                yield delta.get("text", "")
                    except json.JSONDecodeError:
                        continue
    except urllib.error.HTTPError as e:
        raise RuntimeError(f"Anthropic API error {e.code}: {e.read().decode()[:500]}")


def stream_claude(prompt: str):
    """Stream Claude response (via API or CLI fallback)."""
    api_key = get_api_key("claude")
    if api_key:
        yield from stream_claude_api(prompt, api_key)
        return

    # Fallback to CLI
    proc = subprocess.Popen(
        ["claude", "--print", "--output-format", "stream-json", "--verbose"],
        stdin=subprocess.PIPE,
        stdout=subprocess.PIPE,
        stderr=subprocess.DEVNULL,
        text=True,
    )
    proc.stdin.write(prompt)
    proc.stdin.close()

    for line in proc.stdout:
        line = line.strip()
        if not line:
            continue
        try:
            event = json.loads(line)
        except json.JSONDecodeError:
            continue

        if event.get("type") == "result":
            break

        if event.get("type") == "assistant":
            msg = event.get("message", {})
            for block in msg.get("content", []):
                if block.get("type") == "text":
                    yield block.get("text", "")

    proc.wait()


def call_openai_api(prompt: str, api_key: str) -> str:
    """Call OpenAI API directly and return the result text."""
    url = "https://api.openai.com/v1/chat/completions"
    headers = {
        "Authorization": f"Bearer {api_key}",
        "content-type": "application/json",
    }
    body = json.dumps({
        "model": "gpt-4o-mini",
        "messages": [{"role": "user", "content": prompt}],
        "max_tokens": 4096,
    }).encode()

    req = urllib.request.Request(url, data=body, headers=headers, method="POST")
    try:
        with urllib.request.urlopen(req, timeout=120) as resp:
            data = json.loads(resp.read().decode())
            choices = data.get("choices", [])
            if choices:
                return choices[0].get("message", {}).get("content", "")
            return ""
    except urllib.error.HTTPError as e:
        raise RuntimeError(f"OpenAI API error {e.code}: {e.read().decode()[:500]}")


def call_codex(prompt: str) -> str:
    """Call Codex/OpenAI (via API or CLI fallback) and return the result text."""
    api_key = get_api_key("codex")
    if api_key:
        return call_openai_api(prompt, api_key)

    # Fallback to CLI
    result = subprocess.run(
        ["codex", "exec", "--json"],
        input=prompt,
        capture_output=True,
        text=True,
        timeout=120,
    )
    if result.returncode != 0:
        raise RuntimeError(f"codex exited {result.returncode}: {result.stderr[:500]}")
    for line in result.stdout.splitlines():
        try:
            event = json.loads(line)
        except json.JSONDecodeError:
            continue
        if event.get("type") != "item.completed":
            continue
        item = event.get("item", {})
        if item.get("type") == "agent_message":
            return item.get("text", "").strip()
    return result.stdout.strip()


def make_completion_response(content: str, model: str) -> dict:
    return {
        "id": f"chatcmpl-{uuid.uuid4().hex[:12]}",
        "object": "chat.completion",
        "created": int(time.time()),
        "model": model,
        "choices": [
            {
                "index": 0,
                "message": {"role": "assistant", "content": content},
                "finish_reason": "stop",
            }
        ],
        "usage": {"prompt_tokens": 0, "completion_tokens": 0, "total_tokens": 0},
    }


def make_stream_chunk(delta: str, model: str, finish: bool = False) -> str:
    chunk = {
        "id": f"chatcmpl-{uuid.uuid4().hex[:8]}",
        "object": "chat.completion.chunk",
        "created": int(time.time()),
        "model": model,
        "choices": [
            {
                "index": 0,
                "delta": {} if finish else {"content": delta},
                "finish_reason": "stop" if finish else None,
            }
        ],
    }
    return f"data: {json.dumps(chunk)}\n\n"


class ShimHandler(BaseHTTPRequestHandler):
    backend: str = "auto"

    def _backend_for_model(self, model: str) -> str:
        if self.backend != "auto":
            return self.backend
        model_id = model.rsplit("/", 1)[-1]
        if model_id == "codex":
            return "codex"
        return "claude"

    def log_message(self, fmt, *args):
        pass

    def do_GET(self):
        if self.path == "/v1/models":
            models = {
                "object": "list",
                "data": [
                    {"id": "claude-code", "object": "model", "created": 0, "owned_by": "giis"},
                    {"id": "codex", "object": "model", "created": 0, "owned_by": "giis"},
                ],
            }
            self._json(200, models)
        else:
            self._json(404, {"error": "not found"})

    def do_POST(self):
        if self.path != "/v1/chat/completions":
            self._json(404, {"error": "not found"})
            return

        length = int(self.headers.get("Content-Length", 0))
        body = json.loads(self.rfile.read(length))

        messages = body.get("messages", [])
        stream = body.get("stream", False)
        model = body.get("model", self.backend)
        backend = self._backend_for_model(model)
        prompt = build_prompt(messages)

        try:
            if stream and backend == "claude":
                self._stream_response(prompt, model)
            else:
                content = (
                    call_claude(prompt)
                    if backend == "claude"
                    else call_codex(prompt)
                )
                self._json(200, make_completion_response(content, model))
        except BrokenPipeError:
            pass
        except Exception as e:
            self._json(500, {"error": {"message": str(e), "type": "server_error"}})

    def _stream_response(self, prompt: str, model: str):
        self.send_response(200)
        self.send_header("Content-Type", "text/event-stream")
        self.send_header("Cache-Control", "no-cache")
        self.send_header("Connection", "keep-alive")
        self.send_header("Access-Control-Allow-Origin", "http://127.0.0.1")
        self.end_headers()

        try:
            for chunk in stream_claude(prompt):
                if chunk:
                    self.wfile.write(make_stream_chunk(chunk, model).encode())
                    self.wfile.flush()
            self.wfile.write(make_stream_chunk("", model, finish=True).encode())
            self.wfile.write(b"data: [DONE]\n\n")
            self.wfile.flush()
        except BrokenPipeError:
            pass

    def _json(self, status: int, data: dict):
        body = json.dumps(data).encode()
        try:
            self.send_response(status)
            self.send_header("Content-Type", "application/json")
            self.send_header("Content-Length", str(len(body)))
            self.send_header("Access-Control-Allow-Origin", "http://127.0.0.1")
            self.end_headers()
            self.wfile.write(body)
        except BrokenPipeError:
            pass

    def do_OPTIONS(self):
        self.send_response(204)
        self.send_header("Access-Control-Allow-Origin", "http://127.0.0.1")
        self.send_header("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
        self.send_header("Access-Control-Allow-Headers", "Content-Type, Authorization")
        self.end_headers()


def main():
    parser = argparse.ArgumentParser(description="GiiS-Code CLI shim server")
    parser.add_argument("--backend", choices=["auto", "claude", "codex"], default="auto")
    parser.add_argument("--port", type=int, default=8765)
    args = parser.parse_args()

    ShimHandler.backend = args.backend

    server = HTTPServer(("127.0.0.1", args.port), ShimHandler)
    print(f"[giis-shim] Listening on http://127.0.0.1:{args.port}/v1", file=sys.stderr)
    print(f"[giis-shim] Backend: {args.backend}", file=sys.stderr)
    print(f"[giis-shim] Press Ctrl+C to stop", file=sys.stderr)
    try:
        server.serve_forever()
    except KeyboardInterrupt:
        print("\n[giis-shim] Stopped.", file=sys.stderr)


if __name__ == "__main__":
    main()
