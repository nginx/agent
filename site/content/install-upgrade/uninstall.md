---
title: "Uninstall NGINX Agent"
toc: true
weight: 300
docs: DOCS-000
---

## Overview

Learn how to uninstall F5 NGINX Agent from your system.

## Before you begin

### Prerequisites

- NGINX Agent installed [NGINX Agent installed](../installation-oss)
- The user following these steps will need `root` privilege

## Uninstall NGINX Agent
Complete the following steps on each host where you’ve installed NGINX Agent


- [Uninstall NGINX Agent on RHEL, CentOS, Rocky Linux, AlmaLinux, and Oracle Linux](#uninstall-nginx-agent-on-rhel-centos-rocky-linux-almalinux-and-oracle-linux)
- [Uninstall NGINX Agent on Ubuntu](#uninstall-nginx-agent-on-ubuntu)
- [Uninstall NGINX Agent on Debian](#uninstall-nginx-agent-on-debian)
- [Uninstall NGINX Agent on SLES](#uninstall-nginx-agent-on-sles)
- [Uninstall NGINX Agent on Alpine Linux](#uninstall-nginx-agent-on-alpine-linux)
- [Uninstall NGINX Agent on Amazon Linux](#uninstall-nginx-agent-on-amazon-linux)
- [Uninstall NGINX Agent on FreeBSD](#uninstall-nginx-agent-on-freebsd)

### Uninstall NGINX Agent on RHEL, CentOS, Rocky Linux, AlmaLinux, and Oracle Linux

Complete the following steps on each host where you've installed NGINX Agent:

1. Stop NGINX Agent:

   ```shell
   sudo systemctl stop nginx-agent
   ```

1. To uninstall NGINX Agent, run the following command:

   ```shell
   sudo yum remove nginx-agent
   ```

### Uninstall NGINX Agent on Ubuntu

Complete the following steps on each host where you've installed NGINX Agent:

1. Stop NGINX Agent:

   ```shell
   sudo systemctl stop nginx-agent
   ```

1. To uninstall NGINX Agent, run the following command:

   ```shell
   sudo apt-get remove nginx-agent
   ```

   {{< note >}} The `apt-get remove <package>` command will remove the package from your system, while keeping the associated configuration files for possible future use. If you want to completely remove the package and all of its configuration files, you should use `apt-get purge <package>`. {{< /note >}}

### Uninstall NGINX Agent on Debian

Complete the following steps on each host where you've installed NGINX Agent:

1. Stop NGINX Agent:

   ```shell
   sudo systemctl stop nginx-agent
   ```

1. To uninstall NGINX Agent, run the following command:

   ```shell
   sudo apt-get remove nginx-agent
   ```

   {{< note >}} The `apt-get remove <package>` command will remove the package from your system, while keeping the associated configuration files for possible future use. If you want to completely remove the package and all of its configuration files, you should use `apt-get purge <package>`. {{< /note >}}

### Uninstall NGINX Agent on SLES

Complete the following steps on each host where you've installed NGINX Agent:

1. Stop NGINX agent:

   ```shell
   sudo systemctl stop nginx-agent
   ```

1. To uninstall NGINX agent, run the following command:

   ```shell
   sudo zypper remove nginx-agent
   ```

### Uninstall NGINX Agent on Alpine Linux

Complete the following steps on each host where you've installed NGINX agent:

1. Stop NGINX agent:

   ```shell
   sudo rc-service nginx-agent stop
   ```

1. To uninstall NGINX agent, run the following command:

   ```shell
   sudo apk del nginx-agent
   ```

### Uninstall NGINX Agent on Amazon Linux

Complete the following steps on each host where you've installed NGINX agent:

1. Stop NGINX agent:

   ```shell
   sudo systemctl stop nginx-agent
   ```

1. To uninstall NGINX agent, run the following command:

   ```shell
   sudo yum remove nginx-agent
   ```

### Uninstall NGINX Agent on FreeBSD

Complete the following steps on each host where you've installed NGINX agent:

1. Stop NGINX agent:

   ```shell
   sudo service nginx-agent stop
   ```

1. To uninstall NGINX agent, run the following command:

   ```shell
   sudo pkg delete nginx-agent
   ```
