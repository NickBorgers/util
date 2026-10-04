function reduce_framerate() {
	docker run --rm -t --volume=$(pwd):/video/ --workdir=/video/ --network=none nickborgers/mov-to-gif ffmpeg -i "$1" -filter:v fps=15 "/video/low_framerate.$1"
}

function heic_to_jpeg() {
	docker run --rm -t --volume=$(pwd):/image/ --workdir=/image/ --network=none nickborgers/mov-to-gif magick "$1" -quality 90% "$1.jpeg"
}

function mov_to_gif() {
	docker run --rm -t --volume=$(pwd):/content/ --workdir=/content/ --network=none nickborgers/mov-to-gif mov-to-gif "$1"
}

function unrar() {
	docker run --rm -t --volume=$(pwd):/files/ --workdir=/files/ --network=none maxcnunes/unrar unrar e -r "$1"
}

function stabilize_video() {
	if [ -z "$2" ]; then
		ZOOM_PERCENTAGE=5
	else
		ZOOM_PERCENTAGE=$2
	fi
	docker run --rm -t --volume=$(pwd):/video/ --workdir=/video/ --network=none nickborgers/mov-to-gif ash -c "ffmpeg -i \"$1\" -vf vidstabdetect -f null - && ffmpeg -i \"$1\" -vf vidstabtransform=smoothing=30:zoom=50:input="transforms.trf" -c:v libx264 -crf 19 -preset slow -c:a copy \"/video/stabilized.$1\""
}

function smart_crop_video() {
	# Intelligent video cropping with motion and visual analysis
	# Usage: smart_crop_video input.mp4 [output.mp4] [aspect_ratio]
	# Examples:
	#   smart_crop_video video.mp4                          # Outputs: video_cropped.mp4, aspect: 9:16
	#   smart_crop_video video.mp4 output.mp4               # Custom output, aspect: 9:16
	#   smart_crop_video video.mp4 output.mp4 1:1           # Custom output and aspect
	#
	# Environment variables for configuration:
	# - PRESET: FFmpeg encoding preset (ultrafast/fast/medium/slow/veryslow, default: medium)
	# - ANALYSIS_FRAMES: Number of frames to analyze per position (default: 50)
	# - CROP_SCALE: Crop size scale factor 0.0-1.0 (default: 0.75)
	# - SCENE_THRESHOLD: Scene detection sensitivity 0.0-1.0 (default: 0.2, lower=more scenes)
	# - SEGMENT_DURATION: Time-based segment duration in seconds (default: 5.0)

	# Parse arguments with defaults
	local input="$1"
	if [ -z "$input" ]; then
		echo "Error: Input file required"
		echo "Usage: smart_crop_video input.mp4 [output.mp4] [aspect_ratio]"
		return 1
	fi

	local output="${2:-${input%.*}_cropped.mp4}"
	local aspect="${3:-9:16}"

	# Launch browser after delay (background job for bash/zsh)
	(
		sleep 3
		if [[ $(uname) == "Darwin" ]]; then
			open "http://localhost:8765" 2>/dev/null
		elif command -v xdg-open >/dev/null 2>&1; then
			xdg-open "http://localhost:8765" 2>/dev/null
		elif command -v wslview >/dev/null 2>&1; then
			wslview "http://localhost:8765" 2>/dev/null
		fi
	) &

	# Run the docker command (foreground, interactive)
	docker run --rm -it --volume=$(pwd):/content/ --workdir=/content/ -p 8765:8765 \
		-e PRESET="${PRESET:-medium}" \
		-e ANALYSIS_FRAMES="${ANALYSIS_FRAMES:-50}" \
		-e CROP_SCALE="${CROP_SCALE:-0.75}" \
		-e SCENE_THRESHOLD="${SCENE_THRESHOLD:-0.2}" \
		-e SEGMENT_DURATION="${SEGMENT_DURATION:-5.0}" \
		nickborgers/smart-crop-video "$input" "$output" "$aspect"
}

function update_pdf() {
	docker run --rm -t --volume=$(pwd):/content/ --workdir=/content/ --network=none nickborgers/update-pdf ash -c "gs -sDEVICE=pdfwrite -dCompatibilityLevel=1.4 -o /content/1_4.\"$1\" /content/\"$1\""
}

function md_to_pdf() {
	docker run --rm -t --volume=$(pwd):/data/ --workdir=/data/ --network=none ghcr.io/nickborgers/util/md-to-pdf:latest "$1" "${2:-${1%.md}.pdf}"
}

function get_docker_pids() {
	docker ps --format '{{.ID}} {{.Names}}' | while read cid cname; do \
	  for pid in $(docker inspect --format '{{.State.Pid}}' "$cid"); do \
	    uid=$(awk '/^Uid:/ {print $2}' /proc/$pid/status 2>/dev/null); \
	    user=$(getent passwd "$uid" | cut -d: -f1); \
	    [ -z "$user" ] && user="(unknown)"; \
	    echo "$cname $uid $user $pid"; \
	  done; \
	done
}

function network_blip() {
    LOGFILE="/tmp/network_blips.log"
    {
        set -x

        date

        # Determine OS
        if [[ $(uname) == "Darwin" ]]; then
            # macOS: get default gateway
            GATEWAY=$(route -n get default | awk '/gateway/ {print $2}')
            ifconfig
            netstat -rn
        else
            # Linux: get default gateway
            GATEWAY=$(ip route | awk '/^default/ {print $3}')
            ip addr
            ip route
        fi

        arp -a

        ping -c 2 -t 1 8.8.8.8
        ping -c 2 -t 1 $GATEWAY

        set +x
    } >>"$LOGFILE" 2>&1
}

# Add mise shims to PATH if present (provides node/npm for devcontainer CLI).
# Using shims instead of `mise activate` so node resolves inside function bodies,
# not just after the next prompt fires (activate uses PROMPT_COMMAND hook).
if [ -d "$HOME/.local/share/mise/shims" ]; then
	case ":$PATH:" in
		*":$HOME/.local/share/mise/shims:"*) ;;
		*) export PATH="$HOME/.local/share/mise/shims:$PATH" ;;
	esac
fi

function _ensure_devcontainer_cli() {
	if command -v devcontainer &> /dev/null; then
		return 0
	fi
	echo "devcontainer CLI not found, installing..."
	if ! command -v mise &> /dev/null; then
		if [[ "$(uname)" == "Darwin" ]]; then
			brew install mise || { echo "Failed to install mise"; return 1; }
		else
			sudo apt-get update && sudo apt-get install -y gpg sudo wget curl && \
			sudo install -dm 755 /etc/apt/keyrings && \
			wget -qO - https://mise.jdx.dev/gpg-key.pub | gpg --dearmor | sudo tee /etc/apt/keyrings/mise-archive-keyring.gpg 1> /dev/null && \
			echo "deb [signed-by=/etc/apt/keyrings/mise-archive-keyring.gpg arch=amd64] https://mise.jdx.dev/deb stable main" | sudo tee /etc/apt/sources.list.d/mise.list && \
			sudo apt-get update && sudo apt-get install -y mise || { echo "Failed to install mise"; return 1; }
		fi
	fi
	mise use --global node@lts || { echo "Failed to install node via mise"; return 1; }
	# Ensure mise shims are on PATH for the rest of this shell so node/npm/devcontainer
	# resolve immediately (without waiting for PROMPT_COMMAND).
	if [ -d "$HOME/.local/share/mise/shims" ]; then
		case ":$PATH:" in
			*":$HOME/.local/share/mise/shims:"*) ;;
			*) export PATH="$HOME/.local/share/mise/shims:$PATH" ;;
		esac
	fi
	npm config set prefix ~/.local && \
	npm install -g @devcontainers/cli || { echo "Failed to install devcontainer CLI"; return 1; }
}

# Where this checkout lives. The rc file sources the profile by absolute path,
# so bash can locate itself; zsh has no BASH_SOURCE, hence the fallback. Set
# UTIL_DIR yourself if the checkout is somewhere else.
if [ -z "${UTIL_DIR:-}" ]; then
	if [ -n "${BASH_SOURCE:-}" ]; then
		UTIL_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
	else
		UTIL_DIR="$HOME/code/util"
	fi
	export UTIL_DIR
fi

# The account a devcontainer runs as. Nearly every image built on the
# devcontainers base images uses `vscode`; override for one that does not.
: "${UTIL_DEVCONTAINER_USER:=vscode}"

# Where the container's Claude token lives on the host. Not under ~/.claude:
# that directory is the host login's, and this token must stay independent of it.
function _claude_container_token_file() {
	echo "${UTIL_CLAUDE_TOKEN_FILE:-$HOME/.config/util/claude-oauth-token}"
}

# Make sure a container-only Claude token exists and is fresh, creating or
# renewing it so dcs/dcr stay "run it and it works". `claude setup-token` mints
# a long-lived token that is separate from the host login, so a container can
# never rotate the host's refresh token. Needs a terminal and a browser; without
# them it warns and the container starts unauthenticated (or keeps the old token).
#
# Renewed when missing, older than UTIL_CLAUDE_TOKEN_MAX_AGE_DAYS (default 300;
# setup-token tokens last about a year), or when UTIL_REFRESH_CLAUDE_TOKEN=1.
# A failed renewal keeps the existing token. The file is rewritten in place so a
# running container's bind mount sees the new value; new shells pick it up.
function _ensure_claude_container_token() {
	local file max_age
	file="$(_claude_container_token_file)"
	max_age="${UTIL_CLAUDE_TOKEN_MAX_AGE_DAYS:-300}"

	local reason=""
	if [ ! -s "$file" ]; then
		reason="No container Claude token yet"
	elif [ -n "${UTIL_REFRESH_CLAUDE_TOKEN:-}" ]; then
		reason="Refreshing the container Claude token (requested)"
	elif [ -n "$(find "$file" -mtime +"$max_age" 2>/dev/null)" ]; then
		reason="Container Claude token is over $max_age days old"
	fi
	[ -z "$reason" ] && return 0

	command -v claude &>/dev/null || return 0
	if [ ! -t 0 ] || [ ! -t 1 ]; then
		echo "$reason, and there is no terminal to renew it." >&2
		echo "  Run dcs/dcr from a terminal to create it (uses \`claude setup-token\`)." >&2
		return 0
	fi

	echo "$reason. Running \`claude setup-token\`..." >&2
	local log token
	log="$(mktemp)"
	chmod 600 "$log"
	if command -v script &>/dev/null && [ "$(uname)" != "Darwin" ]; then
		script -qec "claude setup-token" "$log" || true
	else
		claude setup-token || true
	fi
	token="$(tr -d '\r' <"$log" 2>/dev/null | sed 's/\x1b\[[0-9;?]*[A-Za-z]//g' | grep -o 'sk-ant-[A-Za-z0-9_-]\{40,\}' | tail -1)"
	rm -f "$log"
	if [ -z "$token" ]; then
		printf 'Could not read the token automatically. Paste it here (blank to skip, input hidden): ' >&2
		read -rs token
		echo >&2
	fi
	if [ -z "$token" ]; then
		echo "No new token saved; keeping what was there." >&2
		return 0
	fi
	mkdir -p "$(dirname "$file")" && chmod 700 "$(dirname "$file")"
	( umask 077; printf '%s\n' "$token" >"$file" )
	echo "Saved container Claude token to $file" >&2
}

# Sets _DC_MOUNTS to the bind mounts every container should get.
#
# A devcontainer is otherwise whatever its base image shipped: no profile, no
# tmux config, and none of the agent CLIs - a project's devcontainer.json
# describes that project, not this machine, and should not have to know about
# either. So mount this checkout and let its bootstrap run inside.
#
# Credentials are mounted per-file rather than by mounting ~/.claude wholesale:
# the container installs its own plugins and settings, and those must not write
# back over the host's. Read-write on purpose - both CLIs rotate their tokens,
# and a container refreshing against a copy would strand the host on a token
# that is no longer valid.
function _devcontainer_util_mounts() {
	_DC_MOUNTS=()
	local home="/home/$UTIL_DEVCONTAINER_USER" rel

	# The devcontainer CLI has no runtime or DNS flag, so route its docker calls
	# through a wrapper that adds them: Kata microVM isolation when Docker has it
	# registered, Tailscale DNS when this host has Tailscale. Both auto-detect and
	# pass through untouched otherwise; see lib/docker-runtime.sh.
	if [ -x "$UTIL_DIR/lib/docker-runtime.sh" ]; then
		_DC_MOUNTS+=(--docker-path "$UTIL_DIR/lib/docker-runtime.sh")
	fi

	# Only mount a real checkout. Docker creates a missing bind source as a
	# root-owned directory on the host, so a wrong UTIL_DIR - the zsh fallback
	# on a machine that keeps the checkout elsewhere, say - would silently
	# litter one and mount an empty /util. The bootstrap tolerates the absence;
	# a root-owned ~/code/util that later blocks a clone is worse.
	if [ -x "$UTIL_DIR/linux_install.sh" ]; then
		_DC_MOUNTS+=(--mount "type=bind,source=$UTIL_DIR,target=/util")
	else
		echo "util checkout not found at $UTIL_DIR; container gets no profile or agent CLIs." >&2
		echo "  Set UTIL_DIR to the checkout to change that." >&2
	fi

	# Claude does not get the host's .credentials.json. Its OAuth refresh token
	# rotates on use, so a container refreshing from a shared copy invalidates
	# the host's - and, when the other side presents the already-rotated token,
	# can get every session logged out at once. Containers authenticate with
	# their own long-lived `claude setup-token` token instead, handed in as a
	# read-only file the container's profile exports as CLAUDE_CODE_OAUTH_TOKEN.
	local token_file
	token_file="$(_claude_container_token_file)"
	if [ -s "$token_file" ]; then
		_DC_MOUNTS+=(--mount "type=bind,source=$token_file,target=/run/util/claude-oauth-token,readonly")
	fi

	for rel in ".codex/auth.json"; do
		[ -f "$HOME/$rel" ] || continue
		_DC_MOUNTS+=(--mount "type=bind,source=$HOME/$rel,target=$home/$rel")
	done

	# Host-authored Claude config the bootstrap cannot recreate: global
	# CLAUDE.md, hand-written agents, synced skills. Mounted read-only under a
	# staging path and symlinked into ~/.claude by the bootstrap, rather than over
	# ~/.claude itself - that keeps the container's own installs writable, avoids
	# docker creating root-owned mount points in its home, and never exposes
	# .credentials.json (a readable refresh token is one the container can rotate).
	for rel in "CLAUDE.md" "agents" "skills/synced"; do
		[ -e "$HOME/.claude/$rel" ] || continue
		_DC_MOUNTS+=(--mount "type=bind,source=$HOME/.claude/$rel,target=/host-claude-config/$rel,readonly")
	done

	# The token alone does not spare you a login: onboarding state lives in
	# ~/.claude.json. That file also holds per-path project history and the
	# container's own MCP config, so it is mounted aside rather than over the
	# container's copy, and the bootstrap lifts only the identity keys out.
	if [ -f "$HOME/.claude.json" ]; then
		_DC_MOUNTS+=(--mount "type=bind,source=$HOME/.claude.json,target=/host-claude.json,readonly")
	fi
}

# Run the bootstrap inside the container, once per container. Stamped because
# `dcs` reuses a running container, and re-running plugin installs on every
# attach would put a network round trip in front of every shell.
# UTIL_FORCE_BOOTSTRAP=1 re-runs it anyway.
function _devcontainer_util_bootstrap() {
	local workspace="$1"
	devcontainer exec --workspace-folder "$workspace" bash -lc '
		stamp="$HOME/.util-bootstrapped"
		if [ -e "$stamp" ] && [ -z "$1" ]; then exit 0; fi
		if [ ! -x /util/linux_install.sh ]; then
			# Mounts are fixed when a container is created, so a container that
			# something else brought up - an editor, a plain devcontainer up -
			# can never gain /util. Recreating is the only fix, so say so
			# rather than leaving a shell with no claude and no explanation.
			echo "util is not mounted at /util; skipping bootstrap." >&2
			echo "  This container was not created by dcs/dcr. Run dcr to recreate it." >&2
			exit 0
		fi
		# Bind-mounting a credential file makes the daemon create its parent
		# directory as root. Hand those back before anything tries to write a
		# settings file into them. Non-recursive on purpose: the mounted files
		# belong to the host, and chowning through a bind mount would retitle
		# them there too.
		sudo chown "$(id -u):$(id -g)" "$HOME/.claude" "$HOME/.codex" 2>/dev/null || true
		# Link the read-only host Claude config (see _devcontainer_util_mounts)
		# into place. Never replaces anything the container already has.
		if [ -d /host-claude-config ]; then
			for rel in CLAUDE.md agents skills/synced; do
				[ -e "/host-claude-config/$rel" ] || continue
				[ -e "$HOME/.claude/$rel" ] || [ -L "$HOME/.claude/$rel" ] && continue
				mkdir -p "$HOME/.claude/$(dirname "$rel")"
				ln -s "/host-claude-config/$rel" "$HOME/.claude/$rel"
			done
		fi
		UTIL_SKIP_PACKAGES=1 /util/linux_install.sh && touch "$stamp"
	' util-bootstrap "${UTIL_FORCE_BOOTSTRAP:-}"
}

function dcs() {
	_ensure_devcontainer_cli || return 1
	local workspace="${1:-.}"
	_ensure_claude_container_token
	_devcontainer_util_mounts
	devcontainer up --workspace-folder "$workspace" "${_DC_MOUNTS[@]}" && \
	_devcontainer_util_bootstrap "$workspace" && \
	devcontainer exec --workspace-folder "$workspace" bash
}

function dcr() {
	_ensure_devcontainer_cli || return 1
	local workspace="${1:-.}"
	_ensure_claude_container_token
	_devcontainer_util_mounts
	devcontainer up --workspace-folder "$workspace" --remove-existing-container "${_DC_MOUNTS[@]}" && \
	_devcontainer_util_bootstrap "$workspace" && \
	devcontainer exec --workspace-folder "$workspace" bash
}

function mosht() {
	local host="$1"
	local session="${2:-main}"
	if [ -z "$host" ]; then
		echo "Usage: mosht <host> [session-name]"
		return 1
	fi
	ssh "$host" "pkill mosh-server; true"
	# Show "mosht <host>:<window>" in the terminal tab. tmux runs on the remote,
	# so it can't know the alias we typed here; pass it in via set-titles-string.
	mosh "$host" -- bash -c "tmux set-option -g set-titles on; tmux set-option -g set-titles-string 'mosht $host:#{window_name}'; tmux attach -t $session || tmux new-session -s $session"
}

function ett() {
	local host="$1"
	local session="${2:-main}"
	if [ -z "$host" ]; then
		echo "Usage: ett <host> [session-name]"
		return 1
	fi
	et "$host" -c "tmux attach -t $session || tmux new-session -s $session"
}

function publish_report() {
	if [ -z "$1" ]; then
		echo "Usage: publish_report <file-or-dir> [remote-path]"
		echo "Examples:"
		echo "  publish_report report.html"
		echo "  publish_report report.html 2026-05/weekly.html"
		echo "  publish_report ./output-dir/ reports/q2"
		return 1
	fi
	local remote_dir="/opt/dockergeneric/reports/html"
	if [ -n "$2" ]; then
		ssh dockergeneric "mkdir -p '$remote_dir/$(dirname "$2")'" && \
		scp -r "$1" "dockergeneric:$remote_dir/$2"
	else
		scp -r "$1" "dockergeneric:$remote_dir/"
	fi
}

alias claude-yolo='claude --dangerously-skip-permissions'
alias codex-yolo='codex --dangerously-bypass-approvals-and-sandbox'

# Inside a devcontainer, authenticate Claude with the container-only token
# dcs/dcr mounted (see _devcontainer_util_mounts) rather than the host login.
if [ -r /run/util/claude-oauth-token ]; then
	CLAUDE_CODE_OAUTH_TOKEN="$(cat /run/util/claude-oauth-token)"
	export CLAUDE_CODE_OAUTH_TOKEN
fi
