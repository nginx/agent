---
title: "Enable gRPC and REST interfaces"
toc: true
weight: 200
docs: DOCS-000 
---

This document describes how to enable the gRPC and REST interfaces for F5 NGINX Agent.

## Before you begin

If it doesn't already exist, create the directory `/etc/nginx-agent/`and copy the `nginx-agent.conf` file into it from the project root directory.

```shell
sudo mkdir /etc/nginx-agent
sudo cp <project_root_directory>/nginx-agent.conf /etc/nginx-agent/
```

Create the `agent-dynamic.conf` file, which is required for NGINX Agent to run.

In Linux environments:
```shell
sudo touch /var/lib/nginx-agent/agent-dynamic.conf
```

In FreeBSD environments:
```shell
sudo touch /var/db/nginx-agent/agent-dynamic.conf
```

---

## Enable the gRPC interface

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

For more information, see [Agent Protocol Definitions and Documentation](https://github.com/nginx/agent/tree/main/docs/proto/README.md).

---

## Enable the REST interface

The NGINX Agent REST interface can be exposed by validating the following lines in the `/etc/nginx-agent/nginx-agent.conf` file are present:

```yaml
api:
  # Set API address to allow remote management
  host: 127.0.0.1
  # Set this value to a secure port number to prevent information leaks
  port: 8038
  # REST TLS parameters
  cert: "<TLS-CERTIFICATE>.crt"
  key: "<PRIVATE-KEY>.key"
```

---

## Start NGINX Agent

To apply the new configuration, NGINX Agent must be started or restarted.

You may want to view the [Start mock control plane interface]({{< relref "/contribute/start-mock-interface.md" >}}) topic to test NGINX Agent, or view the [Configuration overview]({{< relref "/how-to/configuration-overview.md" >}}) for more options.