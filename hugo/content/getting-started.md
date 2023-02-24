---
title: "Getting Started"
draft: false
description: "Learn how to configure and run NGINX Agent."
weight: 300
toc: true
tags: [ "docs" ]
docs: "DOCS-1089"
categories: ["configuration"]
doctypes: ["task"]
---

Follow these steps to configure and run NGINX Agent and a mock interface ("control plane") to which the NGINX Agent will report.

## Install NGINX

Follow the steps in the [Installation]({{< relref "/installation.md" >}}) section to download, install, and run NGINX.

## Clone the NGINX Agent Repository

Using your preferred method, clone the NGINX Agent repository into your development directory. See [Cloning a GitHub Repository](https://docs.github.com/en/repositories/creating-and-managing-repositories/cloning-a-repository) for additional help.

## Install Go

NGINX Agent and the Mock Control Plane are written in Go. Go 1.19 or higher is required to build and run either application from the source code directory. You can [download Go from the official website](https://go.dev/dl/). 

## Start the gRPC Mock Control Plane

Start the mock control plane by running the following command from the `agent` source code root directory:

```bash
go run sdk/examples/server.go

# Command Output
INFO[0000] http listening at 54790 # mock control plane port
INFO[0000] grpc listening at 54789 # grpc control plane port which NGINX Agent will report to
```

## NGINX Agent Settings
If it doesn't already exist, create the `/etc/nginx-agent/` directory and copy the `nginx-agent.conf` file into it from the project root directory. 
```
sudo mkdir /etc/nginx-agent
sudo cp <project_root_directory>/nginx-agent.conf /etc/nginx-agent/
```
Create the `agent-dynamic.conf` file in the `/etc/nginx-agent/` directory, which is required for NGINX Agent to run. 
```
sudo touch /etc/nginx-agent/agent-dynamic.conf
```

### Enable the gRPC interface

Add the the following settings to `/etc/nginx-agent/nginx-agent.conf`:

```yaml
server:
  host: 127.0.0.1 # mock control plane host
  grpcPort: 54789 # mock control plane gRPC port

# gRPC TLS options - DISABLING TLS IS NOT RECOMMENDED FOR PRODUCTION
tls:
  enable: false
  skip_verify: true
```

For more information, see [Agent Protocol Definitions and Documentation](https://github.com/nginx/agent/tree/main/docs/proto/README.md)

### Enable the REST interface
The NGINX Agent REST interface can be exposed by validating the following lines in the `/etc/nginx-agent/nginx-agent.conf` file are present:

```yaml
api:
  port: 8081 # port to expose REST API
  
  # REST TLS parameters
  cert: "<TLS-CERTIFICATE>.crt"
  key: "<PRIVATE-KEY>.key"
```

The mock control plane can use either gRPC or REST protocols to communicate with NGINX Agent.

## Launch Swagger UI
Swagger UI requires goswagger be installed. See [instructions for installing goswagger](https://goswagger.io/install.html) for additional help.

To launch the Swagger UI for the REST interface run the following command

```bash
make launch-swagger-ui
```

## Extensions
An extension is a piece of code, not critical to the main functionality that the NGINX agent is responsible for. This generally falls outside the remit of managing NGINX Configuration and reporting NGINX metrics.

To enable an extension, it must be added to the extensions list in the `/etc/nginx-agent/nginx-agent.conf`. 
Here is an example of enabling the advanced metrics extension:

```yaml
extensions:
  - advanced-metrics
```

## Start NGINX Agent

If already running, restart NGINX Agent to apply the new configuration. Alternatively, if NGINX Agent is not running, you may run it from the source code root directory.

Open another terminal window and start the NGINX Agent. Issue the following command from the `agent` source code root directory.

```bash
sudo make run

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

- **registered** - shows registration information of the data plane
- **nginxes** - lists the nginx instances on the data plane
- **configs** - shows the protobuf payload for NGINX configuration sent to the management plane
- **configs/chunked** - shows the split-up payloads sent to the management plane
- **configs/raw** - shows the actual configuration as it would live on the data plane
- **metrics** - shows a buffer of metrics sent to the management plane (similar to what will be sent back in the REST API)

For more NGINX Agent use cases, refer to the [NGINX Agent SDK examples](https://github.com/nginx/agent/tree/main/sdk/examples).
