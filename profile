function reduce_framerate() {
	docker run --rm -it --volume=$(pwd):/video/ --workdir=/video/ --network=none nickborgers/mov-to-gif ffmpeg -i "$1" -filter:v fps=15 "/video/low_framerate.$1"
}

function heic_to_jpeg() {
	docker run --rm -it --volume=$(pwd):/image/ --workdir=/image/ --network=none nickborgers/mov-to-gif magick "$1" -quality 90% "$1.jpeg"
}

function mov_to_gif() {
	docker run --rm -it --volume=$(pwd):/content/ --workdir=/content/ --network=none nickborgers/mov-to-gif mov-to-gif "$1"
}

function unrar() {
	docker run --rm -it --volume=$(pwd):/files/ --workdir=/files/ --network=none maxcnunes/unrar unrar e -r "$1"
}

function stabilize_video() {
	if [ -z "$2" ]; then
		ZOOM_PERCENTAGE=5
	else
		ZOOM_PERCENTAGE=$2
	fi
	docker run --rm -it --volume=$(pwd):/video/ --workdir=/video/ --network=none nickborgers/mov-to-gif ash -c "ffmpeg -i \"$1\" -vf vidstabdetect -f null - && ffmpeg -i \"$1\" -vf vidstabtransform=smoothing=30:zoom=50:input="transforms.trf" -c:v libx264 -crf 19 -preset slow -c:a copy \"/video/stabilized.$1\""
}

function update_pdf() {
	docker run --rm -it --volume=$(pwd):/content/ --workdir=/content/ --network=none nickborgers/update-pdf ash -c "gs -sDEVICE=pdfwrite -dCompatibilityLevel=1.4 -o /content/1_4.\"$1\" /content/\"$1\""
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
