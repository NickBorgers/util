#!/bin/bash
# Tests for lib/claude-skills.sh.
#
# Each case runs the helper against a throwaway HOME with stubbed `claude` and
# `curl` on PATH, so nothing touches the real ~/.claude or the network.
#
# Run: ./tests/test_claude_skills.sh   (or: make test)

set -uo pipefail

TESTS_DIR="$(cd "$(dirname "$0")" && pwd)"
REPO_DIR="$(cd "$TESTS_DIR/.." && pwd)"

# Exported shell functions outrank executables on PATH, so a `claude` or `curl`
# function inherited from the caller's profile would defeat the stubs - and a
# `curl` function could reach the real network. Drop them before anything runs.
unset -f claude curl mktemp cmp find ln 2>/dev/null || true
unset CURL_SHOULD_FAIL SKILLS_BASE_URL ADVERSARIAL_SKILLS 2>/dev/null || true

PASS=0
FAIL=0
SANDBOX=""

# Hide the real `claude` from PATH, so the "claude CLI not installed" case
# actually sees it missing on a machine where it is installed. Every case then
# gets its plugin state solely from the stub.
# shellcheck source=tests/shim_path.sh
source "$TESTS_DIR/shim_path.sh"
REAL_PATH="$(shim_path claude)"
STUB_BODY="stub skill body"

# Sandbox: temp HOME + temp bin dir prepended to PATH, torn down after each case.
setup() {
    SANDBOX="$(mktemp -d)"
    export HOME="$SANDBOX/home"
    export PATH="$SANDBOX/bin:$REAL_PATH"
    mkdir -p "$HOME/.claude/skills" "$SANDBOX/bin"

    # Stub curl: ignores the URL, writes a fixed body to the -o target.
    # CURL_RACE makes it mutate the destination mid-"download", standing in for
    # another process touching the skill directory while the transfer runs.
    cat >"$SANDBOX/bin/curl" <<STUB
#!/bin/bash
if [ "\${CURL_SHOULD_FAIL:-0}" = "1" ]; then exit 22; fi
out=""
url=""
while [ \$# -gt 0 ]; do
    case "\$1" in
        -o) out="\$2"; shift 2 ;;
        http*) url="\$1"; shift ;;
        *) shift ;;
    esac
done
[ -n "\$out" ] || exit 2
if [ -n "\${CURL_RACE:-}" ] && [ -n "\$url" ]; then
    skill="\$(basename "\$(dirname "\$url")")"
    raced="\$HOME/.claude/skills/\$skill"
    mkdir -p "\$raced"
    echo "content from another process" > "\$raced/SKILL.md"
fi
echo "$STUB_BODY" > "\$out"
STUB
    chmod +x "$SANDBOX/bin/curl"
}

teardown() {
    [ -n "$SANDBOX" ] && rm -rf "$SANDBOX"
    SANDBOX=""
}
trap 'teardown; rm -rf "$REAL_PATH"' EXIT

# stub_claude <mode>: present | missing-plugin | broken | absent
stub_claude() {
    case "$1" in
        absent) return 0 ;;  # no stub written, so `command -v claude` fails
        present)
            cat >"$SANDBOX/bin/claude" <<'STUB'
#!/bin/bash
echo "caveman@caveman"
echo "epic-workflow-github@wi-adam-skills"
STUB
            ;;
        missing-plugin)
            cat >"$SANDBOX/bin/claude" <<'STUB'
#!/bin/bash
echo "caveman@caveman"
STUB
            ;;
        broken)
            cat >"$SANDBOX/bin/claude" <<'STUB'
#!/bin/bash
echo "error: not authenticated" >&2
exit 1
STUB
            ;;
    esac
    chmod +x "$SANDBOX/bin/claude"
}

checksum_of() (
    source "$REPO_DIR/lib/claude-skills.sh"
    _file_checksum "$1"
)

skill_dir() { echo "$HOME/.claude/skills/$1"; }

# A copy this installer version wrote: content plus a matching manifest.
seed_managed_copy() {
    local dir
    dir="$(skill_dir "$1")"
    mkdir -p "$dir"
    echo "${2:-$STUB_BODY}" >"$dir/SKILL.md"
    printf '%s  %s\n' "$(checksum_of "$dir/SKILL.md")" "SKILL.md" \
        >"$dir/.installed-by-util-installer"
}

# A copy from an installer version predating the manifest: SKILL.md only.
seed_legacy_copy() {
    local dir
    dir="$(skill_dir "$1")"
    mkdir -p "$dir"
    echo "${2:-$STUB_BODY}" >"$dir/SKILL.md"
}

run_helper() {
    # Subshell so `source` cannot leak state between cases.
    (
        set -euo pipefail
        ADVERSARIAL_SKILLS="adversarial-code-review adversarial-design-review"
        export ADVERSARIAL_SKILLS
        source "$REPO_DIR/lib/claude-skills.sh"
        install_adversarial_skills
    ) 2>&1
}

ok() { PASS=$((PASS + 1)); echo "  ok - $1"; }
no() { FAIL=$((FAIL + 1)); echo "  FAIL - $1"; }
check() { if [ "$2" = "1" ]; then ok "$1"; else no "$1"; fi; }

exists() { [ -e "$1" ] && echo 1 || echo 0; }
missing() { [ -e "$1" ] && echo 0 || echo 1; }
contains() { grep -qF -- "$2" <<<"$1" && echo 1 || echo 0; }

echo "== plugin present, copy written by this installer =="
setup
stub_claude present
seed_managed_copy adversarial-code-review
OUT="$(run_helper)"
check "removes the duplicate" "$(missing "$(skill_dir adversarial-code-review)")"
check "says it removed it" "$(contains "$OUT" "Removed duplicate standalone adversarial-code-review")"
teardown

echo "== plugin present, legacy copy from an older installer (no manifest) =="
setup
stub_claude present
seed_legacy_copy adversarial-code-review
OUT="$(run_helper)"
check "removes the legacy duplicate" "$(missing "$(skill_dir adversarial-code-review)")"
check "says it was left by an older installer" "$(contains "$OUT" "left by an older installer")"
teardown

echo "== plugin present, legacy copy the user edited =="
setup
stub_claude present
seed_legacy_copy adversarial-code-review "my own notes"
OUT="$(run_helper)"
check "keeps the edited copy" "$(exists "$(skill_dir adversarial-code-review)/SKILL.md")"
check "warns about it" "$(contains "$OUT" "did not write")"
check "prints a cleanup command" "$(contains "$OUT" "rm -rf")"
teardown

echo "== plugin present, legacy copy but download unavailable =="
setup
stub_claude present
seed_legacy_copy adversarial-code-review
export CURL_SHOULD_FAIL=1
OUT="$(run_helper)"
unset CURL_SHOULD_FAIL
check "keeps it when it cannot be verified" "$(exists "$(skill_dir adversarial-code-review)/SKILL.md")"
check "warns rather than guessing" "$(contains "$OUT" "did not write")"
teardown

echo "== plugin present, managed copy the user then edited =="
setup
stub_claude present
seed_managed_copy adversarial-code-review
echo "user edit" >>"$(skill_dir adversarial-code-review)/SKILL.md"
OUT="$(run_helper)"
check "does not delete the edited file" "$(exists "$(skill_dir adversarial-code-review)/SKILL.md")"
check "warns about it" "$(contains "$OUT" "did not write")"
teardown

echo "== plugin present, managed copy with an extra user file =="
setup
stub_claude present
seed_managed_copy adversarial-code-review
echo "notes" >"$(skill_dir adversarial-code-review)/NOTES.md"
OUT="$(run_helper)"
check "keeps the user's extra file" "$(exists "$(skill_dir adversarial-code-review)/NOTES.md")"
check "removes its own SKILL.md" "$(missing "$(skill_dir adversarial-code-review)/SKILL.md")"
check "reports what it kept" "$(contains "$OUT" "kept your additions")"
teardown

echo "== plugin present, no standalone copies =="
setup
stub_claude present
OUT="$(run_helper)"
check "installs nothing" "$(missing "$(skill_dir adversarial-code-review)")"
check "reports the skip" "$(contains "$OUT" "already provided by an epic-workflow plugin")"
teardown

echo "== plugin absent =="
setup
stub_claude missing-plugin
OUT="$(run_helper)"
check "installs code review skill" "$(exists "$(skill_dir adversarial-code-review)/SKILL.md")"
check "installs design review skill" "$(exists "$(skill_dir adversarial-design-review)/SKILL.md")"
check "records a manifest" "$(exists "$(skill_dir adversarial-code-review)/.installed-by-util-installer")"
teardown

echo "== plugin absent, user already has their own copy =="
setup
stub_claude missing-plugin
seed_legacy_copy adversarial-code-review "my own notes"
OUT="$(run_helper)"
check "does not overwrite it" "$(contains "$(cat "$(skill_dir adversarial-code-review)/SKILL.md")" "my own notes")"
check "does not claim ownership of it" "$(missing "$(skill_dir adversarial-code-review)/.installed-by-util-installer")"
check "says it skipped" "$(contains "$OUT" "was not created by this installer")"
teardown

echo "== plugin absent, managed copy the user edited (ADV-007) =="
setup
stub_claude missing-plugin
seed_managed_copy adversarial-code-review
echo "my own additions" >>"$(skill_dir adversarial-code-review)/SKILL.md"
OUT="$(run_helper)"
check "does not overwrite the edit" "$(contains "$(cat "$(skill_dir adversarial-code-review)/SKILL.md")" "my own additions")"
check "says it skipped" "$(contains "$OUT" "has local changes")"
teardown

echo "== plugin present, managed copy with an empty subdirectory (ADV-008) =="
setup
stub_claude present
seed_managed_copy adversarial-code-review
mkdir -p "$(skill_dir adversarial-code-review)/my-notes"
OUT="$(run_helper)"
check "keeps the empty subdirectory" "$(exists "$(skill_dir adversarial-code-review)/my-notes")"
check "still removes its own SKILL.md" "$(missing "$(skill_dir adversarial-code-review)/SKILL.md")"
teardown

echo "== plugin present, legacy copy that is a symlink (ADV-008) =="
setup
stub_claude present
mkdir -p "$(skill_dir adversarial-code-review)" "$SANDBOX/elsewhere"
echo "$STUB_BODY" >"$SANDBOX/elsewhere/SKILL.md"
ln -s "$SANDBOX/elsewhere/SKILL.md" "$(skill_dir adversarial-code-review)/SKILL.md"
OUT="$(run_helper)"
check "does not follow the symlink" "$(exists "$SANDBOX/elsewhere/SKILL.md")"
check "keeps the link" "$(exists "$(skill_dir adversarial-code-review)/SKILL.md")"
check "warns about it" "$(contains "$OUT" "did not write")"
teardown

echo "== plugin present, manifest from an unknown checksum algorithm (ADV-010) =="
setup
stub_claude present
seed_managed_copy adversarial-code-review
printf 'cksum:12345  %s\n' "SKILL.md" >"$(skill_dir adversarial-code-review)/.installed-by-util-installer"
OUT="$(run_helper)"
check "will not delete on an unrecognized algorithm" "$(exists "$(skill_dir adversarial-code-review)/SKILL.md")"
check "warns about it" "$(contains "$OUT" "did not write")"
teardown

echo "== plugin present, skill directory is a symlink (ADV-011) =="
setup
stub_claude present
mkdir -p "$SANDBOX/elsewhere"
echo "$STUB_BODY" >"$SANDBOX/elsewhere/SKILL.md"
ln -s "$SANDBOX/elsewhere" "$(skill_dir adversarial-code-review)"
OUT="$(run_helper)"
check "does not delete through the link" "$(exists "$SANDBOX/elsewhere/SKILL.md")"
check "keeps the link" "$(exists "$(skill_dir adversarial-code-review)")"
check "warns about it" "$(contains "$OUT" "did not write")"
teardown

echo "== plugin present, manifest is a symlink (ADV-011) =="
setup
stub_claude present
seed_managed_copy adversarial-code-review
mkdir -p "$SANDBOX/elsewhere"
mv "$(skill_dir adversarial-code-review)/.installed-by-util-installer" "$SANDBOX/elsewhere/manifest"
ln -s "$SANDBOX/elsewhere/manifest" "$(skill_dir adversarial-code-review)/.installed-by-util-installer"
OUT="$(run_helper)"
check "does not trust a symlinked manifest" "$(exists "$(skill_dir adversarial-code-review)/SKILL.md")"
check "warns about it" "$(contains "$OUT" "did not write")"
teardown

echo "== plugin absent, dangling SKILL.md symlink (ADV-011) =="
setup
stub_claude missing-plugin
seed_managed_copy adversarial-code-review
rm "$(skill_dir adversarial-code-review)/SKILL.md"
ln -s "$SANDBOX/gone" "$(skill_dir adversarial-code-review)/SKILL.md"
OUT="$(run_helper)"
check "does not replace the dangling link" "$(missing "$SANDBOX/gone")"
check "says it skipped" "$(contains "$OUT" "has local changes")"
teardown

echo "== plugin absent, sha256 tooling broken (ADV-012) =="
setup
stub_claude missing-plugin
for t in sha256sum shasum; do
    printf '#!/bin/bash\nexit 1\n' >"$SANDBOX/bin/$t"
    chmod +x "$SANDBOX/bin/$t"
done
OUT="$(run_helper)"
check "installs nothing unverifiable" "$(missing "$(skill_dir adversarial-code-review)/SKILL.md")"
check "warns about the missing tool" "$(contains "$OUT" "no working sha256 tool")"
teardown

echo "== plugin absent, directory appears mid-download (ADV-013) =="
setup
stub_claude missing-plugin
export CURL_RACE=1
OUT="$(run_helper)"
unset CURL_RACE
check "does not overwrite what appeared" "$(contains "$(cat "$(skill_dir adversarial-code-review)/SKILL.md")" "content from another process")"
check "does not claim ownership of it" "$(missing "$(skill_dir adversarial-code-review)/.installed-by-util-installer")"
check "says it skipped" "$(contains "$OUT" "Skipping adversarial-code-review")"
teardown

echo "== plugin enumeration fails =="
setup
stub_claude broken
seed_legacy_copy adversarial-code-review
OUT="$(run_helper)"
check "does not download duplicates" "$(missing "$(skill_dir adversarial-design-review)")"
check "leaves existing skills alone" "$(exists "$(skill_dir adversarial-code-review)/SKILL.md")"
check "warns instead of guessing" "$(contains "$OUT" "'claude plugin list' failed")"
teardown

echo "== claude CLI not installed =="
setup
stub_claude absent
OUT="$(run_helper)"
check "still installs copies" "$(exists "$(skill_dir adversarial-code-review)/SKILL.md")"
check "notes claude is missing" "$(contains "$OUT" "claude CLI not found")"
teardown

echo "== download failure leaves no empty dir =="
setup
stub_claude missing-plugin
export CURL_SHOULD_FAIL=1
OUT="$(run_helper)"
unset CURL_SHOULD_FAIL
check "no leftover dir" "$(missing "$(skill_dir adversarial-code-review)")"
check "warns about the failure" "$(contains "$OUT" "failed to download")"
teardown

echo ""
echo "$PASS passed, $FAIL failed"
[ "$FAIL" -eq 0 ]
