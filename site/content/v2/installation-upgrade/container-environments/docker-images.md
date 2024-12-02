---
title: "Build container images"
draft: false
weight: 100
toc: true
tags: [ "docs" ]
categories: ["configuration"]
doctypes: ["task"]
docs: "DOCS-1410"
---

## Overview

NGINX Agent is a companion daemon for NGINX Open Source or NGINX Plus instances and must run in the same container to work. This document explains multiple ways in which NGINX Agent can be run alongside NGINX in a container.

If you want to use NGINX Agent with NGINX Plus, you need to purchase an NGINX Plus license. Contact your F5 Sales representative for assistance.

See the requirements and supported operating systems in the [NGINX Agent Technical Specifications]({{< relref "technical-specifications.md" >}}) topic.

## Deploy Offical NGINX and NGINX Plus Containers

Docker images are available in the [Deploying NGINX and NGINX Plus on Docker](https://docs.nginx.com/nginx/admin-guide/installing-nginx/installing-nginx-docker/) NGINX documentation.

This guide provides instructions on how to build images with NGINX Agent and NGINX packaged together. It includes steps for downloading the necessary Docker images, configuring your Docker environment, and deploying NGINX and NGINX Plus containers.

## Set up your environment

### Install a container engine

You can use [Docker](https://docs.docker.com/engine/install/) or [Podman](https://podman.io/docs/installation) to manage NGINX Agent container images. Follow the installation instructions for your preferred container engine and be sure the service is running before proceeding with the instructions in this document.

{{<note>}}The examples in this document primarily use Docker commands. You can adapt these using the appropriate [Podman commands](https://docs.podman.io/en/latest/Commands.html) if you're not using Docker.{{</note>}}

### Install the GNU Make package

You need to use the [GNU Make](https://www.gnu.org/software/make/) package to build the NGINX Agent container images provided in the nginx-agent GitHub repository.

If you do not already have Make installed, install it using the appropriate package manager for your operating system.

For example, to install **make** using the Ubuntu Advanced Packaging Tool (APT), run the command **apt install** command shown in the example. In some cases, it may help to update the package source lists in your operating system before proceeding.

1. Update the package source list:

    ```shell
    sudo apt update
    ```

2. Install the `make` package:

    ```shell
    sudo apt install make
    ```

### Clone the nginx-agent repository

The NGINX Agent GitHub repo contains the Dockerfiles and supporting scripts that you will use to build your images.

Run the appropriate command below to clone the GitHub repo by using HTTPS or SSH.

{{<tabs name="clone-repo">}}

{{%tab name="HTTPS"%}}

```shell
git clone https://github.com/nginx/agent.git
```

{{% /tab %}}

{{%tab name="SSH"%}}

```shell
git clone git@github.com:nginx/agent.git
```

{{% /tab %}}

{{% /tabs %}}

### Download the NGINX Plus certificate and key {#myf5-download}

{{< fa "circle-info" "text-muted" >}} **This step is required if you are using NGINX Plus. If you are using NGINX open source, you can skip this section.**

In order to build a container image with NGINX Plus, you must provide the SSL certificate and private key files provided with your NGINX Plus license. These files grant access to the package repository from which the script will download the NGINX Plus package.

1. Log in to the [MyF5](https://my.f5.com) customer portal.
1. Go to **My Products and Plans** > **Subscriptions**.
1. Select the product subscription.
1. Download the **SSL Certificate** and **Private Key** files.
1. Move the SSL certificate and private key files to the directory where you cloned the nginx-agent repo.

   - The Makefile expects to find these files in the path *./build/certs*.  Assuming you cloned the nginx-agent repo to your **$HOME** directory, you would move and rename the files as follows:

     ```shell
     mkdir -p $HOME/nginx-agent/build/certs
     mv nginx-repo-S-X00012345.key $HOME/nginx-agent/build/certs/nginx-repo.key
     mv nginx-repo-S-X00012345.crt $HOME/nginx-agent/build/certs/nginx-repo.crt
     ```

   - Be sure to replace the example certificate and key filenames shown in the example command with your actual file names.
   - The file names in the *build/certs* directory must match those shown in the example.

## Run the NGINX Agent container

To run NGINX Agent container using Docker use the following command:

```shell
docker pull docker-registry.nginx.com/nginx/agent:mainline
```
```shell
docker tag docker-registry.nginx.com/nginx/agent:mainline nginx-agent
```
```shell
docker run --name nginx-agent -d nginx-agent
```

{{<note>}}To learn more about the configuration options, refer to the NGINX Agent [Configuration Overview]({{< relref "/v2/configuration/configuration-overview" >}}).{{</note>}}

### Enable the gRPC interface

To connect your NGINX Agent container to your NGINX One or NGINX Instance Manager instance, you must enable the gRPC interface. To do this, you must edit the NGINX Agent configuration file, *nginx-agent.conf*. For example:

```yaml
server:
  host: 127.0.0.1 # mock control plane host
  grpcPort: 54789 # mock control plane gRPC port

# gRPC TLS options - DISABLING TLS IS NOT RECOMMENDED FOR PRODUCTION
tls:
  enable: false
  skip_verify: true
```

### Enable the REST interface

If your control plane requires REST API, you can expose NGINX Agent's REST API by editing the NGINX Agent configuration file, *nginx-agent.conf*. For example:

```yaml
api:
  host: 0.0.0.0
  port: 8038
```

Once you have updated the *nginx-agent.conf* file, you can run the container with the updated **nginx-agent.conf** mounted and the port **8038** exposed with the following command:

```console
docker run --name nginx-agent -d \
  --mount type=bind,source="$(pwd)"/nginx-agent.conf,target=/etc/nginx-agent/nginx-agent.conf,readonly \
  -p 127.0.0.1:8038:8038/tcp \
  nginx-agent
```

To ensure that the REST Interface is correctly configured, you can use the `curl` command targeting the following endpoint from your terminal:

```shell
curl 0.0.0.0:8038/nginx/
```

If the REST Interface is configured correctly, then you should see a JSON object ouputted to the terminal containing metadata such as NGINX version, path to the NGINX conf, and runtime modules.

**Sample Output:**

```code
[{"nginx_id":"b636d4376dea15405589692d3c5d3869ff3a9b26b0e7bb4bb1aa7e658ace1437","version":"1.27.1","conf_path":"/etc/nginx/nginx.conf","process_id":"7","process_path":"/usr/sbin/nginx","start_time":1725878806000,"built_from_source":false,"loadable_modules":null,"runtime_modules":["http_addition_module","http_auth_request_module","http_dav_module","http_flv_module","http_gunzip_module","http_gzip_static_module","http_mp4_module","http_random_index_module","http_realip_module","http_secure_link_module","http_slice_module","http_ssl_module","http_stub_status_module","http_sub_module","http_v2_module","http_v3_module","mail_ssl_module","stream_realip_module","stream_ssl_module","stream_ssl_preread_module"],"plus":{"enabled":false,"release":""},"ssl":{"ssl_type":0,"details":["OpenSSL","3.3.0","9 Apr 2024 (running with OpenSSL 3.3.1 4 Jun 2024)"]},"status_url":"","configure_args":["","prefix=/etc/nginx","sbin-path=/usr/sbin/nginx","modules-path=/usr/lib/nginx/modules","conf-path=/etc/nginx/nginx.conf","error-log-path=/var/log/nginx/error.log","http-log-path=/var/log/nginx/access.log","pid-path=/var/run/nginx.pid","lock-path=/var/run/nginx.lock","http-client-body-temp-path=/var/cache/nginx/client_temp","http-proxy-temp-path=/var/cache/nginx/proxy_temp","http-fastcgi-temp-path=/var/cache/nginx/fastcgi_temp","http-uwsgi-temp-path=/var/cache/nginx/uwsgi_temp","http-scgi-temp-path=/var/cache/nginx/scgi_temp","with-perl_modules_path=/usr/lib/perl5/vendor_perl","user=nginx","group=nginx","with-compat","with-file-aio","with-threads","with-http_addition_module","with-http_auth_request_module","with-http_dav_module","with-http_flv_module","with-http_gunzip_module","with-http_gzip_static_module","with-http_mp4_module","with-http_random_index_module","with-http_realip_module","with-http_secure_link_module","with-http_slice_module","with-http_ssl_module","with-http_stub_status_module","with-http_sub_module","with-http_v2_module","with-http_v3_module","with-mail","with-mail_ssl_module","with-stream","with-stream_realip_module","with-stream_ssl_module","with-stream_ssl_preread_module","with-cc-opt='-Os -fstack-clash-protection -Wformat -Werror=format-security -g'","with-ld-opt=-Wl,--as-needed,-O1,--sort-common"],"error_log_paths":null}]
```

<hr>

## Build the NGINX Agent images for specific OS targets

{{<important>}}The only **officially supported** base operating system is **Alpine**. The instructions below for other operating systems are provided for informational and **testing purposes only**.{{</important>}}

The NGINX Agent GitHub repo has a set of Make commands that you can use to build a container image for an specific operating system and version:

- `make oss-image` builds an image containing NGINX Agent and NGINX open source.
- `make image` builds an image containing NGINX Agent and NGINX Plus.

You can pass the following arguments when running the **make** command to build an NGINX Agent container image.

{{<bootstrap-table "table table-striped table-border">}}
| Argument | Definition |
| ---------------- | -------------------------|
| OS_RELEASE      | The Linux distribution to use as the base image. <br>Can also be set in the repo Makefile.|
| OS_VERSION      | The version of the Linux distribution to use as the base image. <br>Can also be set in the repo Makefile.|
| AGENT_VERSION      | The versions of NGINX agent that you want installed on the image.|

{{</bootstrap-table>}}

### Build NGINX open source images

Run the following `make` command to build the default image, which uses Alpine as the base image:

```shell
IMAGE_BUILD_TARGET=install-agent-repo make oss-image
```

To build an image with Debian and an older version of NGINX Agent you can run the following command:

```shell
IMAGE_BUILD_TARGET=install-agent-repo NGINX_AGENT_VERSION=2.37.0~bullseye OS_RELEASE=debian OS_VERSION=bullseye-slim make oss-image
```

### Build NGINX Plus images

{{<important>}}You need a license to use NGINX Agent with NGINX Plus. You must complete the steps in the [Download the certificate and key files from MyF5](#myf5-download) section before proceeding.{{</important>}}

Run the following `make` command to build the default image, which uses Ubuntu 24.04 (Noble) as the base image.

```shell
IMAGE_BUILD_TARGET=install-agent-repo make image
```

To build an image with Debian and an older version of NGINX Agent you can run the following command:

```shell
IMAGE_BUILD_TARGET=install-agent-repo NGINX_AGENT_VERSION=2.37.0~bullseye OS_RELEASE=debian OS_VERSION=bullseye-slim make image
```



