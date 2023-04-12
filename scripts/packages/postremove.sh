#!/bin/sh
# Determine OS platform
# shellcheck source=/dev/null
. /etc/os-release

if [ "$ID" = "freebsd" ]; then
    echo "Stop and remove nginx-agent service"
    service nginx-agent onestop >/dev/null 2>&1 || true
    sysrc -x nginx_agent_enable >/dev/null 2>&1 || true
    echo "Removing etc directory"
    rm -rf "/usr/local/etc/nginx-agent"
    echo "Removing db directory"
    rm -rf "/var/db/nginx-agent"
elif command -V systemctl >/dev/null 2>&1; then
    echo "Stop and disable nginx-agent service"
    systemctl stop nginx-agent >/dev/null 2>&1 || true
    systemctl disable nginx-agent >/dev/null 2>&1 || true
    echo "Running daemon-reload"
    systemctl daemon-reload || true
    echo "Removing etc directory"
    rm -rf "/etc/nginx-agent"
    echo "Removing lib directory"
    rm -rf "/var/lib/nginx-agent"
fi

echo "Removing run directory"
rm -rf "/var/run/nginx-agent"
echo "Removing log directory"
rm -rf "/var/log/nginx-agent"
