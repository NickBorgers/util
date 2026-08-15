#!/bin/bash
# shim_path: build a PATH that resolves every real command EXCEPT the named ones.
#
# The tests need programs like `claude` and `npm` to look *absent* so the "not
# installed" branches run. Dropping whole PATH directories - the previous
# approach - is wrong on any host where those programs live in a shared bin dir:
# `claude` installed at /usr/bin/claude took /usr/bin off PATH and the tests lost
# mkdir, grep and rm with it, failing en masse for reasons having nothing to do
# with the code under test.
#
# Instead, symlink each command into one shim directory and skip only the named
# programs. Earlier PATH entries win, as they would have normally.
#
# Usage:
#   source "$TESTS_DIR/shim_path.sh"
#   REAL_PATH="$(shim_path claude codex npm)"
#   trap 'rm -rf "$REAL_PATH"' EXIT
shim_path() {
    local excluded=" $* " dir entry name shim
    shim="$(command mktemp -d)"
    while IFS= read -r dir; do
        [ -n "$dir" ] && [ -d "$dir" ] || continue
        for entry in "$dir"/*; do
            [ -f "$entry" ] && [ -x "$entry" ] || continue
            name="${entry##*/}"
            case "$excluded" in *" $name "*) continue ;; esac
            [ -e "$shim/$name" ] && continue
            command ln -s "$entry" "$shim/$name" 2>/dev/null || true
        done
    done <<<"${PATH//:/$'\n'}"
    echo "$shim"
}
