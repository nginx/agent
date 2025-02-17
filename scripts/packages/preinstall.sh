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

################################
###### Default variables
################################
export AGENT_GROUP="${AGENT_GROUP:-$(id -ng)}"

# Determine OS platform
# shellcheck source=/dev/null
. /etc/os-release

AGENT_CONFIG_FILE=${AGENT_CONFIG_FILE:-"/etc/nginx-agent/nginx-agent.conf"}

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
    sed_cmd="sed -i.bak "

    if [ ! -f "$AGENT_CONFIG_FILE" ]; then
        printf "NGINX Agent config file %s does not exist. Could not be updated\n" "$AGENT_CONFIG_FILE"
        exit 0
    fi

    if [ "${PACKAGE_HOST}" ]; then
        printf "Updating %s ...\n" "${AGENT_CONFIG_FILE}"

        # Replace Host
        ${sed_cmd} "s/host:.*$/host: ${PACKAGE_HOST}/" "${AGENT_CONFIG_FILE}"
    fi
    
    # Check the log-level and set accordingly
    if [ "${LOG_LEVEL}" ]; then
        if [ "$(grep -cP '^(?=[\s]*+[^#])[^#]*(level:)' "${AGENT_CONFIG_FILE}")" -ge 1 ]; then
            printf "Setting existing log level: %s\n" "${LOG_LEVEL}"
            ${sed_cmd} "/^[[:space:]]*#/!s/\(level:.*\)/level: ${LOG_LEVEL}/g" "${AGENT_CONFIG_FILE}"
        else
            printf "Setting log level: %s\n" "${LOG_LEVEL}"
            _log_level_replacement="s/^log:/log:\\
  level: ${LOG_LEVEL}/"

            ${sed_cmd} "${_log_level_replacement}" "${AGENT_CONFIG_FILE}"
            printf "Successfully updated %s\n" "${AGENT_CONFIG_FILE}"
        fi
        printf "Successfully updated %s\n" "${AGENT_CONFIG_FILE}"
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
