#!/bin/bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"

echo "=== macOS Bootstrap ==="
echo ""

# 1. Install Homebrew if missing
if ! command -v brew &>/dev/null; then
    echo "Installing Homebrew..."
    /bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)"
else
    echo "Homebrew already installed."
fi

# 2. Install packages from Brewfile
echo ""
echo "Installing packages from Brewfile..."
brew bundle --file="$SCRIPT_DIR/Brewfile"

# 3. Add source line to ~/.zshrc
echo ""
SOURCE_LINE="source \"$SCRIPT_DIR/profile\""
if [ ! -f "$HOME/.zshrc" ]; then
    echo "$SOURCE_LINE" > "$HOME/.zshrc"
    echo "Created ~/.zshrc with profile source line."
elif grep -qF "$SCRIPT_DIR/profile" "$HOME/.zshrc"; then
    echo "Profile source line already in ~/.zshrc."
else
    echo "" >> "$HOME/.zshrc"
    echo "$SOURCE_LINE" >> "$HOME/.zshrc"
    echo "Added profile source line to ~/.zshrc."
fi

# Warn about old pasted functions
if grep -q "^function reduce_framerate" "$HOME/.zshrc" 2>/dev/null; then
    echo ""
    echo "WARNING: ~/.zshrc contains pasted profile functions (e.g. reduce_framerate)."
    echo "These may conflict with the sourced profile. Consider removing them."
fi

# 4. Symlink tmux.conf
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

# 5. Summary
echo ""
echo "=== Done ==="
echo ""
echo "Reminders:"
if ! command -v docker &>/dev/null; then
    echo "  - Install Docker Desktop: https://www.docker.com/products/docker-desktop/"
fi
echo "  - Restart your shell or run: source ~/.zshrc"
