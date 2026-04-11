#!/bin/bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"

echo "=== Linux Bootstrap ==="
echo ""

# 1. Detect package manager
if command -v apt-get &>/dev/null; then
    PKG_MGR="apt"
elif command -v dnf &>/dev/null; then
    PKG_MGR="dnf"
else
    echo "Error: No supported package manager found (apt or dnf)."
    exit 1
fi
echo "Detected package manager: $PKG_MGR"

# 2. Install packages
echo ""
echo "Installing packages..."
PACKAGES="tmux git jq wget htop curl"
if [ "$PKG_MGR" = "apt" ]; then
    sudo apt-get update
    sudo apt-get install -y $PACKAGES
else
    sudo dnf install -y $PACKAGES
fi

# 3. Detect shell rc file
if [ -n "${ZSH_VERSION:-}" ] || [[ "${SHELL:-}" == */zsh ]]; then
    RC_FILE="$HOME/.zshrc"
else
    RC_FILE="$HOME/.bashrc"
fi
echo ""
echo "Using shell rc file: $RC_FILE"

# 4. Add source line to rc file
SOURCE_LINE="source \"$SCRIPT_DIR/profile\""
if [ ! -f "$RC_FILE" ]; then
    echo "$SOURCE_LINE" > "$RC_FILE"
    echo "Created $RC_FILE with profile source line."
elif grep -qF "$SCRIPT_DIR/profile" "$RC_FILE"; then
    echo "Profile source line already in $RC_FILE."
else
    echo "" >> "$RC_FILE"
    echo "$SOURCE_LINE" >> "$RC_FILE"
    echo "Added profile source line to $RC_FILE."
fi

# Warn about old pasted functions
if grep -q "^function reduce_framerate" "$RC_FILE" 2>/dev/null; then
    echo ""
    echo "WARNING: $RC_FILE contains pasted profile functions (e.g. reduce_framerate)."
    echo "These may conflict with the sourced profile. Consider removing them."
fi

# 5. Symlink tmux.conf
echo ""
if [ -L "$HOME/.tmux.conf" ] && [ "$(readlink "$HOME/.tmux.conf")" = "$SCRIPT_DIR/tmux.conf" ]; then
    echo "~/.tmux.conf symlink already correct."
elif [ -e "$HOME/.tmux.conf" ]; then
    mv "$HOME/.tmux.conf" "$HOME/.tmux.conf.bak"
    ln -s "$SCRIPT_DIR/tmux.conf" "$HOME/.tmux.conf"
    echo "Backed up ~/.tmux.conf to ~/.tmux.conf.bak and created symlink."
else
    ln -s "$SCRIPT_DIR/tmux.conf" "$HOME/.tmux.conf"
    echo "Created ~/.tmux.conf symlink."
fi

# 6. Install caveman hooks for Claude Code
echo ""
echo "Installing caveman hooks for Claude Code..."
if command -v python3 &>/dev/null; then
    bash <(curl -fsSL https://raw.githubusercontent.com/NickBorgers/caveman/main/hooks/install.sh)
else
    echo "WARNING: python3 not found — skipping caveman hook install."
    echo "  Install Python 3.8+ and run: bash <(curl -s https://raw.githubusercontent.com/NickBorgers/caveman/main/hooks/install.sh)"
fi

# 7. Summary
echo ""
echo "=== Done ==="
echo ""
echo "Reminders:"
if ! command -v docker &>/dev/null; then
    echo "  - Install Docker: https://docs.docker.com/engine/install/"
fi
echo "  - Restart your shell or run: source $RC_FILE"
