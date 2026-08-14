#!/bin/bash
# Tests for lib/agent-clis.sh.
#
# Each case runs the helper against a throwaway HOME with stubbed `curl`, `npm`,
# `mise`, `brew` and `uname` on PATH, so nothing touches the real ~/.local and
# nothing reaches the network. Stubs append their argv to $SANDBOX/calls.log,
# and the assertions read that log.
#
# Run: ./tests/test_agent_clis.sh   (or: make test)

set -uo pipefail

TESTS_DIR="$(cd "$(dirname "$0")" && pwd)"
REPO_DIR="$(cd "$TESTS_DIR/.." && pwd)"

# Exported shell functions outrank executables on PATH, so a `curl` or `npm`
# function inherited from the caller's profile would defeat the stubs - and a
# `curl` function could reach the real network. Drop them before anything runs.
unset -f claude codex curl npm mise brew uname mktemp ln 2>/dev/null || true
unset CLAUDE_INSTALL_URL CODEX_NPM_PACKAGE MISE_INSTALL_URL 2>/dev/null || true

PASS=0
FAIL=0
SANDBOX=""

# Hide every real copy of a program we stub, so the "not installed" cases
# actually see it missing on a machine where it is installed. Each case then
# gets its state solely from the stubs it writes.
# shellcheck source=tests/shim_path.sh
source "$TESTS_DIR/shim_path.sh"
REAL_PATH="$(shim_path claude codex npm node mise brew)"

setup() {
    # Exported: a stub body is inserted into the file as an unexpanded variable,
    # so the $SANDBOX inside one is resolved when the stub runs, not when it is
    # written. That is also what keeps "$@" and "$1" intact in the stub bodies.
    export SANDBOX
    SANDBOX="$(mktemp -d)"
    export HOME="$SANDBOX/home"
    export PATH="$SANDBOX/bin:$REAL_PATH"
    export CALLS="$SANDBOX/calls.log"
    mkdir -p "$HOME" "$SANDBOX/bin"
    : >"$CALLS"

    # uname defaults to Linux; the macOS case overrides it. Stubbed rather than
    # branched on in the test, so both platforms' paths run on either platform.
    stub uname 'echo "${FAKE_UNAME:-Linux}"'

    # curl serves two URLs. The Claude installer script creates the binary it
    # would have downloaded; the mise installer does the same for mise. Both are
    # piped into a shell by the helper, so what matters is the emitted script.
    stub curl '
if [ "${CURL_SHOULD_FAIL:-0}" = "1" ]; then exit 22; fi
url=""
for arg in "$@"; do case "$arg" in http*) url="$arg" ;; esac; done
case "$url" in
    *claude*)
        echo "mkdir -p \"$HOME/.local/bin\""
        echo "printf %s\\\\n \"#!/bin/bash\" \"echo 9.9.9 (Claude Code)\" > \"$HOME/.local/bin/claude\""
        echo "chmod +x \"$HOME/.local/bin/claude\""
        ;;
    *mise*)
        echo "mkdir -p \"$HOME/.local/bin\""
        echo "cp \"$SANDBOX/stubs/mise\" \"$HOME/.local/bin/mise\""
        ;;
    *) echo "exit 1" ;;
esac'

    # mise: `use --global node@lts` materialises a shim dir holding an npm stub.
    mkdir -p "$SANDBOX/stubs"
    write_stub "$SANDBOX/stubs/mise" '
mkdir -p "$HOME/.local/share/mise/shims"
cp "$SANDBOX/stubs/npm" "$HOME/.local/share/mise/shims/npm"'

    # npm: `install -g <pkg>` materialises the codex binary under the prefix.
    write_stub "$SANDBOX/stubs/npm" '
case "${1:-}" in
    install)
        mkdir -p "$HOME/.local/bin"
        printf %s\\n "#!/bin/bash" "echo codex-cli 9.9.9" > "$HOME/.local/bin/codex"
        chmod +x "$HOME/.local/bin/codex"
        ;;
esac'
}

teardown() {
    [ -n "$SANDBOX" ] && rm -rf "$SANDBOX"
    SANDBOX=""
}
trap 'teardown; rm -rf "$REAL_PATH"' EXIT

# write_stub <path> <body>: a recording stub. Logs "<name> <args>" to $CALLS
# first, so a call is observable even when the body exits non-zero.
write_stub() {
    local path="$1" body="$2"
    cat >"$path" <<STUB
#!/bin/bash
echo "\$(basename "\$0") \$*" >> "\$CALLS"
$body
STUB
    chmod +x "$path"
}

stub() { write_stub "$SANDBOX/bin/$1" "${2:-}"; }

# The two CLIs, already present before the helper runs.
stub_installed() {
    stub claude 'echo "1.2.3 (Claude Code)"'
    stub codex 'echo "codex-cli 1.2.3"'
}

run_helper() {
    # Subshell so `source` and the PATH edits cannot leak between cases.
    (
        set -euo pipefail
        source "$REPO_DIR/lib/agent-clis.sh"
        install_agent_clis
        echo "EXIT=$?"
        echo "FINAL_PATH=$PATH"
    ) 2>&1
}

ok() { PASS=$((PASS + 1)); echo "  ok - $1"; }
no() { FAIL=$((FAIL + 1)); echo "  FAIL - $1"; }
check() { if [ "$2" = "1" ]; then ok "$1"; else no "$1"; fi; }

exists() { [ -e "$1" ] && echo 1 || echo 0; }
missing() { [ -e "$1" ] && echo 0 || echo 1; }
contains() { grep -qF -- "$2" <<<"$1" && echo 1 || echo 0; }
lacks() { grep -qF -- "$2" <<<"$1" && echo 0 || echo 1; }
eq() { [ "$1" = "$2" ] && echo 1 || echo 0; }
called() { contains "$(cat "$CALLS")" "$1"; }
not_called() { lacks "$(cat "$CALLS")" "$1"; }

echo "== both CLIs already installed =="
setup
stub_installed
OUT="$(run_helper)"
check "does not re-download Claude Code" "$(not_called "curl")"
check "does not re-install Codex" "$(not_called "npm install")"
check "reports the Claude version it found" "$(contains "$OUT" "already installed (1.2.3 (Claude Code))")"
check "reports the Codex version it found" "$(contains "$OUT" "already installed (codex-cli 1.2.3)")"
teardown

echo "== Claude Code missing =="
setup
stub codex 'echo "codex-cli 1.2.3"'
OUT="$(run_helper)"
check "runs the official installer" "$(called "curl -fsSL https://claude.ai/install.sh")"
check "leaves claude on PATH" "$(exists "$HOME/.local/bin/claude")"
check "says so" "$(contains "$OUT" "Claude Code installed.")"
teardown

echo "== Codex missing, npm already present =="
setup
stub claude 'echo "1.2.3 (Claude Code)"'
cp "$SANDBOX/stubs/npm" "$SANDBOX/bin/npm"
OUT="$(run_helper)"
check "installs the published package" "$(called "npm install -g @openai/codex")"
check "keeps the prefix inside \$HOME" "$(called "npm config set prefix $HOME/.local")"
check "does not drag in mise" "$(not_called "mise")"
check "leaves codex on PATH" "$(exists "$HOME/.local/bin/codex")"
teardown

echo "== Codex missing, no Node at all (Linux) =="
setup
stub claude 'echo "1.2.3 (Claude Code)"'
OUT="$(run_helper)"
check "bootstraps mise" "$(called "curl -fsSL https://mise.run")"
check "installs node through it" "$(called "mise use --global node@lts")"
check "then installs Codex" "$(called "npm install -g @openai/codex")"
check "leaves codex on PATH" "$(exists "$HOME/.local/bin/codex")"
teardown

echo "== Codex missing, no Node at all (macOS) =="
setup
stub claude 'echo "1.2.3 (Claude Code)"'
stub brew 'cp "$SANDBOX/stubs/npm" "$SANDBOX/bin/npm"'
export FAKE_UNAME=Darwin
OUT="$(run_helper)"
unset FAKE_UNAME
check "uses Homebrew for Node" "$(called "brew install node")"
check "does not bootstrap mise" "$(not_called "mise")"
check "then installs Codex" "$(called "npm install -g @openai/codex")"
teardown

echo "== macOS without Homebrew =="
setup
stub claude 'echo "1.2.3 (Claude Code)"'
export FAKE_UNAME=Darwin
OUT="$(run_helper)"
unset FAKE_UNAME
check "does not attempt an install it cannot do" "$(not_called "npm install")"
check "explains why" "$(contains "$OUT" "Homebrew not found")"
check "prints the manual command" "$(contains "$OUT" "npm install -g @openai/codex")"
teardown

echo "== node is linked beside codex =="
setup
stub claude 'echo "1.2.3 (Claude Code)"'
mkdir -p "$SANDBOX/shims"
write_stub "$SANDBOX/shims/node" 'echo v24.0.0'
export PATH="$SANDBOX/shims:$PATH"
cp "$SANDBOX/stubs/npm" "$SANDBOX/bin/npm"
OUT="$(run_helper)"
check "links it" "$(exists "$HOME/.local/bin/node")"
check "points at the node it found" "$(eq "$(readlink "$HOME/.local/bin/node")" "$SANDBOX/shims/node")"
check "says why" "$(contains "$OUT" "so its shebang resolves")"
teardown

echo "== an existing codex still gets the node link =="
setup
stub_installed
mkdir -p "$SANDBOX/shims"
write_stub "$SANDBOX/shims/node" 'echo v24.0.0'
export PATH="$SANDBOX/shims:$PATH"
OUT="$(run_helper)"
check "does not reinstall codex" "$(not_called "npm install")"
check "links node anyway" "$(exists "$HOME/.local/bin/node")"
teardown

echo "== node installed but off PATH, as in a non-interactive bootstrap =="
setup
# Exactly the container case: mise has node on disk, but no profile has been
# sourced, so the shim dir is not on PATH and codex cannot run at all.
mkdir -p "$HOME/.local/share/mise/shims"
write_stub "$HOME/.local/share/mise/shims/node" 'echo v24.0.0'
stub claude 'echo "1.2.3 (Claude Code)"'
write_stub "$SANDBOX/bin/codex" 'exec node --version'
OUT="$(run_helper)"
check "finds the node mise installed" "$(exists "$HOME/.local/bin/node")"
check "reports a real version, not 'unknown'" "$(contains "$OUT" "already installed (v24.0.0)")"
teardown

echo "== a node already beside codex is left alone =="
setup
stub_installed
mkdir -p "$SANDBOX/shims" "$HOME/.local/bin"
write_stub "$SANDBOX/shims/node" 'echo v24.0.0'
write_stub "$HOME/.local/bin/node" 'echo v20.0.0'
export PATH="$SANDBOX/shims:$PATH"
OUT="$(run_helper)"
check "not replaced with a symlink" "$(eq "$(readlink "$HOME/.local/bin/node")" "")"
check "still the local one" "$(contains "$(bash "$HOME/.local/bin/node")" "v20.0.0")"
teardown

echo "== no node anywhere, nothing to link =="
setup
stub_installed
OUT="$(run_helper)"
check "no dangling link" "$(missing "$HOME/.local/bin/node")"
check "does not fail the bootstrap" "$(contains "$OUT" "EXIT=0")"
teardown

echo "== network down =="
setup
export CURL_SHOULD_FAIL=1
OUT="$(run_helper)"
unset CURL_SHOULD_FAIL
check "warns about Claude Code" "$(contains "$OUT" "Claude Code install failed")"
check "warns about Codex" "$(contains "$OUT" "no npm available")"
check "does not abort the bootstrap" "$(contains "$OUT" "EXIT=0")"
teardown

echo "== PATH is fixed up for the steps that follow =="
setup
stub_installed
OUT="$(run_helper)"
check "puts ~/.local/bin ahead of the rest" "$(contains "$OUT" "FINAL_PATH=$HOME/.local/bin:")"
check "creates it if absent" "$(exists "$HOME/.local/bin")"
teardown

echo ""
echo "$PASS passed, $FAIL failed"
[ "$FAIL" -eq 0 ]
