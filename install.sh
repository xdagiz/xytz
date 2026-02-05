#!/usr/bin/env bash
set -e
# Usage: curl -fsSL https://raw.githubusercontent.com/xdagiz/xytz/main/install.sh | bash

REPO="xdagiz/xytz"
BINARY_NAME="xytz"

RED="\033[0;31m"
GREEN="\033[0;32m"
YELLOW="\033[1;32m"
CYAN="\033[0;36m"
NC="\033[0m" # No color

info() {
  echo -e "${GREEN}[INFO]${NC} $1"
}

warn() {
  echo -e "${YELLOW}[WARN]${NC} $1"
}

error() {
  echo -e "${RED}[ERROR]${NC} $1"
  exit 1
}

is_macos() {
  local os
  os="$(uname -s)"
  [[ "$os" == "Darwin" ]]
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

install() {
  local platform version download_url tarball_name install_dir binary_path

  platform=$(detect_platform)
  info "Detected platform: $platform"

  version=$(get_latest_version)
  if [[ -z "$version" ]]; then
    error "Failed to get latest version"
  fi

  info "Latest version: v$version"

  tarball_name=$(get_tarball_name "$platform" "$version")
  download_url=$(get_download_url "$version" "$tarball_name")

  info "Downloading from: $download_url"

  tmp_dir=$(mktemp -d)
  trap 'rm -rf "$tmp_dir"' EXIT

  if command -v curl &>/dev/null; then
    if ! curl -fsSL "$download_url" -o "$tmp_dir/$tarball_name"; then
      error "Failed to download from: $download_url"
    fi
  elif command -v wget &>/dev/null; then
    if ! wget -q "$download_url" -O "$tmp_dir/$tarball_name"; then
      error "Failed to download from: $download_url"
    fi
  else
    error "Neither curl nor wget found. Please install one of them."
  fi

  install_dir=$(get_install_dir)
  info "Installing to: $install_dir"

  mkdir -p "$install_dir"
  tar xzf "$tmp_dir/$tarball_name" -C "$tmp_dir"
  cp "$tmp_dir/$BINARY_NAME" "$install_dir/$BINARY_NAME"
  chmod +x "$install_dir/$BINARY_NAME"
  binary_path="$install_dir/$BINARY_NAME"

  info "xytz v$version installed to $binary_path"
  echo ""

  add_to_path "$install_dir"
}

get_latest_version() {
  local api_url="https://api.github.com/repos/$REPO/releases/latest"
  if command -v curl &>/dev/null; then
    curl -s "$api_url" | grep '"tag_name":' | sed -E 's/.*"v([^"]+)".*/\1/'
  elif command -v wget &>/dev/null; then
    wget -qO- "$api_url" | grep '"tag_name":' | sed -E 's/.*"v([^"]+)".*/\1/'
  else
    error "Neither curl nor wget found. Please install one of them."
  fi
}

main() {
  echo -e "${CYAN}██╗  ██╗██╗   ██╗████████╗███████╗"
  echo -e "${CYAN}╚██╗██╔╝╚██╗ ██╔╝╚══██╔══╝╚══███╔╝"
  echo -e "${CYAN} ╚███╔╝  ╚████╔╝    ██║     ███╔╝ "
  echo -e "${CYAN} ██╔██╗   ╚██╔╝     ██║    ███╔╝  "
  echo -e "${CYAN}██╔╝ ██╗   ██║      ██║   ███████╗"
  echo -e "${CYAN}╚═╝  ╚═╝   ╚═╝      ╚═╝   ╚══════╝"
  echo ""
  info "Starting xytz installation..."
  echo ""

  install

  echo ""
  info "Installation complete!"
  info "Run 'xytz --help' to get started."
}

main
