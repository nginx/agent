---
title: "Development Environment Setup"
draft: false
description: "Learn how to setup a Development Environment for NGINX Agent."
weight: 400
toc: true
tags: [ "docs" ]
docs: "DOCS-1088"
categories: ["development"]
doctypes: ["task"]
---

## Select an Operating System

While most Linux or FreeBSD operating systems can be used to contribute to the NGINX Agent project, the following steps have been designed for Ubuntu. Ubuntu is packaged with most libraries required to build and run NGINX Agent, and is the recommended platform for NGINX Agent development.

## Install NGINX

Follow the steps in the [Installation]({{< relref "/installation.md" >}}) section to download, install, and run NGINX and NGINX Agent.

## Clone the NGINX Agent Repository

Using your preferred method, clone the NGINX Agent repository into your development directory. See [Cloning a GitHub Repository](https://docs.github.com/en/repositories/creating-and-managing-repositories/cloning-a-repository) for additional help.

## Install Prerequisite Packages

Depending on the operating system distribution, it may be necessary to install the following packages in order to build NGINX Agent.

Change to the NGINX Agent source directory:

```bash
cd <path_to_development_directory>/agent
```

Install Make:

```bash
sudo apt install make
```

NGINX Agent is written in Go. To install Go, run:

```bash
sudo apt install golang-go
```

## Building NGINX Agent from Source Code

Run the following commands to build and run NGINX Agent:

```
make build
sudo make run
```
