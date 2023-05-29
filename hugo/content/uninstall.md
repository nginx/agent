---
title: "Uninstall NGINX Agent Package"
draft: false
description: "Learn how to uninstall NGINX Agent"
weight: 500
toc: true
tags: [ "docs" ]
docs: "DOCS-1219"
categories: ["configuration"]
doctypes: ["task"]
---

## Prerequisites

- NGINX Agent installed
- `root` privilege


## Uninstall NGINX Agent
Complete the following steps on each host where you’ve installed the NGINX Agent

Before you install NGINX Agent for the first time on your system, you need to set up the `nginx-agent` packages repository. Afterward, you can install and update NGINX Agent from the repository.

- [Uninstalling NGINX Agent on RHEL, CentOS, Rocky Linux, AlmaLinux, and Oracle Linux](#installing-nginx-agent-on-rhel-centos-rocky-linux-almalinux-and-oracle-linux)
- [Uninstalling NGINX Agent on Ubuntu](#installing-nginx-agent-on-ubuntu)
- [Uninstalling NGINX Agent on Debian](#installing-nginx-agent-on-debian)
- [Uninstalling NGINX Agent on SLES](#installing-nginx-agent-on-sles)
- [Uninstalling NGINX Agent on Alpine Linux](#installing-nginx-agent-on-alpine-linux)
- [Uninstalling NGINX Agent on Amazon Linux](#installing-nginx-agent-on-amazon-linux)
- [Uninstalling NGINX Agent on FreeBSD](#installing-nginx-agent-on-freebsd)

### Uninstalling NGINX Agent on RHEL, CentOS, Rocky Linux, AlmaLinux, and Oracle Linux

Complete the following steps on each host where you've installed the NGINX Agent:

1. Stop the NGINX Agent:

   ```bash
   sudo systemctl stop nginx-agent
   ```

2. To uninstall the NGINX Agent, run the following command:

   ```bash
   yum remove nginx-agent
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

### Installing NGINX Agent on Debian

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

### Installing NGINX Agent on SLES

Complete the following steps on each host where you've installed the NGINX Agent:

1. Stop the NGINX Agent:

   ```bash
   sudo systemctl stop nginx-agent
   ```

2. To uninstall the NGINX Agent, run the following command:

   ```bash
   sudo zypper remove nginx-agent
   ``` 

### Installing NGINX Agent on Alpine Linux

Complete the following steps on each host where you've installed the NGINX Agent:

1. Stop the NGINX Agent:

   ```bash
   sudo rc-service nginx-agent stop
   ```

2. To uninstall the NGINX Agent, run the following command:

   ```bash
   sudo apk del nginx-agent
   ``` 

### Installing NGINX Agent on Amazon Linux

Complete the following steps on each host where you've installed the NGINX Agent:

1. Stop the NGINX Agent:

   ```bash
   sudo systemctl stop nginx-agent
   ```

2. To uninstall the NGINX Agent, run the following command:

   ```bash
   yum remove nginx-agent
   ``` 

### Installing NGINX Agent on FreeBSD
 
Complete the following steps on each host where you've installed the NGINX Agent:

1. Stop the NGINX Agent:

   ```bash
   sudo service nginx-agent stop
   ```

2. To uninstall the NGINX Agent, run the following command:

   ```bash
   sudo pkg delete nginx-agent
   ``` 