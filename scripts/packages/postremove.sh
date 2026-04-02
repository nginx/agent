#!/bin/sh

# Determine OS platform
# shellcheck source=/dev/null
. /etc/os-release

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

full_cleanup() {
    echo "Purging all nginx-agent data"
    cleanup
    rm -rf "/etc/nginx-agent"
    rm -rf "/var/log/nginx-agent"
    rm -rf "/var/lib/nginx-agent"
}

case "$ID" in
    debian|ubuntu)
        case "$1" in
            remove)
                stop_agent_systemd
                disable_agent_systemd
                systemd_daemon_reload
                cleanup
                ;;
            purge)
                stop_agent_systemd
                disable_agent_systemd
                systemd_daemon_reload
                full_cleanup
                ;;
        esac
        ;;
    rhel|fedora|centos|amzn|almalinux|rocky)
        if [ "$1" = "0" ]; then
            stop_agent_systemd
            disable_agent_systemd
            systemd_daemon_reload
            # Only purge data if the package is truly absent after this scriptlet runs.
            # During upgrades, rpm -q will still find the package, so cleanup is skipped.
            if ! command -v rpm >/dev/null 2>&1 || ! rpm -q nginx-agent >/dev/null 2>&1; then
                full_cleanup
            fi
        fi
        ;;
    alpine)
        full_cleanup
        ;;
    *)
        stop_agent_systemd
        disable_agent_systemd
        systemd_daemon_reload
        full_cleanup
        ;;
esac
