---
title: "Docker Images"
draft: false
description: "Learn how to build and run NGINX Agent docker images"
weight: 800
toc: true
tags: [ "docs" ]
docs: "DOCS-1230"
categories: ["configuration"]
doctypes: ["task"]
---

## Prerequisites
1. Docker installation
1. nginx-repo.crt and nginx-repo.key files (if using NGINX Plus)

## Building NGINX Agent image with NGINX OSS

To build an image that contains the latest NGINX Agent and the latest mainline version of NGINX OSS run the following command:

```shell
cd scripts/docker/official/nginx-oss-with-nginx-agent/alpine/
docker build -t nginx-agent . \
-f ./Dockerfile.mainline
```

To build an image that contains the latest NGINX Agent and the latest stable version of NGINX OSS run the following command:
```shell
cd scripts/docker/official/nginx-oss-with-nginx-agent/alpine/
docker build -t nginx-agent . \
-f ./Dockerfile.stable
```

## Building NGINX Agent image with NGINX Plus

1. Log in to [MyF5 Customer Portal](https://account.f5.com/myf5) and download your `nginx-repo.crt` and `nginx-repo.key` files. For a trial of NGINX Plus, the files are provided with your trial package.

1. Copy the files to the directory where the Dockerfile is located `scripts/docker/official/nginx-plus-with-nginx-agent/alpine/`.

1. To build an image that contains the latest NGINX Agent and the latest version of NGINX Plus run the following command:

```shell
cd scripts/docker/official/nginx-plus-with-nginx-agent/alpine/
docker build -t nginx-agent . \
--no-cache -f ./Dockerfile \
--secret id=nginx-crt,src=nginx-repo.crt \
--secret id=nginx-key,src=nginx-repo.key
```

## Running NGINX Agent container

Here is an example of how to run a NGINX Agent container:

```console
$ docker run --name nginx-agent -d nginx-agent
```

By default the NGINX Agent REST API is not enabled. To enable it, the NGINX Agent configuration needs to be updated to include the API configuration. Here is an example of what needs to be added to the `nginx-agent.conf`:

```yaml
api:
  host: 0.0.0.0
  port: 8038
```

Once the nginx-agent.conf is updated with the API setting, here is an example of how to run the container with the updated `nginx-agent.conf` mounted and the port `8038` exposed:

```console
$ docker run --name nginx-agent -d \
  --mount type=bind,source="$(pwd)"/nginx-agent.conf,target=/etc/nginx-agent/nginx-agent.conf,readonly \
-p 127.0.0.1:8038:8038/tcp \
nginx-agent
```
