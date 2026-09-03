#!/usr/bin/env sh
set -eu

ROOT=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
TMP_ROOT=${TMPDIR:-"$ROOT/dist/test-tmp"}
mkdir -p "$TMP_ROOT"
TMP=$(mktemp -d "$TMP_ROOT/c0d3r-install-test.XXXXXX")
SERVER_PORT=${C0D3R_TEST_PORT:-18765}
SERVER_PID=
cleanup() {
  if [ -n "${SERVER_PID:-}" ]; then
    kill "$SERVER_PID" >/dev/null 2>&1 || true
    wait "$SERVER_PID" 2>/dev/null || true
  fi
  rm -rf "$TMP"
}
trap cleanup EXIT HUP INT TERM
VERSION=1.2.3-test

case $(uname -s) in Linux) os=linux ;; Darwin) os=darwin ;; *) exit 0 ;; esac
case $(uname -m) in x86_64|amd64) arch=x86_64 ;; arm64|aarch64) arch=arm64 ;; *) exit 0 ;; esac
name="c0d3r_${VERSION}_${os}_${arch}"
mkdir -p "$TMP/releases/$name"
printf '#!/bin/sh\necho test-binary\n' > "$TMP/releases/$name/giis-code"
printf '#!/usr/bin/env python3\n' > "$TMP/releases/$name/giis-shim.py"
chmod 0755 "$TMP/releases/$name/giis-code" "$TMP/releases/$name/giis-shim.py"
tar -czf "$TMP/releases/$name.tar.gz" -C "$TMP/releases" "$name"
printf '%s\n' "$VERSION" > "$TMP/releases/version.txt"
(
  cd "$TMP/releases"
  if command -v sha256sum >/dev/null 2>&1; then sha256sum "$name.tar.gz" > checksums.txt
  else shasum -a 256 "$name.tar.gz" > checksums.txt
  fi
)

python3 -m http.server "$SERVER_PORT" --bind 127.0.0.1 --directory "$TMP/releases" >/dev/null 2>&1 &
SERVER_PID=$!
sleep 1
BASE_URL="http://127.0.0.1:$SERVER_PORT"

HOME="$TMP/home" C0D3R_BASE_URL="$BASE_URL" sh "$ROOT/install.sh" >/dev/null
[ -x "$TMP/home/.local/bin/c0d3r" ]
[ -x "$TMP/home/.local/share/c0d3r/giis-code" ]
[ "$(cat "$TMP/home/.local/share/c0d3r/version")" = "$VERSION" ]

printf '0  %s\n' "$name.tar.gz" > "$TMP/releases/checksums.txt"
if HOME="$TMP/bad-home" C0D3R_BASE_URL="$BASE_URL" sh "$ROOT/install.sh" >/dev/null 2>&1; then
  echo "Installer accepted an invalid checksum" >&2
  exit 1
fi

echo "Installer tests passed"
