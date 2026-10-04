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
#   mounts   The devcontainer CLI's --mount cannot say "readonly", so mounts
#            whose target is one of the host-derived paths below are made
#            read-only here: the Claude access token, the host's Claude config
#            and the host's ~/.claude.json.
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

    # Rewrite `--mount ...,dst=<target>` into a read-only mount for the targets
    # that carry host data. devcontainer emits src=/dst= (and accepts target=).
    args=()
    while [ $# -gt 0 ]; do
        if [ "$1" = --mount ] && [ $# -gt 1 ]; then
            mount="$2"
            case ",$mount," in
                *,readonly,*|*,ro,*) ;;
                *,dst=/run/util/*,*|*,target=/run/util/*,*|\
                *,dst=/host-claude-config/*,*|*,target=/host-claude-config/*,*|\
                *,dst=/host-claude.json,*|*,target=/host-claude.json,*)
                    mount="$mount,readonly" ;;
            esac
            args+=(--mount "$mount")
            shift 2
        else
            args+=("$1")
            shift
        fi
    done

    exec "$docker_bin" "$sub" ${extra[@]+"${extra[@]}"} ${args[@]+"${args[@]}"}
fi
exec "$docker_bin" "$@"
