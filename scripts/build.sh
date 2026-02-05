#!/bin/bash

set -e

VERSION=${1:-"dev"}
VERSION=${VERSION#v}
PLATFORM=${2:-""}
DIST_DIR="dist"
TEMP_DIR=$(mktemp -d)

platforms=(
  "linux-amd64"
  "linux-arm64"
  "darwin-amd64"
  "darwin-arm64"
  "windows-amd64"
  "windows-arm64"
)

show_help() {
  echo "Usage: $0 [version] [platform]"
  echo ""
  echo "Arguments:"
  echo "  version   Version string (default: dev)"
  echo "  platform  Target platform as os-arch (optional)"
  echo ""
  echo "Examples:"
  echo "  $0 v1.0.0               # Build all platforms, version v1.0.0"
  echo "  $0 v1.0.0 linux-amd64   # Build only linux-amd64, version v1.0.0"
  echo "  $0 v1.0.0 darwin-arm64  # Build only darwin-arm64, version v1.0.0"
  echo "  $0 v1.0.0 windows-amd64 # Build only windows-arm64, version v1.0.0"
  echo ""
  echo "Supported platforms:"
  for platform in "${platforms[@]}"; do
    echo "  - $platform"
  done
}

is_valid_platform() {
  local target="$1"
  for platform in "${platforms[@]}"; do
    if [[ "$platform" == "$target" ]]; then
      return 0
    fi
  done
  return 1
}

if [[ "$VERSION" == "-h" || "$VERSION" == "--help" ]]; then
  show_help
  exit 0
fi

echo "Building xytz version: $VERSION"

if [[ -n "$PLATFORM" ]]; then
  if ! is_valid_platform "$PLATFORM"; then
    echo "Error: Invalid platform '$PLATFORM'"
    echo ""
    show_help
    exit 1
  fi

  echo "Building for platform: $PLATFORM"
  platforms=("$PLATFORM")
fi

rm -rf "$DIST_DIR"
mkdir -p "$DIST_DIR"

for platform in "${platforms[@]}"; do
  IFS='-' read -r os arch <<<"$platform"
  archive_name="xytz-v${VERSION}-${os}-${arch}.tar.gz"

  echo "Building for $os-$arch..."
  GOOS=$os GOARCH=$arch go build \
    -ldflags "-s -w -X github.com/xdagiz/xytz/internal/version.Version=${VERSION}" \
    -o "${TEMP_DIR}/xytz${os:+.$os}${arch:+.$arch}" \
    .

  if [ "$os" = "windows" ]; then
    mv "${TEMP_DIR}/xytz${os:+.$os}${arch:+.$arch}" "${TEMP_DIR}/xytz.exe"
  else
    mv "${TEMP_DIR}/xytz${os:+.$os}${arch:+.$arch}" "${TEMP_DIR}/xytz"
  fi

  tar -czf "${DIST_DIR}/${archive_name}" -C "$TEMP_DIR" .
done

rm -rf "$TEMP_DIR"

echo "Generating checksums..."
cd "$DIST_DIR"
sha256sum ./* >checksums.txt
cd ..

echo "Build complete"
ls -la "$DIST_DIR/"

cd "$DIST_DIR"
if command -v sha256sum &>/dev/null; then
  sha256sum ./* >checksums.txt
elif command -v shasum &>/dev/null; then
  shasum -a 256 ./* >checksums.txt
else
  echo "Warning: Neither sha256sum nor shasum found. Skipping checksum generation."
fi
cd ..
