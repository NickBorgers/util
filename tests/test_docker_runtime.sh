#!/bin/bash
# Tests for lib/docker-runtime.sh, the docker wrapper dcs/dcr hand the
# devcontainer CLI. `docker` is a stub that records its argv.
#
# Run: ./tests/test_docker_runtime.sh   (or: make test)

set -uo pipefail

REPO_DIR="$(cd "$(dirname "$0")/.." && pwd)"
WRAPPER="$REPO_DIR/lib/docker-runtime.sh"
REAL_GREP="$(command -v grep)"
PASS=0
FAIL=0
SANDBOX=""

setup() {
    SANDBOX="$(mktemp -d)"
    export CALLS="$SANDBOX/calls.log"
    mkdir -p "$SANDBOX/bin"
    : >"$CALLS"
    # RUNTIMES_JSON is what `docker info --format {{json .Runtimes}}` returns.
    cat >"$SANDBOX/bin/docker" <<'STUB'
#!/bin/bash
if [ "${1:-}" = info ]; then echo "${RUNTIMES_JSON:-{\"runc\":{}}}"; exit 0; fi
printf '%s\n' "$@" >>"$CALLS"
echo "--" >>"$CALLS"
STUB
    chmod +x "$SANDBOX/bin/docker"
    export UTIL_REAL_DOCKER="$SANDBOX/bin/docker"
    ln -s "$REAL_GREP" "$SANDBOX/bin/grep"
    unset UTIL_DOCKER_RUNTIME UTIL_DOCKER_DNS RUNTIMES_JSON 2>/dev/null || true
}
teardown() { rm -rf "$SANDBOX"; }
# Only the wrapper sees the sandbox PATH, so a real tailscale or docker on this
# machine cannot leak into the cases, and the test shell keeps its own tools.
wrap() { PATH="$SANDBOX/bin" "$WRAPPER" "$@"; }
trap teardown EXIT

ok() { PASS=$((PASS + 1)); echo "  ok - $1"; }
no() { FAIL=$((FAIL + 1)); echo "  FAIL - $1"; }
check() { if [ "$2" = "1" ]; then ok "$1"; else no "$1"; fi; }
arg() { grep -qxF -- "$1" "$CALLS" && echo 1 || echo 0; }
not_arg() { grep -qxF -- "$1" "$CALLS" && echo 0 || echo 1; }

echo "== plain docker: nothing injected =="
setup
wrap run --rm alpine true
check "forwards run" "$(arg run)"
check "no --runtime" "$(not_arg "--runtime=kata-clh")"
check "no --dns" "$(not_arg "--dns")"
teardown

echo "== kata-clh registered: used automatically =="
setup
RUNTIMES_JSON='{"runc":{},"kata-clh":{}}' wrap run --rm alpine true
check "adds --runtime=kata-clh" "$(arg "--runtime=kata-clh")"
check "keeps the caller's arguments" "$(arg alpine)"
teardown

echo "== create is covered too =="
setup
RUNTIMES_JSON='{"kata-clh":{}}' wrap create alpine
check "adds --runtime=kata-clh" "$(arg "--runtime=kata-clh")"
teardown

echo "== opting out and overriding =="
setup
RUNTIMES_JSON='{"kata-clh":{}}' UTIL_DOCKER_RUNTIME=none wrap run alpine
check "none leaves the default runtime" "$(not_arg "--runtime=kata-clh")"
: >"$CALLS"
UTIL_DOCKER_RUNTIME=runsc wrap run alpine
check "an explicit runtime wins" "$(arg "--runtime=runsc")"
teardown

echo "== Tailscale DNS =="
setup
printf '#!/bin/sh\n' >"$SANDBOX/bin/tailscale"; chmod +x "$SANDBOX/bin/tailscale"
wrap run alpine
check "adds the Tailscale resolver when installed" "$(arg "100.100.100.100")"
: >"$CALLS"
UTIL_DOCKER_DNS=none wrap run alpine
check "DNS can be skipped" "$(not_arg "--dns")"
: >"$CALLS"
UTIL_DOCKER_DNS=1.1.1.1 wrap run alpine
check "DNS can be overridden" "$(arg "1.1.1.1")"
teardown

echo "== other subcommands are untouched =="
setup
RUNTIMES_JSON='{"kata-clh":{}}' wrap ps -q
check "forwards ps" "$(arg ps)"
check "no injected runtime" "$(not_arg "--runtime=kata-clh")"
teardown

echo ""
echo "$PASS passed, $FAIL failed"
[ "$FAIL" -eq 0 ]
