---
title: Container support and troubleshooting
categories:
- installation
draft: false
tags:
- docs
toc: true
weight: 200
docs: "DOCS-909"
---

## Overview

The NGINX Agent repository includes [Dockerfiles](https://github.com/nginx/agent/tree/main/scripts/docker) that can be used to [build custom container images]({{< relref "/v2/installation-upgrade/container-environments/docker-images.md" >}}). Images are created with an NGINX Open Source or NGINX Plus instance and are available for various operating systems.

See the [Technical Specifications]({{< relref "/technical-specifications.md#container-support" >}}) for a list of supported operationg systems.

NGINX Agent running in a container has some limitations that need to be considered, and are listed below.

## Supported cgroups

To collect metrics about the Docker container that the NGINX Agent is running in, NGINX Agent uses the available cgroup files to calculate metrics like CPU and memory usage.

NGINX Agent supports both versions of cgroups.

- https://www.kernel.org/doc/Documentation/cgroup-v1/
- https://www.kernel.org/doc/Documentation/cgroup-v2.txt

## Metrics

### Unsupported Metrics

The following system metrics are not supported when running NGINX Agent in a Docker container. NGINX Agent returns no values for these metrics:

- system.cpu.idle
- system.cpu.iowait
- system.cpu.stolen
- system.mem.buffered
- system.load.1
- system.load.5
- system.load.15
- system.disk.total
- system.disk.used
- system.disk.free
- system.disk.in_use
- system.io.kbs_r
- system.io.kbs_w
- system.io.wait_r
- system.io.wait_w
- system.io.iops_r
- system.io.iops_w

### Memory Metrics

If no memory limit is set when starting the Docker container, then the memory limit that's shown in the metrics for the container will be the total memory of the Docker host system.

### Swap Memory Metrics

If a warning message similar to the following example is seen in the NGINX Agent logs, the swap memory limit for the Docker container is greater than the swap memory for the Docker host system:

```bash
Swap memory limit specified for the container, ... is greater than the host system swap memory ...
```

The `system.swap.total` metric for the container matches the total swap memory for the Docker host system instead of the swap memory limit specified when starting the Docker container.

If a warning message similar to the following example is seen in the NGINX Agent logs, the Docker host system does not have cgroup swap limit capabilities enabled. To enable these capabilities, follow the steps below.

```bash
Unable to collect Swap metrics because the file ... was not found
```

#### Enable cgroup swap limit capabilities

Run the following command to see if the cgroup swap limit capabilities are enabled:

```bash
$ docker info | grep swap
WARNING: No swap limit support
```

To enable cgroup swap limit capabilities, refer to this Docker guide: [Docker - Linux post-installation steps](https://docs.docker.com/engine/install/linux-postinstall/#your-kernel-does-not-support-cgroup-swap-limit-capabilities).
