build:
	docker build -t mov-to-gif .

run: build
	./prep.sh
	docker run --rm --mount type=bind,source=${CURDIR}/content,destination=/content mov-to-gif
