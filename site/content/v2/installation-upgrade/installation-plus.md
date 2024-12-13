---
title: "Installation from NGINX Plus repository"
draft: false
weight: 400
toc: true
tags: [ "docs" ]
docs: "DOCS-1217"
categories: ["configuration"]
doctypes: ["task"]
---

## Overview

Learn how to install NGINX Agent from NGINX Plus repository

## Prerequisites

- An NGINX Plus subscription (purchased or trial)
- NGINX Plus installed. Once installed, ensure it is running. If you don't have it installed already, follow these steps to install [NGINX Plus](https://docs.nginx.com/nginx/admin-guide/installing-nginx/installing-nginx-plus/)
- A [supported operating system and architecture]({{< relref "/technical-specifications.md#supported-distributions" >}})
- `root` privilege
- Your credentials to the MyF5 Customer Portal, provided by email from F5, Inc.
- Your NGINX Plus certificate and public key (`nginx-repo.crt` and `nginx-repo.key` files), provided by email from F5, Inc.

## Configure NGINX Plus Repository for installing NGINX Agent

Before you install NGINX Agent for the first time on your system, you need to set up the `nginx-agent` packages repository. Afterward, you can install and update NGINX Agent from the repository.

- [Installing NGINX Agent on RHEL, CentOS, Rocky Linux, AlmaLinux, and Oracle Linux](#installing-nginx-agent-on-rhel-centos-rocky-linux-almalinux-and-oracle-linux)
- [Installing NGINX Agent on Ubuntu](#installing-nginx-agent-on-ubuntu)
- [Installing NGINX Agent on Debian](#installing-nginx-agent-on-debian)
- [Installing NGINX Agent on SLES](#installing-nginx-agent-on-sles)
- [Installing NGINX Agent on Alpine Linux](#installing-nginx-agent-on-alpine-linux)
- [Installing NGINX Agent on Amazon Linux 2023](#installing-nginx-agent-on-amazon-linux-2023)
- [Installing NGINX Agent on Amazon Linux](#installing-nginx-agent-on-amazon-linux)
- [Installing NGINX Agent on FreeBSD](#installing-nginx-agent-on-freebsd)

### Installing NGINX Agent on RHEL, CentOS, Rocky Linux, AlmaLinux, and Oracle Linux

1. Create the `/etc/ssl/nginx` directory:

    ```shell
    sudo mkdir -p /etc/ssl/nginx
    ```

1. Log in to [MyF5 Customer Portal](https://account.f5.com/myf5/) and download your `nginx-repo.crt` and `nginx-repo.key` files.

1. Copy the files to the `/etc/ssl/nginx/` directory:

    ```shell
    sudo cp nginx-repo.crt nginx-repo.key /etc/ssl/nginx/
    ```

1. Install the prerequisites:

    ```shell
    sudo yum install yum-utils procps
    ```

1. Set up the yum repository by creating the file `nginx-agent.repo` in `/etc/yum.repos.d`, for example using `vi`:

    ```shell
    sudo vi /etc/yum.repos.d/nginx-agent.repo
    ```

1. Add the following lines to `nginx-agent.repo`:

    ```
    [nginx-agent]
    name=nginx agent repo
    baseurl=https://pkgs.nginx.com/nginx-agent/centos/$releasever/$basearch/
    sslclientcert=/etc/ssl/nginx/nginx-repo.crt
    sslclientkey=/etc/ssl/nginx/nginx-repo.key
    gpgcheck=0
    enabled=1
    ```

1. To install `nginx-agent`, run the following command:

    ```shell
    sudo yum install nginx-agent
    ```

    When prompted to accept the GPG key, verify that the fingerprint matches `8540 A6F1 8833 A80E 9C16 53A4 2FD2 1310 B49F 6B46`, `573B FD6B 3D8F BC64 1079 A6AB ABF5 BD82 7BD9 BF62`, `9E9B E90E ACBC DE69 FE9B 204C BCDC D8A3 8D88 A2B3`, and if so, accept it.

1. Verify the installation:

    ```shell
    sudo nginx-agent -v
    ```

### Installing NGINX Agent on Ubuntu

1. Create the `/etc/ssl/nginx` directory:

    ```shell
    sudo mkdir -p /etc/ssl/nginx
    ```

1. Log in to [MyF5 Customer Portal](https://account.f5.com/myf5/) and download your `nginx-repo.crt` and `nginx-repo.key` files.

1. Copy the files to the `/etc/ssl/nginx/` directory:

    ```shell
    sudo cp nginx-repo.crt nginx-repo.key /etc/ssl/nginx/
    ```

1. Install the prerequisites:

    ```shell
    sudo apt-get install apt-transport-https lsb-release ca-certificates wget gnupg2 ubuntu-keyring
    ```

1. Download and add [NGINX signing key](https://cs.nginx.com/static/keys/nginx_signing.key):

    ```shell
    wget -qO - https://cs.nginx.com/static/keys/nginx_signing.key | gpg --dearmor | sudo tee /usr/share/keyrings/nginx-archive-keyring.gpg >/dev/null
    ```

1. Create `apt` configuration `/etc/apt/apt.conf.d/90pkgs-nginx`:

    ```
    Acquire::https::pkgs.nginx.com::Verify-Peer "true";
    Acquire::https::pkgs.nginx.com::Verify-Host "true";
    Acquire::https::pkgs.nginx.com::SslCert     "/etc/ssl/nginx/nginx-repo.crt";
    Acquire::https::pkgs.nginx.com::SslKey      "/etc/ssl/nginx/nginx-repo.key";
    ```

1. Add the `nginx-agent` repository:

    ```shell
    echo "deb [signed-by=/usr/share/keyrings/nginx-archive-keyring.gpg] https://pkgs.nginx.com/nginx-agent/ubuntu/ `lsb_release -cs` agent" \
      | sudo tee /etc/apt/sources.list.d/nginx-agent.list
    ```

1. To install `nginx-agent`, run the following commands:

    ```shell
    sudo apt update
    sudo apt install nginx-agent
    ```

1. Verify the installation:

    ```shell
    sudo nginx-agent -v
    ```


### Installing NGINX Agent on Debian

1. Create the `/etc/ssl/nginx` directory:

    ```shell
    sudo mkdir -p /etc/ssl/nginx
    ```

1. Log in to [MyF5 Customer Portal](https://account.f5.com/myf5/) and download your `nginx-repo.crt` and `nginx-repo.key` files.

1. Copy the files to the `/etc/ssl/nginx/` directory:

    ```shell
    sudo cp nginx-repo.crt nginx-repo.key /etc/ssl/nginx/
    ```

1. Install the prerequisites:

    ```shell
    sudo apt install curl gnupg2 ca-certificates lsb-release debian-archive-keyring
    ```

1. Add the `nginx-agent` repository:

    ```shell
    echo "deb https://pkgs.nginx.com/nginx-agent/debian/ `lsb_release -cs` agent" \
      | sudo tee /etc/apt/sources.list.d/nginx-agent.list
    ```

1. Create apt configuration `/etc/apt/apt.conf.d/90pkgs-nginx`:

    ```
    Acquire::https::pkgs.nginx.com::Verify-Peer "true";
    Acquire::https::pkgs.nginx.com::Verify-Host "true";
    Acquire::https::pkgs.nginx.com::SslCert     "/etc/ssl/nginx/nginx-repo.crt";
    Acquire::https::pkgs.nginx.com::SslKey      "/etc/ssl/nginx/nginx-repo.key";
    ```

1. To install `nginx-agent`, run the following commands:

    ```shell
    sudo apt update
    sudo apt install nginx-agent
    ```

1. Verify the installation:

    ```shell
    sudo nginx-agent -v
    ```

### Installing NGINX Agent on SLES

1. Create the `/etc/ssl/nginx` directory:

    ```shell
    sudo mkdir -p /etc/ssl/nginx
    ```

1. Log in to [MyF5 Customer Portal](https://account.f5.com/myf5/) and download your `nginx-repo.crt` and `nginx-repo.key` files.

1. Copy the files to the `/etc/ssl/nginx/` directory:

    ```shell
    sudo cp nginx-repo.crt nginx-repo.key /etc/ssl/nginx/
    ```

1. Create a file bundle of the certificate and key:

    ```shell
    cat /etc/ssl/nginx/nginx-repo.crt /etc/ssl/nginx/nginx-repo.key > /etc/ssl/nginx/nginx-repo-bundle.crt
    ```

1. Install the prerequisites:

    ```shell
    sudo zypper install curl ca-certificates gpg2 gawk
    ```

1. To set up the zypper repository for `nginx-agent` packages, run the following command:

    ```shell
    sudo zypper addrepo --refresh --check \
        'https://pkgs.nginx.com/nginx-agent/sles/$releasever_major?ssl_clientcert=/etc/ssl/nginx/nginx-repo-bundle.crt&ssl_verify=peer' nginx-agent
    ```

1. Next, import an official NGINX signing key so `zypper`/`rpm` can verify the package's authenticity. Fetch the key:

    ```shell
    curl -o /tmp/nginx_signing.key https://nginx.org/keys/nginx_signing.key
    ```

1. Verify that the downloaded file contains the proper key:

    ```shell
    gpg --with-fingerprint --dry-run --quiet --no-keyring --import --import-options import-show /tmp/nginx_signing.key
    ```

1. The output should contain the full fingerprints `8540 A6F1 8833 A80E 9C16 53A4 2FD2 1310 B49F 6B46`, `573B FD6B 3D8F BC64 1079 A6AB ABF5 BD82 7BD9 BF62`, `9E9B E90E ACBC DE69 FE9B 204C BCDC D8A3 8D88 A2B3` as follows:

    ```
      pub   rsa4096 2024-05-29 [SC]
            8540A6F18833A80E9C1653A42FD21310B49F6B46
      uid                      nginx signing key <signing-key-2@nginx.com>

      pub   rsa2048 2011-08-19 [SC] [expires: 2027-05-24]
            573BFD6B3D8FBC641079A6ABABF5BD827BD9BF62
      uid                      nginx signing key <signing-key@nginx.com>

      pub   rsa4096 2024-05-29 [SC]
            9E9BE90EACBCDE69FE9B204CBCDCD8A38D88A2B3
      uid                      nginx signing key <signing-key-3@nginx.com>
    ```

1. Finally, import the key to the rpm database:

    ```shell
    sudo rpmkeys --import /tmp/nginx_signing.key
    ```

1. To install `nginx-agent`, run the following command:

    ```shell
    sudo zypper install nginx-agent
    ```

1. Verify the installation:

    ```shell
    sudo nginx-agent -v
    ```

### Installing NGINX Agent on Alpine Linux

1. Log in to [MyF5 Customer Portal](https://account.f5.com/myf5/) and download your `nginx-repo.crt` and `nginx-repo.key` files.

1. Copy the files to the `/etc/apk/` directory:

    ```shell
    sudo cp nginx-repo.key /etc/apk/cert.key
    sudo cp nginx-repo.crt /etc/apk/cert.pem
    ```

1. Install the prerequisites:

    ```shell
    sudo apk add openssl curl ca-certificates
    ```

1. To set up the apk repository for `nginx-agent` packages, run the following command:

    ```shell
    printf "%s%s%s\n" \
        "https://pkgs.nginx.com/nginx-agent/alpine/v" \
        `grep -o -E '^[0-9]+\.[0-9]+' /etc/alpine-release` \
        "/main" \
        | sudo tee -a /etc/apk/repositories
    ```

1. Next, import an official NGINX signing key so apk can verify the package's authenticity. Fetch the key:

    ```shell
    curl -o /tmp/nginx_signing.rsa.pub https://nginx.org/keys/nginx_signing.rsa.pub
    ```

1. Verify that downloaded file contains the proper key:

    ```shell
    openssl rsa -pubin -in /tmp/nginx_signing.rsa.pub -text -noout
    ```

    The output should contain the following modulus:

    ```
    Public-Key: (2048 bit)
    Modulus:
        00:fe:14:f6:0a:1a:b8:86:19:fe:cd:ab:02:9f:58:
        2f:37:70:15:74:d6:06:9b:81:55:90:99:96:cc:70:
        5c:de:5b:e8:4c:b2:0c:47:5b:a8:a2:98:3d:11:b1:
        f6:7d:a0:46:df:24:23:c6:d0:24:52:67:ba:69:ab:
        9a:4a:6a:66:2c:db:e1:09:f1:0d:b2:b0:e1:47:1f:
        0a:46:ac:0d:82:f3:3c:8d:02:ce:08:43:19:d9:64:
        86:c4:4e:07:12:c0:5b:43:ba:7d:17:8a:a3:f0:3d:
        98:32:b9:75:66:f4:f0:1b:2d:94:5b:7c:1c:e6:f3:
        04:7f:dd:25:b2:82:a6:41:04:b7:50:93:94:c4:7c:
        34:7e:12:7c:bf:33:54:55:47:8c:42:94:40:8e:34:
        5f:54:04:1d:9e:8c:57:48:d4:b0:f8:e4:03:db:3f:
        68:6c:37:fa:62:14:1c:94:d6:de:f2:2b:68:29:17:
        24:6d:f7:b5:b3:18:79:fd:31:5e:7f:4c:be:c0:99:
        13:cc:e2:97:2b:dc:96:9c:9a:d0:a7:c5:77:82:67:
        c9:cb:a9:e7:68:4a:e1:c5:ba:1c:32:0e:79:40:6e:
        ef:08:d7:a3:b9:5d:1a:df:ce:1a:c7:44:91:4c:d4:
        99:c8:88:69:b3:66:2e:b3:06:f1:f4:22:d7:f2:5f:
        ab:6d
    Exponent: 65537 (0x10001)
    ```

1. Finally, move the key to apk trusted keys storage:

    ```shell
    sudo mv /tmp/nginx_signing.rsa.pub /etc/apk/keys/
    ```

1. To install `nginx-agent`, run the following command:

    ```shell
    sudo apk add nginx-agent
    ```

1. Verify the installation:

    ```shell
    sudo nginx-agent -v
    ```

### Installing NGINX Agent on Amazon Linux 2023

1. Create the `/etc/ssl/nginx` directory:

    ```shell
    sudo mkdir -p /etc/ssl/nginx
    ```

1. Log in to [MyF5 Customer Portal](https://account.f5.com/myf5/) and download your `nginx-repo.crt` and `nginx-repo.key` files.

1. Copy the `nginx-repo.crt` and `nginx-repo.key` files to the `/etc/ssl/nginx/` directory:

    ```shell
    sudo cp nginx-repo.crt nginx-repo.key /etc/ssl/nginx/
    ```

1. Install the prerequisites:

    ```shell
    sudo dnf install yum-utils procps-ng ca-certificates
    ```

1. To set up the dnf repository for Amazon Linux 2023, create the file named `/etc/yum.repos.d/nginx-agent.repo` with the following contents:

    ```
    [nginx-agent]
    name=nginx-agent repo
    baseurl=https://packages.nginx.org/nginx-agent/amzn/2023/$basearch/
    sslclientcert=/etc/ssl/nginx/nginx-repo.crt
    sslclientkey=/etc/ssl/nginx/nginx-repo.key
    gpgcheck=0
    enabled=1
    ```

1. To install `nginx-agent`, run the following command:

    ```shell
    sudo dnf install nginx-agent
    ```

1. When prompted to accept the GPG key, verify that the fingerprint matches `8540 A6F1 8833 A80E 9C16 53A4 2FD2 1310 B49F 6B46`, `573B FD6B 3D8F BC64 1079 A6AB ABF5 BD82 7BD9 BF62`, `9E9B E90E ACBC DE69 FE9B 204C BCDC D8A3 8D88 A2B3`, and if so, accept it.

1. Verify the installation:

    ```shell
    sudo nginx-agent -v
    ```

### Installing NGINX Agent on Amazon Linux 2

1. Create the `/etc/ssl/nginx` directory:

    ```shell
    sudo mkdir -p /etc/ssl/nginx
    ```

1. Log in to [MyF5 Customer Portal](https://account.f5.com/myf5/) and download your `nginx-repo.crt` and `nginx-repo.key` files.

1. Copy the `nginx-repo.crt` and `nginx-repo.key` files to the `/etc/ssl/nginx/` directory:

    ```shell
    sudo cp nginx-repo.crt nginx-repo.key /etc/ssl/nginx/
    ```

1. Install the prerequisites:

    ```shell
    sudo yum install yum-utils procps ca-certificates
    ```

1. To set up the yum repository for Amazon Linux 2, create the file named `/etc/yum.repos.d/nginx-agent.repo` with the following contents:

    ```
    [nginx-agent]
    name=nginx-agent repo
    baseurl=https://pkgs.nginx.com/nginx-agent/amzn/2023/$releasever/$basearch
    sslclientcert=/etc/ssl/nginx/nginx-repo.crt
    sslclientkey=/etc/ssl/nginx/nginx-repo.key
    gpgcheck=0
    enabled=1
    ```

1. To install `nginx-agent`, run the following command:

    ```shell
    sudo yum install nginx-agent
    ```

1. When prompted to accept the GPG key, verify that the fingerprint matches `8540 A6F1 8833 A80E 9C16 53A4 2FD2 1310 B49F 6B46`, `573B FD6B 3D8F BC64 1079 A6AB ABF5 BD82 7BD9 BF62`, `9E9B E90E ACBC DE69 FE9B 204C BCDC D8A3 8D88 A2B3`, and if so, accept it.

1. Verify the installation:

    ```shell
    sudo nginx-agent -v
    ```

### Installing NGINX Agent on FreeBSD

1. Create the `/etc/ssl/nginx` directory:

    ```shell
    sudo mkdir -p /etc/ssl/nginx
    ```

1. Log in to [MyF5 Customer Portal](https://account.f5.com/myf5/) and download your `nginx-repo.crt` and `nginx-repo.key` files.

1. Copy the files to the `/etc/ssl/nginx/` directory:

    ```shell
    sudo cp nginx-repo.crt nginx-repo.key /etc/ssl/nginx/
    ```

1. Install the prerequisite `ca_root_nss` package:

    ```shell
    sudo pkg install ca_root_nss
    ```

1. To setup the pkg repository create the file named `/etc/pkg/nginx-agent.conf` with the following content:

    ```
    nginx-agent: {
    URL: pkg+https://pkgs.nginx.com/nginx-agent/freebsd/${ABI}/latest
    ENABLED: yes
    MIRROR_TYPE: SRV
    }
    ```

1. Add the following lines to the `/usr/local/etc/pkg.conf` file:

    ```
    PKG_ENV: { SSL_NO_VERIFY_PEER: "1",
    SSL_CLIENT_CERT_FILE: "/etc/ssl/nginx/nginx-repo.crt",
    SSL_CLIENT_KEY_FILE: "/etc/ssl/nginx/nginx-repo.key" }
    ```

1. To install `nginx-agent`, run the following command:

    ```shell
    sudo pkg install nginx-agent
    ```

1. Verify the installation:

    ```shell
    sudo nginx-agent -v
    ```
