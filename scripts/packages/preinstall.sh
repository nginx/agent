#!/bin/sh
set -e

# Copyright (C) Nginx, Inc. 2022.
#
# Description:
# NGINX Agent install script for downloading the NGINX Agent package from the appropriate repository
#
################################
###### Changeable variables
################################

LOG_LEVEL=""
INSTANCE_GROUP=""

################################
###### Default variables
################################
export AGENT_GROUP="${AGENT_GROUP:-$(id -ng)}"
RED='\033[0;31m'
NC='\033[0m'

# Determine OS platform
# shellcheck source=/dev/null
. /etc/os-release

if [ "$ID" = "freebsd" ]; then
    AGENT_CONFIG_FILE=${AGENT_CONFIG_FILE:-"/usr/local/etc/nginx-agent/nginx-agent.conf"}
    AGENT_DYNAMIC_CONFIG_DIR="/var/db/nginx-agent"
    # Old location of agent-dynamic.conf 
    OLD_DYNAMIC_CONFIG_DIR="/etc/nginx-agent"
    mkdir -p /var/log/nginx-agent/
else
    AGENT_CONFIG_FILE=${AGENT_CONFIG_FILE:-"/etc/nginx-agent/nginx-agent.conf"}
    AGENT_DYNAMIC_CONFIG_DIR="/var/lib/nginx-agent"
    # Old location of agent-dynamic.conf 
    OLD_DYNAMIC_CONFIG_DIR="/etc/nginx-agent"
fi

AGENT_DYNAMIC_CONFIG_FILE="${AGENT_DYNAMIC_CONFIG_DIR}/agent-dynamic.conf"
OLD_DYNAMIC_CONFIG_FILE="${OLD_DYNAMIC_CONFIG_DIR}/agent-dynamic.conf"
AGENT_DYNAMIC_CONFIG_COMMENT="#
# agent-dynamic.conf
#
# Dynamic configuration file for NGINX Agent.
#
# The purpose of this file is to track NGINX Agent configuration
# values that can be dynamically changed via the API and the NGINX Agent install script.
# You may edit this file, but API calls that modify the tags on this system will
# overwrite the tag values in this file.
#
# The NGINX Agent configuration values that API calls can modify are as follows:
#    - tags
#
# The NGINX Agent configuration value(s) that the NGINX Agent install script can modify are as follows:
#    - instance_group

"

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

create_config_file() {
    mkdir -p ${AGENT_DYNAMIC_CONFIG_DIR}
    printf "%s" "${AGENT_DYNAMIC_CONFIG_COMMENT}" | tee ${AGENT_DYNAMIC_CONFIG_FILE} > /dev/null
    chmod 0640 ${AGENT_DYNAMIC_CONFIG_FILE}
    printf "Successfully created %s\n" "${AGENT_DYNAMIC_CONFIG_FILE}"
}

load_config_values() {
    if [ ! -f "$AGENT_DYNAMIC_CONFIG_FILE" ]; then
        if [ -f "$OLD_DYNAMIC_CONFIG_FILE" ]; then
            printf "Moving %s to %s\n" "$OLD_DYNAMIC_CONFIG_FILE" "$AGENT_DYNAMIC_CONFIG_FILE"
            mkdir -p ${AGENT_DYNAMIC_CONFIG_DIR}
            mv "$OLD_DYNAMIC_CONFIG_FILE" "$AGENT_DYNAMIC_CONFIG_FILE"
            printf "Creating symlink %s at %s\n" "$AGENT_DYNAMIC_CONFIG_FILE" "$OLD_DYNAMIC_CONFIG_FILE"
            ln -s "$AGENT_DYNAMIC_CONFIG_FILE" "$OLD_DYNAMIC_CONFIG_FILE" 
        else
            printf "Could not find %s ... Creating file\n" ${AGENT_DYNAMIC_CONFIG_FILE}
            create_config_file
        fi
        
    fi

    # Check if there are existing values
    _instance_group="$(grep "^instance_group:" "${AGENT_DYNAMIC_CONFIG_FILE}"  | head -n 1 | cut -d : -f 2 | sed "s/^[[:space:]]//")"

    if [ "$_instance_group" ] && [ ! "${INSTANCE_GROUP}" ]; then
        INSTANCE_GROUP=$_instance_group
    fi
}

update_config_file() {
    echo "Checking what version of NGINX Agent is already installed"
    check_version="nginx-agent --version"
    nginx_agent_version=$($check_version 2>&1)
    if [ $? -eq 0 ]; then 
        echo "Currently NGINX Agent version is $nginx_agent_version"
        
        if [ -z "${nginx_agent_version##nginx-agent version v2*}" ]; then
            echo "Updating NGINX Agent V2 configuration to V3 configuration"
            echo "Backing up NGINX Agent V2 configuration to /etc/nginx-agent/nginx-agent-v2-backup.conf"
            cp $AGENT_CONFIG_FILE /etc/nginx-agent/nginx-agent-v2-backup.conf
            
            nginx_one_host="agent.connect.nginx.com"
            v2_config_file=$AGENT_CONFIG_FILE
            v3_config_file=$AGENT_CONFIG_FILE
            
            if grep -q "nginx_one_host" ${v2_config_file}; then
                echo "N1 connected agent"
            else 
                echo "${RED}Previous version of NGINX Agent was not connected to NGINX One. Stopping upgrade.${NC}" 
                exit 1
            fi
            
            token=`grep "token:" "${v2_config_file}"`
            token=`echo $token | cut -d ":" -f 2 | xargs`
            
            config_dirs=`grep "config_dirs:" "${v2_config_file}"`
            config_dirs=`echo $config_dirs | cut -d "\"" -f 2`
            
            allowed_directories=""
            export IFS=":"
            for config_dir in $config_dirs; do
              allowed_directories="${allowed_directories}\n  - ${config_dir}"
            done
                   
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

command:
    server:
        host: ${nginx_one_host}
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
          host: ${nginx_one_host}
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
            
            echo "${v3_config_contents}" > $v3_config_file
        fi
    fi
}

#
# Main body of the script
#
{
    title
    ensure_sudo
    load_config_values
    update_config_file
}
