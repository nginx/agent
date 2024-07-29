#!/bin/sh
set -e

NEWVER="$1"
OLDVER="$2"

restart_agent_if_required() {
    if service nginx-agent status; then
        printf "PostUpgrade: Restarting nginx agent (upgraded to %s from %s)\n" "$NEWVER" "$OLDVER"
        service nginx-agent restart || true
    fi
}

# Determine OS platform

. /etc/os-release

case "$ID" in
    alpine)
        restart_agent_if_required
        ;;
esac
