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

# 3. Reinstall formulae whose libraries moved out from under them
#
# When Homebrew upgrades a library, dependents keep pointing at the old
# filename. mosh breaks this way every time protobuf bumps a major version:
#   dyld: Library not loaded: /opt/homebrew/opt/protobuf/lib/libprotobuf.35.1.0.dylib
# The upgrade in step 2 usually fixes it, but only once Homebrew ships a
# rebuilt bottle. Until then the dependent stays broken, so check directly.
#
# `brew linkage --test` reports missing libraries and exits non-zero. It is a
# developer command, so running it turns developer mode on. Save the setting
# and put it back.
echo ""
echo "Checking installed formulae for broken library links..."
LINKAGE_DEV_WAS_ON=0
if brew developer 2>/dev/null | grep -q "is enabled"; then
    LINKAGE_DEV_WAS_ON=1
fi
# A newline-separated string, not an array: macOS ships bash 3.2, where
# expanding an empty array under `set -u` aborts the script.
BROKEN_FORMULAE=""
while read -r formula; do
    [ -n "$formula" ] || continue
    if ! brew linkage --test "$formula" >/dev/null 2>&1; then
        BROKEN_FORMULAE="$BROKEN_FORMULAE$formula"$'\n'
    fi
done < <(brew list --formula --installed-on-request 2>/dev/null || true)
if [ "$LINKAGE_DEV_WAS_ON" -eq 0 ]; then
    brew developer off >/dev/null 2>&1 || true
fi
if [ -z "$BROKEN_FORMULAE" ]; then
    echo "All formulae link correctly."
else
    echo "Broken library links: $(echo "$BROKEN_FORMULAE" | tr '\n' ' ')"
    while read -r formula; do
        [ -n "$formula" ] || continue
        echo "  Reinstalling $formula..."
        brew reinstall "$formula" || echo "  WARNING: could not reinstall $formula."
    done <<< "$BROKEN_FORMULAE"
fi

# 4. Add source line to ~/.zshrc
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

# 6. Install the agent CLIs themselves
echo ""
echo "Installing agent CLIs..."
# shellcheck source=lib/agent-clis.sh
source "$SCRIPT_DIR/lib/agent-clis.sh"
install_agent_clis

# 7. Install Claude Code output styles
echo ""
echo "Installing Claude Code output styles..."
# shellcheck source=lib/claude-output-styles.sh
source "$SCRIPT_DIR/lib/claude-output-styles.sh"
install_claude_output_styles "$SCRIPT_DIR/claude-output-styles"
configure_claude_output_style "PlainTech"

# 8. Install plugins for Claude Code
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

    # Caveman ships a statusline badge script but a plugin cannot wire itself
    # into settings.json, so without this it asks to be set up every session.
    # The marketplace clone is the stable path; the plugin cache dir carries a
    # version hash that changes on every update.
    CAVEMAN_STATUSLINE="$HOME/.claude/plugins/marketplaces/caveman/hooks/caveman-statusline.sh"
    if [ ! -f "$CAVEMAN_STATUSLINE" ]; then
        CAVEMAN_STATUSLINE="$(find "$HOME/.claude/plugins/cache" \
            -path '*caveman*/hooks/caveman-statusline.sh' 2>/dev/null | head -1 || true)"
    fi
    if [ -n "$CAVEMAN_STATUSLINE" ] && [ -f "$CAVEMAN_STATUSLINE" ]; then
        SETTINGS="$HOME/.claude/settings.json"
        mkdir -p "$HOME/.claude"
        [ -f "$SETTINGS" ] || echo '{}' > "$SETTINGS"
        CURRENT_STATUSLINE="$(jq -r '.statusLine.command // ""' "$SETTINGS" 2>/dev/null || true)"
        if [ -n "$CURRENT_STATUSLINE" ] && [[ "$CURRENT_STATUSLINE" != *caveman-statusline* ]]; then
            echo "  NOTE: $SETTINGS already sets a statusLine; leaving it alone."
            echo "    To show the caveman badge: bash \"$CAVEMAN_STATUSLINE\""
        else
            TMP="$(mktemp)"
            if jq --arg cmd "bash \"$CAVEMAN_STATUSLINE\"" \
                '.statusLine = {type: "command", command: $cmd}' "$SETTINGS" > "$TMP" 2>/dev/null; then
                mv "$TMP" "$SETTINGS"
                echo "  Caveman statusline badge wired into $SETTINGS."
            else
                rm -f "$TMP"
                echo "  WARNING: could not update $SETTINGS (invalid JSON or jq missing)."
            fi
        fi
    else
        echo "  NOTE: caveman statusline script not found; badge not configured."
    fi

    if command -v codex &>/dev/null; then
        # Re-add so overrides apply on re-runs. approval_policy=never stops
        # Codex from surfacing interactive approval modals — the prompts that
        # lock up mosh + tmux sessions.
        #
        # Deliberately do NOT pin sandbox_mode here. It belongs to the user's
        # ~/.codex/config.toml: a flag on this line silently overrides it, which
        # breaks hosts where Codex's sandbox cannot start (e.g. a bundled bwrap
        # blocked by apparmor_restrict_unprivileged_userns) and leaves every
        # command Codex runs failing.
        #
        # -s user, not `claude mcp add`'s default `local` scope: local writes the
        # server under projects/<cwd> in ~/.claude.json, so Codex existed only in
        # the directory this installer happened to run from and every other repo
        # reported it unregistered. The local remove clears that stale entry.
        claude mcp remove -s local codex >/dev/null 2>&1 || true
        claude mcp remove -s user codex >/dev/null 2>&1 || true
        # Not silenced: a failed add used to still print "added", so a broken
        # /adversarial-code-review looked like a clean install.
        if claude mcp add -s user codex -- codex mcp-server \
            -c approval_policy="never"; then
            echo "Codex MCP server added (user scope, non-interactive approvals)."
        else
            echo "  WARNING: could not register the Codex MCP server."
            echo "    Retry manually: claude mcp add -s user codex -- codex mcp-server -c approval_policy=never"
        fi
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
        echo "    claude mcp add -s user codex -- codex mcp-server -c approval_policy=never"
    fi
else
    echo "WARNING: claude CLI not found — skipping plugin installs."
    echo "  Install Claude Code and run:"
    echo "    claude plugin marketplace add NickBorgers/caveman"
    echo "    claude plugin install caveman; claude plugin update caveman"
    echo "    claude plugin install playground@claude-plugins-official; claude plugin update playground@claude-plugins-official"
    echo "  If Codex is installed, also run:"
    echo "    claude mcp add -s user codex -- codex mcp-server -c approval_policy=never"
fi

# 9. Install Claude Code skills
echo ""
echo "Installing Claude Code skills..."
# shellcheck source=lib/claude-skills.sh
source "$SCRIPT_DIR/lib/claude-skills.sh"
install_adversarial_skills

# 10. Summary
echo ""
echo "=== Done ==="
echo ""
echo "Reminders:"
if ! command -v docker &>/dev/null; then
    echo "  - Install Docker Desktop: https://www.docker.com/products/docker-desktop/"
fi
echo "  - Restart your shell or run: source ~/.zshrc"
