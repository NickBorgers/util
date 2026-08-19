#!/bin/bash
# Tests for lib/claude-output-styles.sh.
#
# Runs the helper against a throwaway HOME and a throwaway repo-style source
# directory, so nothing touches the real ~/.claude/output-styles.
#
# Run: ./tests/test_claude_output_styles.sh   (or: make test)

set -uo pipefail

TESTS_DIR="$(cd "$(dirname "$0")" && pwd)"
REPO_DIR="$(cd "$TESTS_DIR/.." && pwd)"

PASS=0
FAIL=0
SANDBOX=""

setup() {
    SANDBOX="$(mktemp -d)"
    export HOME="$SANDBOX/home"
    mkdir -p "$HOME" "$SANDBOX/src"
}

teardown() {
    [ -n "$SANDBOX" ] && rm -rf "$SANDBOX"
    SANDBOX=""
}
trap teardown EXIT

run_helper() {
    (
        set -euo pipefail
        source "$REPO_DIR/lib/claude-output-styles.sh"
        install_claude_output_styles "$@"
    ) 2>&1
}

ok() { PASS=$((PASS + 1)); echo "  ok - $1"; }
no() { FAIL=$((FAIL + 1)); echo "  FAIL - $1"; }
check() { if [ "$2" = "1" ]; then ok "$1"; else no "$1"; fi; }

contains() { grep -qF -- "$2" <<<"$1" && echo 1 || echo 0; }
eq() { [ "$1" = "$2" ] && echo 1 || echo 0; }
is_symlink_to() { [ -L "$1" ] && [ "$(readlink "$1")" = "$2" ] && echo 1 || echo 0; }

echo "== a fresh install symlinks every style =="
setup
echo "style one" >"$SANDBOX/src/One.md"
echo "style two" >"$SANDBOX/src/Two.md"
OUT="$(run_helper "$SANDBOX/src")"
check "One.md is linked" "$(is_symlink_to "$HOME/.claude/output-styles/One.md" "$SANDBOX/src/One.md")"
check "Two.md is linked" "$(is_symlink_to "$HOME/.claude/output-styles/Two.md" "$SANDBOX/src/Two.md")"
check "says so" "$(contains "$OUT" "Linked One.md")"
teardown

echo "== re-running is a no-op, not a re-link =="
setup
echo "style one" >"$SANDBOX/src/One.md"
run_helper "$SANDBOX/src" >/dev/null
OUT="$(run_helper "$SANDBOX/src")"
check "still linked correctly" "$(is_symlink_to "$HOME/.claude/output-styles/One.md" "$SANDBOX/src/One.md")"
check "reports already linked" "$(contains "$OUT" "One.md already linked")"
teardown

echo "== an existing real file is backed up, not clobbered =="
setup
mkdir -p "$HOME/.claude/output-styles"
echo "user's own copy" >"$HOME/.claude/output-styles/One.md"
echo "repo copy" >"$SANDBOX/src/One.md"
OUT="$(run_helper "$SANDBOX/src")"
check "the repo copy is now linked" "$(is_symlink_to "$HOME/.claude/output-styles/One.md" "$SANDBOX/src/One.md")"
check "the user's copy survives as .bak" "$(eq "$(cat "$HOME/.claude/output-styles/One.md.bak")" "user's own copy")"
check "says so" "$(contains "$OUT" "Backed up existing One.md")"
teardown

echo "== a symlink pointing elsewhere is backed up too =="
setup
mkdir -p "$HOME/.claude/output-styles" "$SANDBOX/elsewhere"
echo "somewhere else" >"$SANDBOX/elsewhere/One.md"
ln -s "$SANDBOX/elsewhere/One.md" "$HOME/.claude/output-styles/One.md"
echo "repo copy" >"$SANDBOX/src/One.md"
run_helper "$SANDBOX/src" >/dev/null
check "now points at the repo copy" "$(is_symlink_to "$HOME/.claude/output-styles/One.md" "$SANDBOX/src/One.md")"
check "old symlink preserved as .bak" "$(is_symlink_to "$HOME/.claude/output-styles/One.md.bak" "$SANDBOX/elsewhere/One.md")"
teardown

echo "== an existing .bak is never overwritten =="
setup
mkdir -p "$HOME/.claude/output-styles"
echo "older backup, still needed" >"$HOME/.claude/output-styles/One.md.bak"
echo "user's current copy" >"$HOME/.claude/output-styles/One.md"
echo "repo copy" >"$SANDBOX/src/One.md"
OUT="$(run_helper "$SANDBOX/src")"
check "the old backup survives untouched" "$(eq "$(cat "$HOME/.claude/output-styles/One.md.bak")" "older backup, still needed")"
check "the user's file is left in place, not linked" "$(eq "$(cat "$HOME/.claude/output-styles/One.md")" "user's current copy")"
check "says so" "$(contains "$OUT" "Skipping One.md")"
teardown

echo "== no source directory is a silent no-op =="
setup
OUT="$(run_helper "$SANDBOX/does-not-exist")"
check "nothing written" "$([ ! -e "$HOME/.claude/output-styles" ] && echo 1 || echo 0)"
check "nothing said" "$(eq "$OUT" "")"
teardown

echo "== an empty source directory is a silent no-op =="
setup
OUT="$(run_helper "$SANDBOX/src")"
check "output-styles dir created but empty" "$([ -d "$HOME/.claude/output-styles" ] && [ -z "$(ls -A "$HOME/.claude/output-styles")" ] && echo 1 || echo 0)"
check "nothing said" "$(eq "$OUT" "")"
teardown

run_configure() {
    (
        set -euo pipefail
        source "$REPO_DIR/lib/claude-output-styles.sh"
        configure_claude_output_style "$@"
    ) 2>&1
}
q() { jq -r "$1" "$HOME/.claude/settings.json" 2>/dev/null; }

echo "== configure: no settings.json yet =="
setup
OUT="$(run_configure "PlainTech")"
check "outputStyle is set" "$(eq "$(q .outputStyle)" "PlainTech")"
check "says so" "$(contains "$OUT" "outputStyle set to PlainTech")"
teardown

echo "== configure: settings.json exists with other keys =="
setup
mkdir -p "$HOME/.claude"
echo '{"model": "sonnet"}' >"$HOME/.claude/settings.json"
run_configure "PlainTech" >/dev/null
check "outputStyle is set" "$(eq "$(q .outputStyle)" "PlainTech")"
check "existing key survives" "$(eq "$(q .model)" "sonnet")"
teardown

echo "== configure: already set to the same style is a no-op =="
setup
mkdir -p "$HOME/.claude"
echo '{"outputStyle": "PlainTech"}' >"$HOME/.claude/settings.json"
OUT="$(run_configure "PlainTech")"
check "says already set" "$(contains "$OUT" "already set to PlainTech")"
teardown

echo "== configure: a different deliberate choice is left alone =="
setup
mkdir -p "$HOME/.claude"
echo '{"outputStyle": "Explanatory"}' >"$HOME/.claude/settings.json"
OUT="$(run_configure "PlainTech")"
check "outputStyle unchanged" "$(eq "$(q .outputStyle)" "Explanatory")"
check "warns instead of overwriting" "$(contains "$OUT" "already sets outputStyle")"
teardown

echo "== configure: jq missing falls back to instructions =="
setup
SANDBOX_BIN="$SANDBOX/bin"
mkdir -p "$SANDBOX_BIN"
OUT="$(PATH="$SANDBOX_BIN"; run_configure "PlainTech")"
check "warns" "$(contains "$OUT" "jq not available")"
check "gives manual instructions" "$(contains "$OUT" "/output-style PlainTech")"
check "writes nothing" "$([ ! -e "$HOME/.claude/settings.json" ] && echo 1 || echo 0)"
teardown

echo "== configure: invalid existing JSON is reported, not silently dropped =="
setup
mkdir -p "$HOME/.claude"
echo 'not json at all' >"$HOME/.claude/settings.json"
OUT="$(run_configure "PlainTech")"
check "warns" "$(contains "$OUT" "WARNING: could not update")"
check "leaves the file as-is" "$(eq "$(cat "$HOME/.claude/settings.json")" "not json at all")"
teardown

echo ""
echo "$PASS passed, $FAIL failed"
[ "$FAIL" -eq 0 ]
