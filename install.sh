#!/usr/bin/env bash
set -e
# Usage: curl -fsSL https://raw.githubusercontent.com/xdagiz/xytz/main/install.sh | bash

REPO="xdagiz/xytz"
BINARY_NAME="xytz"

RED="\033[0;31m"
GREEN="\033[0;32m"
YELLOW="\033[1;33m"
CYAN="\033[0;36m"
NC="\033[0m"

info() {
  echo -e "${GREEN}INFO${NC} $1"
}

warn() {
  echo -e "${YELLOW}WARN${NC} $1"
}

error() {
  echo -e "${RED}ERROR${NC} $1" >&2
  exit 1
}

detect_platform() {
  local os arch
  os="$(uname -s)"
  arch="$(uname -m)"

  case "$os" in
  Linux)
    case "$arch" in
    x86_64) echo "linux-amd64" ;;
    aarch64 | arm64) echo "linux-arm64" ;;
    *) error "Unsupported architecture: $arch" ;;
    esac
    ;;
  Darwin)
    case "$arch" in
    x86_64) echo "darwin-amd64" ;;
    arm64) echo "darwin-arm64" ;;
    *) error "Unsupported architecture: $arch" ;;
    esac
    ;;
  MINGW* | MSYS* | CYGWIN* | Windows*)
    error "Windows is not supported via Bash. Please download binaries from: https://github.com/xdagiz/xytz/releases"
    ;;
  *)
    error "Unsupported OS: $os"
    ;;
  esac
}

get_install_dir() {
  echo "$HOME/.local/bin"
}

get_tarball_name() {
  local platform="$1"
  local version="$2"
  echo "xytz-v${version}-${platform}.tar.gz"
}

get_download_url() {
  local version="$1"
  local tarball_name="$2"
  echo "https://github.com/$REPO/releases/download/v${version}/${tarball_name}"
}

fetch() {
  local url="$1"
  local out="$2"
  if command -v curl &>/dev/null; then
    curl -fsSL "$url" -o "$out"
  elif command -v wget &>/dev/null; then
    wget -qO "$out" "$url"
  else
    error "Neither curl nor wget found. Please install one of them."
  fi
}

sha256_of() {
  if command -v sha256sum &>/dev/null; then
    sha256sum "$1" | cut -d' ' -f1
  elif command -v shasum &>/dev/null; then
    shasum -a 256 "$1" | cut -d' ' -f1
  else
    echo ""
  fi
}

add_to_path() {
  local install_dir="$1"
  local shell_rc=""

  if [[ -n "$ZSH_VERSION" ]]; then
    shell_rc="$HOME/.zshrc"
  elif [[ -n "$BASH_VERSION" ]]; then
    shell_rc="$HOME/.bashrc"
  else
    shell_rc="$HOME/.bashrc"
  fi

  if echo "$PATH" | grep -q "$install_dir"; then
    info "$install_dir is already in your PATH"
  else
    warn "$install_dir not found in your PATH"
    echo ""
    echo "Add this to your $shell_rc:"
    echo ""
    echo -e "${CYAN}    export PATH=\"\$PATH:$install_dir\"${NC}"
    echo ""
    warn "Then run: source $shell_rc"
  fi
}

get_latest_version() {
  local api_url="https://api.github.com/repos/$REPO/releases/latest"
  local payload=""
  if command -v curl &>/dev/null; then
    payload=$(curl -fsSL "$api_url" 2>/dev/null) || payload=""
  elif command -v wget &>/dev/null; then
    payload=$(wget -qO- "$api_url" 2>/dev/null) || payload=""
  else
    error "Neither curl nor wget found. Please install one of them."
  fi

  if [[ -z "$payload" ]]; then
    return 0
  fi

  printf '%s' "$payload" | grep '"tag_name":' | sed -E 's/.*"v([^"]+)".*/\1/' || true
}

verify_checksum() {
  local dir="$1"
  local tarball_name="$2"
  local version="$3"
  local checksums_url="https://github.com/$REPO/releases/download/v${version}/checksums.txt"
  local expected actual

  if ! command -v sha256sum &>/dev/null && ! command -v shasum &>/dev/null; then
    error "no sha256 tool found (install sha256sum or shasum); refusing to install unverified binary"
  fi

  if ! fetch "$checksums_url" "$dir/checksums.txt"; then
    error "checksums.txt not available for v$version; refusing to install unverified binary"
  fi

  expected=$(awk -v n="$tarball_name" '$2 == n || $2 == "*"n { print $1; exit }' "$dir/checksums.txt")
  actual=$(sha256_of "$dir/$tarball_name")

  if [[ -z "$expected" ]]; then
    error "no checksum entry for $tarball_name in checksums.txt; refusing to install unverified binary"
  fi
  if [[ -z "$actual" ]]; then
    error "failed to hash $tarball_name"
  fi

  expected=$(printf '%s' "$expected" | tr '[:upper:]' '[:lower:]')
  actual=$(printf '%s' "$actual" | tr '[:upper:]' '[:lower:]')

  if [[ "$expected" != "$actual" ]]; then
    error "Checksum mismatch for $tarball_name (expected $expected, got $actual)"
  fi

  info "Checksum verified"
}

install() {
  local platform version download_url tarball_name install_dir binary_path backup

  platform=$(detect_platform)
  info "Detected platform: $platform"

  version=$(get_latest_version)
  if [[ -z "$version" ]]; then
    error "Failed to get latest version (GitHub API rate limiting can cause this; try again later)"
  fi

  info "Latest version: v$version"

  tarball_name=$(get_tarball_name "$platform" "$version")
  download_url=$(get_download_url "$version" "$tarball_name")

  info "Downloading from: $download_url"

  tmp_dir=$(mktemp -d)
  trap 'rm -rf "$tmp_dir"' EXIT

  if ! fetch "$download_url" "$tmp_dir/$tarball_name"; then
    error "Failed to download from: $download_url"
  fi

  verify_checksum "$tmp_dir" "$tarball_name" "$version"

  tar xzf "$tmp_dir/$tarball_name" -C "$tmp_dir"

  install_dir=$(get_install_dir)
  info "Installing to: $install_dir"

  mkdir -p "$install_dir"
  binary_path="$install_dir/$BINARY_NAME"

  backup=""
  if [[ -f "$binary_path" ]]; then
    backup="$tmp_dir/${BINARY_NAME}.bak"
    cp "$binary_path" "$backup"
  fi

  cp "$tmp_dir/$BINARY_NAME" "$binary_path"
  chmod +x "$binary_path"

  info "Verifying installation..."
  if ! "$binary_path" --help >/dev/null 2>&1; then
    if [[ -n "$backup" ]]; then
      cp "$backup" "$binary_path"
      warn "New binary failed verification; previous version restored"
    fi
    error "Installation verification failed. Binary may be corrupted or incompatible."
  fi
  info "Verification successful!"

  info "xytz v$version installed to $binary_path"
  echo ""

  add_to_path "$install_dir"
}

main() {
  echo -e "${CYAN}██╗  ██╗██╗   ██╗████████╗███████╗"
  echo -e "${CYAN}╚██╗██╔╝╚██╗ ██╔╝╚══██╔══╝╚══███╔╝"
  echo -e "${CYAN} ╚███╔╝  ╚████╔╝    ██║     ███╔╝ "
  echo -e "${CYAN} ██╔██╗   ╚██╔╝     ██║    ███╔╝  "
  echo -e "${CYAN}██╔╝ ██╗   ██║      ██║   ███████╗"
  echo -e "${CYAN}╚═╝  ╚═╝   ╚═╝      ╚═╝   ╚══════╝"
  echo ""
  info "Starting installation..."

  install

  echo ""
  info "Installation complete!"
}

main
