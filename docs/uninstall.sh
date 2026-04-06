#!/bin/bash
set -euo pipefail

# git-switch uninstaller
# Usage: curl -fsSL https://ishan-sharma-me.github.io/git-switch/uninstall.sh | bash

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

# Remove binary
if [ -f "${INSTALL_DIR}/${BINARY}" ]; then
  rm -f "${INSTALL_DIR}/${BINARY}"
  success "Removed ${BOLD}${BINARY}${NC} from ${INSTALL_DIR}/"
else
  warn "${BINARY} not found in ${INSTALL_DIR}/"
fi

# Remove zsh completions
ZSH_COMP="${HOME}/.local/share/zsh/site-functions/_git-switch"
ZSH_FPATH_LINE='fpath=(~/.local/share/zsh/site-functions $fpath)'
if [ -f "$ZSH_COMP" ]; then
  rm -f "$ZSH_COMP"
  info "Removed zsh completion file"
fi
if [ -f "${HOME}/.zshrc" ] && grep -Fq "$ZSH_FPATH_LINE" "${HOME}/.zshrc"; then
  grep -Fv "$ZSH_FPATH_LINE" "${HOME}/.zshrc" > "${HOME}/.zshrc.tmp" && mv "${HOME}/.zshrc.tmp" "${HOME}/.zshrc"
  info "Removed fpath entry from ~/.zshrc"
fi

# Remove bash completions
BASH_COMP="${HOME}/.local/share/bash-completion/completions/git-switch"
if [ -f "$BASH_COMP" ]; then
  rm -f "$BASH_COMP"
  info "Removed bash completion file"
fi

# Remove fish completions
FISH_COMP="${HOME}/.config/fish/completions/git-switch.fish"
if [ -f "$FISH_COMP" ]; then
  rm -f "$FISH_COMP"
  info "Removed fish completion file"
fi

# Remove Claude Code integration
CLAUDE_MD="${HOME}/.claude/CLAUDE.md"
if [ -f "$CLAUDE_MD" ] && grep -q "git-switch-start" "$CLAUDE_MD"; then
  sed '/<!-- git-switch-start -->/,/<!-- git-switch-end -->/d' "$CLAUDE_MD" > "${CLAUDE_MD}.tmp" && mv "${CLAUDE_MD}.tmp" "$CLAUDE_MD"
  info "Removed Claude Code integration from CLAUDE.md"
fi

echo ""
success "${BOLD}git-switch${NC} has been uninstalled."
echo ""
echo "  Your config (~/.config/git-switch/) and SSH keys were NOT removed."
echo "  To remove config: rm -rf ~/.config/git-switch"
echo ""
echo "  Open a new terminal tab for changes to take effect."
