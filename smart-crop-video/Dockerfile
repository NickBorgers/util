FROM alpine:3.19

RUN apk add --no-cache \
    ffmpeg \
    python3 \
    py3-pip

# Install Flask for ephemeral webserver
RUN pip3 install --no-cache-dir flask --break-system-packages

COPY smart-crop-video.py /usr/local/bin/smart-crop-video

RUN chmod +x /usr/local/bin/smart-crop-video
