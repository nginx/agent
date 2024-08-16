#!/bin/sh
set -e

# Note: >/dev/null 2>&1 is used in multiple if statements in this file. 
# This is to hide expected error messages from being outputted.

# Determine OS platform

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
    echo "Removing nginx-agent group"
    if command -V groupdel >/dev/null 2>&1; then
        if [ "$(getent group nginx-agent)" ]; then
            groupdel nginx-agent
        fi
    fi

    if [ "$ID" = "freebsd" ]; then
        pw groupdel nginx-agent
    fi

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
