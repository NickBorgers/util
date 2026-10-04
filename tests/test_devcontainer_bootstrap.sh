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
    echo '{}' >"$HOME/.claude.json"
    mkdir -p "$HOME/.claude/agents" "$HOME/.claude/skills/synced"
    echo hi >"$HOME/.claude/CLAUDE.md"
    mkdir -p "$HOME/.config/util"
    # A fresh access token, far from expiry, and a refresh token that must never
    # reach the container.
    local exp=$(( ($(date +%s) + 7200) * 1000 ))
    printf '{"claudeAiOauth":{"accessToken":"sk-ant-oat01-ACCESS","refreshToken":"sk-ant-ort01-REFRESH","expiresAt":%s}}' "$exp" >"$HOME/.claude/.credentials.json"
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
# For the helpers' own stderr, which never reaches the devcontainer stub.
not_called_out() { grep -qF -- "$2" <<<"$1" && echo 0 || echo 1; }
contains() { grep -qF -- "$2" <<<"$1" && echo 1 || echo 0; }
count_of() { grep -cF -- "$1" "$CALLS"; }
eq() { [ "$1" = "$2" ] && echo 1 || echo 0; }

echo "== dcr mounts the checkout and the credentials =="
setup
seed_credentials
run dcr /work >/dev/null
check "mounts this checkout at /util" "$(called "type=bind,source=$REPO_DIR,target=/util")"
check "mounts the container Claude token read-only" \
    "$(called "type=bind,source=$HOME/.config/util/claude-oauth-token,target=/run/util/claude-oauth-token")"
check "never mounts the host Claude login (refresh token rotates)" "$(not_called ".claude/.credentials.json")"
check "mounts the host Claude config read-only" "$(called "target=/host-claude.json")"
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
check "mounts the container Claude token" "$(called "target=/run/util/claude-oauth-token")"
check "does not recreate the container" "$(not_called "--remove-existing-container")"
teardown

echo "== credentials that do not exist are not mounted =="
setup
run dcr /work >/dev/null
check "mounts the checkout regardless" "$(called "target=/util")"
check "no Claude token mount" "$(not_called "claude-oauth-token")"
check "no Codex mount" "$(not_called "auth.json")"
teardown

echo "== the container Claude token comes from the host login, without prompts =="
setup
seed_credentials
OUT="$(run dcr /work)"
TOKEN_FILE="$HOME/.config/util/claude-oauth-token"
check "writes the access token" "$(eq "$(cat "$TOKEN_FILE" 2>/dev/null)" "sk-ant-oat01-ACCESS")"
check "never writes the refresh token" "$([ -f "$TOKEN_FILE" ] && grep -q REFRESH "$TOKEN_FILE" && echo 0 || echo 1)"
check "keeps the token file private" "$(eq "$(stat -c %a "$TOKEN_FILE" 2>/dev/null)" "600")"
check "says how long it lasts" "$(contains "$OUT" "valid for about")"
check "mounts it read-only" "$(called "source=$TOKEN_FILE,target=/run/util/claude-oauth-token")"
teardown

echo "== dcs refreshes the token in place =="
setup
seed_credentials
run dcs /work >/dev/null
INODE_BEFORE="$(stat -c %i "$HOME/.config/util/claude-oauth-token")"
printf '{"claudeAiOauth":{"accessToken":"sk-ant-oat01-NEWER","expiresAt":%s}}' "$(( ($(date +%s) + 7200) * 1000 ))" >"$HOME/.claude/.credentials.json"
run dcs /work >/dev/null
check "picks up the host's newer token" "$(eq "$(cat "$HOME/.config/util/claude-oauth-token")" "sk-ant-oat01-NEWER")"
# A bind-mounted file only follows writes made in place; a rename would leave a
# running container on the old inode.
check "rewrites the same file" "$(eq "$(stat -c %i "$HOME/.config/util/claude-oauth-token")" "$INODE_BEFORE")"
teardown

echo "== an expired host token is not handed over =="
setup
seed_credentials
printf '{"claudeAiOauth":{"accessToken":"sk-ant-oat01-OLD","expiresAt":1000}}' >"$HOME/.claude/.credentials.json"
OUT="$(run dcr /work)"
check "says to refresh on the host" "$(contains "$OUT" "has expired")"
check "writes no token" "$([ -f "$HOME/.config/util/claude-oauth-token" ] && echo 0 || echo 1)"
check "mounts no token" "$(not_called "claude-oauth-token")"
teardown

echo "== no host login: container is left to log in itself =="
setup
OUT="$(run dcr /work)"
check "still brings up the container" "$(called "up")"
check "mounts no token" "$(not_called "claude-oauth-token")"
teardown

echo "== host-authored Claude config is shared read-only, never the login =="
setup
seed_credentials
run dcr /work >/dev/null
check "mounts CLAUDE.md read-only" "$(called "source=$HOME/.claude/CLAUDE.md,target=/host-claude-config/CLAUDE.md")"
check "mounts agents read-only" "$(called "target=/host-claude-config/agents")"
check "mounts synced skills read-only" "$(called "target=/host-claude-config/skills/synced")"
check "does not mount the whole ~/.claude" "$(not_called "target=/home/vscode/.claude,")"
check "bootstrap links it into place" "$(called 'ln -s "/host-claude-config/$rel"')"
teardown

echo "== host-authored Claude config that does not exist is not mounted =="
setup
run dcr /work >/dev/null
check "no host-claude-config mounts" "$(not_called "target=/host-claude-config")"
teardown

echo "== the docker wrapper is handed to the devcontainer CLI =="
setup
run dcr /work >/dev/null
check "passes --docker-path" "$(arg "--docker-path")"
check "pointing at lib/docker-runtime.sh" "$(arg "$REPO_DIR/lib/docker-runtime.sh")"
teardown

echo "== this project's memory is shared, and only this project's =="
setup
seed_credentials
mkdir -p "$SANDBOX/proj"
KEY="$(printf '%s' "$(cd "$SANDBOX/proj" && pwd -P)" | sed 's/[^A-Za-z0-9]/-/g')"
mkdir -p "$HOME/.claude/projects/$KEY/memory" "$HOME/.claude/projects/-other-project/memory"
echo note >"$HOME/.claude/projects/$KEY/memory/logs.md"
run dcr "$SANDBOX/proj" >/dev/null
check "mounts the project memory where Claude looks" "$(called "source=$HOME/.claude/projects/$KEY/memory,target=/home/vscode/.claude/projects/$KEY/memory")"
check "does not mount other projects" "$(not_called "-other-project")"
check "bootstrap hands back the mount-created parents" "$(called 'sudo chown "$(id -u):$(id -g)" "$HOME/.claude/projects"')"
check "passes the project key to the bootstrap" "$(arg "$KEY")"
teardown

echo "== memory sharing can be turned off =="
setup
seed_credentials
mkdir -p "$SANDBOX/proj"
KEY="$(printf '%s' "$(cd "$SANDBOX/proj" && pwd -P)" | sed 's/[^A-Za-z0-9]/-/g')"
mkdir -p "$HOME/.claude/projects/$KEY/memory"
UTIL_SHARE_MEMORY=none run dcr "$SANDBOX/proj" >/dev/null
check "no memory mount" "$(not_called "target=/home/vscode/.claude/projects/")"
teardown

echo "== a project with no memory mounts nothing =="
setup
seed_credentials
mkdir -p "$SANDBOX/proj"
run dcr "$SANDBOX/proj" >/dev/null
check "no memory mount" "$(not_called "target=/home/vscode/.claude/projects/")"
teardown

echo "== dcs says so when the running container lacks a mount =="
setup
seed_credentials
mkdir -p "$SANDBOX/proj"
KEY="$(printf '%s' "$(cd "$SANDBOX/proj" && pwd -P)" | sed 's/[^A-Za-z0-9]/-/g')"
mkdir -p "$HOME/.claude/projects/$KEY/memory"
# docker stub: `ps` finds a container; `inspect` lists the destinations in
# $HAVE_MOUNTS, one per line.
cat >"$SANDBOX/bin/docker" <<'STUB'
#!/bin/bash
case "${1:-}" in
    ps) echo abc123 ;;
    inspect) printf '%s\n' $HAVE_MOUNTS ;;
esac
STUB
chmod +x "$SANDBOX/bin/docker"
OUT="$(HAVE_MOUNTS="/util /run/util/claude-oauth-token" run dcs "$SANDBOX/proj")"
check "names the missing project memory" "$(contains "$OUT" "/home/vscode/.claude/projects/$KEY/memory")"
check "names a missing host config mount" "$(contains "$OUT" "/host-claude-config/CLAUDE.md")"
check "does not name a mount it has" "$(not_called_out "$OUT" "created without: /util")"
check "says to run dcr" "$(contains "$OUT" "Run dcr to recreate it")"
check "still brings the container up" "$(called "up")"
: >"$CALLS"
ALL="$(printf '%s ' /util /run/util/claude-oauth-token /host-claude.json /home/vscode/.codex/auth.json /host-claude-config/CLAUDE.md /host-claude-config/agents /host-claude-config/skills/synced "/home/vscode/.claude/projects/$KEY/memory")"
OUT="$(HAVE_MOUNTS="$ALL" run dcs "$SANDBOX/proj")"
check "silent when nothing is missing" "$(not_called_out "$OUT" "created without")"
teardown

echo "== dcs is quiet when there is no running container =="
setup
seed_credentials
cat >"$SANDBOX/bin/docker" <<'STUB'
#!/bin/bash
exit 0
STUB
chmod +x "$SANDBOX/bin/docker"
OUT="$(run dcs /work)"
check "no warning" "$(not_called_out "$OUT" "created without")"
teardown

echo "== a UTIL_DIR that is not a checkout is not mounted =="
setup
seed_credentials
# Docker would create the missing source as a root-owned directory on the host
# and mount an empty /util, so the mount has to be withheld, not merely survived.
OUT="$(UTIL_DIR="$SANDBOX/not-a-checkout" run dcr /work)"
check "no /util mount" "$(not_called "target=/util")"
check "says what is missing" "$(contains "$OUT" "util checkout not found")"
check "names the path it looked at" "$(contains "$OUT" "$SANDBOX/not-a-checkout")"
check "still brings up the container" "$(called "up")"
check "still mounts the credentials" "$(called "target=/home/vscode/.codex/auth.json")"
check "bootstrap degrades inside the container" "$(called "util is not mounted at /util")"
teardown

echo "== the container user is overridable =="
setup
seed_credentials
UTIL_DEVCONTAINER_USER=node run dcr /work >/dev/null
check "targets that user's home" "$(called "target=/home/node/.codex/auth.json")"
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
check "says how to fix it" "$(called "Run dcr to recreate it")"
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
