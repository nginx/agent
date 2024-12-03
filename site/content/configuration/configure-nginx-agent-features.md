---
title: "Configure NGINX Agent Features"
draft: false
weight: 300
toc: true
tags: [ "docs" ]
categories: ["configuration"]
doctypes: ["task"]
---

## Overview

This guide describes the F5 NGINX Agent features, and how to enable and disable features using the NGINX Agent configuration file.

## Before you begin

Before you start, make sure that you have:

- NGINX Agent has been installed. 
- Access to the NGINX Agent configuration file.


## Features

The following table details the NGINX Agent features available. 

| **Feature Name**      | **Description**      |
| ------------- | ------------- |
| registration | Registering the NGINX Agent with the management plane.|
| nginx-config-async | Enable the publishing and uploading of NGINX configurations from the management plane.|
| metrics | Enable collecting of NGINX metrics.|
| metrics-throttle | Batch metrics before sending.|
| metrics-sender | Reports metrics over the gRPC connection.|
| dataplane-status | Report the health of the NGINX Instance.|
| process-watcher | Observe changes to the NGINX process.|
| file-watcher | Observe changes to the NGINX configuration or any changes to files on disk.|
| activity-events | Send NGINX or NGINX Agent related events to the management plane.|
| agent-api | Enable the NGINX Agent REST API.|


### Use Cases

#### Enable metrics only
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

At this stage, users will be able to view metrics data being sent, but will not have the capability to push NGINX configuration changes.

