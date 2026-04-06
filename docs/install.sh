#!/bin/bash
set -euo pipefail

# git-switch installer
# Usage: curl -fsSL https://ishan-sharma-me.github.io/git-switch/install.sh | bash

REPO="ishan-sharma-me/git-switch"
INSTALL_DIR="${HOME}/.local/bin"
BINARY="git-switch"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
CYAN='\033[0;36m'
BOLD='\033[1m'
NC='\033[0m'

info() { echo -e "${CYAN}${BOLD}==>${NC} $1"; }
success() { echo -e "${GREEN}${BOLD}==>${NC} $1"; }
warn() { echo -e "${YELLOW}${BOLD}==>${NC} $1"; }
error() { echo -e "${RED}${BOLD}error:${NC} $1"; exit 1; }

# Detect OS
OS="$(uname -s | tr '[:upper:]' '[:lower:]')"
case "$OS" in
  darwin) ;;
  linux) ;;
  *) error "Unsupported OS: $OS. Only macOS and Linux are supported." ;;
esac

# Detect architecture
ARCH="$(uname -m)"
case "$ARCH" in
  x86_64)  ARCH="amd64" ;;
  amd64)   ARCH="amd64" ;;
  arm64)   ARCH="arm64" ;;
  aarch64) ARCH="arm64" ;;
  *) error "Unsupported architecture: $ARCH" ;;
esac

ASSET="git-switch-${OS}-${ARCH}"

info "Detected ${BOLD}${OS}/${ARCH}${NC}"

# Get latest release tag
info "Finding latest release..."
LATEST=$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" | grep '"tag_name"' | sed -E 's/.*"([^"]+)".*/\1/')
if [ -z "$LATEST" ]; then
  error "Could not find latest release. Check https://github.com/${REPO}/releases"
fi
info "Latest version: ${BOLD}${LATEST}${NC}"

# Download
DOWNLOAD_URL="https://github.com/${REPO}/releases/download/${LATEST}/${ASSET}"
info "Downloading ${BOLD}${ASSET}${NC}..."

TMPDIR=$(mktemp -d)
trap 'rm -rf "$TMPDIR"' EXIT

HTTP_CODE=$(curl -fsSL -w '%{http_code}' -o "${TMPDIR}/${BINARY}" "$DOWNLOAD_URL" 2>/dev/null || true)
if [ ! -f "${TMPDIR}/${BINARY}" ] || [ "$HTTP_CODE" != "200" ]; then
  error "Download failed. URL: ${DOWNLOAD_URL}"
fi

# Install
mkdir -p "$INSTALL_DIR"
mv "${TMPDIR}/${BINARY}" "${INSTALL_DIR}/${BINARY}"
chmod 755 "${INSTALL_DIR}/${BINARY}"

success "Installed ${BOLD}${BINARY}${NC} to ${INSTALL_DIR}/"

# Check PATH
if ! echo "$PATH" | tr ':' '\n' | grep -qx "$INSTALL_DIR"; then
  warn "${INSTALL_DIR} is not in your PATH"
  echo ""
  SHELL_NAME="$(basename "$SHELL")"
  case "$SHELL_NAME" in
    zsh)
      echo "  Run this to fix it:"
      echo "    echo 'export PATH=\"\$HOME/.local/bin:\$PATH\"' >> ~/.zshrc && source ~/.zshrc"
      ;;
    bash)
      echo "  Run this to fix it:"
      echo "    echo 'export PATH=\"\$HOME/.local/bin:\$PATH\"' >> ~/.bashrc && source ~/.bashrc"
      ;;
    fish)
      echo "  Run this to fix it:"
      echo "    fish_add_path ${INSTALL_DIR}"
      ;;
    *)
      echo "  Add this to your shell profile:"
      echo "    export PATH=\"\$HOME/.local/bin:\$PATH\""
      ;;
  esac
  echo ""
fi

# Set up completions
SHELL_NAME="$(basename "$SHELL")"
case "$SHELL_NAME" in
  zsh)
    COMP_DIR="${HOME}/.local/share/zsh/site-functions"
    mkdir -p "$COMP_DIR"
    "${INSTALL_DIR}/${BINARY}" completion zsh > "${COMP_DIR}/_git-switch"
    FPATH_LINE='fpath=(~/.local/share/zsh/site-functions $fpath)'
    if [ -f "${HOME}/.zshrc" ] && ! grep -Fq "$FPATH_LINE" "${HOME}/.zshrc"; then
      ZSHRC=$(cat "${HOME}/.zshrc")
      printf '%s\n%s\n' "$FPATH_LINE" "$ZSHRC" > "${HOME}/.zshrc"
    fi
    success "Zsh completions installed"
    ;;
  bash)
    COMP_DIR="${HOME}/.local/share/bash-completion/completions"
    mkdir -p "$COMP_DIR"
    "${INSTALL_DIR}/${BINARY}" completion bash > "${COMP_DIR}/git-switch"
    success "Bash completions installed"
    ;;
  fish)
    COMP_DIR="${HOME}/.config/fish/completions"
    mkdir -p "$COMP_DIR"
    "${INSTALL_DIR}/${BINARY}" completion fish > "${COMP_DIR}/git-switch.fish"
    success "Fish completions installed"
    ;;
esac

echo ""
success "${BOLD}git-switch ${LATEST}${NC} installed successfully!"
echo ""
echo "  Get started:"
echo "    git-switch add       Import an existing SSH key"
echo "    git-switch create    Generate new SSH/GPG keys"
echo "    git-switch --help    Show all commands"
echo ""
echo "  Open a new terminal tab for completions to take effect."
