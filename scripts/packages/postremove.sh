#!/bin/sh
# vim:sw=4:ts=4:et:

# Determine OS platform
# shellcheck source=/dev/null
. /etc/os-release

_stop_agent_freebsd() {
    echo "Stopping nginx-agent service"
    service nginx-agent onestop >/dev/null 2>&1 || true
}

_disable_agent_freebsd() {
    echo "Disabling nginx-agent service"
    sysrc -x nginx_agent_enable >/dev/null 2>&1 || true
}

_stop_agent_systemd() {
    echo "Stopping nginx-agent service"
    systemctl stop nginx-agent >/dev/null 2>&1 || true
}

_disable_agent_systemd() {
    echo "Disabling nginx-agent service"
    systemctl disable nginx-agent >/dev/null 2>&1 || true
}

_systemd_daemon_reload() {
    echo "Running daemon-reload"
    systemctl daemon-reload || true
}

_cleanup() {
    echo "Removing /var/run/nginx-agent directory"
    rm -rf "/var/run/nginx-agent"
}

case "$ID" in
    freebsd)
        _stop_agent_freebsd
        _disable_agent_freebsd
        _cleanup
        ;;
    debian|ubuntu)
        if [ "$1" = "remove" ]; then
            _stop_agent_systemd
            _disable_agent_systemd
            _systemd_daemon_reload
            _cleanup
        fi
        ;;
    rhel|fedora|centos|amzn|almalinux|rocky)
        if [ "$1" = "0" ]; then
            _stop_agent_systemd
            _disable_agent_systemd
            _systemd_daemon_reload
            _cleanup
        fi
        ;;
    alpine)
        _cleanup
        ;;
    *)
        _stop_agent_systemd
        _disable_agent_systemd
        _systemd_daemon_reload
        _cleanup
        ;;
esac
