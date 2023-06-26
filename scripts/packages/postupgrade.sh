#!/bin/sh

NEWVER="$1"
OLDVER="$2"

_restart_agent_if_required() {
    if service nginx-agent status >/dev/null 2>&1; then
        printf "PostUpgrade: Restarting nginx agent (upgraded to %s from %s)\n" "$NEWVER" "$OLDVER"
        service nginx-agent restart || true
    fi
}

# Determine OS platform
# shellcheck source=/dev/null
. /etc/os-release

case "$ID" in
    alpine)
        _restart_agent_if_required
        ;;
esac
