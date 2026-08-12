#!/bin/bash
# Shared helper that installs the agent CLIs the rest of the bootstrap only ever
# configured: Claude Code and Codex.
#
# Sourced by linux_install.sh and mac_install.sh ahead of the plugin step. Before
# this existed both installers guarded everything behind `command -v claude` and
# printed "install it yourself" on any machine that did not already have one -
# which is every fresh box, and every devcontainer built from a stock base image.
#
# Written for bash 3.2 so it works with the /bin/bash that ships on macOS:
# no associative arrays, no empty-array expansion under `set -u`.

# Anthropic's own installer drops a self-updating native binary in ~/.local/bin
# and needs no Node at all. Codex has no equivalent, so it comes from npm.
CLAUDE_INSTALL_URL="${CLAUDE_INSTALL_URL:-https://claude.ai/install.sh}"
CODEX_NPM_PACKAGE="${CODEX_NPM_PACKAGE:-@openai/codex}"
MISE_INSTALL_URL="${MISE_INSTALL_URL:-https://mise.run}"

# Everything here installs under ~/.local. Put its bin dir on PATH for the rest
# of this run: without it the plugin step that follows would still report
# "claude CLI not found" until the user opened a new login shell, and a first
# run would silently configure nothing.
_ensure_local_bin_on_path() {
    mkdir -p "$HOME/.local/bin"
    case ":$PATH:" in
        *":$HOME/.local/bin:"*) ;;
        *) PATH="$HOME/.local/bin:$PATH"; export PATH ;;
    esac
}

# Same reasoning for mise's shims, which is where a mise-managed node/npm lives.
# Shims rather than `mise activate`, matching the profile: activate only takes
# effect on the next prompt, which never arrives inside a script.
_ensure_mise_shims_on_path() {
    [ -d "$HOME/.local/share/mise/shims" ] || return 0
    case ":$PATH:" in
        *":$HOME/.local/share/mise/shims:"*) ;;
        *) PATH="$HOME/.local/share/mise/shims:$PATH"; export PATH ;;
    esac
}

# Node is an implementation detail of Codex's packaging, so it is obtained the
# same way the profile obtains it for the devcontainer CLI - mise on Linux, brew
# on macOS - rather than from the distro, whose Node is usually too old for it.
_ensure_npm() {
    _ensure_mise_shims_on_path
    command -v npm &>/dev/null && return 0

    if [ "$(uname)" = "Darwin" ]; then
        if ! command -v brew &>/dev/null; then
            echo "  WARNING: Homebrew not found, so Node cannot be installed."
            return 1
        fi
        echo "  Installing Node via Homebrew..."
        brew install node || return 1
    else
        if ! command -v mise &>/dev/null; then
            echo "  Installing mise (to provide Node)..."
            curl -fsSL "$MISE_INSTALL_URL" | sh || return 1
            _ensure_local_bin_on_path
        fi
        echo "  Installing Node via mise..."
        mise use --global node@lts || return 1
        _ensure_mise_shims_on_path
    fi

    command -v npm &>/dev/null
}

# Install only when missing. Both CLIs update themselves - Claude Code on its own
# schedule, Codex via npm - so re-running the bootstrap should not spend a
# network round trip re-installing what is already there.
install_claude_cli() {
    if command -v claude &>/dev/null; then
        echo "Claude Code already installed ($(claude --version 2>/dev/null || echo 'version unknown'))."
        return 0
    fi

    echo "Installing Claude Code..."
    if ! curl -fsSL "$CLAUDE_INSTALL_URL" | bash; then
        echo "  WARNING: Claude Code install failed."
        echo "    Retry manually: curl -fsSL $CLAUDE_INSTALL_URL | bash"
        return 1
    fi

    _ensure_local_bin_on_path
    if ! command -v claude &>/dev/null; then
        echo "  WARNING: the installer finished but 'claude' is still not on PATH."
        return 1
    fi
    echo "Claude Code installed."
}

install_codex_cli() {
    # Before the first `codex` call, not just before installing one: this runs
    # from a non-interactive shell during the devcontainer bootstrap, where
    # nothing has sourced the profile, so a mise-managed node is present on disk
    # but absent from PATH. Without this, `codex --version` reports "unknown"
    # and the node link below finds nothing to link.
    _ensure_mise_shims_on_path

    if command -v codex &>/dev/null; then
        echo "Codex already installed ($(codex --version 2>/dev/null || echo 'version unknown'))."
        # Also on this path: a codex installed before this helper existed is
        # exactly the one missing the node link.
        _link_node_beside_codex
        return 0
    fi

    echo "Installing Codex..."
    if ! _ensure_npm; then
        echo "  WARNING: no npm available, so Codex was skipped."
        echo "    Install Node, then: npm install -g $CODEX_NPM_PACKAGE"
        return 1
    fi

    # Same prefix the profile uses for its own global installs, so packages land
    # in ~/.local/bin and no step of this bootstrap ever needs sudo for npm.
    npm config set prefix "$HOME/.local" || true
    if ! npm install -g "$CODEX_NPM_PACKAGE"; then
        echo "  WARNING: Codex install failed."
        echo "    Retry manually: npm install -g $CODEX_NPM_PACKAGE"
        return 1
    fi

    _ensure_local_bin_on_path
    if ! command -v codex &>/dev/null; then
        echo "  WARNING: npm reported success but 'codex' is still not on PATH."
        return 1
    fi
    _link_node_beside_codex
    echo "Codex installed."
}

# codex is a `#!/usr/bin/env node` script, so it is only as reachable as node is.
# When node comes from mise, node lives on PATH solely because the profile put
# the shim dir there - and Debian's stock .bashrc returns early for a
# non-interactive shell, so nothing sourced the profile. `codex` then resolves
# and immediately dies with "env: node: No such file or directory", which is
# what any hook, script, or MCP server launched outside a login shell sees.
#
# Putting node beside codex fixes it for exactly the shells that can find codex
# at all: if ~/.local/bin is on PATH, both resolve; if it is not, neither does.
_link_node_beside_codex() {
    local node_path
    node_path="$(command -v node 2>/dev/null)" || return 0
    [ -n "$node_path" ] || return 0

    # Already the copy we would link to, or a real node installed there.
    case "$node_path" in "$HOME/.local/bin/"*) return 0 ;; esac
    [ -e "$HOME/.local/bin/node" ] && return 0

    ln -s "$node_path" "$HOME/.local/bin/node" 2>/dev/null \
        && echo "  Linked node beside codex so its shebang resolves without the profile."
    return 0
}

# Never fatal. A machine with no network, or one where only one of the two
# installs works, should still get the rest of the bootstrap - the plugin step
# after this already degrades gracefully when a CLI is absent.
install_agent_clis() {
    _ensure_local_bin_on_path
    install_claude_cli || true
    install_codex_cli || true
    return 0
}
