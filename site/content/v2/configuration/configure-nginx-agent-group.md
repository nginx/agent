---
title: "Add NGINX users to nginx-agent group"
draft: false
weight: 300
toc: true
tags: [ "docs" ]
docs: "DOCS-933"
categories: ["configuration"]
doctypes: ["task"]
---

## Overview

During installation, NGINX Agent detects the NGINX user (typically `nginx`) for the master and worker processes and adds this user to a group called `nginx-agent`.

If you change the NGINX username after installing the NGINX Agent, you'll need to add the new username to the `nginx-agent` group so that the NGINX socket has the proper permissions.

A failure to update the `nginx-agent` group when the NGINX username changes may result in non-compliance errors for NGINX Plus.


## NGINX Socket

NGINX Agent creates a socket in the default location `/var/run/nginx-agent/nginx.sock`. You can customize this location by editing the `nginx-agent.conf` file and setting the path similar to the following example:

```nginx configuration
nginx:
 ...
 socket: "unix:/var/run/nginx-agent/nginx.sock"
```

The socket server starts when the NGINX socket configuration is enabled; the socket configuration is enabled by default.


## Add NGINX Users to nginx-agent Group

To manually add NGINX users to the `nginx-agent` group, take the following steps:

1. Verify the `nginx-agent` group exists:

    ```bash
    sudo getent group | grep nginx-agent
    ```

    The output looks similar to the following example:

    ```bash
    nginx-agent:x:1001:root,nginx
    ```

    If the group doesn't exist, create it by running the following command:

    ```bash
    sudo groupadd nginx-agent
    ```

2. Verify the ownership of `/var/run/nginx-agent` directory:

    ```bash
    ls -l /var/run/nginx-agent
    ```

    The output looks similar to the following:

    ```bash
    total 0
    srwxrwxr-x 1 root nginx-agent 0 Jun 13 10:51 nginx.sockvv
    ```

    If the group ownership is not `nginx-agent`, change the ownership by running the following command:

    ```bash
    sudo chown :nginx-agent /var/run/nginx-agent
    ```

3. To add NGINX user(s) to the `nginx-agent` group, run the following command:

    ```bash
    sudo usermod -a -G nginx-agent <username>
    ```

    For example to add the `nginx` user, take the following step:

    ```bash
    sudo usermod -a -G nginx-agent nginx
    ```

    Repeat for all NGINX users.
