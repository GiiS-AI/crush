#!/bin/bash
# Start shim if not already running.
if ! curl -s http://127.0.0.1:8787/health > /dev/null 2>&1; then
  "$(dirname "$0")/giis-code" bridge --addr 127.0.0.1:8787 > /dev/null 2>&1 &
  sleep 1
fi
if ! curl -s http://127.0.0.1:8765/v1/models > /dev/null 2>&1; then
  python3 "$(dirname "$0")/giis-shim.py" --backend auto &
  sleep 1
fi
exec "$(dirname "$0")/giis-code" "$@"
