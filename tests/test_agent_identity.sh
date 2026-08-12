#!/bin/bash
# Tests for lib/agent-identity.sh.
#
# Runs the helper against a throwaway HOME with a stand-in host config, so
# nothing touches the real ~/.claude.json.
#
# Run: ./tests/test_agent_identity.sh   (or: make test)

set -uo pipefail

TESTS_DIR="$(cd "$(dirname "$0")" && pwd)"
REPO_DIR="$(cd "$TESTS_DIR/.." && pwd)"

unset -f jq mktemp 2>/dev/null || true
unset CLAUDE_HOST_CONFIG CLAUDE_IDENTITY_KEYS 2>/dev/null || true

PASS=0
FAIL=0
SANDBOX=""

setup() {
    SANDBOX="$(mktemp -d)"
    export HOME="$SANDBOX/home"
    export PATH="$SANDBOX/bin:$REAL_PATH"
    mkdir -p "$HOME" "$SANDBOX/bin"
}

teardown() {
    [ -n "$SANDBOX" ] && rm -rf "$SANDBOX"
    SANDBOX=""
}
trap teardown EXIT

REAL_PATH="$PATH"

# A host config with the identity keys plus the things that must NOT travel.
seed_host() {
    cat >"$SANDBOX/host.json" <<'JSON'
{
  "hasCompletedOnboarding": true,
  "oauthAccount": {"emailAddress": "someone@example.com"},
  "userID": "user-abc",
  "theme": "dark",
  "numStartups": 256,
  "projects": {"/home/someone/code/private": {"history": ["secret prompt"]}},
  "mcpServers": {"host-only": {"command": "nope"}}
}
JSON
}

run_helper() {
    (
        set -euo pipefail
        source "$REPO_DIR/lib/agent-identity.sh"
        seed_claude_identity "$@"
    ) 2>&1
}

ok() { PASS=$((PASS + 1)); echo "  ok - $1"; }
no() { FAIL=$((FAIL + 1)); echo "  FAIL - $1"; }
check() { if [ "$2" = "1" ]; then ok "$1"; else no "$1"; fi; }

contains() { grep -qF -- "$2" <<<"$1" && echo 1 || echo 0; }
exists() { [ -e "$1" ] && echo 1 || echo 0; }
missing() { [ -e "$1" ] && echo 0 || echo 1; }
eq() { [ "$1" = "$2" ] && echo 1 || echo 0; }
q() { jq -r "$1" "$HOME/.claude.json" 2>/dev/null; }

echo "== identity is carried over =="
setup
seed_host
OUT="$(run_helper "$SANDBOX/host.json")"
check "onboarding is marked complete" "$(eq "$(q .hasCompletedOnboarding)" "true")"
check "the account comes across" "$(eq "$(q .oauthAccount.emailAddress)" "someone@example.com")"
check "so does the user id" "$(eq "$(q .userID)" "user-abc")"
check "and the theme" "$(eq "$(q .theme)" "dark")"
check "says so" "$(contains "$OUT" "no second login needed")"
teardown

echo "== host-only state does not travel =="
setup
seed_host
run_helper "$SANDBOX/host.json" >/dev/null
# Project history is keyed by absolute path and is the user's prompt log; it has
# no meaning in the container and no business being copied there.
check "no project history" "$(eq "$(q .projects)" "null")"
check "no host MCP servers" "$(eq "$(q .mcpServers)" "null")"
check "no unrelated counters" "$(eq "$(q .numStartups)" "null")"
teardown

echo "== the container's own config survives =="
setup
seed_host
cat >"$HOME/.claude.json" <<'JSON'
{"mcpServers": {"codex": {"command": "codex"}}, "machineID": "container-machine"}
JSON
run_helper "$SANDBOX/host.json" >/dev/null
check "keeps the codex MCP server" "$(eq "$(q .mcpServers.codex.command)" "codex")"
check "keeps the container machine id" "$(eq "$(q .machineID)" "container-machine")"
check "still marks onboarding complete" "$(eq "$(q .hasCompletedOnboarding)" "true")"
teardown

echo "== a key the host never set is left alone, not nulled =="
setup
echo '{"hasCompletedOnboarding": true}' >"$SANDBOX/host.json"
echo '{"theme": "light"}' >"$HOME/.claude.json"
run_helper "$SANDBOX/host.json" >/dev/null
# Writing an explicit null would read as "not onboarded" and prompt anyway.
check "the container's theme is untouched" "$(eq "$(q .theme)" "light")"
check "no null keys introduced" "$(eq "$(q '[to_entries[]|select(.value==null)]|length')" "0")"
teardown

echo "== no host config mounted, as on a real host =="
setup
OUT="$(run_helper "$SANDBOX/absent.json")"
check "writes nothing at all" "$(missing "$HOME/.claude.json")"
check "and says nothing" "$(eq "$OUT" "")"
teardown

echo "== jq missing =="
setup
seed_host
# A PATH with no jq on it, so the helper takes its degraded path. Scoped inside
# the command substitution: as a bare prefix it would be a plain assignment and
# would strip PATH for every case after this one.
OUT="$(PATH="$SANDBOX/bin"; run_helper "$SANDBOX/host.json")"
check "warns instead of failing" "$(contains "$OUT" "jq not available")"
check "says what to expect" "$(contains "$OUT" "one-time login")"
teardown

echo "== an unreadable host config =="
setup
echo 'not json at all' >"$SANDBOX/host.json"
OUT="$(run_helper "$SANDBOX/host.json")"
check "warns" "$(contains "$OUT" "could not read the host")"
check "leaves a usable file behind" "$(eq "$(q .)" "{}")"
teardown

echo ""
echo "$PASS passed, $FAIL failed"
[ "$FAIL" -eq 0 ]
