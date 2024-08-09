#!/bin/bash

set -euxo pipefail

handle_term()
{
    echo "received TERM signal"
    echo "stopping nginx ..."
    kill -TERM "${nginx_pid}" 2>/dev/null
}

trap 'handle_term' TERM

# Launch nginx
echo "starting nginx ..."
/usr/sbin/nginx -g "daemon off;" &

nginx_pid=$!

wait_term()
{
    wait ${nginx_pid}
}

wait_term
