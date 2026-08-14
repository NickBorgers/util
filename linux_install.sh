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
    PKG_MGR=""
fi

# 2. Install the system packages that are actually missing.
#
# This step used to install the whole list unconditionally and abort the script
# when neither apt nor dnf existed. That is wrong inside a devcontainer, where
# `dcr` runs this on every fresh container: the base image already ships git and
# curl, the run may have no sudo at all, and none of that should stop the shell
# config and agent CLIs that follow - which are the parts that actually matter.
echo ""
PACKAGES="tmux et git jq wget htop curl"
MISSING=""
for pkg in $PACKAGES; do
    command -v "$pkg" &>/dev/null || MISSING="${MISSING:+$MISSING }$pkg"
done

SUDO=""
if [ "$(id -u)" -ne 0 ]; then
    command -v sudo &>/dev/null && SUDO="sudo"
fi

if [ -n "${UTIL_SKIP_PACKAGES:-}" ]; then
    echo "Skipping system packages (UTIL_SKIP_PACKAGES is set)."
elif [ -z "$MISSING" ]; then
    echo "All system packages already present."
elif [ -z "$PKG_MGR" ]; then
    echo "WARNING: no apt or dnf here; install these yourself: $MISSING"
elif [ "$(id -u)" -ne 0 ] && [ -z "$SUDO" ]; then
    echo "WARNING: not root and no sudo; install these yourself: $MISSING"
else
    echo "Detected package manager: $PKG_MGR"
    echo "Installing packages: $MISSING"
    # Non-fatal: a stale index or an offline machine must not cost the caller
    # everything downstream of here.
    if [ "$PKG_MGR" = "apt" ]; then
        # shellcheck disable=SC2086  # deliberate word splitting: $MISSING is a package list
        $SUDO apt-get update && $SUDO apt-get install -y $MISSING \
            || echo "WARNING: package install failed; continuing."
    else
        # shellcheck disable=SC2086  # deliberate word splitting: $MISSING is a package list
        $SUDO dnf install -y $MISSING || echo "WARNING: package install failed; continuing."
    fi
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

# 6. Install the agent CLIs themselves
echo ""
echo "Installing agent CLIs..."
# shellcheck source=lib/agent-clis.sh
source "$SCRIPT_DIR/lib/agent-clis.sh"
install_agent_clis

# No-op unless the host's config has been mounted in, which only dcs/dcr do.
# shellcheck source=lib/agent-identity.sh
source "$SCRIPT_DIR/lib/agent-identity.sh"
seed_claude_identity

# 7. Install plugins for Claude Code
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

# 8. Install Claude Code skills
echo ""
echo "Installing Claude Code skills..."
# shellcheck source=lib/claude-skills.sh
source "$SCRIPT_DIR/lib/claude-skills.sh"
install_adversarial_skills

# 9. Summary
echo ""
echo "=== Done ==="
echo ""
echo "Reminders:"
if ! command -v docker &>/dev/null; then
    echo "  - Install Docker: https://docs.docker.com/engine/install/"
fi
echo "  - Restart your shell or run: source $RC_FILE"
