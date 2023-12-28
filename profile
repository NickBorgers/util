function reduce_framerate() {
	docker run --rm -it --volume=$(pwd):/video/ --workdir=/video/ --network=none nborgers/mov-to-gif ffmpeg -i "$1" -filter:v fps=15 "/video/low_framerate.$1"
}

function heic_to_jpeg() {
	docker run --rm -it --volume=$(pwd):/image/ --workdir=/image/ --network=none nborgers/mov-to-gif magick "$1" -quality 90% "$1.jpeg"
}

function mov_to_gif() {
	docker run --rm -it --volume=$(pwd):/content/ --workdir=/content/ --network=none nborgers/mov-to-gif mov-to-gif "$1"
}
