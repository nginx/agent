#!/bin/bash

set -uxo pipefail

handle_term() {
    echo "received TERM signal"
    echo "stopping nginx-agent ..."
    kill -TERM "${agent_pid}" 2>/dev/null || true
    wait "${agent_pid}" 2>/dev/null || true
    echo "stopping nginx ..."
    kill -TERM "${nginx_pid}" 2>/dev/null || true
    wait "${nginx_pid}" 2>/dev/null || true
    # stop app protect plugin if running
    if [ -n "${plugin_pid:-}" ]; then
        echo "stopping bd-socket-plugin ..."
        kill -TERM "${plugin_pid}" 2>/dev/null || true
        wait "${plugin_pid}" 2>/dev/null || true
    fi
}

handle_quit() {
    echo "received QUIT signal"
    echo "stopping nginx-agent ..."
    kill -QUIT "${agent_pid}" 2>/dev/null || true
    wait "${agent_pid}" 2>/dev/null || true
    echo "stopping nginx ..."
    kill -QUIT "${nginx_pid}" 2>/dev/null || true
    wait "${nginx_pid}" 2>/dev/null || true
    # stop app protect plugin if running
    if [ -n "${plugin_pid:-}" ]; then
        echo "stopping bd-socket-plugin ..."
        kill -QUIT "${plugin_pid}" 2>/dev/null || true
        wait "${plugin_pid}" 2>/dev/null || true
    fi
}

trap 'handle_term' TERM 
trap 'handle_quit' QUIT

set -e

mkdir -p \
  /tmp/proxy_temp \
  /tmp/client_temp \
  /tmp/fastcgi_temp \
  /tmp/uwsgi_temp \
  /tmp/scgi_temp

# Start App Protect plugin only if the binary exists (NGINX Plus)
if [ -x "/usr/share/ts/bin/bd-socket-plugin" ]; then
    mkdir -p /var/log/app_protect
    chown nginx:nginx /var/log/app_protect 2>/dev/null || true
    echo "starting bd-socket-plugin as nginx user..."
    /usr/share/ts/bin/bd-socket-plugin tmm_count 4 proc_cpuinfo_cpu_mhz 2000000 total_xml_memory 307200000 total_umu_max_size 3129344 sys_max_account_id 1024 no_static_config >> /var/log/app_protect/bd-socket-plugin.log 2>&1 &
    plugin_pid=$!
    echo "Started bd-socket-plugin with PID ${plugin_pid}"
else
    echo "bd-socket-plugin not found, skipping (this is expected for OSS builds)"
fi

nginx -g "daemon off;" &
nginx_pid=$!

echo "Started nginx with PID $nginx_pid"

timeout=30
elapsed=0

while [ "$elapsed" -lt "$timeout" ]; do
    if pgrep nginx >/dev/null 2>&1; then
        echo "nginx is running"
        break
    fi

    sleep 1
    elapsed=$((elapsed + 1))
done

if [ "$elapsed" -ge "$timeout" ]; then
    echo "nginx did not start within ${timeout}s"
    exit 1
fi

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
