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

function dcs() {
	_ensure_devcontainer_cli || return 1
	local workspace="${1:-.}"
	local session
	session=$(basename "$workspace")
	devcontainer up --workspace-folder "$workspace" && \
	devcontainer exec --workspace-folder "$workspace" bash
}

function dcr() {
	_ensure_devcontainer_cli || return 1
	local workspace="${1:-.}"
	devcontainer up --workspace-folder "$workspace" --remove-existing-container && \
	devcontainer exec --workspace-folder "$workspace" bash
}

function mosht() {
	local host="$1"
	local session="${2:-main}"
	if [ -z "$host" ]; then
		echo "Usage: mosht <host> [session-name]"
		return 1
	fi
	mosh "$host" -- bash -c "tmux attach -t $session || tmux new-session -s $session"
}

alias claude-yolo='claude --dangerously-skip-permissions'

