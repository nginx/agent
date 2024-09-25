---
title: Start mock control plane interface
weight: 100
docs: DOCS-000
---

## Overview

This document describes how to configure and run NGINX Agent using a mock interface ("control plane") for NGINX Agent to report to.

## Install NGINX

Follow the [Install NGINX Agent]({{< relref "/installation-upgrade/" >}}) topic to download and install NGINX Agent.

## Clone the NGINX Agent repository

```shell
git clone git@github.com:nginx/agent.git
```

Read [Cloning a GitHub Repository](https://docs.github.com/en/repositories/creating-and-managing-repositories/cloning-a-repository) for more information

## Install Go

NGINX Agent and the Mock Control Plane are written in Go. Go 1.22.2 or higher is required to build and run either application from the source code directory. You can [download Go from the official website](https://go.dev/dl/).

## Start the gRPC mock control plane

Start the mock control plane by running the following command from the `agent` source code root directory:

```shell
go run sdk/examples/server.go

# Command Output
INFO[0000] http listening at 54790 # mock control plane port
INFO[0000] grpc listening at 54789 # grpc control plane port which NGINX Agent will report to
```

The mock control plane can use either gRPC or REST protocols to communicate with NGINX Agent.

## Launch Swagger UI

Swagger UI requires goswagger be installed. See [instructions for installing goswagger](https://goswagger.io/install.html) for additional help.

To launch the Swagger UI for the REST interface run the following command:

```shell
make launch-swagger-ui
```

## Apply new NGINX Agent configuration

Start or restart NGINX Agent to apply the new configuration.

Open another terminal window and start NGINX Agent. Issue the following command from the `agent` source code root directory.

```shell
sudo make run
```
```text
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

## Logs

NGINX Agent uses formatted log files to collect metrics. Expanding log formats and instance counts will also increase the size of the NGINX Agent log files. We recommend adding a separate partition for `/var/log/nginx-agent`.

{{< important >}}
Without log rotation or storage on a separate partition, log files could use up all the free drive space and cause your system to become unresponsive to certain services.

For more information, see [NGINX Agent Log Rotation]({{< relref "configuration/configuration-overview.md#nginx-agent-log-rotation" >}}).
{{< /important >}}

## Extensions

An extension is a piece of code, not critical to the main functionality that NGINX agent is responsible for. This generally falls outside the remit of managing NGINX Configuration and reporting NGINX metrics.

To enable an extension, it must be added to the extensions list in the `/etc/nginx-agent/nginx-agent.conf`.
Here is an example of enabling the advanced metrics extension:

```yaml
extensions:
  - advanced-metrics
```