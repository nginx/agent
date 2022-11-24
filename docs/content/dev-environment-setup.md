---
title: "Development Environment Setup"
draft: false
description: "Learn how to setup a Development Environment for NGINX Agent."
weight: 400
toc: true
tags: [ "docs" ]
docs: "DOCS-000"
categories: ["development"]
doctypes: ["task"]
---

## Selecting an Operating System

While most Linux or FreeBSD operating systems can be used to contribute to the Agent project, the following steps have been designed for Ubuntu. Ubuntu is packaged with most libraries required to build and run Agent, and is the recommended platform for Agent development.

## Installing NGINX

Follow steps in the [Installation]({{< relref "/installation.md" >}}) section to download, install, and run NGINX and Agent.

## Cloning the Agent Repository

Run the following command from your development directory to clone Agent source code from the GitHub repository. See [Cloning a GitHub Repository](https://docs.github.com/en/repositories/creating-and-managing-repositories/cloning-a-repository) for additional help.

```bash
git clone git@github.com:nginx/agent.git
```

## Installing Prerequisite Packages

Depending on the operating system distribution, it may be necessary to install the following packages in order to build Agent.

Change to the Agent source directory:

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

## Building Agent from Source Code

Run the following commands to build and run Agent:

```
make build
make run
```
