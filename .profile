function reduce_framerate() {
	docker run --rm -it --volume=$(pwd):/video/ --network=none linuxserver/ffmpeg -i "/video/$1" -filter:v fps=15 "/video/low_framerate.$1"
}
