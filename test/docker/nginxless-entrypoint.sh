#!/bin/bash

set -euxo pipefail

handle_term()
{
    echo "received TERM signal"
    echo "stopping nginx-agent ..."
    kill -TERM "${agent_pid}" 2>/dev/null
    wait -n ${agent_pid}
}

handle_quit() {
    echo "received QUIT signal"
    echo "stopping nginx-agent ..."
    kill -QUIT "${agent_pid}" 2>/dev/null
    wait -n ${agent_pid}
}

trap 'handle_term' TERM 
trap 'handle_quit' QUIT

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
}

wait_term

echo "nginx-agent process has stopped, exiting."
