#!/bin/sh
set -e

# Copyright (C) Nginx, Inc. 2021.
#
# Description:
# NGINX Agent install script for downloading the NGINX Agent package from the appropriate repository
#
# args:
#   $INSTANCE_GROUP         | -g | --instance-group  Instance group name (e.g. nginx-01)
#   $LOG_LEVEL              | -o | --log-level       Log level for this script (info; trace; debug; error; fatal; panic;)

################################
###### Changeable variables
################################

LOG_LEVEL=""
INSTANCE_GROUP=""

################################
###### Default variables
################################
export AGENT_GROUP=${AGENT_GROUP:-$(id -ng)}
export AGENT_INSTALL_LOG=${AGENT_INSTALL_LOG:-/tmp/agent-install.log}

# Determine OS platform
. /etc/os-release
if [ $ID = "freebsd" ]; then
    AGENT_CONFIG_FILE=${AGENT_CONFIG_FILE:-"/usr/local/etc/nginx-agent/nginx-agent.conf"}
    AGENT_DYNAMIC_CONFIG_DIR="/usr/local/etc/nginx-agent"
else
    AGENT_CONFIG_FILE=${AGENT_CONFIG_FILE:-"/etc/nginx-agent/nginx-agent.conf"}
    AGENT_DYNAMIC_CONFIG_DIR="/etc/nginx-agent"
fi

AGENT_DYNAMIC_CONFIG_FILE="${AGENT_DYNAMIC_CONFIG_DIR}/agent-dynamic.conf"
AGENT_DYNAMIC_CONFIG_COMMENT="#
# dynamic-agent.conf
#
# Dynamic configuration file for NGINX Agent.
#
# The purpose of this file is to track agent configuration
# values that can be dynamically changed via the API and the agent install script.
# You may edit this file, but API calls that modify the tags on this system will
# overwrite the tag values in this file.
#
# The agent configuration values that API calls can modify are as follows:
#    - tags
#
# The agent configuration value(s) that the agent install script can modify are as follows:
#    - instance_group

"

ALLOWABLE_LOG_LEVELS="trace debug info error fatal panic"

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
    printf " --- Will install the NGINX Agent in 5 seconds ---\n"
    printf " --- Install log is outputted to: %s\n" "${AGENT_INSTALL_LOG}"
    sleep 5
}

print_help() {
  title
  cat <<EOF
Install NGINX Agent.

Usage:
  $(basename "$0") [-g | --instance-group <instance_group>] [-o | --log-level default: info; trace; debug; error; fatal; panic]


Options:
  -g | --instance-group  Instance group name (e.g. nginx-01)
  -o | --log-level       Logging level of nginx-agent (default: info; trace; debug; error; fatal; panic)
}
EOF
}

ensure_sudo() {
    if [ "$(id -u)" = "0" ]; then
        echo "Sudo permissions detected"
    else
        err_exit "No sudo permission detected, please run as sudo"
    fi
}

is_empty() {
    if [ -z "$2" ]; then
        printf "\nValue for '%s' blank... skipping" "$1"
        return 0
    fi
    return 1
}

parse_args() {
    while [ "$1" != "" ]; do
        arg=$1
        case "$arg" in
            -g | --instance-group)
                [ "$#" -eq 1 ] && print_help && err_exit "\nMissing value for '$arg'"
                shift
                is_empty "$arg" "$1" && shift && continue
                INSTANCE_GROUP=$1
                printf "Overriding instance_group value from command line: %s ...\n" "${INSTANCE_GROUP}"
                ;;
            -o | --log-level)
                [ "$#" -eq 1 ] && print_help && err_exit "\nMissing value for '$arg'"
                shift
                is_empty "$arg" "$1" && shift && continue

                valid_log_level="false"
                for LEVEL in ${ALLOWABLE_LOG_LEVELS}; do
                    if [ "$LEVEL" = "$1" ]; then
                        valid_log_level="true"
                    fi
                done

                if [ "$valid_log_level" != "true" ]; then
                    printf "Exiting due to invalid option ($1) for -o | --log-level. Please select a valid option: trace, debug, info, error, fatal, panic.\n"
                    exit 1
                fi

                LOG_LEVEL=$1
                printf "Overriding log-level value from command line: %s ...\n" "${LOG_LEVEL}"
                ;;
            *)
                print_help
                exit 0
                ;;
        esac
        shift
    done
}

load_config_values() {

    # If the file doesn't exist attempt to create it
    if [ ! -f "$AGENT_DYNAMIC_CONFIG_FILE" ]; then
        printf "Could not find ${AGENT_DYNAMIC_CONFIG_FILE} ... Creating file\n"
        mkdir -p ${AGENT_DYNAMIC_CONFIG_DIR}
        printf "${AGENT_DYNAMIC_CONFIG_COMMENT}" | tee ${AGENT_DYNAMIC_CONFIG_FILE} > /dev/null
        printf "Successfully created ${AGENT_DYNAMIC_CONFIG_FILE}\n"
    fi

    # Check if there are existing values
    local _instance_group=$(grep "^instance_group:" "${AGENT_DYNAMIC_CONFIG_FILE}"  | head -n 1 | cut -d : -f 2 | sed "s/^[[:space:]]//")

    if [ "$_instance_group" ] && [ ! "${INSTANCE_GROUP}" ]; then
        INSTANCE_GROUP=$_instance_group
    fi
}

update_config_file() {
    agent_config_updated=""
    dynamic_config_updated=""

    printf "Updating ${AGENT_DYNAMIC_CONFIG_FILE} ...\n"
    local sed_cmd="sed -i.bak "
    if [ ! -f "$AGENT_CONFIG_FILE" ]; then
        err_exit "$AGENT_CONFIG_FILE does not exist"
    fi
    if [ ! -f "$AGENT_DYNAMIC_CONFIG_FILE" ]; then
        err_exit "$AGENT_DYNAMIC_CONFIG_FILE does not exist"
    fi

    # Replace Host
    ${sed_cmd} "s/host:.*$/host: ${PACKAGE_HOST}/" "${AGENT_CONFIG_FILE}"

    # Check the instance group and set accordingly
    if [ "${INSTANCE_GROUP}" ]; then
        if [ "$(grep -cP '^(?=[\s]*+[^#])[^#]*(instance_group)' "${AGENT_DYNAMIC_CONFIG_FILE}")" -ge 1 ]; then
            printf "Setting existing instance_group: ${INSTANCE_GROUP}\n"
            ${sed_cmd} "/^[[:space:]]*#/!s/\(instance_group:.*\)/instance_group: ${INSTANCE_GROUP}/g" "${AGENT_DYNAMIC_CONFIG_FILE}"
        else
            printf "Setting instance_group: ${INSTANCE_GROUP}\n"
            printf "instance_group: %s\n" "${INSTANCE_GROUP}" >> "${AGENT_DYNAMIC_CONFIG_FILE}"
        fi
        dynamic_config_updated="true" 
    fi

    # Check the log-level and set accordingly
    if [ "${LOG_LEVEL}" ]; then
        if [ "$(grep -cP '^(?=[\s]*+[^#])[^#]*(level:)' "${AGENT_CONFIG_FILE}")" -ge 1 ]; then
            printf "Setting existing log level: ${LOG_LEVEL}\n"
            ${sed_cmd} "/^[[:space:]]*#/!s/\(level:.*\)/level: ${LOG_LEVEL}/g" "${AGENT_CONFIG_FILE}"
        else
            printf "Setting log level: ${LOG_LEVEL}\n"
            local log_level_replacement="s/^log:/log:\\
  level: ${LOG_LEVEL}/"
                    
            ${sed_cmd} "${log_level_replacement}" "${AGENT_CONFIG_FILE}"
            printf "Successfully updated ${AGENT_CONFIG_FILE} \n"
        fi
        agent_config_updated="true" 
    fi

    if [ "${dynamic_config_updated=}" ]; then
        printf "Successfully updated ${AGENT_DYNAMIC_CONFIG_FILE}\n"
    fi

    if [ "${agent_config_updated=}" ]; then
        printf "Successfully updated ${AGENT_CONFIG_FILE}\n"
    fi   
}

#
# Main body of the script
#
{
  parse_args "$@"
  title
  ensure_sudo
  load_config_values
  update_config_file
} | tee ${AGENT_INSTALL_LOG}
