---
title: "Features configuration"
draft: false
weight: 150
toc: true
tags: [ "docs" ]
categories: ["configuration"]
doctypes: ["task"]
---

## Overview

This guide describes the F5 NGINX Agent features, and how to enable and disable features using the NGINX Agent configuration file.

## Before you begin

Before you start, make sure that you have:

- NGINX Agent installed in your system.
- Access to the NGINX Agent configuration file.


## Features

The following table lists the NGINX Agent features.

{{<bootstrap-table "table table-striped table-bordered">}}
| **Feature Name**      | **Description**      | **Default/Non-default**      |
| ------------- | ------------- | ------------- |
| registration | Registering the NGINX Agent with the management plane.| Default |
| nginx-config-async | Enable the publishing and uploading of NGINX configurations from the management plane.| Default |
| metrics | Enable collecting of NGINX metrics.| Default |
| metrics-throttle | Batch metrics before sending.| Non-default |
| metrics-sender | Reports metrics over the gRPC connection.| Non-default |
| dataplane-status | Report the health of the NGINX Instance.| Default |
| process-watcher | Observe changes to the NGINX process.| Default |
| file-watcher | Observe changes to the NGINX configuration or any changes to files on disk.| Default |
| activity-events | Send NGINX or NGINX Agent related events to the management plane.| Default |
| agent-api | Enable the NGINX Agent REST API.| Default |
{{</bootstrap-table>}}

## Use Cases

### Enable metrics only
1. **Access the NGINX Instance:** SSH to the virtual machine/server where NGINX Agent is running.

```
    ssh user@your-nginx-instance
```

2. **Edit NGINX Agent configuration:** 

```
    sudo vim /etc/nginx-agent/nginx-agent.conf
```

3. **Add Features section:** Add the following yaml to the end of the file:

```
    features:
      - metrics
      - metrics-throttle
      - dataplane-status
``` 

4. **Restart the NGINX Agent service:** Restart the NGINX Agent service to enable changes.

Once the steps have been completed, users will be able to view metrics data being sent but will not have the capability to push NGINX configuration changes.

### Enable the publishing of NGINX configurations and disable the collection of metrics.
1. **Access the NGINX Instance:** SSH to the virtual machine/server where NGINX Agent is running.

```
    ssh user@your-nginx-instance
```

2. **Edit NGINX Agent configuration:** 

```
    sudo vim /etc/nginx-agent/nginx-agent.conf
```

3. **Add Features section:** Add the following yaml to the end of the file:

```
    features:
      - nginx-config-async
      - dataplane-status
      - file-watcher
``` 

4. **Restart the NGINX Agent service:** Restart the NGINX Agent service to enable changes.

Once the steps have been completed, users will be able to publish NGINX configurations but metrics data will not be collected by the NGINX Agent. 

