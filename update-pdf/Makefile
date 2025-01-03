build:
	docker build -t update-pdf .

run: build
	./prep.sh
	docker run --rm --mount type=bind,source=${CURDIR}/content,destination=/content update-pdf
