#!/usr/bin/env sh
set -eu

BASE_URL=${C0D3R_BASE_URL:-https://giis.ai/downloads/c0d3r/latest}
INSTALL_DIR=${C0D3R_INSTALL_DIR:-"$HOME/.local/share/c0d3r"}
BIN_DIR=${C0D3R_BIN_DIR:-"$HOME/.local/bin"}
VERSION=${C0D3R_VERSION:-}

fail() {
  printf 'c0d3r installer: %s\n' "$*" >&2
  exit 1
}

need() {
  command -v "$1" >/dev/null 2>&1 || fail "Required command not found: $1"
}

need curl
need tar
need python3

case $(uname -s) in
  Linux) os=linux ;;
  Darwin) os=darwin ;;
  *) fail "Unsupported operating system: $(uname -s). Use Linux, macOS, or WSL." ;;
esac

case $(uname -m) in
  x86_64|amd64) arch=x86_64 ;;
  arm64|aarch64) arch=arm64 ;;
  *) fail "Unsupported architecture: $(uname -m). Supported: x86_64 and arm64." ;;
esac

if [ -z "$VERSION" ]; then
  VERSION=$(curl -fsSL "$BASE_URL/version.txt") || fail "Could not determine the latest version"
fi
VERSION=$(printf '%s' "$VERSION" | tr -d '\r\n')
case "$VERSION" in
  ''|*[!0-9A-Za-z._+-]*) fail "Invalid release version: $VERSION" ;;
esac

archive="c0d3r_${VERSION}_${os}_${arch}.tar.gz"
TMP_ROOT=${TMPDIR:-"$HOME/.cache/c0d3r/tmp"}
mkdir -p "$TMP_ROOT"
tmp_dir=$(mktemp -d 2>/dev/null || mktemp -d "$TMP_ROOT/c0d3r-install.XXXXXX")
trap 'rm -rf "$tmp_dir"' EXIT HUP INT TERM

printf 'Downloading c0d3r %s for %s/%s...\n' "$VERSION" "$os" "$arch"
curl -fsSL "$BASE_URL/$archive" -o "$tmp_dir/$archive" || fail "Release archive not found: $archive"
curl -fsSL "$BASE_URL/checksums.txt" -o "$tmp_dir/checksums.txt" || fail "Release checksums not found"

expected=$(awk -v file="$archive" '$2 == file || $2 == "*" file || $2 == "./" file { print $1; exit }' "$tmp_dir/checksums.txt")
[ -n "$expected" ] || fail "No checksum published for $archive"
if command -v sha256sum >/dev/null 2>&1; then
  actual=$(sha256sum "$tmp_dir/$archive" | awk '{print $1}')
elif command -v shasum >/dev/null 2>&1; then
  actual=$(shasum -a 256 "$tmp_dir/$archive" | awk '{print $1}')
else
  fail "sha256sum or shasum is required to verify the download"
fi
[ "$actual" = "$expected" ] || fail "Checksum verification failed for $archive"

tar -xzf "$tmp_dir/$archive" -C "$tmp_dir"
payload="$tmp_dir/c0d3r_${VERSION}_${os}_${arch}"
[ -x "$payload/giis-code" ] || fail "Archive does not contain an executable giis-code binary"
[ -f "$payload/giis-shim.py" ] || fail "Archive does not contain giis-shim.py"

mkdir -p "$INSTALL_DIR" "$BIN_DIR"
install -m 0755 "$payload/giis-code" "$INSTALL_DIR/giis-code.new"
install -m 0755 "$payload/giis-shim.py" "$INSTALL_DIR/giis-shim.py.new"
mv "$INSTALL_DIR/giis-code.new" "$INSTALL_DIR/giis-code"
mv "$INSTALL_DIR/giis-shim.py.new" "$INSTALL_DIR/giis-shim.py"
printf '%s\n' "$VERSION" > "$INSTALL_DIR/version"

cat > "$BIN_DIR/c0d3r" <<EOF_LAUNCHER
#!/bin/sh
set -eu
# VS Code (Snap-packaged, e.g. code-insiders) injects its own XDG_DATA_HOME/
# XDG_CONFIG_HOME/XDG_STATE_HOME pointing into a per-revision confinement
# path like ~/snap/code-insiders/2489/.local/share, into every process it
# spawns (including its integrated terminal and extension hosts). c0d3r
# honors those vars for where it stores provider config/auth, so every
# Insiders update silently moves all configured providers/logins to a new,
# empty-looking directory - this is what makes provider selection look
# "stuck"/reset: the selection is saved, just to a location that vanishes on
# the next update. Strip these if they look Snap-confined so c0d3r falls
# back to its real, stable default (\$HOME/.local/share etc.) instead.
case "\${XDG_DATA_HOME:-}" in
  */snap/*) unset XDG_DATA_HOME ;;
esac
case "\${XDG_CONFIG_HOME:-}" in
  */snap/*) unset XDG_CONFIG_HOME ;;
esac
case "\${XDG_STATE_HOME:-}" in
  */snap/*) unset XDG_STATE_HOME ;;
esac
C0D3R_HOME=\${C0D3R_INSTALL_DIR:-"$INSTALL_DIR"}
STATE_DIR=\${XDG_STATE_HOME:-"\$HOME/.local/state"}/c0d3r
mkdir -p "\$STATE_DIR"
if ! curl -fsS http://127.0.0.1:8787/health >/dev/null 2>&1; then
  "\$C0D3R_HOME/giis-code" bridge --addr 127.0.0.1:8787 >>"\$STATE_DIR/bridge.log" 2>&1 &
fi
if ! curl -fsS http://127.0.0.1:8765/v1/models >/dev/null 2>&1; then
  python3 "\$C0D3R_HOME/giis-shim.py" --backend auto >>"\$STATE_DIR/shim.log" 2>&1 &
fi
exec "\$C0D3R_HOME/giis-code" "\$@"
EOF_LAUNCHER
chmod 0755 "$BIN_DIR/c0d3r"

add_path() {
  profile=$1
  line='export PATH="$HOME/.local/bin:$PATH"'
  [ -f "$profile" ] || : > "$profile"
  grep -Fqs '$HOME/.local/bin' "$profile" || printf '\n%s\n' "$line" >> "$profile"
}

case ":$PATH:" in
  *":$BIN_DIR:"*) ;;
  *)
    add_path "$HOME/.profile"
    [ -d "$HOME/.zsh" ] || [ -f "$HOME/.zshrc" ] || [ "${SHELL:-}" = "/bin/zsh" ] || add_path "$HOME/.bashrc"
    if [ -f "$HOME/.zshrc" ] || [ "${SHELL:-}" = "/bin/zsh" ]; then
      add_path "$HOME/.zshrc"
    fi
    ;;
esac

printf '\nc0d3r %s installed successfully.\n' "$VERSION"
if command -v c0d3r >/dev/null 2>&1; then
  printf 'Run: c0d3r\n'
else
  printf 'Open a new terminal, or run:\n  export PATH="%s:$PATH"\n  c0d3r\n' "$BIN_DIR"
fi
