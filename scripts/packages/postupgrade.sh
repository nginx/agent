#!/bin/sh
set -e

# Note: >/dev/null 2>&1 is used in multiple if statements in this file. 
# This is to hide expected error messages from being outputted.

NEWVER="$1"
OLDVER="$2"

restart_agent_if_required() {
    if service nginx-agent status >/dev/null 2>&1; then
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
