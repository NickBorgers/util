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

# 6. Install plugins for Claude Code
echo ""
echo "Installing plugins for Claude Code..."
if command -v claude &>/dev/null; then
    claude plugin marketplace add NickBorgers/caveman 2>/dev/null || true
    claude plugin install caveman 2>/dev/null || true
    claude plugin update caveman 2>/dev/null || true
    echo "Caveman plugin installed/updated."
    claude plugin install playground@claude-plugins-official 2>/dev/null || true
    claude plugin update playground@claude-plugins-official 2>/dev/null || true
    echo "Playground plugin installed/updated."
    if command -v codex &>/dev/null; then
        # Re-add so overrides apply on re-runs. approval_policy=never +
        # sandbox_mode=workspace-write stop Codex from surfacing interactive
        # approval modals — the prompts that lock up mosh + tmux sessions.
        claude mcp remove codex 2>/dev/null || true
        claude mcp add codex -- codex mcp-server \
            -c approval_policy="never" -c sandbox_mode="workspace-write" 2>/dev/null || true
        echo "Codex MCP server added (non-interactive approvals)."
        # Belt-and-suspenders: pre-authorize the codex MCP tool in Claude Code so
        # it doesn't prompt when permission checks are active (non-skip runs).
        SETTINGS="$HOME/.claude/settings.json"
        mkdir -p "$HOME/.claude"
        [ -f "$SETTINGS" ] || echo '{}' > "$SETTINGS"
        TMP="$(mktemp)"
        if jq '.permissions.allow = ((.permissions.allow // []) + ["mcp__codex"] | unique)' "$SETTINGS" > "$TMP" 2>/dev/null; then
            mv "$TMP" "$SETTINGS"
            echo "  Pre-authorized mcp__codex in $SETTINGS."
        else
            rm -f "$TMP"
            echo "  WARNING: could not update $SETTINGS (invalid JSON or jq missing)."
        fi
    else
        echo "NOTE: codex CLI not found — skipping MCP server setup."
        echo "  Install Codex and run:"
        echo "    claude mcp add codex -- codex mcp-server -c approval_policy=never -c sandbox_mode=workspace-write"
    fi
else
    echo "WARNING: claude CLI not found — skipping plugin installs."
    echo "  Install Claude Code and run:"
    echo "    claude plugin marketplace add NickBorgers/caveman"
    echo "    claude plugin install caveman; claude plugin update caveman"
    echo "    claude plugin install playground@claude-plugins-official; claude plugin update playground@claude-plugins-official"
    echo "  If Codex is installed, also run:"
    echo "    claude mcp add codex -- codex mcp-server -c approval_policy=never -c sandbox_mode=workspace-write"
fi

# 7. Install Claude Code skills
echo ""
echo "Installing Claude Code skills..."
SKILLS_BASE_URL="https://raw.githubusercontent.com/wi-adam/agent-skills/main/plugins/claude/epic-workflow-tkt/skills"
SKILLS="adversarial-code-review adversarial-design-review"
CLAUDE_PLUGINS="$(claude plugin list 2>/dev/null || true)"
if grep -Eq 'epic-workflow-(github|tkt)@wi-adam-skills' <<<"$CLAUDE_PLUGINS"; then
    echo "  Adversarial review skills already provided by an epic-workflow plugin; skipping standalone copies."
else
    for SKILL in $SKILLS; do
        SKILL_DIR="$HOME/.claude/skills/$SKILL"
        mkdir -p "$SKILL_DIR"
        if curl -fsSL "$SKILLS_BASE_URL/$SKILL/SKILL.md" -o "$SKILL_DIR/SKILL.md"; then
            echo "  $SKILL installed."
        else
            echo "  WARNING: failed to download $SKILL"
        fi
    done
fi

# 8. Summary
echo ""
echo "=== Done ==="
echo ""
echo "Reminders:"
if ! command -v docker &>/dev/null; then
    echo "  - Install Docker: https://docs.docker.com/engine/install/"
fi
echo "  - Restart your shell or run: source $RC_FILE"
