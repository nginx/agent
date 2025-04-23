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
    if [ -z "${nginx_agent_version##nginx-agent version v2*}" ]; then
        echo "Updating NGINX Agent V2 configuration to V3 configuration"
        echo "Backing up NGINX Agent V2 configuration to /etc/nginx-agent/nginx-agent-v2-backup.conf"
        cp $AGENT_CONFIG_FILE /etc/nginx-agent/nginx-agent-v2-backup.conf
        
        v2_config_file=$AGENT_CONFIG_FILE
        v3_config_file=$AGENT_CONFIG_FILE
        
        echo "NGINX Agent server host should be ${NGINX_ONE_HOST}"
        
        if grep -q "$NGINX_ONE_HOST" ${v2_config_file}; then
            echo "NGINX Agent is configured to connect to NGINX One"
        else 
            echo "${RED_COLOUR}Previous version of NGINX Agent was not configured to connect to NGINX One. Stopping upgrade${NO_COLOUR}" 
            exit 1
        fi
        
        token=`grep "token:" "${v2_config_file}"`
        token=`echo $token | cut -d ":" -f 2 | xargs`
        
        instance_group=`grep "instance_group:" "${AGENT_DYNAMIC_CONFIG_FILE}"`
        instance_group=`echo $instance_group | cut -d ":" -f 2 | xargs`
        
        labels=""
        
        if [ -n "${instance_group}" ]; then 
            echo "Adding config sync group to NGINX Agent configuration"
            labels="
labels:
  config-sync-group: ${instance_group}
"
        fi
        
        config_dirs=`grep "config_dirs:" "${v2_config_file}"`
        config_dirs=`echo $config_dirs | cut -d "\"" -f 2`
        
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
        "
            
        echo "${v3_config_contents}" > $v3_config_file
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
