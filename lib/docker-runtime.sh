#!/bin/bash
# docker wrapper for the devcontainer CLI, which has no runtime or DNS flag.
# dcs/dcr pass it as --docker-path; it injects flags into `docker run` and
# `docker create` and forwards everything else untouched.
#
#   runtime  UTIL_DOCKER_RUNTIME if set (empty or "none" = leave Docker's
#            default), else kata-clh when Docker has it registered (a microVM
#            per container; see lib/kata-install.sh), else Docker's default.
#   dns      UTIL_DOCKER_DNS if set ("none" = skip), else Tailscale's resolver
#            (100.100.100.100) when `tailscale` is installed on this host.
#            Docker otherwise puts the LAN resolver first, which answers
#            NXDOMAIN for *.ts.net, so tailnet hosts do not resolve in
#            containers.
#
# Written for bash 3.2.

docker_bin="${UTIL_REAL_DOCKER:-docker}"

if [ "${1:-}" = run ] || [ "${1:-}" = create ]; then
    sub="$1"
    shift
    extra=()

    runtime="${UTIL_DOCKER_RUNTIME-__auto__}"
    if [ "$runtime" = "__auto__" ]; then
        runtime=""
        if "$docker_bin" info --format '{{json .Runtimes}}' 2>/dev/null | grep -q '"kata-clh"'; then
            runtime="kata-clh"
        fi
    fi
    if [ -n "$runtime" ] && [ "$runtime" != none ]; then
        extra+=(--runtime="$runtime")
    fi

    dns="${UTIL_DOCKER_DNS-__auto__}"
    if [ "$dns" = "__auto__" ]; then
        dns=""
        command -v tailscale >/dev/null 2>&1 && dns="100.100.100.100"
    fi
    if [ -n "$dns" ] && [ "$dns" != none ]; then
        extra+=(--dns "$dns")
    fi

    exec "$docker_bin" "$sub" ${extra[@]+"${extra[@]}"} "$@"
fi
exec "$docker_bin" "$@"
