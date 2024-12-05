---
title: "Upgrade NGINX Agent package"
draft: false
weight: 600
toc: true
tags: [ "docs" ]
docs: "DOCS-1227"
categories: ["configuration"]
doctypes: ["task"]
---

## Overview

Learn how to upgrade NGINX Agent.

## Upgrade NGINX Agent from version v2.31.0 or greater

{{< note >}} Starting from version v2.31.0, NGINX Agent will automatically restart itself during an upgrade. {{< /note >}}

To upgrade NGINX Agent, follow these steps:

1. Open an SSH connection to the server where you’ve installed NGINX Agent and log in.

1. Make a backup copy of the following locations to ensure that you can successfully recover if the upgrade has issues:

    - `/etc/nginx-agent`
    - `config_dirs` values for any configuration specified in `/etc/nginx-agent/nginx-agent.conf`

1. Install the updated version of NGINX Agent:

    - CentOS, RHEL, RPM-Based

        ```shell
        sudo yum -y makecache
        sudo yum update -y nginx-agent
        ```

    - Debian, Ubuntu, Deb-Based

        ```shell
        sudo apt-get update
        sudo apt-get install -y --only-upgrade nginx-agent -o Dpkg::Options::="--force-confold"
        ```



## Upgrade NGINX Agent from a version less than v2.31.0

To upgrade NGINX Agent, take the following steps:

1. Open an SSH connection to the server where you’ve installed NGINX Agent and log in.

1. Make a backup copy of the following locations to ensure that you can successfully recover if the upgrade has issues:

    - `/etc/nginx-agent`
    - `config_dirs` values for any configuration specified in `/etc/nginx-agent/nginx-agent.conf`

1. Stop NGINX Agent:

    ```shell
    sudo systemctl stop nginx-agent
    ```

1. Install the updated version of NGINX Agent:

    - CentOS, RHEL, RPM-Based

        ```shell
        sudo yum -y makecache
        sudo yum update -y nginx-agent
        ```

    - Debian, Ubuntu, Deb-Based

        ```shell
        sudo apt-get update
        sudo apt-get install -y --only-upgrade nginx-agent -o Dpkg::Options::="--force-confold"
        ```

1. Start NGINX Agent:

    ```shell
    sudo systemctl start nginx-agent
    ```
