FROM ubuntu:20.04

# Avoid interactive prompts during package installation
ENV DEBIAN_FRONTEND=noninteractive

# Copy the list of packages into the container
COPY packages.txt /tmp/packages.txt
COPY entrypoint.sh /entrypoint.sh

# Install packages and clean up apt cache
RUN apt-get update && \
    xargs -a /tmp/packages.txt apt-get install -y --no-install-recommends && \
    rm -rf /var/lib/apt/lists/* /tmp/packages.txt && \
    chmod +x /entrypoint.sh

ENTRYPOINT ["/entrypoint.sh"]
