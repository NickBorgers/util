FROM alpine:3.19

RUN apk add --no-cache \
    ffmpeg \
    python3

COPY smart-crop-video.py /usr/local/bin/smart-crop-video

RUN chmod +x /usr/local/bin/smart-crop-video
