#!/bin/sh
set -e

# Determine OS platform
# shellcheck source=/dev/null
. /etc/os-release

stop_agent_freebsd() {
    echo "Stopping nginx-agent service"
    service nginx-agent onestop >/dev/null 2>&1 || true
}

disable_agent_freebsd() {
    echo "Disabling nginx-agent service"
    sysrc -x nginx_agent_enable >/dev/null 2>&1 || true
}

stop_agent_systemd() {
    echo "Stopping nginx-agent service"
    systemctl stop nginx-agent >/dev/null 2>&1 || true
}

disable_agent_systemd() {
    echo "Disabling nginx-agent service"
    systemctl disable nginx-agent >/dev/null 2>&1 || true
}

systemd_daemon_reload() {
    echo "Running daemon-reload"
    systemctl daemon-reload || true
}

cleanup() {
    echo "Removing /var/run/nginx-agent directory"
    rm -rf "/var/run/nginx-agent"
}

case "$ID" in
    freebsd)
        stop_agent_freebsd
        disable_agent_freebsd
        cleanup
        ;;
    debian|ubuntu)
        if [ "$1" = "remove" ]; then
            stop_agent_systemd
            disable_agent_systemd
            systemd_daemon_reload
            cleanup
        fi
        ;;
    rhel|fedora|centos|amzn|almalinux|rocky)
        if [ "$1" = "0" ]; then
            stop_agent_systemd
            disable_agent_systemd
            systemd_daemon_reload
            cleanup
        fi
        ;;
    alpine)
        cleanup
        ;;
    *)
        stop_agent_systemd
        disable_agent_systemd
        systemd_daemon_reload
        cleanup
        ;;
esac
