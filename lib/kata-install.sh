#!/bin/bash
# Optional: install Kata Containers and register `kata-clh` with Docker, so
# dcs/dcr devcontainers run as Cloud Hypervisor microVMs (own guest kernel)
# instead of plain runc containers. lib/docker-runtime.sh picks it up
# automatically once registered. Linux x86_64 with /dev/kvm; needs sudo.
#
# Cloud Hypervisor rather than Firecracker on purpose: Firecracker has no
# filesystem sharing, and a devcontainer bind-mounts the checkout.
#
# Run: ./lib/kata-install.sh        (KATA_VERSION=4.2.0 by default)

set -euo pipefail

KATA_VERSION="${KATA_VERSION:-4.2.0}"
PREFIX=/opt/kata

[ "$(uname -s)" = Linux ] && [ "$(uname -m)" = x86_64 ] || { echo "Linux x86_64 only." >&2; exit 1; }
[ -e /dev/kvm ] || { echo "/dev/kvm not found; enable virtualization first." >&2; exit 1; }
command -v docker >/dev/null || { echo "docker not found." >&2; exit 1; }
command -v zstd >/dev/null || { echo "zstd not found (apt install zstd)." >&2; exit 1; }
command -v jq >/dev/null || { echo "jq not found (apt install jq)." >&2; exit 1; }

tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT

if [ ! -x "$PREFIX/runtime-rs/bin/containerd-shim-kata-v2" ]; then
    echo "Downloading Kata $KATA_VERSION (~1 GB)..."
    curl -fL -o "$tmp/kata.tar.zst" \
        "https://github.com/kata-containers/kata-containers/releases/download/$KATA_VERSION/kata-static-$KATA_VERSION-amd64.tar.zst"
    sudo tar --zstd -xf "$tmp/kata.tar.zst" -C /
fi

# The Rust shim only accepts a config at its default path (a KATA_CONF_FILE
# pointing at a shipped file is rejected), so install the CLH one there.
sudo mkdir -p /etc/kata-containers/runtime-rs
sudo cp "$PREFIX/share/defaults/kata-containers/runtime-rs/configuration-clh-runtime-rs.toml" \
    /etc/kata-containers/runtime-rs/configuration.toml

# Docker finds io.containerd.kata-clh.v2 as containerd-shim-kata-clh-v2 on PATH.
printf '#!/bin/bash\nexec %s/runtime-rs/bin/containerd-shim-kata-v2 "$@"\n' "$PREFIX" |
    sudo tee /usr/local/bin/containerd-shim-kata-clh-v2 >/dev/null
sudo chmod +x /usr/local/bin/containerd-shim-kata-clh-v2

daemon=/etc/docker/daemon.json
[ -f "$daemon" ] || echo '{}' | sudo tee "$daemon" >/dev/null
sudo cp "$daemon" "$daemon.bak"
jq '.runtimes["kata-clh"] = {runtimeType: "io.containerd.kata-clh.v2"}' "$daemon" |
    sudo tee "$daemon.new" >/dev/null
sudo mv "$daemon.new" "$daemon"
echo "Restarting Docker (backup at $daemon.bak)..."
sudo systemctl restart docker

docker run --rm --runtime=kata-clh alpine uname -r
echo "kata-clh works. dcs/dcr will use it automatically; UTIL_DOCKER_RUNTIME=none opts out."
