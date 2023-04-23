function reduce_framerate() {
	docker run --rm -it --volume=$(pwd):/video/ --workdir=/video/ --network=none linuxserver/ffmpeg -i "$1" -filter:v fps=15 "/video/low_framerate.$1"
}

function heic_to_jpeg() {
	docker run --rm -it --volume=$(pwd):/image/ --workdir=/image/ --network=none dpokidov/imagemagick "$1" -quality 90% "$1.jpeg"
}
