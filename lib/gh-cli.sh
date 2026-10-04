#!/bin/bash
# Installs the GitHub CLI where it is missing - which is every devcontainer built
# from a stock base image - and wires git to use it.
#
# Credentials are not handled here: dcs/dcr mount the host's token at
# /run/util/gh-token and the profile exports it as GH_TOKEN, which gh and the
# git credential helper below both honour.
#
# Written for bash 3.2.

GH_RELEASES_URL="${GH_RELEASES_URL:-https://github.com/cli/cli/releases}"

# A release tarball under ~/.local: no sudo, no apt repo, works on any distro.
install_gh() {
    if command -v gh &>/dev/null; then
        echo "  gh already installed."
        _configure_gh_git
        return 0
    fi
    [ "$(uname)" = "Linux" ] || { echo "  gh not found; install it with your package manager."; return 0; }

    local arch tag ver tmp
    case "$(uname -m)" in
        x86_64 | amd64) arch=amd64 ;;
        aarch64 | arm64) arch=arm64 ;;
        *) echo "  WARNING: no gh release for $(uname -m); skipping."; return 0 ;;
    esac

    # The redirect from /latest names the tag without touching the rate-limited API.
    tag="$(curl -fsSLI -o /dev/null -w '%{url_effective}' "$GH_RELEASES_URL/latest" 2>/dev/null | sed 's|.*/tag/||')"
    case "$tag" in
        v[0-9]*) ver="${tag#v}" ;;
        *) echo "  WARNING: could not work out the latest gh release; skipping."; return 0 ;;
    esac

    echo "  Installing gh $ver ($arch)..."
    tmp="$(mktemp -d)"
    if curl -fsSL -o "$tmp/gh.tar.gz" "$GH_RELEASES_URL/download/$tag/gh_${ver}_linux_${arch}.tar.gz" \
        && tar -xzf "$tmp/gh.tar.gz" -C "$tmp"; then
        mkdir -p "$HOME/.local/bin"
        install -m 755 "$tmp/gh_${ver}_linux_${arch}/bin/gh" "$HOME/.local/bin/gh"
        PATH="$HOME/.local/bin:$PATH"; export PATH
        echo "  gh installed."
        _configure_gh_git
    else
        echo "  WARNING: gh download failed; skipping."
    fi
    rm -rf "$tmp"
}

# Let git push and pull over https with the same token. Only inside a container
# that was handed one: on a host, the user's own git config stands.
_configure_gh_git() {
    [ -r /run/util/gh-token ] || return 0
    command -v git &>/dev/null || return 0
    GH_TOKEN="$(cat /run/util/gh-token)" gh auth setup-git 2>/dev/null \
        && echo "  git credential helper set to gh."
    return 0
}
