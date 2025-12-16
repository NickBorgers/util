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

# Install nputop - Intel NPU monitoring tool
# https://github.com/ZoLArk173/nputop
RUN apt-get update && \
    apt-get install -y --no-install-recommends curl build-essential && \
    curl --proto '=https' --tlsv1.2 -sSf https://sh.rustup.rs | sh -s -- -y && \
    . $HOME/.cargo/env && \
    git clone --depth 1 https://github.com/ZoLArk173/nputop.git /tmp/nputop && \
    cd /tmp/nputop && \
    cargo build --release && \
    cp target/release/nputop /usr/local/bin/ && \
    chmod +x /usr/local/bin/nputop && \
    cd / && \
    rm -rf /tmp/nputop $HOME/.cargo $HOME/.rustup && \
    apt-get purge -y build-essential && \
    apt-get autoremove -y && \
    rm -rf /var/lib/apt/lists/*

ENTRYPOINT ["/entrypoint.sh"]
