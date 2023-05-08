---
title: "Development Environment Setup"
draft: false
description: "Learn how to setup a Development Environment for NGINX Agent."
weight: 600
toc: true
tags: [ "docs" ]
docs: "DOCS-1088"
categories: ["development"]
doctypes: ["task"]
---

## Select an Operating System

While most Linux or FreeBSD operating systems can be used to contribute to the NGINX Agent project, the following steps have been designed for Ubuntu. Ubuntu is packaged with most libraries required to build and run NGINX Agent, and is the recommended platform for NGINX Agent development.

## Install NGINX

Follow the steps in the [Installation]({{< relref "/installation-github.md" >}}) section to download, install, and run NGINX and NGINX Agent.

## Clone the NGINX Agent Repository

Using your preferred method, clone the NGINX Agent repository into your development directory. See [Cloning a GitHub Repository](https://docs.github.com/en/repositories/creating-and-managing-repositories/cloning-a-repository) for additional help.

## Installing Prerequisite Packages
Depending on the operating system distribution, it may be necessary to install the following packages in order to build NGINX Agent.

Change to the NGINX Agent source directory:
```bash
cd <path_to_development_directory>/agent
```

Install Make:
```bash
sudo apt install make
```

NGINX Agent is written in Go. You may [download Go](https://go.dev/doc/install) and follow installation instructions on the same page or run:
```bash
sudo apt install golang-go
```

Install Protoc:
```bash
sudo apt install -y protobuf-compiler
```

Install NGINX Agent tools and dependencies:

Before starting development on the NGINX Agent, it is important to download and install the necessary tool and dependencies required by the NGINX Agent. You can do this by running the following `make` command:
```bash
make install-tools deps
```

## Building NGINX Agent from Source Code

Run the following commands to build and run NGINX Agent:

```bash
make build
sudo make run
```
