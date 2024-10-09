---
title: "Development environment setup"
toc: true
weight: 200
docs: DOCS-000
---

## Overview

This page describes how to configure a development environment for F5 NGINX Agent.

While most Linux or FreeBSD operating systems can be used to contribute to the NGINX Agent project, the following steps have been designed for Ubuntu. 

Ubuntu is the recommended operating system for development, as it comes with most packages requires to build and run NGINX Agent.

## Before you begin

To begin this task, you will require the following:

- A [working NGINX Agent instance]({{< ref "/install-upgrade/install.md" >}}).
- A [Go installation](https://go.dev/dl/) of version 1.22.2 or newer.
- A [Protocol Buffer Compiler](https://grpc.io/docs/protoc-installation/) installation.

You will also need a copy of the NGINX Agent repository, which you can clone using `git`:

```shell
git clone git@github.com:nginx/agent.git
```

Read [Cloning a repository](https://docs.github.com/en/repositories/creating-and-managing-repositories/cloning-a-repository) for more information

Follow the steps in the [Installation]({{< relref "/install-upgrade/install.md" >}}) topic to install NGINX Agent.

## Install prerequisite packages
Depending on the operating system distribution, it may be necessary to install the following packages in order to build NGINX Agent.

Change to the NGINX Agent source directory:
```shell
cd <path_to_development_directory>/agent
```

Install Make:
```shell
sudo apt install make
```

Install NGINX Agent tools and dependencies:

Before starting development on NGINX Agent, it is important to download and install the necessary tool and dependencies required by NGINX Agent. You can do this by running the following `make` command:
```shell
make install-tools deps
```

## Build NGINX Agent from source code

Run the following commands to build and run NGINX Agent:

```shell
make build
sudo make run
```
