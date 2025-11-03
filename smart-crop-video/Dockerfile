FROM alpine:3.19

RUN apk add --no-cache \
    ffmpeg \
    python3 \
    py3-pip

# Install Flask for ephemeral webserver
RUN pip3 install --no-cache-dir flask --break-system-packages

# Copy the entire package (includes smart_crop modules)
COPY smart-crop-video.py /app/
COPY smart_crop/ /app/smart_crop/

# Set Python path to find smart_crop modules
ENV PYTHONPATH=/app

# Create wrapper script in PATH for easy command invocation
RUN echo '#!/bin/sh' > /usr/local/bin/smart-crop-video && \
    echo 'exec python3 /app/smart-crop-video.py "$@"' >> /usr/local/bin/smart-crop-video && \
    chmod +x /usr/local/bin/smart-crop-video

# Set working directory to /content (where volume is mounted)
# This ensures preview images are created in the mounted directory
WORKDIR /content

# Use CMD instead of ENTRYPOINT to allow tests to override the command
# while still providing a default for normal docker run usage
CMD ["smart-crop-video"]
