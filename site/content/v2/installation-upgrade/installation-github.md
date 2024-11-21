---
title: "Installation from GitHub release"
draft: false
weight: 200
toc: true
tags: [ "docs" ]
docs: "DOCS-1090"
categories: ["configuration"]
doctypes: ["task"]
---

## Overview

Learn how to install NGINX Agent from a GitHub Release.

## Install NGINX

NGINX Agent interfaces directly with an NGINX server process installed on the same system. If you don't have it already, follow these steps to install [NGINX Open Source](https://www.nginx.com/resources/wiki/start/topics/tutorials/install/) or [NGINX Plus](https://docs.nginx.com/nginx/admin-guide/installing-nginx/installing-nginx-plus/). Once installed, ensure the NGINX instance is running.

## Install NGINX Agent from Package Files

To install NGINX Agent on your system, go to [GitHub Releases](https://github.com/nginx/agent/releases) and download the latest package supported by your OS distribution and CPU architecture.

Use your system's package manager to install the package. Some examples:

- Debian, Ubuntu, and other distributions using the `dpkg` package manager.

  ```shell
  sudo dpkg -i nginx-agent-<agent-version>.deb
  ```

- RHEL, CentOS RHEL, Amazon Linux, Oracle Linux, and other distributions using the `yum` package manager

  ```shell
  sudo yum localinstall nginx-agent-<agent-version>.rpm
  ```

- RHEL and other distributions using the `rpm` package manager

  ```shell
  sudo rpm -i nginx-agent-<agent-version>.rpm
  ```

- Alpine Linux

  ```shell
  sudo apk add nginx-agent-<agent-version>.apk
  ```

- FreeBSD

  ```shell
  sudo pkg add nginx-agent-<agent-version>.pkg
  ```
