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

# Install unraid-diskmv scripts (diskmv and consld8) for disk management
# https://github.com/trinapicot/unraid-diskmv
RUN git clone --depth 1 https://github.com/trinapicot/unraid-diskmv.git /tmp/unraid-diskmv && \
    cp /tmp/unraid-diskmv/diskmv /tmp/unraid-diskmv/consld8 /usr/local/bin/ && \
    chmod +x /usr/local/bin/diskmv /usr/local/bin/consld8 && \
    rm -rf /tmp/unraid-diskmv

ENTRYPOINT ["/entrypoint.sh"]
