#!/bin/bash
# Shared helper for the standalone adversarial review skills.
#
# Sourced by linux_install.sh and mac_install.sh. Kept in its own file so the
# state-dependent logic can be exercised with a stubbed `claude`, a stubbed
# `curl`, and a temporary HOME - see tests/test_claude_skills.sh.
#
# Written for bash 3.2 so it works with the /bin/bash that ships on macOS:
# no associative arrays, no empty-array expansion under `set -u`.

SKILLS_BASE_URL="${SKILLS_BASE_URL:-https://raw.githubusercontent.com/wi-adam/agent-skills/main/plugins/claude/epic-workflow-tkt/skills}"
ADVERSARIAL_SKILLS="${ADVERSARIAL_SKILLS:-adversarial-code-review adversarial-design-review}"

# Manifest written into every skill directory this installer creates. Each line
# is "<checksum>  <path relative to the skill directory>". Deletion is driven by
# this manifest rather than by mere presence of the file: a directory being
# installer-created does not make everything inside it ours to delete, since the
# user may have edited or added files afterwards.
SKILL_MARKER=".installed-by-util-installer"

# Echoes "sha256:<digest>", or returns 1 if no SHA tooling is available. The
# algorithm is recorded alongside the digest so a manifest written on one machine
# is never silently compared against a different algorithm later. There is no
# weaker fallback on purpose: a checksum authorizes deletion, and cksum is not
# collision-resistant enough for that.
_file_checksum() {
    local out digest

    if command -v sha256sum &>/dev/null; then
        out="$(sha256sum "$1")" || return 1
    elif command -v shasum &>/dev/null; then
        out="$(shasum -a 256 "$1")" || return 1
    else
        return 1
    fi

    # Validate rather than trust: a tool that failed while still exiting 0, or
    # printed something unexpected, must not yield a manifest entry that can
    # never match - such an entry would strand the file forever.
    digest="${out%% *}"
    [ "${#digest}" -eq 64 ] || return 1
    case "$digest" in *[!0-9a-f]*) return 1 ;; esac

    echo "sha256:$digest"
}

# Every entry in a skill directory except the top-level manifest, one relative
# path per line. Deliberately not limited to regular files: symlinks, empty
# subdirectories and other entries are user content too, and must be able to
# block deletion rather than be silently stepped over.
_dir_entries() {
    (cd "$1" && find . -mindepth 1 ! -path "./$SKILL_MARKER" | sed 's|^\./||' | LC_ALL=C sort)
}

# Echoes the checksum the manifest recorded for a relative path, or returns 1.
_manifest_checksum_for() {
    local line
    while IFS= read -r line; do
        [ -n "$line" ] || continue
        if [ "${line#*  }" = "$2" ]; then
            echo "${line%%  *}"
            return 0
        fi
    done <"$1/$SKILL_MARKER"
    return 1
}

# Echoes exactly one of: provided | absent | unknown | no-claude
#
# `unknown` is deliberately distinct from `absent`. A failed enumeration (auth
# expired, CLI too old, transient error) must not be read as "no plugin", or the
# installer re-downloads standalone copies and recreates the duplication this
# logic exists to prevent.
detect_epic_workflow_plugin() {
    if ! command -v claude &>/dev/null; then
        echo "no-claude"
        return 0
    fi

    local output status=0
    output="$(claude plugin list 2>/dev/null)" || status=$?
    if [ "$status" -ne 0 ]; then
        echo "unknown"
        return 0
    fi

    if grep -Eq 'epic-workflow-(github|tkt)@wi-adam-skills' <<<"$output"; then
        echo "provided"
    else
        echo "absent"
    fi
}

# Delete only the manifest files that still match the checksum recorded at
# install time, then the manifest itself. Returns 0 if the directory came out
# empty and was removed, 1 if user files remained and it was preserved.
_remove_manifest_files() {
    local dir="$1" line sum rel actual all_removed=1

    while IFS= read -r line; do
        [ -n "$line" ] || continue
        sum="${line%%  *}"
        rel="${line#*  }"
        [ -e "$dir/$rel" ] || continue

        # Only ever delete a plain regular file whose digest still matches, and
        # only when the manifest recorded an algorithm we can still compute.
        if [ -L "$dir/$rel" ] || [ ! -f "$dir/$rel" ]; then
            all_removed=0
            continue
        fi
        case "$sum" in
            sha256:*) ;;
            *) all_removed=0; continue ;;
        esac

        actual="$(_file_checksum "$dir/$rel")" || actual=""
        if [ -n "$actual" ] && [ "$actual" = "$sum" ]; then
            rm -f "$dir/$rel"
        else
            all_removed=0
        fi
    done <"$dir/$SKILL_MARKER"

    if [ "$all_removed" -eq 1 ]; then
        rm -f "$dir/$SKILL_MARKER"
        # rmdir, never rm -rf: anything that appeared meanwhile blocks removal
        # instead of being destroyed.
        rmdir "$dir" 2>/dev/null && return 0
    fi
    return 1
}

# A copy from an installer version that predates the manifest. Treat it as ours
# to delete only if it is exactly one pristine SKILL.md matching what upstream
# serves today. Anything else - extra files, local edits, an older upstream
# revision, or no network to check against - is preserved.
_legacy_copy_is_pristine() {
    local dir="$1" skill="$2" tmp status=0

    # Exactly one entry, and it must be a real file rather than a symlink.
    [ "$(_dir_entries "$dir")" = "SKILL.md" ] || return 1
    [ -f "$dir/SKILL.md" ] && [ ! -L "$dir/SKILL.md" ] || return 1

    tmp="$(mktemp)"
    if curl -fsSL "$SKILLS_BASE_URL/$skill/SKILL.md" -o "$tmp"; then
        cmp -s "$tmp" "$dir/SKILL.md" || status=1
    else
        status=1
    fi
    rm -f "$tmp"
    return $status
}

# Clean up standalone copies now that a plugin provides the same skills.
remove_managed_skill_copies() {
    local skill dir kept=""

    for skill in $ADVERSARIAL_SKILLS; do
        dir="$HOME/.claude/skills/$skill"
        [ -e "$dir" ] || [ -L "$dir" ] || continue

        # A symlinked skill directory would make every check below apply to the
        # link while the deletions land on someone else's files. Never follow it.
        if [ -L "$dir" ] || [ ! -d "$dir" ]; then
            kept="$kept$dir"$'\n'
            continue
        fi

        if [ -f "$dir/$SKILL_MARKER" ] && [ ! -L "$dir/$SKILL_MARKER" ]; then
            if _remove_manifest_files "$dir"; then
                echo "  Removed duplicate standalone $skill (plugin now provides it)."
            else
                echo "  Removed this installer's files from $skill, kept your additions."
                kept="$kept$dir"$'\n'
            fi
        # Legacy cleanup applies only to directories with no manifest at all.
        # A manifest that exists but is not a plain file (a symlink, a
        # directory) is a signal something unexpected is going on - preserve.
        elif [ ! -e "$dir/$SKILL_MARKER" ] && [ ! -L "$dir/$SKILL_MARKER" ] &&
            _legacy_copy_is_pristine "$dir" "$skill"; then
            # rm the verified file then rmdir, so anything created between the
            # check and here survives instead of being swept up by rm -rf.
            rm -f "$dir/SKILL.md"
            if rmdir "$dir" 2>/dev/null; then
                echo "  Removed duplicate standalone $skill left by an older installer."
            else
                echo "  Removed the stale $skill SKILL.md, kept the rest of $dir."
                kept="$kept$dir"$'\n'
            fi
        else
            kept="$kept$dir"$'\n'
        fi
    done

    if [ -n "$kept" ]; then
        echo "  WARNING: these standalone skills duplicate the plugin but hold files this installer did not write:"
        printf '%s' "$kept" | while IFS= read -r dir; do
            [ -n "$dir" ] && echo "    $dir"
        done
        echo "  They were left in place in case you customized them. To remove:"
        printf '%s' "$kept" | while IFS= read -r dir; do
            [ -n "$dir" ] && echo "    rm -rf \"$dir\""
        done
    fi
}

# Download to a temp file first, so a failed download never leaves an empty,
# manifest-less directory behind that a later run would read as user content.
# Echoes why a skill directory must not be written to, or nothing if it is safe.
#
# Never overwrite, and never claim ownership of, anything we did not create -
# doing so would authorize a later run to delete user files. A symlinked
# directory or manifest counts as not ours. The -L arms matter for dangling
# symlinks, which fail -e.
_install_block_reason() {
    local dir="$1" recorded actual

    if [ -L "$dir" ]; then
        echo "$dir is a symlink"
        return 0
    fi
    if [ -e "$dir" ]; then
        if [ ! -d "$dir" ] || [ ! -f "$dir/$SKILL_MARKER" ] || [ -L "$dir/$SKILL_MARKER" ]; then
            echo "$dir already exists and was not created by this installer"
            return 0
        fi
    fi

    # A manifest says we wrote the file, not that the user left it alone.
    # Refresh only when it is still a plain file matching what we recorded.
    if [ -e "$dir/SKILL.md" ] || [ -L "$dir/SKILL.md" ]; then
        recorded=""
        actual=""
        if [ -f "$dir/SKILL.md" ] && [ ! -L "$dir/SKILL.md" ]; then
            recorded="$(_manifest_checksum_for "$dir" "SKILL.md")" || recorded=""
            [ -n "$recorded" ] && { actual="$(_file_checksum "$dir/SKILL.md")" || actual=""; }
        fi
        if [ -z "$recorded" ] || [ -z "$actual" ] || [ "$recorded" != "$actual" ]; then
            echo "$dir/SKILL.md has local changes"
            return 0
        fi
    fi
}

install_standalone_skills() {
    local skill dir tmp manifest_tmp sum reason

    for skill in $ADVERSARIAL_SKILLS; do
        dir="$HOME/.claude/skills/$skill"

        reason="$(_install_block_reason "$dir")"
        if [ -n "$reason" ]; then
            echo "  Skipping $skill: $reason."
            continue
        fi

        tmp="$(mktemp)"
        if ! curl -fsSL "$SKILLS_BASE_URL/$skill/SKILL.md" -o "$tmp"; then
            rm -f "$tmp"
            echo "  WARNING: failed to download $skill"
            continue
        fi

        # Hash before installing. A file we cannot checksum could never be
        # verified for cleanup later, so decline to write it at all.
        sum="$(_file_checksum "$tmp")" || sum=""
        if [ -z "$sum" ]; then
            rm -f "$tmp"
            echo "  WARNING: no working sha256 tool - skipping $skill rather than installing an unverifiable copy."
            continue
        fi

        # The checks above predate the download. Repeat them, since the whole
        # transfer is a window in which the destination can change underneath us.
        reason="$(_install_block_reason "$dir")"
        if [ -n "$reason" ]; then
            rm -f "$tmp"
            echo "  Skipping $skill: $reason."
            continue
        fi

        # Plain mkdir, not mkdir -p: it fails if anything appeared at the path
        # while we were downloading, rather than adopting it.
        if [ ! -d "$dir" ]; then
            mkdir -p "$(dirname "$dir")"
            if ! mkdir "$dir" 2>/dev/null; then
                rm -f "$tmp"
                echo "  Skipping $skill: $dir appeared while downloading."
                continue
            fi
        fi

        mv "$tmp" "$dir/SKILL.md"
        # Write the manifest via a temp file so the redirection cannot follow a
        # symlink out of the skill directory.
        manifest_tmp="$(mktemp)"
        printf '%s  %s\n' "$sum" "SKILL.md" >"$manifest_tmp"
        mv "$manifest_tmp" "$dir/$SKILL_MARKER"
        echo "  $skill installed."
    done
}

install_adversarial_skills() {
    local state
    state="$(detect_epic_workflow_plugin)"

    case "$state" in
        provided)
            echo "  Adversarial review skills already provided by an epic-workflow plugin; skipping standalone copies."
            remove_managed_skill_copies
            ;;
        unknown)
            echo "  WARNING: 'claude plugin list' failed, so plugin state is unknown."
            echo "  Leaving standalone skills untouched rather than risking duplicates."
            echo "  Re-run this installer once 'claude plugin list' works."
            ;;
        no-claude)
            echo "  NOTE: claude CLI not found - installing standalone copies for later."
            install_standalone_skills
            ;;
        absent)
            install_standalone_skills
            ;;
    esac
}
