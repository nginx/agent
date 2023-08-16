---
title: "Uninstall NGINX Agent Package"
draft: false
description: "Learn how to uninstall NGINX Agent"
weight: 700
toc: true
tags: [ "docs" ]
docs: "DOCS-1230"
categories: ["configuration"]
doctypes: ["task"]
---

## Prerequisites

- NGINX Agent installed [NGINX Agent installed](../installation-oss)
- The user following these steps will need `root` privilege

## Uninstalling NGINX Agent
Complete the following steps on each host where you’ve installed the NGINX Agent


- [Uninstalling NGINX Agent on RHEL, CentOS, Rocky Linux, AlmaLinux, and Oracle Linux](#uninstalling-nginx-agent-on-rhel-centos-rocky-linux-almalinux-and-oracle-linux)
- [Uninstalling NGINX Agent on Ubuntu](#uninstalling-nginx-agent-on-ubuntu)
- [Uninstalling NGINX Agent on Debian](#uninstalling-nginx-agent-on-debian)
- [Uninstalling NGINX Agent on SLES](#uninstalling-nginx-agent-on-sles)
- [Uninstalling NGINX Agent on Alpine Linux](#uninstalling-nginx-agent-on-alpine-linux)
- [Uninstalling NGINX Agent on Amazon Linux](#uninstalling-nginx-agent-on-amazon-linux)
- [Uninstalling NGINX Agent on FreeBSD](#uninstalling-nginx-agent-on-freebsd)

### Uninstalling NGINX Agent on RHEL, CentOS, Rocky Linux, AlmaLinux, and Oracle Linux

Complete the following steps on each host where you've installed the NGINX Agent:

1. Stop the NGINX Agent:

   ```bash
   sudo systemctl stop nginx-agent
   ```

2. To uninstall the NGINX Agent, run the following command:

   ```bash
   sudo yum remove nginx-agent
   ```

### Uninstalling NGINX Agent on Ubuntu

Complete the following steps on each host where you've installed the NGINX Agent:

1. Stop the NGINX Agent:

   ```bash
   sudo systemctl stop nginx-agent
   ```

2. To uninstall the NGINX Agent, run the following command:

   ```bash
   sudo apt-get remove nginx-agent
   ```

   > **Note:** The `apt-get remove <package>` command will remove the package from your system, while keeping the associated configuration files for possible future use. If you want to completely remove the package and all of its configuration files, you should use `apt-get purge <package>`.

### Uninstalling NGINX Agent on Debian

Complete the following steps on each host where you've installed the NGINX Agent:

1. Stop the NGINX Agent:

   ```bash
   sudo systemctl stop nginx-agent
   ```

2. To uninstall the NGINX Agent, run the following command:

   ```bash
   sudo apt-get remove nginx-agent
   ```

   > **Note:** The `apt-get remove <package>` command will remove the package from your system, while keeping the associated configuration files for possible future use. If you want to completely remove the package and all of its configuration files, you should use `apt-get purge <package>`.

### Uninstalling NGINX Agent on SLES

Complete the following steps on each host where you've installed the NGINX Agent:

1. Stop the NGINX Agent:

   ```bash
   sudo systemctl stop nginx-agent
   ```

2. To uninstall the NGINX Agent, run the following command:

   ```bash
   sudo zypper remove nginx-agent
   ``` 

### Uninstalling NGINX Agent on Alpine Linux

Complete the following steps on each host where you've installed the NGINX Agent:

1. Stop the NGINX Agent:

   ```bash
   sudo rc-service nginx-agent stop
   ```

2. To uninstall the NGINX Agent, run the following command:

   ```bash
   sudo apk del nginx-agent
   ``` 

### Uninstalling NGINX Agent on Amazon Linux

Complete the following steps on each host where you've installed the NGINX Agent:

1. Stop the NGINX Agent:

   ```bash
   sudo systemctl stop nginx-agent
   ```

2. To uninstall the NGINX Agent, run the following command:

   ```bash
   sudo yum remove nginx-agent
   ``` 

### Uninstalling NGINX Agent on FreeBSD
 
Complete the following steps on each host where you've installed the NGINX Agent:

1. Stop the NGINX Agent:

   ```bash
   sudo service nginx-agent stop
   ```

2. To uninstall the NGINX Agent, run the following command:

   ```bash
   sudo pkg delete nginx-agent
   ``` 
