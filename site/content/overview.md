---
title: "Overview"
draft: false
weight: 100
toc: true
tags: [ "docs" ]
docs: "DOCS-1091"
categories: ["configuration"]
doctypes: ["task"]
---

## Overview

NGINX Agent is a companion daemon for your NGINX Open Source or NGINX Plus instance. It enables:

- Remote management of NGINX configurations
- Collection and reporting of real-time NGINX performance and operating system metrics
- Notifications of NGINX events


{{< img src="grafana-dashboard-example.png" caption="Grafana  dashboard showing metrics reported by NGINX Agent" alt="Grafana dashboard showing metrics reported by NGINX Agent" width="99%">}}

## How it Works

NGINX Agent runs as a companion process on a system running NGINX. It provides gRPC and REST interfaces for configuration management and metrics collection from the NGINX process and operating system. NGINX Agent enables remote interaction with NGINX using common Linux tools and unlocks the ability to build sophisticated monitoring and control systems that can manage large collections of NGINX instances.

{{< img src="agent-flow.png" caption="How Agent works" alt="How NGINX Agent works" width="99%">}}


## Configuration Management

NGINX Agent provides an API interface for submission of updated configuration files. Upon receipt of a new file, it checks the output of `nginx -V` to determine the location of existing configurations. It then validates the new configuration with `nginx -t` before applying it via a signal HUP to the NGINX master process.

## Collecting Metrics

NGINX Agent interfaces with NGINX process information and parses NGINX logs to calculate and report metrics. When interfacing with NGINX Plus, NGINX Agent pulls relevant information from the NGINX Plus API. Reported metrics may be aggregated by [Prometheus](https://prometheus.io/) and visualized with tools like [Grafana](https://grafana.com/).

### NGINX Open Source

When running alongside an open source instance of NGINX, NGINX Agent requires that NGINX Access and Error logs are turned on and contain all default variables.

### NGINX Plus

For NGINX Agent to work properly with an NGINX Plus instance, the API needs to be configured in that instance's nginx.conf. See [Instance Metrics Overview](https://docs.nginx.com/nginx-instance-manager/monitoring/overview-metrics/) for more details. Once NGINX Plus is configured with the `/api/` endpoint, the Agent will automatically use it on startup.

## Event Notifications

NGINX Agent allows a gRPC connected control system to register a listener for a specific event. The control mechanism is then invoked when NGINX Agent sends an associated system signal. The source of a notification can be either the NGINX instance or NGINX Agent itself. Here's a list of currently supported events:


{{< raw-html>}}<div class="table-responsive">{{</raw-html>}}
{{<bootstrap-table "table table-striped table-bordered">}}
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
{{</bootstrap-table>}}
{{< raw-html>}}</div>{{</raw-html>}}


