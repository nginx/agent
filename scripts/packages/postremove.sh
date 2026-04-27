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

is_agent_installed() {
    # Check if nginx-agent binary still exists
    if [ -f "/usr/bin/nginx-agent" ]; then
        return 0  # Agent is installed
    else
        return 1  # Agent is not installed
    fi
}

is_package_installed() {
    # Check if nginx-agent package is still installed using package manager
    case "$ID" in
        debian|ubuntu)
            dpkg -l | grep -q '^ii.*nginx-agent' && return 0 || return 1
            ;;
        rhel|fedora|centos|amzn|almalinux|rocky|ol)
            rpm -q nginx-agent >/dev/null 2>&1 && return 0 || return 1
            ;;
        alpine)
            apk info nginx-agent >/dev/null 2>&1 && return 0 || return 1
            ;;
        *)
            return 1
            ;;
    esac
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
    rhel|fedora|centos|amzn|almalinux|rocky|ol)
        if [ "$1" = "0" ]; then
            # Package is being completely removed
            echo "PostRemove: Package being removed (not upgraded)"
            stop_agent_systemd
            disable_agent_systemd
            systemd_daemon_reload
                    
            # Check if agent binary still exists (it shouldn't in a real removal)
            if ! is_agent_installed; then
                echo "PostRemove: Agent binary not found, performing full cleanup"
                full_cleanup
            else
                echo "PostRemove: Agent binary still present, performing partial cleanup only"
                cleanup
            fi
        elif [ "$1" = "1" ]; then
            # Package is being upgraded
            echo "PostRemove: Agent is being upgraded, performing partial cleanup only"
            cleanup
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
