---
title: "Getting started"
draft: false
description: "Learn how to configure and run NGINX Agent."
weight: 300
toc: true
tags: [ "docs" ]
docs: "DOCS-000"
categories: ["configuration"]
doctypes: ["task"]
---

Follow these steps to configure and run Agent and a mock interface ("control plane") to which the Agent will report.

## Installing NGINX and Agent

Follow steps in the [Installation]({{< relref "/installation.md" >}}) section to download, install, and run NGINX and Agent.

## Cloning the Agent Repository

Run the following command in your development directory to clone the Agent source code from the GitHub repository. See [Cloning a GitHub Repository](https://docs.github.com/en/repositories/creating-and-managing-repositories/cloning-a-repository) for additional help.

```
git clone git@github.com:nginx/agent.git
```

## Starting the Mock Control Plane

Start the mock control plane by running the following command from the `agent` source code root directory:

```
go run sdk/examples/server.go

# Command Output
INFO[0000] http listening at 54790 # mock control plane port
INFO[0000] grpc listening at 54789 # grpc control plane port which Agent will report to
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
  port: 9090 # port to expose REST API
  
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
- nginxes - lists the nginx instances on the dataplane
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

Agent is written in Go. To install Go, run:
```
sudo apt install golang-go
```

## Building Agent from Source Code
Run the following commands to build and run Agent:
```
make build
make run
```