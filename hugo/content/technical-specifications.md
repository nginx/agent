---
title: "Technical Specifications"
draft: false
description: "This document describes the requirements for NGINX Agent."
weight: 500
toc: true
tags: [ "docs" ]
docs: "DOCS-1092"
categories: ["development"]
doctypes: ["task"]
---

## Supported Distributions

NGINX Agent can run in most environments. The below list are the supported distributions.

AlmaLinux
- 8 (x86_64, aarch64)
- 9 (x86_64, aarch64)

Alpine Linux
- 3.13 (x86_64, aarch64)
- 3.14 (x86_64, aarch64)
- 3.15 (x86_64, aarch64)
- 3.16 (x86_64, aarch64)

Amazon Linux 2
- LTS (x86_64, aarch64)

CentOS
- 7.4+ (x86_64, aarch64)

Debian
- 11 (x86_64, aarch64)

FreeBSD
- 12.1+ (amd64)
- 13 (amd64)

Oracle Linux
- 7.4+ (x86_64)
- 8.1+ (x86_64, aarch64)

Red Hat Enterprise Linux (RHEL)
- 7.4+ (x86_64)
- 8.1+ (x86_64)
- 9.0+ (x86_64)

Rocky Linux
- 8 (x86_64, aarch64)
- 9 (x86_64, aarch64)

SUSE Linux Enterprise Server (SLES)
- 12 SP5 (x86_64)
- 15 SP2 (x86_64)

Ubuntu
- 18.04 LTS (x86_64, aarch64)
- 20.04 LTS (x86_64, aarch64)
- 22.04 LTS (x86_64, aarch64)

## Supported Deployment Environments 

NGINX Agent can be deployed in the following environments:

- Bare Metal
- Container
- Public Cloud: AWS, Google Cloud Platform, and Microsoft Azure
- Virtual Machine

## Supported NGINX Versions 

NGINX Agent works with all supported versions of NGINX Open Source and NGINX Plus.


## Sizing Recommendations

Minimum system sizing recommendations for NGINX Agent:

| CPU        | Memory   | Network   | Storage |
|------------|----------|-----------|---------|
| 1 CPU core | 1 GB RAM | 1 GbE NIC | 20 GB   |
