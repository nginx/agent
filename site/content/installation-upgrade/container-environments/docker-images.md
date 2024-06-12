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

NGINX Agent is a companion daemon for NGINX Open Source or NGINX Plus instances and must run in the same container to work. The NGINX Agent GitHub repository includes [Dockerfiles](https://github.com/nginx/agent/tree/main/scripts/docker) that you can use to build custom container images that bundle NGINX Agent with NGINX or NGINX Plus.

If you want to use NGINX Agent with NGINX Plus, you need to purchase an NGINX Plus license. Contact your F5 Sales representative for assistance.

See the requirements and supported operating systems in the [NGINX Agent Technical Specifications]({{< relref "technical-specifications.md" >}}) topic.


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

### Download the agent binary {#agent-binary}

Before you can build a container image with NGINX, you must build or download the **agent** binary.

The **agent** binary packages are available from the [NGINX Agent Releases](https://github.com/nginx/agent/releases) page on GitHub.

Download the binary package for the operating system that you will use in the container image.

Note the location and name of the downloaded package. You will need to use this when running the **make** command to build the image (referred to as `[PATH-TO-PACKAGE]` in the example commands below).

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

## Build the official NGINX Agent image with Docker

{{<tabs name="build-image">}}

{{%tab name="NGINX Open Source"%}}

Change to the directory where the Dockerfile is located:

```shell
$ cd scripts/docker/official/nginx-oss-with-nginx-agent/alpine/
```

- To build an image that contains the latest NGINX Agent and the latest mainline version of NGINX run the following command:

   ```shell
   $ docker build -t nginx-agent . --no-cache -f ./Dockerfile.mainline
   ```

- To build an image that contains the latest NGINX Agent and the latest stable version of NGINX run the following command:

   ```shell
   $ docker build -t nginx-agent . --no-cache -f ./Dockerfile.stable
   ```

{{% /tab %}}

{{%tab name="NGINX Plus"%}}

1. Log in to [MyF5 Customer Portal](https://account.f5.com/myf5) and download your "nginx-repo.crt" and "nginx-repo.key" files. These files are also provided with the NGINX Plus trial package.

1. Copy the files to the directory where the Dockerfile is located **scripts/docker/official/nginx-plus-with-nginx-agent/alpine/**.

1. To build an image that contains the latest NGINX Agent and the latest version of NGINX Plus change to the directory where the Dockerfile is located:

```shell
$ cd scripts/docker/official/nginx-plus-with-nginx-agent/alpine/
```

1. Run the following command to build the image:

```shell
$ docker build -t nginx-agent . \
  --no-cache -f ./Dockerfile \
  --secret id=nginx-crt,src=nginx-repo.crt \
  --secret id=nginx-key,src=nginx-repo.key
```
{{% /tab %}}
{{% /tabs %}}


## Run the NGINX Agent container

Here is an example of how to run the NGINX Agent container using Docker:

```console
docker run --name nginx-agent -d nginx-agent
```

{{<note>}}To learn more about the configuration options, refer to the NGINX Agent [Configuration Overview]({{< relref "/configuration/configuration-overview" >}}).{{</note>}}

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

<hr>

## Build the NGINX Agent images for specific OS targets

{{<important>}}The only **officially supported** base operating system is **Alpine**. The instructions below for other operating systems are provided for informational and **testing purposes only**.{{</important>}}

The NGINX Agent GitHub repo has a set of Make commands that you can use to build a container image for an specific operating system and version:

- `make official-oss-image` builds an image containing NGINX Agent and NGINX open source.
- `make official-plus-image` builds an image containing NGINX Agent and NGINX Plus.

You can pass the following arguments when running the **make** command to build an NGINX Agent container image.

{{<bootstrap-table "table table-striped table-border">}}
| Argument | Definition |
| ---------------- | -------------------------|
| PACKAGE_NAME     | **Required.** The full path to the downloaded [agent binary package](#agent-binary). <br>Must precede the **make** command. |
| OS_RELEASE      | The Linux distribution to use as the base image. <br>Can also be set in the repo Makefile.|
| OS_VERSION      | The version of the Linux distribution to use as the base image. <br>Can also be set in the repo Makefile.|
{{</bootstrap-table>}}

Refer to the [Supported distributions](technical-specifications.md/#supported-distributions) table to find out which base images you can use.
You can find the official images and versions for each distribution on [Docker Hub](https://hub.docker.com/search?image_filter=official&q=&type=image).

Keep the following information in mind when using the NGINX Agent [Dockerfiles](https://github.com/nginx/agent/tree/main/scripts/docker) to build container images:

- On some operating systems, you need root privileges (**sudo**) to run **make** commands.
- If you choose to run the **docker build** or **podman build** command instead of using the **make** commands provided, you must do so from the nginx-agent repository's root directory.

### Build NGINX open source images

Run the following **make** command to build the default image, which uses Alpine 3.19 as the base image.

```shell
PACKAGE_NAME=[PATH-TO-PACKAGE] make official-oss-image
```

### Example build commands by distribution

{{<tabs name="build-oss-image">}}

{{%tab name="alma linux"%}}

```shell
PACKAGE_NAME=[PATH-TO-PACKAGE] OS_RELEASE=almalinux make oss-image
```

{{% /tab %}}

{{%tab name="alpine linux"%}}

```shell
PACKAGE_NAME=[PATH-TO-PACKAGE] OS_RELEASE=alpine make oss-image
```

{{% /tab %}}

{{%tab name="amazon linux"%}}

```shell
PACKAGE_NAME=[PATH-TO-PACKAGE] OS_RELEASE=amazonlinux make oss-image
```

{{% /tab %}}

{{%tab name="debian"%}}

```shell
PACKAGE_NAME=[PATH-TO-PACKAGE] OS_RELEASE=debian make oss-image
```

{{% /tab %}}

{{%tab name="oracle linux"%}}

```shell
PACKAGE_NAME=[PATH-TO-PACKAGE] OS_RELEASE=oraclelinux make oss-image
```

{{% /tab %}}

{{%tab name="rocky linux"%}}

```shell
PACKAGE_NAME=[PATH-TO-PACKAGE] OS_RELEASE=rockylinux make oss-image
```

{{% /tab %}}

{{%tab name="ubuntu"%}}

The command below creates a base image using the most recent LTS version of Ubuntu as the base image:

```shell
PACKAGE_NAME=[PATH-TO-PACKAGE] make oss-image OS_RELEASE=ubuntu
```

{{% /tab %}}

{{% /tabs %}}

### Build NGINX Plus images

{{<important>}}You need a license to use NGINX Agent with NGINX Plus. You must complete the steps in the [Download the certificate and key files from MyF5](#myf5-download) section before proceeding.{{</important>}}

Run the following `make` command to build the default image, which uses Ubuntu 24.04 (Noble) as the base image.

```shell
PACKAGE_NAME=[PATH-TO-PACKAGE] make official-plus-image
```

### Example NGINX Plus build commands by distribution

{{<tabs name="build-image">}}

{{%tab name="alpine linux"%}}

```shell
PACKAGE_NAME=[PATH-TO-PACKAGE] OS_RELEASE=alpine make official-plus-image
```

{{% /tab %}}

{{%tab name="amazon linux"%}}

```shell
PACKAGE_NAME=[PATH-TO-PACKAGE] OS_RELEASE=amazonlinux make official-plus-image
```

{{% /tab %}}

{{%tab name="centos"%}}

```shell
PACKAGE_NAME=[PATH-TO-PACKAGE] OS_RELEASE=centos OS_VERSION=7 make official-plus-image
```

{{% /tab %}}

{{%tab name="debian"%}}

```shell
PACKAGE_NAME=[PATH-TO-PACKAGE] OS_RELEASE=debian OS_VERSION=bullseye-slim make official-plus-image
```

{{% /tab %}}

{{%tab name="oracle linux"%}}

```shell
PACKAGE_NAME=[PATH-TO-PACKAGE] OS_RELEASE=oraclelinux make official-plus-image
```

{{% /tab %}}

{{%tab name="rhel"%}}

```shell
PACKAGE_NAME=[PATH-TO-PACKAGE] OS_RELEASE=redhatenterprise make official-plus-image
```

{{% /tab %}}

{{%tab name="suse"%}}

```shell
PACKAGE_NAME=[PATH-TO-PACKAGE] OS_RELEASE=suse OS_VERSION=sle15 make official-plus-image
```

{{% /tab %}}

{{%tab name="ubuntu"%}}

```shell
PACKAGE_NAME=[PATH-TO-PACKAGE] OS_RELEASE=ubuntu make official-plus-image
```

{{% /tab %}}

{{% /tabs %}}
