#!/bin/sh
set -e

# Determine OS platform
# shellcheck source=/dev/null
. /etc/os-release

if [ "$ID" = "freebsd" ]; then
    BSD_HIER="/usr/local"
    AGENT_EXE="${BSD_HIER}/bin/nginx-agent"
else
    AGENT_EXE="/usr/bin/nginx-agent"
    BSD_HIER=""
fi
AGENT_RUN_DIR="/var/run/nginx-agent"
AGENT_LOG_DIR="/var/log/nginx-agent"
AGENT_UNIT_LOCATION="/etc/systemd/system"
AGENT_UNIT_FILE="nginx-agent.service"
AGENT_USER=$(id -nu)
WORKER_USER=""
AGENT_GROUP="nginx-agent"

detect_nginx_users() {
    if command -V systemctl >/dev/null 2>&1; then
        printf "PostInstall: Reading NGINX systemctl unit file for user information\n"
        nginx_unit_file=$(systemctl status nginx | grep -Po "\(\K\/.*service")
        pid_file=$(grep -Po "PIDFile=\K.*$" "${nginx_unit_file}")

        if [ ! -f "$pid_file" ]; then
            printf "%s does not exist\n" "${pid_file}"
        else
            pidId=$(cat "${pid_file}")
            nginx_user=$(ps --no-headers -u -p "${pidId}" | head -1 | awk '{print $1}')
        fi

        if [ ! "${nginx_user}" ]; then
            printf "No NGINX user found\n"
        fi
    fi

    if [ -z "${nginx_user}" ]; then
        printf "PostInstall: Reading NGINX process information to determine NGINX user\n"
        nginx_pid=""
        for pid in /proc/[0-9]*; do
            pid=${pid##*/}
            if [ -r /proc/"$pid"/cmdline ]; then
                if grep -q "nginx: master process" /proc/"$pid"/cmdline 2>/dev/null; then
                    nginx_pid=$pid
                    break
                fi
            fi
        done 
        if [ "${nginx_pid}" ]; then
            nginx_user=$(awk '/^Uid:/ {print $2}' /proc/"$nginx_pid"/status | xargs -I {} getent passwd {} | cut -d: -f1)
        fi

        if [ -z "${nginx_user}" ]; then
            printf "No NGINX user found\n"
        fi
    fi

    if [ "${nginx_user}" ]; then
        echo "NGINX processes running as user '${nginx_user}'. nginx-agent will be configured to run as same user"
        AGENT_USER=${nginx_user}
    else
        echo "WARNING: No NGINX processes detected."
    fi

    if [ -z "${worker_user}" ]; then
        printf "PostInstall: Reading NGINX process information to determine NGINX user\n"
        worker_pid=""
        for pid in /proc/[0-9]*; do
            pid=${pid##*/}
            if [ -r /proc/"$pid"/cmdline ]; then
                if grep -q "nginx: worker process" /proc/"$pid"/cmdline 2>/dev/null; then
                    worker_pid=$pid
                    break
                fi
            fi
        done
        if [ "${worker_pid}" ]; then
            worker_user=$(awk '/^Uid:/ {print $2}' /proc/"$worker_pid"/status | xargs -I {} getent passwd {} | cut -d: -f1)
        fi

        if [ -z "${worker_user}" ]; then
            printf "No NGINX worker user found\n"
        fi
    fi

    if [ "${worker_user}" ]; then
        echo "NGINX processes running as user '${worker_user}'. nginx-agent will try add that user to '${AGENT_GROUP}'"
        WORKER_USER=${worker_user}
    else
        echo "WARNING: No NGINX worker processes detected."
    fi

    if [ -z "${AGENT_USER}" ]; then
        echo "\$USER not defined. Running as root"
        USER=root
        AGENT_USER=root
    fi
}

ensure_sudo() {
    if [ "$(id -u)" = "0" ]; then
        echo "Sudo permissions detected"
    else
        echo "No sudo permission detected, please run as sudo"
        exit 1
    fi
}

ensure_agent_path() {
    if [ ! -f "${AGENT_EXE}" ]; then
        echo "nginx-agent not in default path, exiting..."
        exit 1
    fi

    printf "Found nginx-agent %s\n" "${AGENT_EXE}"
}

create_agent_group() {
    printf "PostInstall: Adding nginx-agent group %s\n" "${AGENT_GROUP}"

    if command -V groupadd >/dev/null 2>&1; then
        if [ ! "$(getent group $AGENT_GROUP)" ]; then
            groupadd "${AGENT_GROUP}"
        fi

        printf "PostInstall: Adding NGINX / agent user %s to group %s\n" "${AGENT_USER}" "${AGENT_GROUP}"
        usermod -a -G "${AGENT_GROUP}" "${AGENT_USER}"
        if [ "${WORKER_USER}" ]; then
            printf "PostInstall: Adding NGINX Worker user %s to group %s\n" "${WORKER_USER}" "${AGENT_GROUP}"
            usermod -a -G "${AGENT_GROUP}" "${WORKER_USER}"
        fi
    fi

    if [ "$ID" = "freebsd" ]; then
        pw groupadd "${AGENT_GROUP}"

        printf "PostInstall: Adding NGINX / agent user %s to group %s\n" "${AGENT_USER}" "${AGENT_GROUP}"
        pw groupmod "${AGENT_GROUP}" -M "${AGENT_USER}"
        if [ "${WORKER_USER}" ]; then
            printf "PostInstall: Adding NGINX Worker user %s to group %s\n" "${WORKER_USER}" "${AGENT_GROUP}"
            pw groupmod "${AGENT_GROUP}" -M "${WORKER_USER}"
        fi
    fi

    if [ "$ID" = "alpine" ]; then
        addgroup "${AGENT_GROUP}"

        printf "PostInstall: Adding NGINX / agent user %s to group %s\n" "${AGENT_USER}" "${AGENT_GROUP}"
        addgroup "${AGENT_USER}" "${AGENT_GROUP}"
        if [ "${WORKER_USER}" ]; then
            printf "PostInstall: Adding NGINX Worker user %s to group %s\n" "${WORKER_USER}" "${AGENT_GROUP}"
            addgroup "${WORKER_USER}" "${AGENT_GROUP}"
        fi
    fi
}

create_run_dir() {
    printf "PostInstall: Creating NGINX Agent run directory \n"
    mkdir -p "${AGENT_RUN_DIR}"

    printf "PostInstall: Modifying group ownership of NGINX Agent run directory \n"
    chown "${AGENT_USER}":"${AGENT_GROUP}" "${AGENT_RUN_DIR}"
}

update_unit_file() {
    # Fill in data to unit file that's acquired post install
    if command -V systemctl >/dev/null 2>&1; then
        printf "PostInstall: Modifying NGINX Agent unit file with correct locations and user information\n"
        EXE_CMD="s|\${AGENT_EXE}|${AGENT_EXE}|g"
        sed -i -e $EXE_CMD ${AGENT_UNIT_LOCATION}/${AGENT_UNIT_FILE}

        LOG_DIR_CMD="s|\${AGENT_LOG_DIR}|${AGENT_LOG_DIR}|g"
        sed -i -e $LOG_DIR_CMD ${AGENT_UNIT_LOCATION}/${AGENT_UNIT_FILE}

        RUN_DIR_CMD="s|\${AGENT_RUN_DIR}|${AGENT_RUN_DIR}|g"
        sed -i -e $RUN_DIR_CMD ${AGENT_UNIT_LOCATION}/${AGENT_UNIT_FILE}

        USER_CMD="s/\${AGENT_USER}/${AGENT_USER}/g"
        sed -i -e $USER_CMD ${AGENT_UNIT_LOCATION}/${AGENT_UNIT_FILE}

        GROUP_CMD="s/\${AGENT_GROUP}/${AGENT_GROUP}/g"
        sed -i -e $GROUP_CMD ${AGENT_UNIT_LOCATION}/${AGENT_UNIT_FILE}

        printf "PostInstall: Reload the service unit from disk\n"
        systemctl daemon-reload
        printf "PostInstall: Unmask the service unit from disk\n"
        systemctl unmask "${AGENT_UNIT_FILE}"
        printf "PostInstall: Set the preset flag for the service unit\n"
        systemctl preset "${AGENT_UNIT_FILE}"
        printf "PostInstall: Set the enabled flag for the service unit\n"
        systemctl enable "${AGENT_UNIT_FILE}"
    fi

    if [ "$ID" = "freebsd" ]; then
        printf "PostInstall: Enabling NGINX Agent Service\n"
        sysrc nginx_agent_enable=YES
    fi
}

add_default_config_file() {
    if [ ! -f "${BSD_HIER}"/etc/nginx-agent/nginx-agent.conf ]; then
        printf "PostInstall: Creating default nginx-agent.conf file\n"
        cat <<EOF > "${BSD_HIER}"/etc/nginx-agent/nginx-agent.conf
#
# /etc/nginx-agent/nginx-agent.conf
#
# Configuration file for NGINX Agent.
#
# This file is to track NGINX Agent configuration values that are meant to be statically set. There  
# are additional NGINX Agent configuration values that are set via the API and NGINX Agent install script
# which can be found in /var/lib/nginx-agent/agent-dynamic.conf. 

# specify the server grpc port to connect to
server:
  # host of the control plane
  host: 127.0.0.1
  grpcPort: 54789
  # provide servername overrides if using SNI
  # metrics: ""
  # command: ""
# tls options
tls:
  # enable tls in the nginx-agent setup for grpcs
  # default to enable to connect with tls connection but without client cert for mtls
  enable: false
  # specify the absolute path to the CA certificate file to use for verifying
  # the server certificate (also requires 'skip_verify: false' below)
  # by default, this will be the trusted root CAs found in the OS CA store
  # ca: /etc/nginx-agent/ca.pem
  # specify the absolute path to the client cert, when mtls is enabled
  # cert: /etc/nginx-agent/client.crt
  # specify the absolute path to the client cert key, when mtls is enabled
  # key: /etc/nginx-agent/client.key
  # controls whether the server certificate chain and host name are verified.
  # for production use, see instructions for configuring TLS
  skip_verify: true
log:
  # set log level (panic, fatal, error, info, debug, trace; default "info")
  level: info
  # set log path. if empty, don't log to file.
  path: /var/log/nginx-agent/
# data plane status message / 'heartbeat'
nginx:
  # path of NGINX logs to exclude
  exclude_logs: ""
  socket: "unix:/var/run/nginx-agent/nginx.sock"

dataplane:
  status:
    # poll interval for data plane status - the frequency the NGINX Agent will query the dataplane for changes
    poll_interval: 30s
    # report interval for data plane status - the maximum duration to wait before syncing dataplane information if no updates have being observed
    report_interval: 24h
metrics:
  # specify the size of a buffer to build before sending metrics
  bulk_size: 20
  # specify metrics poll interval
  report_interval: 1m
  collection_interval: 15s
  mode: aggregated

# OSS NGINX default config path
# path to aux file dirs can also be added
config_dirs: "/etc/nginx:/usr/local/etc/nginx:/usr/share/nginx/modules:/etc/nms"

api:
  # default port for NGINX Agent API, this is for the server configuration of the REST API
  port: 8081
EOF
    printf "PostInstall: Updating file permissions for nginx-agent.conf to 0640\n"
    chmod 0640 "${BSD_HIER}"/etc/nginx-agent/nginx-agent.conf
    fi
}

upgrade_config_file() {
    if [ -f "${BSD_HIER}"/etc/nginx-agent/nginx-agent.conf ]; then
        extensions=""
        if grep -q "advanced_metrics:" "${BSD_HIER}"/etc/nginx-agent/nginx-agent.conf; then
            extensions="${extensions} advanced-metrics"
        fi
        if grep -q "nginx_app_protect:" "${BSD_HIER}"/etc/nginx-agent/nginx-agent.conf; then
            extensions="${extensions} nginx-app-protect"
        fi
        if grep -q "nap_monitoring:" "${BSD_HIER}"/etc/nginx-agent/nginx-agent.conf; then
            extensions="${extensions} nap-monitoring"
        fi
        if ! grep -q "extensions:" "${BSD_HIER}"/etc/nginx-agent/nginx-agent.conf && [ "${#extensions}" -ne "0" ]; then
            printf "PostInstall: Updating nginx-agent.conf to include extensions array\n"
            printf "\nextensions:\n" >> "${BSD_HIER}"/etc/nginx-agent/nginx-agent.conf
            for extension in ${extensions}; do
                echo "  - $extension" >> "${BSD_HIER}"/etc/nginx-agent/nginx-agent.conf
            done
        fi
    fi
}

restart_agent_if_required() {
    if [ "${ID}" = "freebsd" ]; then
        # https://github.com/freebsd/pkg/pull/2128
        return
    fi
    if service nginx-agent status >/dev/null 2>&1; then
        printf "PostInstall: Restarting nginx agent\n"
        service nginx-agent restart || true
    fi
}

summary() {
    echo "----------------------------------------------------------------------"
    echo " NGINX Agent package has been successfully installed."
    echo ""
    echo " Please follow the next steps to start the software:"
    if [ "$ID" = "freebsd" ]; then
        echo "    sudo service nginx-agent start"
        echo ""
    else
        echo "    sudo systemctl start nginx-agent"
        echo ""
    fi
    echo " Configuration settings can be adjusted here:"
    echo "    ${BSD_HIER}/etc/nginx-agent/nginx-agent.conf"
    echo ""
    echo "----------------------------------------------------------------------"
}

#
# Main body of the script
#
{
    detect_nginx_users
    ensure_sudo
    ensure_agent_path
    create_agent_group
    create_run_dir
    update_unit_file
    add_default_config_file
    upgrade_config_file
    restart_agent_if_required
    summary
}
