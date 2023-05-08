---
title: "Installation From NGINX Plus Repository"
draft: false
description: "Learn how to install NGINX Agent from NGINX Plus repository."
weight: 400
toc: true
tags: [ "docs" ]
docs: "DOCS-1092"
categories: ["configuration"]
doctypes: ["task"]
---

## Install NGINX Plus

NGINX Agent interfaces directly with an NGINX server process installed on the same system. If you don't have it already, follow these steps to install [NGINX Plus](https://docs.nginx.com/nginx/admin-guide/installing-nginx/installing-nginx-plus/). Once installed, ensure the NGINX instance is running.

## Install NGINX Agent from NGINX Plus Repository

First ensure that you are running a supported distribution and architecture: [technical specifications](../technical-specifications/)

- [RHEL, CentOS, Rocky Linux, AlmaLinux and Oracle Linux](#rhel-centos-rocky-linux-almalinux-and-oracle-linux)
- [Ubuntu](#ubuntu)
- [Debian](#debian)
- [SLES](#sles)
- [Alpine Linux](#alpine-linux)
- [Amazon Linux](#amazon-linux)

### RHEL, CentOS, Rocky Linux, AlmaLinux and Oracle Linux
  - Create the `/etc/ssl/nginx` directory:
    ```
    sudo mkdir -p /etc/ssl/nginx
    ```

  - Log in to [MyF5 Customer Portal](https://account.f5.com/myf5/) and download your `nginx-repo.crt` and `nginx-repo.key` files.

  - Copy the files to the `/etc/ssl/nginx/` directory:
    ```
    sudo cp nginx-repo.crt nginx-repo.key /etc/ssl/nginx/
    ```

  - Install the prerequisites:
    ```
    sudo yum install yum-utils procps
    ```

  - To set up the yum repository, create the file named `/etc/yum.repos.d/nginx-agent.repo` with the following contents:
    ```
    [nginx-agent]
    name=nginx agent repo
    baseurl=https://pkgs.nginx.com/nginx-agent/centos/$releasever/$basearch/
    sslclientcert=/etc/ssl/nginx/nginx-repo.crt
    sslclientkey=/etc/ssl/nginx/nginx-repo.key
    gpgcheck=0
    enabled=1
    ```

  - To install nginx agent, run the following command:
    ```
    sudo yum install nginx-agent
    ```
    When prompted to accept the GPG key, verify that the fingerprint matches `573B FD6B 3D8F BC64 1079 A6AB ABF5 BD82 7BD9 BF62`, and if so, accept it.

### Ubuntu
  - Create the `/etc/ssl/nginx` directory:
    ```
    sudo mkdir -p /etc/ssl/nginx
    ```

  - Log in to [MyF5 Customer Portal](https://account.f5.com/myf5/) and download your `nginx-repo.crt` and `nginx-repo.key` files.

  - Copy the files to the `/etc/ssl/nginx/` directory:
    ```
    sudo cp nginx-repo.crt nginx-repo.key /etc/ssl/nginx/
    ```

  - Install the prerequisites:
    ```
    sudo apt-get install apt-transport-https lsb-release ca-certificates wget gnupg2 ubuntu-keyring
    ```

  - Create apt configuration `/etc/apt/apt.conf.d/90pkgs-nginx`:
    ```
    Acquire::https::pkgs.nginx.com::Verify-Peer "true";
    Acquire::https::pkgs.nginx.com::Verify-Host "true";
    Acquire::https::pkgs.nginx.com::SslCert     "/etc/ssl/nginx/nginx-repo.crt";
    Acquire::https::pkgs.nginx.com::SslKey      "/etc/ssl/nginx/nginx-repo.key";
    ```

  - Add the `nginx-agent` repository:
    ```
    echo "deb https://pkgs.nginx.com/nginx-agent/ubuntu/ `lsb_release -cs` agent" \
      | sudo tee /etc/apt/sources.list.d/nginx-agent.list
    ```

### Debian
  - Create the `/etc/ssl/nginx` directory:
    ```
    sudo mkdir -p /etc/ssl/nginx
    ```

  - Log in to [MyF5 Customer Portal](https://account.f5.com/myf5/) and download your `nginx-repo.crt` and `nginx-repo.key` files.

  - Copy the files to the `/etc/ssl/nginx/` directory:
    ```
    sudo cp nginx-repo.crt nginx-repo.key /etc/ssl/nginx/
    ```

  - Install the prerequisites:
    ```
    sudo apt install curl gnupg2 ca-certificates lsb-release debian-archive-keyring
    ```

  - Add the `nginx-agent` repository:
    ```
    echo "deb https://pkgs.nginx.com/nginx-agent/debian/ `lsb_release -cs` agent" \
      | sudo tee /etc/apt/sources.list.d/nginx-agent.list
    ```

  - Create apt configuration `/etc/apt/apt.conf.d/90pkgs-nginx`:
    ```
    Acquire::https::pkgs.nginx.com::Verify-Peer "true";
    Acquire::https::pkgs.nginx.com::Verify-Host "true";
    Acquire::https::pkgs.nginx.com::SslCert     "/etc/ssl/nginx/nginx-repo.crt";
    Acquire::https::pkgs.nginx.com::SslKey      "/etc/ssl/nginx/nginx-repo.key";
    ```

  - To install `nginx-agent`, run the following commands:
    ```
    sudo apt update
    sudo apt install nginx-agent
    ```

### SLES
  - Create the `/etc/ssl/nginx` directory:
    ```
    sudo mkdir -p /etc/ssl/nginx
    ```

  - Log in to [MyF5 Customer Portal](https://account.f5.com/myf5/) and download your `nginx-repo.crt` and `nginx-repo.key` files.

  - Copy the files to the `/etc/ssl/nginx/` directory:
    ```
    sudo cp nginx-repo.crt nginx-repo.key /etc/ssl/nginx/
    ```

  - Create a file bundle of the certificate and key:
    ```
    cat /etc/ssl/nginx/nginx-repo.crt /etc/ssl/nginx/nginx-repo.key > /etc/ssl/nginx/nginx-repo-bundle.crt
    ```

  - Install the prerequisites:
    ```
    sudo zypper install curl ca-certificates gpg2 gawk
    ```

  - To set up the zypper repository for `nginx-agent` packages, run the following command:
    ```
    sudo zypper addrepo --refresh --check \
        'https://pkgs.nginx.com/nginx-agent/sles/$releasever_major?ssl_clientcert=/etc/ssl/nginx/nginx-repo-bundle.crt&ssl_verify=peer' nginx-agent
    ```
  - Next, import an official nginx signing key so `zypper`/`rpm` could verify the packages authenticity. Fetch the key:
    ```
    curl -o /tmp/nginx_signing.key https://nginx.org/keys/nginx_signing.key
    ```

  - Verify that the downloaded file contains the proper key:
    ```
    gpg --with-fingerprint /tmp/nginx_signing.key
    ```

  - The output should contain the full fingerprint `573B FD6B 3D8F BC64 1079 A6AB ABF5 BD82 7BD9 BF62` as follows:
    ```
    pub  2048R/7BD9BF62 2011-08-19 [expires: 2024-06-14]
          Key fingerprint = 573B FD6B 3D8F BC64 1079  A6AB ABF5 BD82 7BD9 BF62
    uid nginx signing key <signing-key@nginx.com>
    ```

  - Finally, import the key to the rpm database:
    ```
    sudo rpmkeys --import /tmp/nginx_signing.key

  - To install `nginx-agent`, run the following command:
    ```
    sudo zypper install nginx-agent
    ```

### Alpine Linux
  - Log in to [MyF5 Customer Portal](https://account.f5.com/myf5/) and download your `nginx-repo.crt` and `nginx-repo.key` files.

  - Copy the files to the `/etc/apk/` directory:
    ```
    sudo cp nginx-repo.crt nginx-repo.key /etc/apk/
    ```
    <!-- ```
    sudo cp nginx-repo.key /etc/apk/cert.key
    sudo cp nginx-repo.crt /etc/apk/cert.pem
    ``` -->

  - Install the prerequisites:
    ```
    sudo apk add openssl curl ca-certificates
    ```

  - To set up the apk repository for `nginx-agent` packages, run the following command:
    ```
    printf "%s%s%s%s\n" \
        "@nginx-agent " \
        "https://pkgs.nginx.com/nginx-agent/alpine/v" \
        `grep -o -E '^[0-9]+\.[0-9]+' /etc/alpine-release` \
        "/main" \
        | sudo tee -a /etc/apk/repositories
    ```

  - Next, import an official nginx signing key so apk could verify the packages authenticity. Fetch the key:
    ```
    curl -o /tmp/nginx_signing.rsa.pub https://nginx.org/keys/nginx_signing.rsa.pub
    ```

  - Verify that downloaded file contains the proper key:
    ```
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

  - Finally, move the key to apk trusted keys storage:
    ```
    sudo mv /tmp/nginx_signing.rsa.pub /etc/apk/keys/
    ```

  - To install nginx-agent, run the following command:
    ```
    sudo apk add nginx-agent@nginx-agent
    ```

### Amazon Linux
  - Create the `/etc/ssl/nginx` directory:
    ```
    sudo mkdir -p /etc/ssl/nginx
    ```

  - Log in to [MyF5 Customer Portal](https://account.f5.com/myf5/) and download your `nginx-repo.crt` and `nginx-repo.key` files.

  - Copy the `nginx-repo.crt` and `nginx-repo.key` files to the `/etc/ssl/nginx/` directory:
    ```
    sudo cp nginx-repo.crt nginx-repo.key /etc/ssl/nginx/
    ```

  - Install the prerequisites:
    ```
    sudo yum install yum-utils procps ca-certificates
    ```

  - To set up the yum repository for Amazon Linux 2, create the file named `/etc/yum.repos.d/nginx-agent.repo` with the following contents:
    ```
    [nginx-agent]
    name=nginx-agent repo
    baseurl=https://pkgs.nginx.com/nginx-agent/amzn2/$releasever/$basearch
    sslclientcert=/etc/ssl/nginx/nginx-repo.crt
    sslclientkey=/etc/ssl/nginx/nginx-repo.key
    gpgcheck=0
    enabled=1
    ```

  - To install `nginx-agent`, run the following command:
    ```
    sudo yum install nginx-agent
    ```

  - When prompted to accept the GPG key, verify that the fingerprint matches `573B FD6B 3D8F BC64 1079 A6AB ABF5 BD82 7BD9 BF62`, and if so, accept it.
