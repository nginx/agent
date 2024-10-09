---
title: Start mock control plane interface
toc: true
weight: 300
docs: DOCS-000
---

This document describes how to configure and run F5 NGINX Agent using a mock interface ("control plane") for NGINX Agent to report to. 

The mock interface is useful when developing NGINX Agent, as it allows you to view what metrics are being reported.

## Before you begin

To begin this task, you will require the following:

- A [working NGINX Agent instance]({{< ref "/install-upgrade/install.md" >}}).
- A [Go installation](https://go.dev/dl/) of version 1.22.2 or newer.
- A [go-swagger](https://goswagger.io/go-swagger/install/) installation.

You will also need a copy of the NGINX Agent repository, which you can clone using `git`:

```shell
git clone git@github.com:nginx/agent.git
```

Read [Cloning a repository](https://docs.github.com/en/repositories/creating-and-managing-repositories/cloning-a-repository) for more information.

## Start the gRPC mock control plane

Start the mock control plane by running the following command from the `agent` source code root directory:

```shell
go run sdk/examples/server.go
```
```text
INFO[0000] http listening at 54790 # mock control plane port
INFO[0000] grpc listening at 54789 # grpc control plane port which NGINX Agent will report to
```

The mock control plane can use either gRPC or REST protocols to communicate with NGINX Agent.

To enable them, view the [Enable gRPC and REST interfaces]({{< relref "/how-to/enable-interfaces.md" >}}) topic.

## Launch Swagger UI

To launch the Swagger UI for the REST interface run the following command:

```shell
make launch-swagger-ui
```

## Start NGINX Agent

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