#!/bin/sh
set -e

# Copyright (C) Nginx, Inc. 2022.
#
# Description:
# NGINX Agent pre install script
#
################################
###### Default variables
################################
export NGINX_ONE_HOST="${NGINX_AGENT_SERVER_HOST:-agent.connect.nginx.com}"
export AGENT_GROUP="${AGENT_GROUP:-$(id -ng)}"

RED_COLOUR='\033[0;31m'
NO_COLOUR='\033[0m'

# Determine OS platform
# shellcheck source=/dev/null
. /etc/os-release

AGENT_CONFIG_FILE=${AGENT_CONFIG_FILE:-"/etc/nginx-agent/nginx-agent.conf"}
AGENT_DYNAMIC_CONFIG_FILE=${AGENT_DYNAMIC_CONFIG_FILE:-"/var/lib/nginx-agent/agent-dynamic.conf"}

#
# Functions
#
err_exit() {
    printf "\n%b" "$1"
    printf " exiting.\n"
    exit 1
}

title() {
    printf "\n --- NGINX Agent Package Installer --- \n\n"
}

ensure_sudo() {
    if [ "$(id -u)" = "0" ]; then
        echo "Sudo permissions detected"
    else
        err_exit "No sudo permission detected, please run as sudo"
    fi
}

update_config_file() {
    # Check for existing agent binary and version
    echo "Checking what version of NGINX Agent is already installed"
    if command -v nginx-agent >/dev/null 2>&1; then
        nginx_agent_version=$(nginx-agent -v 2>&1)
        echo "Existing NGINX Agent version: $nginx_agent_version"
    else
        echo "No existing NGINX Agent installation found, skipping migration"
        return 0
    fi

    # Only proceed if it's v2
    if echo "$nginx_agent_version" | grep -qE '^nginx-agent version v2'; then
        echo "Migrating NGINX Agent configuration from V2 to V3 format"

        # Backup v2 config if present
        if [ -f "$AGENT_CONFIG_FILE" ]; then
            echo "Backing up existing NGINX Agent V2 configuration to $AGENT_CONFIG_FILE.v2-backup"
            cp -v "$AGENT_CONFIG_FILE" "$AGENT_CONFIG_FILE.v2-backup" \
                || err_exit "Failed to back up v2 config"
        else
            echo "No existing NGINX Agent V2 config file found, skipping backup"
        fi

        v2_config_file="$AGENT_CONFIG_FILE"
        v3_config_file="$AGENT_CONFIG_FILE"

        echo "Verifying configured NGINX One host: ${NGINX_ONE_HOST}"
        if [ -f "$v2_config_file" ] && grep -q "$NGINX_ONE_HOST" "$v2_config_file"; then
            echo "NGINX Agent is configured to connect to NGINX One"
        else
            err_exit "Upgrade aborted: existing NGINX Agent V2 is not configured for NGINX One"
        fi

        # Extract token
        if token_line=$(grep "token:" "$v2_config_file"); then
            token=$(echo "$token_line" | cut -d ":" -f 2 | xargs)
        else
            err_exit "Upgrade aborted: no token found in v2 config"
        fi

        # Extract instance_group if present
        instance_group=""
        if [ -f "$AGENT_DYNAMIC_CONFIG_FILE" ] && instance_line=$(grep "instance_group:" "$AGENT_DYNAMIC_CONFIG_FILE"); then
            instance_group=$(echo "$instance_line" | cut -d ":" -f 2 | xargs)
            echo "Migrating existing Config Sync Group: $instance_group"
            labels="
labels:
  config-sync-group: ${instance_group}
"
        else
            labels=""
        fi

        # Extract config_dirs if present
        config_dirs=""
        if config_line=$(grep "config_dirs:" "$v2_config_file"); then
            config_dirs=$(echo "$config_line" | cut -d '"' -f 2)
        fi

        allowed_directories=""
        IFS=":"
        for config_dir in $config_dirs; do
            allowed_directories="${allowed_directories}\n  - ${config_dir}"
        done
        allowed_directories="${allowed_directories}\n  - /var/log/nginx"
        allowed_directories="${allowed_directories}\n  - /etc/app_protect"

        echo "Writing new v3 configuration to $v3_config_file"
        v3_config_contents="
# /etc/nginx-agent/nginx-agent.conf

log:
  level: info
  path: /var/log/nginx-agent/

allowed_directories: ${allowed_directories}
${labels}
command:
    server:
        host: ${NGINX_ONE_HOST}
        port: 443
    auth:
        token: ${token}
    tls:
        skip_verify: false
"

        if [ -n "$( echo -e )" ]; then
            echo "$v3_config_contents" > "$v3_config_file" \
                || err_exit "Failed to write v3 config"
        else
            echo -e "$v3_config_contents" > "$v3_config_file" \
                || err_exit "Failed to write v3 config"
        fi
    else
        echo "Existing NGINX Agent version is not v2, skipping config migration"
    fi
}

#
# Main body of the script
#
{
    title
    ensure_sudo
    update_config_file
}
