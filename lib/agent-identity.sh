#!/bin/bash
# Carries the host's Claude Code identity into a devcontainer.
#
# Mounting ~/.claude/.credentials.json is necessary but not sufficient: the
# onboarding state lives in a different file, ~/.claude.json, which is
# container-local. With a perfectly good token already in place, a fresh
# container still has hasCompletedOnboarding unset and walks the first-run
# flow - which is why "the credentials are mounted" and "I had to log in
# again" are both true at the same time.
#
# Only identity and onboarding keys are copied. Not `projects`, which is
# keyed by absolute path and would be meaningless here; not `mcpServers`,
# which the container configures for itself.
#
# Written for bash 3.2 so it works with the /bin/bash that ships on macOS.

# Where dcs/dcr mounts the host's copy, read-only. Absent on a real host, which
# is what makes this a no-op outside a container.
CLAUDE_HOST_CONFIG="${CLAUDE_HOST_CONFIG:-/host-claude.json}"

# The keys worth carrying across. Deliberately excludes anything that would
# answer a security question on the user's behalf - the folder-trust prompt is
# theirs to answer, in the container as on the host.
CLAUDE_IDENTITY_KEYS="${CLAUDE_IDENTITY_KEYS:-hasCompletedOnboarding oauthAccount userID theme}"

seed_claude_identity() {
    local src="${1:-$CLAUDE_HOST_CONFIG}"
    local dest="$HOME/.claude.json"
    local tmp filter key

    [ -r "$src" ] || return 0

    if ! command -v jq &>/dev/null; then
        echo "  NOTE: jq not available, so the host's Claude Code identity was not carried over."
        echo "    Expect a one-time login in this container."
        return 0
    fi

    [ -f "$dest" ] || echo '{}' >"$dest"

    # Build {key: $host.key, ...} from the key list rather than hardcoding it,
    # so the set stays overridable and the filter cannot drift from it.
    filter=""
    for key in $CLAUDE_IDENTITY_KEYS; do
        filter="${filter:+$filter, }$key: \$host.$key"
    done

    tmp="$(mktemp)" || return 0
    # Host values win for these keys only; everything already in the container's
    # file survives. with_entries drops keys the host has not set, so a missing
    # one is left alone instead of being written as null - which Claude Code
    # would read as "not onboarded" and prompt for anyway.
    if jq -s ".[0] as \$host | .[1] + ({$filter} | with_entries(select(.value != null)))" \
        "$src" "$dest" >"$tmp" 2>/dev/null && [ -s "$tmp" ]; then
        cat "$tmp" >"$dest"
        rm -f "$tmp"
        echo "  Carried the host's Claude Code identity over; no second login needed."
    else
        rm -f "$tmp"
        echo "  WARNING: could not read the host's Claude Code identity from $src."
        echo "    Expect a one-time login in this container."
    fi
}
