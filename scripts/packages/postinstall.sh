#!/bin/sh
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

if command -v systemctl; then
    printf "PostInstall: Reading NGINX systemctl unit file for user information\n"
    nginx_unit_file=$(systemctl status nginx | grep -Po "\(\K\/.*service")
    pid_file=$(grep -Po "PIDFile=\K.*$" "${nginx_unit_file}")

    if [ ! -f "$pid_file" ]; then
        echo "${pid_file} does not exist"
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
    nginx_user=$(ps aux | grep "nginx: master process" | grep -v grep | head -1 | awk '{print $1}')

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
    worker_user=$(ps aux | grep "nginx: worker process" | grep -v grep | head -1 | awk '{print $1}')

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

if [ "$(id -u)" = "0" ]; then
    echo "Sudo permissions detected"
else
    echo "No sudo permission detected, please run as sudo"
    exit 1
fi

if [ -z "${AGENT_EXE}" ]; then
    printf "Could not determine path to nginx-agent, using default...\n"

    if [ ! -f "${AGENT_EXE}" ]; then
        echo "nginx-agent not in default path, exiting..."
        exit 1
    fi
fi

printf "Found nginx-agent %s\n" "${AGENT_EXE}"

# Fill in data to unit file that's acquired post install
if command -v systemctl; then
    printf "PostInstall: Adding nginx-agent group %s\n" "${AGENT_GROUP}"
    groupadd "${AGENT_GROUP}"

    printf "PostInstall: Adding NGINX / agent user %s to group %s\n" "${AGENT_USER}" "${AGENT_GROUP}"
    usermod -a -G "${AGENT_GROUP}" "${AGENT_USER}"
    if [ "${WORKER_USER}" ]; then
        printf "PostInstall: Adding NGINX Worker user %s to group %s\n" "${WORKER_USER}" "${AGENT_GROUP}"
        usermod -a -G "${AGENT_GROUP}" "${WORKER_USER}"
    fi

    printf "PostInstall: Creating NGINX Agent run directory \n"
    mkdir "${AGENT_RUN_DIR}"

    printf "PostInstall: Modifying group ownership of NGINX Agent run directory \n"
    chown "${AGENT_USER}":"${AGENT_GROUP}" "${AGENT_RUN_DIR}"

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
