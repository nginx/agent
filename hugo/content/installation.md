---
title: "Installation"
draft: false
description: "Learn how to install NGINX Agent."
weight: 200
toc: true
tags: [ "docs" ]
docs: "DOCS-1090"
categories: ["configuration"]
doctypes: ["task"]
---

## Install NGINX

NGINX Agent interfaces directly with an NGINX server process installed on the same system. If you don't have it already, follow these steps to install [NGINX Open Source](https://www.nginx.com/resources/wiki/start/topics/tutorials/install/) or [NGINX Plus](https://docs.nginx.com/nginx/admin-guide/installing-nginx/installing-nginx-plus/). Once installed, ensure the NGINX instance is running.

## Install Agent from Package Files

To install NGINX Agent on your system, go to [Releases](https://github.com/nginx/agent/releases) and download `nginx-agent.tar.gz`. Create a new subdirectory and extract the archive into it. Change into the subdirectory matching the package manager format appropriate for your operating system distribution.

Depending on OS distribution and CPU architecture type, use your system's package manager to install the package. Some examples:

- Debian, Ubuntu, and other distributions using the `dpkg` package manager.

  ```
  sudo dpkg -i nginx-agent-<agent-version>.deb
  ```

- RHEL, CentOS RHEL, Amazon Linux, Oracle Linux, and other distributions using the `yum` package manager
  
  ```
  sudo yum localinstall nginx-agent-<agent-version>.rpm
  ```

- RHEL and other distributions using the `rpm` package manager

  ```
  sudo rpm -i nginx-agent-<agent-version>.rpm
  ```

- Alpine Linux

  ```
  sudo apk add nginx-agent-<agent-version>.apk
  ```

- FreeBSD
 
  ```
  sudo pkg add nginx-agent-<agent-version>
  ```

## Start and Enable Start on Boot

To start the NGINX Agent on `systemd` systems, run the following command:

```
sudo systemctl start nginx-agent
```

To enable the NGINX Agent to start on boot, run the following command:

```
sudo systemctl enable nginx-agent
```

## Logs 

NGINX Agent uses formatted log files to collect metrics. Expanding log formats and instance counts will also increase the size of the NGINX Agent log files. We recommend adding a separate partition for `/var/log/nginx-agent`. 

{{< important >}}Without log rotation or storage on a separate partition, log files could use up all the free drive space and cause your system to become unresponsive to certain services.{{< /important >}}
