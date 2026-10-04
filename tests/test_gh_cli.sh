#!/bin/bash
# Tests for lib/gh-cli.sh. `curl`, `gh` and `uname` are stubs on a throwaway
# PATH, so nothing reaches the network or touches the real ~/.local.
#
# Run: ./tests/test_gh_cli.sh   (or: make test)

set -uo pipefail

REPO_DIR="$(cd "$(dirname "$0")/.." && pwd)"
unset -f gh curl uname 2>/dev/null || true
unset GH_RELEASES_URL 2>/dev/null || true

PASS=0
FAIL=0
SANDBOX=""
# shellcheck source=tests/shim_path.sh
source "$(dirname "$0")/shim_path.sh"
REAL_PATH="$(shim_path gh curl)"

setup() {
    export SANDBOX
    SANDBOX="$(mktemp -d)"
    export HOME="$SANDBOX/home"
    export CALLS="$SANDBOX/calls.log"
    mkdir -p "$HOME" "$SANDBOX/bin" "$SANDBOX/fake/gh_9.9.9_linux_amd64/bin"
    : >"$CALLS"
    printf '#!/bin/sh\necho fake gh\n' >"$SANDBOX/fake/gh_9.9.9_linux_amd64/bin/gh"
    chmod +x "$SANDBOX/fake/gh_9.9.9_linux_amd64/bin/gh"
    tar -czf "$SANDBOX/gh.tar.gz" -C "$SANDBOX/fake" gh_9.9.9_linux_amd64
    # -w means the tag lookup; anything else is the download, written to -o.
    cat >"$SANDBOX/bin/curl" <<'STUB'
#!/bin/bash
echo "curl $*" >>"$CALLS"
out=""; w=0
while [ $# -gt 0 ]; do
    case "$1" in -o) out="$2"; shift ;; -w) w=1 ;; esac
    shift
done
if [ "$w" = 1 ]; then printf 'https://github.com/cli/cli/releases/tag/%s' "${FAKE_TAG-v9.9.9}"; exit 0; fi
[ -n "$out" ] && cp "$SANDBOX/gh.tar.gz" "$out"
STUB
    chmod +x "$SANDBOX/bin/curl"
    export PATH="$SANDBOX/bin:$REAL_PATH"
}
teardown() { rm -rf "$SANDBOX"; }
trap teardown EXIT

ok() { PASS=$((PASS + 1)); echo "  ok - $1"; }
no() { FAIL=$((FAIL + 1)); echo "  FAIL - $1"; }
check() { if [ "$2" = "1" ]; then ok "$1"; else no "$1"; fi; }
called() { grep -qF -- "$1" "$CALLS" && echo 1 || echo 0; }
not_called() { grep -qF -- "$1" "$CALLS" && echo 0 || echo 1; }
install_it() { ( source "$REPO_DIR/lib/gh-cli.sh"; install_gh ) 2>&1; }

echo "== gh missing: installed from the release tarball =="
setup
OUT="$(install_it)"
check "looks up the latest tag" "$(called "releases/latest")"
check "downloads the matching asset" "$(called "download/v9.9.9/gh_9.9.9_linux_amd64.tar.gz")"
check "puts gh in ~/.local/bin" "$([ -x "$HOME/.local/bin/gh" ] && echo 1 || echo 0)"
teardown

echo "== arm64 gets the arm64 asset =="
setup
printf '#!/bin/sh\necho aarch64\n' >"$SANDBOX/bin/uname"; chmod +x "$SANDBOX/bin/uname"
printf '#!/bin/sh\n[ "$1" = "-m" ] && echo aarch64 || echo Linux\n' >"$SANDBOX/bin/uname"
mkdir -p "$SANDBOX/fake/gh_9.9.9_linux_arm64/bin"
cp "$SANDBOX/fake/gh_9.9.9_linux_amd64/bin/gh" "$SANDBOX/fake/gh_9.9.9_linux_arm64/bin/gh"
tar -czf "$SANDBOX/gh.tar.gz" -C "$SANDBOX/fake" gh_9.9.9_linux_arm64
install_it >/dev/null
check "downloads the arm64 asset" "$(called "gh_9.9.9_linux_arm64.tar.gz")"
teardown

echo "== gh already present: left alone =="
setup
printf '#!/bin/sh\nexit 0\n' >"$SANDBOX/bin/gh"; chmod +x "$SANDBOX/bin/gh"
OUT="$(install_it)"
check "no download" "$(not_called "curl")"
teardown

echo "== an unreadable release lookup skips, never fails =="
setup
OUT="$(FAKE_TAG="" install_it)"
check "says so" "$(grep -qF "could not work out" <<<"$OUT" && echo 1 || echo 0)"
check "downloads nothing" "$(not_called "download/")"
check "installs nothing" "$([ -e "$HOME/.local/bin/gh" ] && echo 0 || echo 1)"
teardown

echo "== git is wired to gh only when a token was handed in =="
setup
printf '#!/bin/sh\necho "gh $*" >>"$CALLS"\n' >"$SANDBOX/bin/gh"; chmod +x "$SANDBOX/bin/gh"
install_it >/dev/null
check "no setup-git without /run/util/gh-token" "$(not_called "auth setup-git")"
teardown

echo ""
echo "$PASS passed, $FAIL failed"
[ "$FAIL" -eq 0 ]
