![Agent Banner](docs/agent-banner.png "Agent Banner")

NGINX Agent is a companion daemon for your NGINX Open Source or NGINX Plus instance. It enables:
- Remote management of NGINX configurations
- Collection and reporting of real-time NGINX performance and operating system metrics
- Notifications of NGINX events

![Grafana dashboard showing Agent reported metrics](docs/grafana-dashboard-example.png "Grafana dashboard showing Agent reported metrics")
[Grafana](https://grafana.com/) dashboard showing metrics reported by Agent

# Table of Contents
- [How it Works](#how-it-works)
  - [Configuration Management](#configuration-management)
  - [Collecting Metrics](#collecting-metrics)
  - [Event Notifications](#event-notifications)
- [Installation](#installation)
  - [Installing NGINX](#installing-nginx)
  - [Installing Go](#installing-go)
  - [Installing Agent from Package Files](#installing-agent-from-package-files)
  - [Starting and Enabling Start on Boot](#starting-and-enabling-start-on-boot)
  - [Logging](#logging)
- [Getting Started with Agent](#getting-started-with-agent)
  - [Installing NGINX and Agent](#installing-nginx-and-agent)
  - [Starting the Mock Control Plane](#starting-the-mock-control-plane)
  - [Agent Settings](#agent-settings)
  - [Starting Agent](#starting-agent)
- [Development Environment Setup](#development-environment-setup)
  - [Selecting an Operating System](#selecting-an-operating-system)
  - [Installing NGINX](#installing-nginx)
  - [Cloning the Agent Repository](#cloning-the-agent-repository)
  - [Installing Prerequisite Packages](#installing-prerequisite-packages)
  - [Building Agent from Source Code](#building-agent-from-source-code)
- [Agent Technical Specifications](#agent-technical-specifications)
  - [Supported Distributions](#supported-distributions)
  - [Supported Deployment Environments](#supported-deployment-environments)
  - [Supported NGINX Versions](#supported-nginx-versions)
  - [Sizing Recommendations](#sizing-recommendations)
- [Community](#community)
- [Contributing](#contributing)
- [Change Log](#change-log)
- [License](#license)

# How it Works
Agent runs as a companion process on a system running NGINX. It provides gRPC and REST interfaces for configuration management and metrics collection from the NGINX process and operating system. Agent enables remote interaction with NGINX using common Linux tools and unlocks the ability to build sophisticated monitoring and control systems that can manage large collections of NGINX instances.

![How agent works](docs/agent-flow.png "How it works")

## Configuration Management
Agent provides an API interface for submission of updated configuration files. Upon receipt of a new file, it checks the output of `nginx -V` to determine the location of existing configurations. It then validates the new configuration with `nginx -t` before applying it via a NOHUP signal to the NGINX master process.

## Collecting Metrics
Agent interfaces with NGINX process information and parses NGINX logs to calculate and report metrics. When interfacing with NGINX Plus, Agent pulls relevant information from the NGINX Plus API. Reported metrics may be aggregated by [Prometheus](https://prometheus.io/) and visualized with tools like [Grafana](https://grafana.com/).

### NGINX Open Source
When running alongside an open source instance of NGINX, Agent requires that NGINX Access and Error logs are turned on and contain all default variables.

### NGINX Plus
For Agent to work properly with an NGINX Plus instance, the API needs to be configured in that instance's nginx.conf. See [Instance Metrics Overview](https://docs.nginx.com/nginx-management-suite/nim/about/overview-metrics/) for more details. Once NGINX Plus is configured with the `/api/` endpoint, Agent will automatically use it on startup.

## Event Notifications
Agent allows a gRPC connected control system to register a listener for a specific event. The control mechanism is then invoked when Agent sends an associated system signal. The source of a notification can be either the NGINX instance or Agent itself. Here's a list of currently supported events:

| Event                            | Description                                  |
| -------------------------------- | -------------------------------------------- |
| AGENT_START_MESSAGE              | Agent process started                        |
| AGENT_STOP_MESSAGE               | Agent process stopped                        |
| NGINX_FOUND_MESSAGE              | NGINX master process detected on system      |
| NGINX_STOP_MESSAGE               | NGINX master process stopped                 |
| NGINX_RELOAD_SUCCESS_MESSAGE     | NGINX master process reloaded successfully   |
| NGINX_RELOAD_FAILED_MESSAGE      | NGINX master process failed to reload        |
| NGINX_WORKER_START_MESSAGE       | New NGINX worker process started             |
| NGINX_WORKER_STOP_MESSAGE        | NGINX worker process stopped                 |
| CONFIG_APPLY_SUCCESS_MESSAGE     | Successfully applied new NGINX configuration |
| CONFIG_APPLY_FAILURE_MESSAGE     | Failed to apply new NGINX configuration      |
| CONFIG_ROLLBACK_SUCCESS_MESSAGE  | Successfully rolled back NGINX configuration |
| CONFIG_ROLLBACK_FAILURE_MESSAGE  | Failed to roll back NGINX configuration      |

# Installation
## Installing NGINX
Agent interfaces directly with an NGINX server process installed on the same system. If you don't have it already, follow these steps to install [NGINX Open Source](https://docs.nginx.com/nginx/admin-guide/installing-nginx/installing-nginx-open-source/) or [NGINX Plus](https://docs.nginx.com/nginx/admin-guide/installing-nginx/installing-nginx-plus/). Once installed, ensure the NGINX instance is running.

## Installing Go
Agent is written in Go and requires Go 1.19 or higher to be installed. You can [download Go from the official website](https://go.dev/dl/). 

## Installing Agent from Package Files
To install Agent on your system, go to [Releases](https://github.com/nginx/agent/releases) and download `nginx-agent.tar.gz`. Create a new subdirectory and extract the archive into it. Change into the subdirectory matching the package manager format appropriate for your operating system distribution.

Depending on OS distribution and CPU architecture type, use your system's package manager to install the package. Some examples:

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
sudo pkg add nginx-agent-<agent-version>
```

## Starting and Enabling Start on Boot
To start the Agent on systemd systems, run the following command:
```
sudo systemctl start nginx-agent
```
To enable the Agent to start on boot, run the following command:
```
sudo systemctl enable nginx-agent
```

## Logging 
Agent uses formatted log files to collect metrics. Expanding log formats and instance counts will also increase the size of Agent log files. We recommend adding a separate partition for `/var/log/nginx-agent`. Without log rotation or storage on a separate partition, log files could use up all the free drive space and cause your system to become unresponsive to certain services.

# Getting Started with Agent
Follow these steps to configure and run Agent and a mock interface ("control plane") to which the Agent will report.

## Installing NGINX and Agent
Follow steps in the [Installation](#installation) section to download, install, and run NGINX and Agent.

## Cloning the Agent Repository
Run the following command in your development directory to clone the Agent source code from the GitHub repository. See [Cloning a GitHub Repository](https://docs.github.com/en/repositories/creating-and-managing-repositories/cloning-a-repository) for additional help.

```
git clone https://github.com/nginx/agent.git
```

## Starting the gRPC Mock Control Plane
Start the mock control plane by running the following command from the `agent` source code root directory:

```
go run sdk/examples/server.go

# Command Output
INFO[0000] http listening at 54790 # mock control plane port
INFO[0000] gRPC listening at 54789 # gRPC control plane port which Agent will report to
```

## Agent Settings
If it doesn't already exist, create the `/etc/nginx-agent/nginx-agent.conf` file
```
sudo mkdir /etc/nginx-agent
sudo cp nginx-agent.conf /etc/nginx-agent/
```

### Enabling the gRPC interface
Update the `/etc/nginx-agent/nginx-agent.conf` file to include the following settings:

```yaml
server:
  host: 127.0.0.1 # mock control plane host
  grpcPort: 54789 # mock control plane gRPC port

# gRPC TLS options - DISABLING TLS IS NOT RECOMMENDED FOR PRODUCTION
tls:
  enable: false
  skip_verify: true
```

### Enabling the REST interface
The Agent REST interface can be exposed by adding the following lines to the `nginx-agent.conf` file.

```yaml
api:
  port: 8081 # port to expose REST API
  
  # REST TLS parameters
  cert: "<TLS-CERTIFICATE>.crt"
  key: "<PRIVATE-KEY>.key"
```

The mock control plane can use either gRPC or REST protocols to communicate with Agent.

## Starting Agent
If already running, restart Agent to apply the new configuration. Alternatively, if Agent is not running, you may run it from the source code root directory.

Open another terminal window and start the Agent. Issue the following command from the `agent` source code root directory.
```
make run

# Command Output snippet
WARN[0000] Log level is info                            
INFO[0000] setting displayName to XXX            
INFO[0000] NGINX Agent at with pid 12345, clientID=XXXXXX-XXXXXX-XXXXXX-XXXXXX-XXXXXX name=XXX
INFO[0000] NginxBinary initializing                     
INFO[0000] Commander initializing                       
INFO[0000] Comms initializing                           
INFO[0000] OneTimeRegistration initializing             
INFO[0000] Registering XXXXXX-XXXXXX-XXXXXX-XXXXXX-XXXXXX 
INFO[0000] Metrics initializing                         
INFO[0000] MetricsThrottle initializing                 
INFO[0000] DataPlaneStatus initializing                 
INFO[0000] MetricsThrottle waiting for report ready     
INFO[0000] Metrics waiting for handshake to be completed 
INFO[0000] ProcessWatcher initializing                  
INFO[0000] Extensions initializing                      
INFO[0000] FileWatcher initializing                     
INFO[0000] FileWatchThrottle initializing
INFO[0001] Events initializing                          
INFO[0001] OneTimeRegistration completed
```

Open a web browser to view the mock control plane at [http://localhost:54790](http://localhost:54790). The following links will be shown in the web interface:

- registered - shows registration information of the dataplane
- nginxes - lists the NGINX instances on the dataplane
- configs - shows the protobuf payload for NGINX configuration sent to the management plane
- configs/chunked - shows the split up payloads sent to the management plane
- configs/raw - shows the actual configuration as it would live on the dataplane
- metrics - shows a buffer of metrics sent to the management plane (similar to what will be sent back in the REST API)

For more Agent use-cases, refer to https://github.com/nginx/agent/tree/main/sdk/examples

# Development Environment Setup
## Selecting an Operating System
While most Linux or FreeBSD operating systems can be used to contribute to the Agent project, the following steps have been designed for Ubuntu. Ubuntu is packaged with most libraries required to build and run Agent, and is the recommended platform for Agent development.

## Installing NGINX
Follow steps in the [Installation](#installation) section to download and install NGINX. Once installed ensure NGINX instance is running.

## Cloning the Agent Repository
Run the following command from your development directory to clone Agent source code from the GitHub repository. See [Cloning a GitHub Repository](https://docs.github.com/en/repositories/creating-and-managing-repositories/cloning-a-repository) for additional help.

```
git clone git@github.com:nginx/agent.git
```

## Installing Prerequisite Packages
Depending on the operating system distribution, it may be necessary to install the following packages in order to build Agent.

Change to the Agent source directory:
```
cd <path_to_development_directory>/agent
```

Install Make:
```
sudo apt install make
```

Agent is written in Go. You may [download Go](https://go.dev/doc/install) and follow installation instructions on the same page or run:
```
sudo apt install golang-go
```

## Building Agent from Source Code
Run the following commands to build and run Agent:
```
make build
sudo make run
```

# Agent Technical Specifications
## Supported Distributions
Agent can run in most environments. For a list of supported distributions, see the [NGINX Technical Specs](https://docs.nginx.com/nginx/technical-specs/#supported-distributions) guide.

## Supported Deployment Environments 
Agent can be deployed in the following environments:

- Bare Metal
- Container
- Public Cloud: AWS, Google Cloud Platform, and Microsoft Azure
- Virtual Machine

## Supported NGINX Versions 
Agent works with all supported versions of NGINX Open Source and NGINX Plus.

## Sizing Recommendations
Minimum system sizing recommendations for Agent:

| CPU        | Memory   | Network   | Storage |
|------------|----------|-----------|---------|
| 1 CPU core | 1 GB RAM | 1 GbE NIC | 20 GB   |

# Community
- Our [Slack channel #nginx-agent](https://nginxcommunity.slack.com/), is the go-to place to start asking questions and sharing your thoughts.

- Our [GitHub issues page](https://github.com/nginx/agent/issues) offers space for a more technical discussion at your own pace.


# Contributing
Get involved with the project by contributing! Please see our [contributing guide](CONTRIBUTING.md) for details.

# Change Log
See our [changelog](docs/CHANGELOG.md) to keep track of updates.

# License
[Apache License, Version 2.0](LICENSE)
