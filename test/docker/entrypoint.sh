#!/bin/bash

set -euxo pipefail

handle_term() {
    echo "received TERM signal"
    echo "stopping nginx-agent ..."
    kill -TERM "${agent_pid}" 2>/dev/null
    wait -n ${agent_pid}
    echo "stopping nginx ..."
    kill -TERM "${nginx_pid}" 2>/dev/null
    wait -n ${nginx_pid}
}

handle_quit() {
    echo "received QUIT signal"
    echo "stopping nginx-agent ..."
    kill -QUIT "${agent_pid}" 2>/dev/null
    wait -n ${agent_pid}
    echo "stopping nginx ..."
    kill -QUIT "${nginx_pid}" 2>/dev/null
    wait -n ${nginx_pid}
}

trap 'handle_term' TERM 
trap 'handle_quit' QUIT

# Launch nginx
echo "starting nginx ..."
/bin/su -s /bin/sh -c "/usr/share/ts/bin/bd-socket-plugin tmm_count 4 proc_cpuinfo_cpu_mhz 2000000 total_xml_memory 307200000 total_umu_max_size 3129344 sys_max_account_id 1024 no_static_config 2>&1 >> /var/log/app_protect/bd-socket-plugin.log &" nginx
/usr/sbin/nginx -g "daemon off;" &

nginx_pid=$!

SECONDS=0
while [[ ! -f /var/run/nginx.pid ]] && [[ ! -f /var/run/nginx/nginx.pid ]]; do
    if (( SECONDS > 30 )); then
        echo "couldn't find nginx master process"
        exit 1
    fi
    sleep 1
done

cat /etc/nginx-agent/nginx-agent.conf;

# start nginx-agent, pass args
echo "starting nginx-agent ..."
nginx-agent "$@" &

agent_pid=$!

if [ $? != 0 ]; then
    echo "couldn't start the agent, please check the log file"
    exit 1
fi

wait_term()
{
    wait ${agent_pid}
    wait ${nginx_pid}
}

wait_term

echo "nginx-agent process has stopped, exiting."
