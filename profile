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

