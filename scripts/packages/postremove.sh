#!/bin/sh
# Determine OS platform
. /etc/os-release
if [ $ID = "freebsd" ]; then
    echo "Stop and remove nginx-agent service"
    service nginx-agent onestop || true
    sysrc -x nginx_agent_enable || true
elif command -v systemctl; then
    echo "Stop and disable nginx-agent service"
    systemctl stop nginx-agent || true
    systemctl disable nginx-agent || true
    echo "Running daemon-reload"
    systemctl daemon-reload || true
fi

echo "Removing run directory"
rm -rf "/var/run/nginx-agent"
