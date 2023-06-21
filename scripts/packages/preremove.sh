#!/bin/sh
# vim:sw=4:ts=4:et:
# Pre Remove Steps

# Determine OS platform
# shellcheck source=/dev/null
. /etc/os-release

_stop_agent_openrc() {
    echo "Stopping nginx-agent service"
    service nginx-agent stop 2>&1 || true
}

case "$ID" in
    alpine)
        _stop_agent_openrc
        ;;
esac
