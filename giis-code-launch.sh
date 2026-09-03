#!/bin/bash
# VS Code (Snap-packaged, e.g. code-insiders) injects its own XDG_DATA_HOME/
# XDG_CONFIG_HOME/XDG_STATE_HOME pointing into a per-revision confinement
# path like ~/snap/code-insiders/2489/.local/share, into every process it
# spawns. giis-code honors those vars for where it stores provider
# config/auth, so every Insiders update silently moves all configured
# providers/logins to a new, empty-looking directory - strip these if they
# look Snap-confined so it falls back to its real, stable default instead.
case "${XDG_DATA_HOME:-}" in
  */snap/*) unset XDG_DATA_HOME ;;
esac
case "${XDG_CONFIG_HOME:-}" in
  */snap/*) unset XDG_CONFIG_HOME ;;
esac
case "${XDG_STATE_HOME:-}" in
  */snap/*) unset XDG_STATE_HOME ;;
esac
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
