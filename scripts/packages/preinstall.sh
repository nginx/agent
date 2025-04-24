#!/bin/sh
set -e

# Copyright (C) Nginx, Inc. 2022.
#
# Description:
# NGINX Agent install script for downloading the NGINX Agent package from the appropriate repository
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
    echo "Checking what version of NGINX Agent is already installed"
    check_version="nginx-agent --version"
    nginx_agent_version=$($check_version 2>&1) || true
    echo "Existing NGINX Agent version: $nginx_agent_version"

    if [ -z "${nginx_agent_version##nginx-agent version v2*}" ]; then
        echo "Migrating NGINX Agent configuration from V2 to V3 format"
        echo "Backing up existing NGINX Agent V2 configuration to /etc/nginx-agent/nginx-agent-v2-backup.conf"
        cp "$AGENT_CONFIG_FILE" /etc/nginx-agent/nginx-agent-v2-backup.conf

        v2_config_file=$AGENT_CONFIG_FILE
        v3_config_file=$AGENT_CONFIG_FILE

        echo "Verifying configured NGINX One host: ${NGINX_ONE_HOST}"

        if grep -q "$NGINX_ONE_HOST" "$v2_config_file"; then
            echo "NGINX Agent is configured to connect to NGINX One"
        else
            echo "${RED_COLOUR}Upgrade aborted: existing Agent V2 is not configured for NGINX One.${NO_COLOUR}"
            exit 1
        fi

        token=$(grep "token:" "$v2_config_file" | cut -d ":" -f 2 | xargs)

        # extract instance_group if present
        instance_group=""
        if instance_line=$(grep "instance_group:" "$AGENT_DYNAMIC_CONFIG_FILE"); then
            instance_group=$(echo "$instance_line" | cut -d ":" -f 2 | xargs)
        fi

        labels=""
        if [ -n "$instance_group" ]; then
            echo "Migrating existing config sync group into NGINX Agent V3 configuration"
            labels="
labels:
  config-sync-group: ${instance_group}
"
        fi

        # extract config_dirs if present
        config_dirs=""
        if config_line=$(grep "config_dirs:" "$v2_config_file"); then
            config_dirs=$(echo "$config_line" | cut -d "\"" -f 2)
        fi

        allowed_directories=""
        IFS=":"
        for config_dir in $config_dirs; do
            allowed_directories="${allowed_directories}\n  - ${config_dir}"
        done
        allowed_directories="${allowed_directories}\n  - /var/log/nginx"

        v3_config_contents="
#
# /etc/nginx-agent/nginx-agent.conf
#
# Configuration file for NGINX Agent.
#

log:
  # set log level (error, info, debug; default \"info\")
  level: info
  # set log path. if empty, don't log to file.
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

collector:
  receivers:
    host_metrics:
      scrapers:
        cpu: {}
        memory: {}
        disk: {}
        network: {}
        filesystem: {}
  processors:
    batch: {}
  exporters:
    otlp_exporters:
      - server:
          host: ${NGINX_ONE_HOST}
          port: 443
        authenticator: headers_setter
        tls:
          skip_verify: false
  extensions:
    headers_setter:
      headers:
        - action: insert
          key: \"authorization\"
          value: ${token}
"

        echo "${v3_config_contents}" > "$v3_config_file"
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
