---
title: "Technical specifications"
weight: 100
toc: true
docs: "DOCS-1092"
---

This document describes the requirements for NGINX Agent v2.

## Supported distributions

NGINX Agent can run in most environments. We support the following distributions:

{{< bootstrap-table "table table-striped table-bordered" >}}
| | AlmaLinux | Alpine Linux | Amazon Linux | Amazon Linux 2 | CentOS | Debian | 
|-|-----------|--------------|--------------|----------------|--------|--------|
|**Version**|8 <br><hr>9 |  3.16<br><hr>3.17<br><hr> 3.18<br><hr> 3.19|  2023|  LTS|  7.4+|  11<br><hr> 12|
|**Architecture**| x86_84<br><hr>aarch64| x86_64<br><hr>aarch64 | x86_64<br><hr>aarch64 | x86_64<br><hr>aarch64 | x86_64<br><hr>aarch64 | x86_64<br><hr>aarch64 |
{{< /bootstrap-table >}}

{{< bootstrap-table "table table-striped table-bordered" >}}
| |FreeBSD | Oracle Linux | Red Hat <br>Enterprise Linux <br>(RHEL) | Rocky Linux | SUSE Linux <br>Enterprise Server <br>(SLES) | Ubuntu |
|-|--------|--------------|---------------------------------|-------------|-------------------------------------|--------|
|**Version**|13<br><hr>14|7.4+<br><hr>8.1+<br><hr>9|7.4+<br><hr>8.1+<br><hr>9.0+|8<br><hr>9|12 SP5<br><hr>15 SP2|20.04 LTS<br><hr>22.04 LTS|
|**Architecture**|amd64|x86_64|x86_64<br><hr>aarch64|x86_64<br><hr>aarch64|x86_64|x86_64<br><hr>aarch64|
{{< /bootstrap-table >}}


## Supported deployment environments 

NGINX Agent can be deployed in the following environments:

- Bare Metal
- Container
- Public Cloud: AWS, Google Cloud Platform, and Microsoft Azure
- Virtual Machine

## Supported NGINX versions 

NGINX Agent works with all supported versions of NGINX Open Source and NGINX Plus.


## Sizing recommendations

Minimum system sizing recommendations for NGINX Agent:
{{< bootstrap-table "table table-striped table-bordered" >}}
| CPU        | Memory   | Network   | Storage |
|------------|----------|-----------|---------|
| 1 CPU core | 1 GB RAM | 1 GbE NIC | 20 GB   |
{{< /bootstrap-table >}}

## Logging

NGINX Agent utilizes log files and formats to collect metrics. Increasing the log formats and instance counts will result in increased log file sizes. To prevent system storage issues due to a growing log directory, it is recommended to add a separate partition for `/var/log/nginx-agent` and enable [log rotation](http://nginx.org/en/docs/control.html#logs).