#!/bin/sh
set -e

# Copyright (C) Nginx, Inc. 2022.
#
# Description:
# NGINX Agent script for converting NGINX AGENT V2 config format to V3 config format

export NGINX_ONE_HOST="${NGINX_AGENT_SERVER_HOST:-agent.connect.nginx.com}"

RED_COLOUR='\033[0;31m'
NO_COLOUR='\033[0m'

for i in "$@"; do
  case $i in
    --v2-config-file=*)
      v2_config_file="${i#*=}"
      shift
      ;;
    --v3-config-file=*)
      v3_config_file="${i#*=}"
      shift
      ;;
    -*|--*)
      echo "Unknown option $i"
      exit 1
      ;;
    *)
      ;;
  esac
done

echo "NGINX Agent server host should be ${NGINX_ONE_HOST}"

if grep -q "$NGINX_ONE_HOST" ${v2_config_file}; then
    echo "NGINX Agent is configured to connect to NGINX One"
else 
    echo "${RED_COLOUR}Previous version of NGINX Agent was not configured to connect to NGINX One. Stopping upgrade${NO_COLOUR}" 
    exit 1
fi

token=`grep "token:" "${v2_config_file}"`
token=`echo $token | cut -d ":" -f 2 | xargs`

config_dirs=`grep "config_dirs:" "${v2_config_file}"`
config_dirs=`echo $config_dirs | cut -d "\"" -f 2`

allowed_directories=""
IFS=":"
for config_dir in $config_dirs; do
  allowed_directories="${allowed_directories}\n  - ${config_dir}"
done

allowed_directories="${allowed_directories}\n  - /var/log/nginx"
allowed_directories="${allowed_directories}\n  - /etc/app_protect"

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
        host: ${NGINX_ONE_HOST}
        port: 443
    auth:
        token: ${token}
    tls:
        skip_verify: false
"

echo "${v3_config_contents}" > $v3_config_file

echo "NGINX Agent configuration successfully upgraded"
