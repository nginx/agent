![GitHub go.mod Go version](https://img.shields.io/github/go-mod/go-version/nginx/agent)
![GitHub Release](https://img.shields.io/github/v/release/nginx/agent)
![GitHub License](https://img.shields.io/github/license/nginx/agent)
![Contributions Welcome](https://img.shields.io/badge/contributions-welcome-brightgreen.svg?style=flat)

![NGINX Agent Banner](docs/agent-banner.png "NGINX Agent Banner")

NGINX Agent is a companion daemon for your NGINX Open Source or NGINX Plus instance. It enables:
- Remote management of NGINX configurations
- Collection and reporting of real-time NGINX performance and operating system metrics

![Grafana dashboard showing NGINX Agent reported metrics](docs/grafana-dashboard-example.png "Grafana dashboard showing NGINX Agent reported metrics")
[Grafana](https://grafana.com/) dashboard showing metrics reported by NGINX Agent

# Table of Contents
- [How it Works](#how-it-works)
  - [Configuration Management](#configuration-management)
  - [Collecting Metrics](#collecting-metrics)
- [Installation](#installation)
  - [Installing NGINX](#installing-nginx)
  - [Installing NGINX Agent from Package Files](#installing-nginx-agent-from-package-files)
  - [Starting and Enabling Start on Boot](#starting-and-enabling-start-on-boot)
  - [Logging](#logging)
  - [Building Docker Image](#building-docker-image)
- [Getting Started with NGINX Agent](#getting-started-with-nginx-agent)
  - [Installing NGINX](#installing-nginx)
  - [Cloning the NGINX Agent Repository](#cloning-the-nginx-agent-repository)
  - [Installing Go](#installing-go)
  - [Starting the gRPC Mock Control Plane](#starting-the-grpc-mock-control-plane)
  - [NGINX Agent Settings](#nginx-agent-settings)
  - [Extensions](#extensions)
  - [Starting NGINX Agent](#starting-nginx-agent)
- [Development Environment Setup](#development-environment-setup)
  - [Selecting an Operating System](#selecting-an-operating-system)
  - [Installing NGINX](#installing-nginx)
  - [Cloning the NGINX Agent Repository](#cloning-the-nginx-agent-repository)
  - [Installing Prerequisite Packages](#installing-prerequisite-packages)
  - [Building NGINX Agent from Source Code](#building-nginx-agent-from-source-code)
- [NGINX Agent Technical Specifications](#nginx-agent-technical-specifications)
  - [Supported Distributions](#supported-distributions)
  - [Supported Deployment Environments](#supported-deployment-environments)
  - [Supported NGINX Versions](#supported-nginx-versions)
  - [Sizing Recommendations](#sizing-recommendations)
- [Community](#community)
- [Contributing](#contributing)
- [Change Log](#change-log)
- [License](#license)

# How it Works
NGINX Agent runs as a companion process on a system running NGINX. It provides gRPC and REST interfaces for configuration management and metrics collection from the NGINX process and operating system. NGINX Agent enables remote interaction with NGINX using common Linux tools and unlocks the ability to build sophisticated monitoring and control systems that can manage large collections of NGINX instances.

![How NGINX Agent works](docs/agent-flow.png "How NGINX Agent works")

## Configuration Management
NGINX Agent provides an API interface for submission of updated configuration files. Upon receipt of a new file, it checks the output of `nginx -V` to determine the location of existing configurations. It then validates the new configuration with `nginx -t` before applying it via a signal HUP to the NGINX master process.

## Collecting Metrics
NGINX Agent interfaces with NGINX process information and parses NGINX logs to calculate and report metrics. When interfacing with NGINX Plus, NGINX Agent pulls relevant information from the NGINX Plus API. Reported metrics may be aggregated by [Prometheus](https://prometheus.io/) and visualized with tools like [Grafana](https://grafana.com/).

The NGINX Agent keeps a connection open to the NGINX Plus API and queries based on the collection interval. This connection gets reported in the metrics and depending on the phase of reporting this connection can show up as idle or active.

### NGINX Open Source
When running alongside an open source instance of NGINX, NGINX Agent requires that NGINX Access and Error logs are turned on and contain all default variables.

### NGINX Plus
For NGINX Agent to work properly with an NGINX Plus instance, the API needs to be configured in that instance's nginx.conf. See [Instance Metrics Overview](https://docs.nginx.com/nginx-instance-manager/monitoring/overview-metrics/) for more details. Once NGINX Plus is configured with the `/api/` endpoint, NGINX Agent will automatically use it on startup.

# Installation
## Installing NGINX
NGINX Agent interfaces directly with an NGINX server process installed on the same system. If you don't have it already, follow these steps to install [NGINX Open Source](https://docs.nginx.com/nginx/admin-guide/installing-nginx/installing-nginx-open-source/) or [NGINX Plus](https://docs.nginx.com/nginx/admin-guide/installing-nginx/installing-nginx-plus/). Once installed, ensure the NGINX instance is running.

## Installing NGINX Agent from Package Files
To install NGINX Agent on your system, go to [Releases](https://github.com/nginx/agent/releases) and download the latest package supported by your OS distribution and CPU architecture.

Use your system's package manager to install the package. Some examples:

Debian, Ubuntu, and other distributions using the `dpkg` package manager.

```
sudo dpkg -i nginx-agent-<agent-version>.deb
```
RHEL, CentOS RHEL, Amazon Linux, Oracle Linux, and other distributions using the `yum` package manager
```
sudo yum localinstall nginx-agent-<agent-version>.rpm
```
RHEL and other distributions using the `rpm` package manager
```
sudo rpm -i nginx-agent-<agent-version>.rpm
```
Alpine Linux
```
sudo apk add nginx-agent-<agent-version>.apk
```
FreeBSD
```
sudo pkg add nginx-agent-<agent-version>.pkg
```

## Starting and Enabling Start on Boot
To start the NGINX Agent on systemd systems, run the following command:
```
sudo systemctl start nginx-agent
```
To enable the NGINX Agent to start on boot, run the following command:
```
sudo systemctl enable nginx-agent
```

On Alpine Linux, use the following command to start the agent:
```
sudo service nginx-agent start
```
To enable the agent to start at boot time on Alpine Linux, run this:
```
sudo rc-update add nginx-agent default
```

## Logging
NGINX Agent uses formatted log files to collect metrics. Expanding log formats and instance counts will also increase the size of NGINX Agent log files. We recommend adding a separate partition for `/var/log/nginx-agent`. Without log rotation or storage on a separate partition, log files could use up all the free drive space and cause your system to become unresponsive to certain services.

### Log Rotation
By default, NGINX Agent rotates logs daily using logrotate with the following configuration: 

```yaml
/var/log/nginx-agent/*.log
{
   # log files are rotated every day
   daily
   # log files are rotated if they grow bigger than 5M
   size 5M
   # truncate the original log file after creating a copy
   copytruncate
   # remove rotated logs older than 10 days
   maxage 10
   # log files are rotated 10 times before being removed
   rotate 10
   # old log files are compressed
   compress
   # if the log file is missing it will go on to the next one without issuing an error message
   missingok 
   # do not rotate the log if it is empty
   notifempty
}
```

If you need to make changes to the default configuration you can update the file here `/etc/logrotate.d/nginx-agent`

For more detail on logrotate configuration see [Logrotate Configuration Options](https://linux.die.net/man/8/logrotate)


# NGINX Agent Technical Specifications
## Supported Distributions
NGINX Agent can run in most environments. For a list of supported distributions, see the [NGINX Technical Specs](https://docs.nginx.com/nginx/technical-specs/#supported-distributions) guide.

## Supported Deployment Environments
NGINX Agent can be deployed in the following environments:

- Bare Metal
- Container
- Public Cloud: AWS, Google Cloud Platform, and Microsoft Azure
- Virtual Machine

## Supported NGINX Versions
NGINX Agent works with all supported versions of NGINX Open Source and NGINX Plus.

## Sizing Recommendations
Minimum system sizing recommendations for NGINX Agent:

| CPU        | Memory   | Network   | Storage |
|------------|----------|-----------|---------|
| 1 CPU core | 1 GB RAM | 1 GbE NIC | 20 GB   |

# Community
- Our [NGINX Community](https://community.nginx.org/), is the go-to place to start asking questions and sharing your thoughts.

- Our [GitHub issues page](https://github.com/nginx/agent/issues) offers space for a more technical discussion at your own pace.


# Contributing
Get involved with the project by contributing! Please see our [contributing guide](CONTRIBUTING.md) for details.

# Change Log
See our [release page](https://github.com/nginx/agent/releases) to keep track of updates.

# License
[Apache License, Version 2.0](LICENSE)
