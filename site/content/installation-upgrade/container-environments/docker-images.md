---
title: "Build Container Images"
draft: false
weight: 100
toc: true
tags: [ "docs" ]
categories: ["configuration"]
doctypes: ["task"]
docs: "DOCS-1410"
---

## Overview

Learn how to build NGINX Agent container images with Docker.

## Prerequisites
1. Docker must be [installed and running](https://docs.docker.com/engine/install/)
1. (Optional) If you plan to use NGINX Plus, you will need the nginx-repo.crt and nginx-repo.key files. You can download them from [MyF5](https://my.f5.com).

## Building NGINX Agent image with Docker

{{<tabs name="build-image">}}

{{%tab name="NGINX Open Source"%}}

To build an image that contains the latest NGINX Agent and the latest mainline version of NGINX run the following command:

```shell
$ cd scripts/docker/official/nginx-oss-with-nginx-agent/alpine/
$ docker build -t nginx-agent . --no-cache -f ./Dockerfile.mainline
```

To build an image that contains the latest NGINX Agent and the latest stable version of NGINX run the following command:
```shell
$ cd scripts/docker/official/nginx-oss-with-nginx-agent/alpine/
$ docker build -t nginx-agent . --no-cache -f ./Dockerfile.stable
```
  
{{% /tab %}}
  
{{%tab name="NGINX Plus"%}}

1. Log in to [MyF5 Customer Portal](https://account.f5.com/myf5) and download your `nginx-repo.crt` and `nginx-repo.key` files. These files are also provided with the NGINX Plus trial package.

1. Copy the files to the directory where the Dockerfile is located `scripts/docker/official/nginx-plus-with-nginx-agent/alpine/`.

1. To build an image that contains the latest NGINX Agent and the latest version of NGINX Plus run the following command:

```shell
$ cd scripts/docker/official/nginx-plus-with-nginx-agent/alpine/
$ docker build -t nginx-agent . \
  --no-cache -f ./Dockerfile \
  --secret id=nginx-crt,src=nginx-repo.crt \
  --secret id=nginx-key,src=nginx-repo.key
```
{{% /tab %}}
{{% /tabs %}}

<hr>


## Running NGINX Agent container

Here is an example of how to run a NGINX Agent container with Docker:

```console
$ docker run --name nginx-agent -d nginx-agent
```

By default, the NGINX Agent REST API is not enabled. To enable the API, the NGINX Agent configuration file `nginx-agent.conf` needs to be updated to include the API configuration, for example:

```yaml
api:
  host: 0.0.0.0
  port: 8038
```

Once the `nginx-agent.conf` file is updated with the API setting, following the previous example, you can run the container with the updated `nginx-agent.conf` mounted and the port `8038` exposed with the following command:

```console
$ docker run --name nginx-agent -d \
  --mount type=bind,source="$(pwd)"/nginx-agent.conf,target=/etc/nginx-agent/nginx-agent.conf,readonly \
  -p 127.0.0.1:8038:8038/tcp \
  nginx-agent
```
