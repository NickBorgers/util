#!/bin/bash
# Tests for the devcontainer helpers in `profile` (dcs, dcr and the mount and
# bootstrap helpers behind them).
#
# `devcontainer` is stubbed and records its argv, so no container is ever built.
# The bootstrap script the helper sends into the container is captured as text
# and asserted on, rather than executed.
#
# Run: ./tests/test_devcontainer_bootstrap.sh   (or: make test)

set -uo pipefail

TESTS_DIR="$(cd "$(dirname "$0")" && pwd)"
REPO_DIR="$(cd "$TESTS_DIR/.." && pwd)"

unset -f devcontainer 2>/dev/null || true
unset UTIL_DIR UTIL_DEVCONTAINER_USER UTIL_FORCE_BOOTSTRAP 2>/dev/null || true

PASS=0
FAIL=0
SANDBOX=""

setup() {
    export SANDBOX
    SANDBOX="$(mktemp -d)"
    export HOME="$SANDBOX/home"
    export PATH="$SANDBOX/bin:$PATH"
    export CALLS="$SANDBOX/calls.log"
    mkdir -p "$HOME" "$SANDBOX/bin"
    : >"$CALLS"

    # One line per invocation, arguments NUL-free and newline-separated inside
    # the line, so a multi-line bootstrap script stays greppable as one record.
    cat >"$SANDBOX/bin/devcontainer" <<'STUB'
#!/bin/bash
{ echo "=== devcontainer"; printf '%s\n' "$@"; } >> "$CALLS"
STUB
    chmod +x "$SANDBOX/bin/devcontainer"
}

teardown() {
    [ -n "$SANDBOX" ] && rm -rf "$SANDBOX"
    SANDBOX=""
}
trap teardown EXIT

seed_credentials() {
    mkdir -p "$HOME/.claude" "$HOME/.codex"
    echo '{}' >"$HOME/.claude/.credentials.json"
    echo '{}' >"$HOME/.codex/auth.json"
}

# Subshell so the sourced profile cannot leak functions or PATH between cases.
run() {
    (
        set -uo pipefail
        # The profile ends dcs/dcr with an interactive `devcontainer exec ... bash`;
        # the stub makes that a no-op, so nothing blocks on a terminal.
        source "$REPO_DIR/profile" >/dev/null 2>&1
        "$@"
    ) 2>&1
}

ok() { PASS=$((PASS + 1)); echo "  ok - $1"; }
no() { FAIL=$((FAIL + 1)); echo "  FAIL - $1"; }
check() { if [ "$2" = "1" ]; then ok "$1"; else no "$1"; fi; }

called() { grep -qF -- "$1" "$CALLS" && echo 1 || echo 0; }
not_called() { grep -qF -- "$1" "$CALLS" && echo 0 || echo 1; }
# The stub logs one argument per line, so this matches a whole argument rather
# than a substring of one - the difference between asserting an argument is "1"
# and matching every line that happens to contain a 1.
arg() { grep -qxF -- "$1" "$CALLS" && echo 1 || echo 0; }
count_of() { grep -cF -- "$1" "$CALLS"; }
eq() { [ "$1" = "$2" ] && echo 1 || echo 0; }

echo "== dcr mounts the checkout and the credentials =="
setup
seed_credentials
run dcr /work >/dev/null
check "mounts this checkout at /util" "$(called "type=bind,source=$REPO_DIR,target=/util")"
check "mounts the Claude credentials" \
    "$(called "type=bind,source=$HOME/.claude/.credentials.json,target=/home/vscode/.claude/.credentials.json")"
check "mounts the Codex credentials" \
    "$(called "type=bind,source=$HOME/.codex/auth.json,target=/home/vscode/.codex/auth.json")"
check "still recreates the container" "$(arg "--remove-existing-container")"
check "passes the workspace through" "$(arg "/work")"
teardown

echo "== dcs mounts the same, without recreating =="
setup
seed_credentials
run dcs /work >/dev/null
check "mounts this checkout at /util" "$(called "type=bind,source=$REPO_DIR,target=/util")"
check "mounts the Claude credentials" "$(called "target=/home/vscode/.claude/.credentials.json")"
check "does not recreate the container" "$(not_called "--remove-existing-container")"
teardown

echo "== credentials that do not exist are not mounted =="
setup
run dcr /work >/dev/null
check "mounts the checkout regardless" "$(called "target=/util")"
check "no Claude mount" "$(not_called ".credentials.json")"
check "no Codex mount" "$(not_called "auth.json")"
teardown

echo "== the container user is overridable =="
setup
seed_credentials
UTIL_DEVCONTAINER_USER=node run dcr /work >/dev/null
check "targets that user's home" "$(called "target=/home/node/.claude/.credentials.json")"
check "and not the default" "$(not_called "/home/vscode/")"
teardown

echo "== the bootstrap sent into the container =="
setup
run dcr /work >/dev/null
check "runs the Linux bootstrap from the mount" "$(called "/util/linux_install.sh")"
check "skips the apt step inside a container" "$(called "UTIL_SKIP_PACKAGES=1")"
check "stamps the container so it runs once" "$(called 'touch "$stamp"')"
check "reclaims the mount-created config dirs" "$(called "sudo chown")"
check "degrades if the mount is missing" "$(called "util is not mounted at /util")"
teardown

echo "== bootstrap force flag is passed through =="
setup
UTIL_FORCE_BOOTSTRAP=1 run dcr /work >/dev/null
check "forwards the flag as an argument" "$(arg "util-bootstrap")"
# Argument order matters: $0 is the script name, $1 is the force flag the
# in-container script tests. A missing flag must still occupy the slot.
check "sends the flag itself" "$(arg "1")"
teardown

echo "== defaults to the current directory =="
setup
run dcr >/dev/null
check "uses . as the workspace" "$(arg ".")"
teardown

echo "== a failed 'up' stops before the shell =="
setup
cat >"$SANDBOX/bin/devcontainer" <<'STUB'
#!/bin/bash
{ echo "=== devcontainer"; printf '%s\n' "$@"; } >> "$CALLS"
case "${1:-}" in up) exit 1 ;; esac
STUB
chmod +x "$SANDBOX/bin/devcontainer"
run dcr /work >/dev/null
check "does not bootstrap a container that never came up" "$(not_called "linux_install.sh")"
check "only the failed up was attempted" "$(eq "$(count_of "=== devcontainer")" "1")"
teardown

echo ""
echo "$PASS passed, $FAIL failed"
[ "$FAIL" -eq 0 ]
