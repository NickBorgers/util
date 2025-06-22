FROM ubuntu:noble
# This is a copy of progrium's config
# https://github.com/progrium/docker-stress/blob/master/README.md

RUN apt-get update && apt-get install -y stress && rm -rf /var/lib/apt/lists/*

COPY script.sh /usr/lib/load.sh

CMD /usr/lib/load.sh
