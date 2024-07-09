#!/bin/sh
set -e

# Pre Remove Steps

# Determine OS platform
# shellcheck source=/dev/null
. /etc/os-release

stop_agent_openrc() {
    echo "Stopping nginx-agent service"
    service nginx-agent stop 2>&1 || true
}

case "$ID" in
    alpine)
        stop_agent_openrc
        ;;
esac
