#!/usr/bin/env sh
set -eu

APP=giis-code
INSTALL_DIR="$HOME/.giis-code/bin"
BASE_URL="https://github.com/GiiS-AI/GiiS-Code/releases/latest/download"
SHIM_URL="https://raw.githubusercontent.com/GiiS-AI/GiiS-Code/main/giis-shim.py"

os_name=$(uname -s)
arch_name=$(uname -m)

case "$os_name" in
	Linux) platform_os=linux ;;
	Darwin) platform_os=mac ;;
	*)
		echo "Unsupported operating system: $os_name" >&2
		exit 1
		;;
esac

case "$arch_name" in
	x86_64|amd64) platform_arch=x86_64 ;;
	aarch64|arm64) platform_arch=arm64 ;;
	*)
		echo "Unsupported architecture: $arch_name" >&2
		exit 1
		;;
esac

asset="$APP-$platform_os-$platform_arch"
tmp_file=$(mktemp)
shim_file=$(mktemp)

mkdir -p "$INSTALL_DIR"
curl -fsSL "$BASE_URL/$asset" -o "$tmp_file"
chmod +x "$tmp_file"
mv "$tmp_file" "$INSTALL_DIR/$APP"

curl -fsSL "$SHIM_URL" -o "$shim_file"
chmod +x "$shim_file"
mv "$shim_file" "$INSTALL_DIR/giis-shim.py"

cat > "$INSTALL_DIR/c0d3r" <<'EOF'
#!/bin/sh
set -eu

DIR="$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)"
if ! curl -s http://127.0.0.1:8787/health > /dev/null 2>&1; then
  "$DIR/giis-code" bridge --addr 127.0.0.1:8787 > /dev/null 2>&1 &
  sleep 1
fi
if ! curl -s http://127.0.0.1:8765/v1/models > /dev/null 2>&1; then
  python3 "$DIR/giis-shim.py" --backend auto &
  sleep 1
fi
exec "$DIR/giis-code" "$@"
EOF
chmod +x "$INSTALL_DIR/c0d3r"

add_path() {
	profile_file=$1
	line="export PATH=\"$INSTALL_DIR:\$PATH\""
	if [ -f "$profile_file" ] && grep -Fqs "$INSTALL_DIR" "$profile_file"; then
		return 0
	fi
	printf '\n%s\n' "$line" >> "$profile_file"
}

touch "$HOME/.bashrc" "$HOME/.zshrc"
add_path "$HOME/.bashrc"
add_path "$HOME/.zshrc"

echo "Run: c0d3r to start"
