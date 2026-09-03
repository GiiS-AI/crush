#!/usr/bin/env sh
set -eu

VERSION=${1:-}
[ -n "$VERSION" ] || { echo "Usage: $0 VERSION [OUTPUT_DIR]" >&2; exit 2; }
OUTPUT_DIR=${2:-dist/release}
ROOT=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)

case "$VERSION" in
  v*) VERSION=${VERSION#v} ;;
esac
case "$VERSION" in
  ''|*[!0-9A-Za-z._+-]*) echo "Invalid version: $VERSION" >&2; exit 2 ;;
esac

rm -rf "$OUTPUT_DIR"
mkdir -p "$OUTPUT_DIR"
OUTPUT_DIR=$(CDPATH= cd -- "$OUTPUT_DIR" && pwd)
BUILD_TMP_DIR="$OUTPUT_DIR/.build-tmp"
BUILD_CACHE_DIR=${C0D3R_GOCACHE:-"$ROOT/dist/.go-build-cache"}
mkdir -p "$BUILD_TMP_DIR" "$BUILD_CACHE_DIR"

build_archive() {
  os=$1
  goarch=$2
  archive_arch=$3
  name="c0d3r_${VERSION}_${os}_${archive_arch}"
  payload="$OUTPUT_DIR/$name"
  mkdir -p "$payload"

  echo "Building $name"
  (
    cd "$ROOT"
    CGO_ENABLED=0 GOOS="$os" GOARCH="$goarch" GOEXPERIMENT=greenteagc \
      GOCACHE="$BUILD_CACHE_DIR" GOTMPDIR="$BUILD_TMP_DIR" \
      go build -trimpath \
      -ldflags "-s -w -X github.com/GiiS-AI/GiiS-Code/internal/version.Version=$VERSION" \
      -o "$payload/giis-code" .
  )
  cp "$ROOT/giis-shim.py" "$payload/giis-shim.py"
  cp "$ROOT/LICENSE.md" "$payload/LICENSE.md"
  chmod 0755 "$payload/giis-code" "$payload/giis-shim.py"
  tar -czf "$OUTPUT_DIR/$name.tar.gz" -C "$OUTPUT_DIR" "$name"
  rm -rf "$payload"
}

build_archive linux amd64 x86_64
build_archive linux arm64 arm64
build_archive darwin amd64 x86_64
build_archive darwin arm64 arm64

(
  cd "$OUTPUT_DIR"
  if command -v sha256sum >/dev/null 2>&1; then
    sha256sum ./*.tar.gz > checksums.txt
  else
    shasum -a 256 ./*.tar.gz > checksums.txt
  fi
  printf '%s\n' "$VERSION" > version.txt
)

rm -rf "$BUILD_TMP_DIR"
echo "Release artifacts written to $OUTPUT_DIR"
