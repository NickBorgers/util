#!/bin/bash
# Symlinks this repo's custom Claude Code output styles into
# ~/.claude/output-styles/.
#
# Sourced by linux_install.sh and mac_install.sh. Mirrors the tmux.conf
# symlink pattern in those scripts: the repo copy is the single source of
# truth, so editing a style here updates every machine that re-runs the
# installer, and nothing needs downloading or checksumming since the file
# never leaves the repo it was cloned as part of.
#
# Written for bash 3.2 so it works with the /bin/bash that ships on macOS.

# Symlinks every *.md file in $1 (the repo's claude-output-styles directory)
# into $HOME/.claude/output-styles/. Safe to re-run: already-correct symlinks
# are left alone, and anything else found at the destination (a real file, a
# symlink elsewhere) is backed up to "<name>.bak" first rather than clobbered.
# If "<name>.bak" already exists (an earlier backup this function itself
# never got a chance to reconcile), that file is skipped rather than
# overwriting the older backup.
install_claude_output_styles() {
    local src_dir="$1" dest_dir="$HOME/.claude/output-styles" src dest name

    [ -d "$src_dir" ] || return 0
    mkdir -p "$dest_dir"

    for src in "$src_dir"/*.md; do
        [ -e "$src" ] || continue  # no *.md files matched
        name="$(basename "$src")"
        dest="$dest_dir/$name"

        if [ -L "$dest" ] && [ "$(readlink "$dest")" = "$src" ]; then
            echo "  $name already linked."
        elif [ -e "$dest" ] || [ -L "$dest" ]; then
            if [ -e "$dest.bak" ] || [ -L "$dest.bak" ]; then
                echo "  Skipping $name: $dest.bak already exists, so backing up $dest would"
                echo "    overwrite it. Move $dest.bak aside and re-run to link $name."
                continue
            fi
            mv "$dest" "$dest.bak"
            ln -s "$src" "$dest"
            echo "  Backed up existing $name to $name.bak and linked to repo copy."
        else
            ln -s "$src" "$dest"
            echo "  Linked $name."
        fi
    done
}

# Sets settings.json's top-level "outputStyle" key to $1, so Claude Code uses
# it without a manual /output-style run. Mirrors the caveman statusline wiring
# elsewhere in these installers: a deliberate choice already in settings.json
# is never overwritten, only an unset or already-matching value is touched.
configure_claude_output_style() {
    local style="$1" settings="$HOME/.claude/settings.json" current tmp

    if ! command -v jq &>/dev/null; then
        echo "  NOTE: jq not available; outputStyle not configured."
        echo "    To use it: run '/output-style $style' inside Claude Code, or add"
        echo "    \"outputStyle\": \"$style\" to $settings."
        return 0
    fi

    mkdir -p "$HOME/.claude"
    [ -f "$settings" ] || echo '{}' >"$settings"

    current="$(jq -r '.outputStyle // ""' "$settings" 2>/dev/null || true)"
    if [ "$current" = "$style" ]; then
        echo "  outputStyle already set to $style."
        return 0
    fi
    if [ -n "$current" ]; then
        echo "  NOTE: $settings already sets outputStyle to \"$current\"; leaving it alone."
        echo "    To switch: run '/output-style $style' inside Claude Code."
        return 0
    fi

    tmp="$(mktemp)"
    if jq --arg style "$style" '.outputStyle = $style' "$settings" >"$tmp" 2>/dev/null; then
        mv "$tmp" "$settings"
        echo "  outputStyle set to $style in $settings."
    else
        rm -f "$tmp"
        echo "  WARNING: could not update $settings (invalid JSON or jq missing)."
    fi
}
