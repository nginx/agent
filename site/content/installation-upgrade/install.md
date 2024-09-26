---
title: Install NGINX Agent
toc: true
weight: 100
docs: DOCS-000
---

This document describes the three main ways to install F5 NGINX agent:

- Using the NGINX Open Source repository
- Using the NGINX Plus repository
- Using the GitHub package files

## Before you begin

There are a few prerequisites shared between all installation methods: 

- A [supported operating system and architecture](../technical-specifications/#supported-distributions)
- `root` privilege

## NGINX Open Source repository

Before you install NGINX Agent, you must install and run NGINX. 

If you don't have it installed already, read the [Installing NGINX Open Source
](https://docs.nginx.com/nginx/admin-guide/installing-nginx/installing-nginx-open-source/) topic.


### Configure NGINX OSS Repository for installing NGINX Agent

Before you install NGINX Agent for the first time on your system, you need to set up the `nginx-agent` packages repository. Afterward, you can install and update NGINX Agent from the repository.

- [Install NGINX Agent on RHEL, CentOS, Rocky Linux, AlmaLinux, and Oracle Linux](#install-nginx-agent-on-rhel-centos-rocky-linux-almalinux-and-oracle-linux)
- [Install NGINX Agent on Ubuntu](#install-nginx-agent-on-ubuntu)
- [Install NGINX Agent on Debian](#install-nginx-agent-on-debian)
- [Install NGINX Agent on SLES](#install-nginx-agent-on-sles)
- [Install NGINX Agent on Alpine Linux](#install-nginx-agent-on-alpine-linux)
- [Install NGINX Agent on Amazon Linux](#install-nginx-agent-on-amazon-linux)
- [Install NGINX Agent on FreeBSD](#install-nginx-agent-on-freebsd)

#### Install NGINX Agent on RHEL, CentOS, Rocky Linux, AlmaLinux, and Oracle Linux

1. Install the prerequisites:

    ```shell
    sudo yum install yum-utils
    ```

1. To set up the yum repository, create the file named `/etc/yum.repos.d/nginx-agent.repo` with the following contents:

    ```
    [nginx-agent]
    name=nginx agent repo
    baseurl=http://packages.nginx.org/nginx-agent/centos/$releasever/$basearch/
    gpgcheck=1
    enabled=1
    gpgkey=https://nginx.org/keys/nginx_signing.key
    module_hotfixes=true
    ```

1. To install `nginx-agent`, run the following command:

    ```shell
    sudo yum install nginx-agent
    ```

    When prompted to accept the GPG key, verify that the fingerprint matches `573B FD6B 3D8F BC64 1079 A6AB ABF5 BD82 7BD9 BF62`, and if so, accept it.

#### Install NGINX Agent on Ubuntu

1. Install the prerequisites:

    ```shell
    sudo apt install curl gnupg2 ca-certificates lsb-release ubuntu-keyring
    ```

1. Import an official nginx signing key so apt could verify the packages authenticity. Fetch the key:

    ```shell
    curl https://nginx.org/keys/nginx_signing.key | gpg --dearmor \
        | sudo tee /usr/share/keyrings/nginx-archive-keyring.gpg >/dev/null
    ```

1. Verify that the downloaded file contains the proper key:

    ```shell
    gpg --dry-run --quiet --no-keyring --import --import-options import-show /usr/share/keyrings/nginx-archive-keyring.gpg
    ```

    The output should contain the full fingerprint `573BFD6B3D8FBC641079A6ABABF5BD827BD9BF62` as follows:

    ```
    pub   rsa2048 2011-08-19 [SC] [expires: 2024-06-14]
          573BFD6B3D8FBC641079A6ABABF5BD827BD9BF62
    uid                      nginx signing key <signing-key@nginx.com>
    ```

    {{< important >}}If the fingerprint is different, remove the file.{{< /important >}}

1. Add the nginx agent repository:

    ```shell
    echo "deb [signed-by=/usr/share/keyrings/nginx-archive-keyring.gpg] \
      http://packages.nginx.org/nginx-agent/ubuntu/ `lsb_release -cs` agent" \
      | sudo tee /etc/apt/sources.list.d/nginx-agent.list
    ```

1. To install `nginx-agent`, run the following commands:

    ```shell
    sudo apt update
    sudo apt install nginx-agent
    ```

#### Install NGINX Agent on Debian

1. Install the prerequisites:

    ```shell
    sudo apt install curl gnupg2 ca-certificates lsb-release debian-archive-keyring
    ```

1. Import an official nginx signing key so apt could verify the packages authenticity. Fetch the key:

    ```shell
    curl https://nginx.org/keys/nginx_signing.key | gpg --dearmor \
        | sudo tee /usr/share/keyrings/nginx-archive-keyring.gpg >/dev/null
    ```

1. Verify that the downloaded file contains the proper key:

    ```shell
    gpg --dry-run --quiet --no-keyring \
      --import --import-options import-show /usr/share/keyrings/nginx-archive-keyring.gpg
    ```

    The output should contain the full fingerprint `573BFD6B3D8FBC641079A6ABABF5BD827BD9BF62` as follows:

    ```
    pub   rsa2048 2011-08-19 [SC] [expires: 2024-06-14]
          573BFD6B3D8FBC641079A6ABABF5BD827BD9BF62
    uid                      nginx signing key <signing-key@nginx.com>
    ```

    {{< important >}}If the fingerprint is different, remove the file.{{< /important >}}

1. Add the `nginx-agent` repository:

    ```shell
    echo "deb [signed-by=/usr/share/keyrings/nginx-archive-keyring.gpg] \
      http://packages.nginx.org/nginx-agent/debian/ `lsb_release -cs` agent" \ | sudo tee /etc/apt/sources.list.d/nginx-agent.list
    ```

1. To install `nginx-agent`, run the following commands:

    ```shell
    sudo apt update
    sudo apt install nginx-agent
    ```

#### Install NGINX Agent on SLES

1. Install the prerequisites:

    ```shell
    sudo zypper install curl ca-certificates gpg2 gawk
    ```

1. To set up the zypper repository for `nginx-agent` packages, run the following command:

    ```shell
    sudo zypper addrepo --gpgcheck --refresh --check \
        'http://packages.nginx.org/nginx-agent/sles/$releasever_major' nginx-agent
    ```

1. Next, import an official NGINX signing key so `zypper`/`rpm` can verify the package's authenticity. Fetch the key:

    ```shell
    curl -o /tmp/nginx_signing.key https://nginx.org/keys/nginx_signing.key
    ```

1. Verify that the downloaded file contains the proper key:

    ```shell
    gpg --with-fingerprint --dry-run --quiet --no-keyring --import --import-options import-show /tmp/nginx_signing.key
    ```

1. The output should contain the full fingerprint `573B FD6B 3D8F BC64 1079 A6AB ABF5 BD82 7BD9 BF62` as follows:

    ```
    pub   rsa2048 2011-08-19 [SC] [expires: 2024-06-14]
        573B FD6B 3D8F BC64 1079  A6AB ABF5 BD82 7BD9 BF62
    uid                      nginx signing key <signing-key@nginx.com>
    ```

1. Finally, import the key to the rpm database:

    ```shell
    sudo rpmkeys --import /tmp/nginx_signing.key
    ```

1. To install `nginx-agent`, run the following command:

    ```shell
    sudo zypper install nginx-agent
    ```

#### Install NGINX Agent on Alpine Linux

1. Install the prerequisites:

    ```shell
    sudo apk add openssl curl ca-certificates
    ```

1. To set up the apk repository for `nginx-agent` packages, run the following command:

    ```shell
    printf "%s%s%s\n" \
        "http://packages.nginx.org/nginx-agent/alpine/v" \
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

#### Install NGINX Agent on Amazon Linux

1. Install the prerequisites:

    ```shell
    sudo yum install yum-utils procps
    ```

1. To set up the yum repository for Amazon Linux 2, create the file named `/etc/yum.repos.d/nginx-agent.repo` with the following contents:
    ```
    [nginx-agent]
    name=nginx agent repo
    baseurl=http://packages.nginx.org/nginx-agent/amzn2/$releasever/$basearch/
    gpgcheck=1
    enabled=1
    gpgkey=https://nginx.org/keys/nginx_signing.key
    module_hotfixes=true
    ```

1. To install `nginx-agent`, run the following command:

    ```shell
    sudo yum install nginx-agent
    ```

1. When prompted to accept the GPG key, verify that the fingerprint matches `573B FD6B 3D8F BC64 1079 A6AB ABF5 BD82 7BD9 BF62`, and if so, accept it.

#### Install NGINX Agent on FreeBSD

1. To setup the pkg repository create the file named `/etc/pkg/nginx-agent.conf` with the following content:

    ```
    nginx-agent: {
    URL: pkg+http://packages.nginx.org/nginx-agent/freebsd/${ABI}/latest
    ENABLED: true
    MIRROR_TYPE: SRV
    }
    ```

1. To install `nginx-agent`, run the following command:

    ```shell
    sudo pkg install nginx-agent
    ```

## NGINX Plus repository

Before you install NGINX Agent, you must install and run NGINX Plus.

If you don’t have it installed already, read the [Installing NGINX Plus
](https://docs.nginx.com/nginx/admin-guide/installing-nginx/installing-nginx-plus/) topic.

You will also need the following:

- Your credentials to the MyF5 Customer Portal, provided by email from F5, Inc.
- An NGINX Plus subscription (Full or trial)
- Your NGINX Plus certificate and public key (`nginx-repo.crt` and `nginx-repo.key` files), provided by email from F5, Inc.

### Configure NGINX Plus Repository for installing NGINX Agent

Before you install NGINX Agent for the first time on your system, you need to set up the `nginx-agent` packages repository. Afterward, you can install and update NGINX Agent from the repository.

- [Install NGINX Agent on RHEL, CentOS, Rocky Linux, AlmaLinux, and Oracle Linux](#install-nginx-agent-on-rhel-centos-rocky-linux-almalinux-and-oracle-linux)
- [Install NGINX Agent on Ubuntu](#install-nginx-agent-on-ubuntu)
- [Install NGINX Agent on Debian](#install-nginx-agent-on-debian)
- [Install NGINX Agent on SLES](#install-nginx-agent-on-sles)
- [Install NGINX Agent on Alpine Linux](#install-nginx-agent-on-alpine-linux)
- [Install NGINX Agent on Amazon Linux](#install-nginx-agent-on-amazon-linux)
- [Install NGINX Agent on FreeBSD](#install-nginx-agent-on-freebsd)

#### Install NGINX Agent on RHEL, CentOS, Rocky Linux, AlmaLinux, and Oracle Linux

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

    When prompted to accept the GPG key, verify that the fingerprint matches `573B FD6B 3D8F BC64 1079 A6AB ABF5 BD82 7BD9 BF62`, and if so, accept it.

#### Install NGINX Agent on Ubuntu

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

#### Install NGINX Agent on Debian

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

#### Install NGINX Agent on SLES

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

1. The output should contain the full fingerprint `573B FD6B 3D8F BC64 1079 A6AB ABF5 BD82 7BD9 BF62` as follows:

    ```
    pub   rsa2048 2011-08-19 [SC] [expires: 2024-06-14]
        573B FD6B 3D8F BC64 1079  A6AB ABF5 BD82 7BD9 BF62
    uid                      nginx signing key <signing-key@nginx.com>
    ```

1. Finally, import the key to the rpm database:

    ```shell
    sudo rpmkeys --import /tmp/nginx_signing.key
    ```

1. To install `nginx-agent`, run the following command:

    ```shell
    sudo zypper install nginx-agent
    ```

#### Install NGINX Agent on Alpine Linux

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

#### Install NGINX Agent on Amazon Linux

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
    baseurl=https://pkgs.nginx.com/nginx-agent/amzn2/$releasever/$basearch
    sslclientcert=/etc/ssl/nginx/nginx-repo.crt
    sslclientkey=/etc/ssl/nginx/nginx-repo.key
    gpgcheck=0
    enabled=1
    ```

1. To install `nginx-agent`, run the following command:

    ```shell
    sudo yum install nginx-agent
    ```

1. When prompted to accept the GPG key, verify that the fingerprint matches `573B FD6B 3D8F BC64 1079 A6AB ABF5 BD82 7BD9 BF62`, and if so, accept it.

#### Install NGINX Agent on FreeBSD

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

## GitHub package files

To install NGINX Agent on your system, go to the [GitHub releases page](https://github.com/nginx/agent/releases) and download the latest package supported by your operating system distribution and CPU architecture.

Use your system's package manager to install the package. Some examples:

- Debian, Ubuntu, and other distributions using the `dpkg` package manager.

  ```shell
  sudo dpkg -i nginx-agent-<agent-version>.deb
  ```

- RHEL, CentOS RHEL, Amazon Linux, Oracle Linux, and other distributions using the `yum` package manager

  ```shell
  sudo yum localinstall nginx-agent-<agent-version>.rpm
  ```

- RHEL and other distributions using the `rpm` package manager

  ```shell
  sudo rpm -i nginx-agent-<agent-version>.rpm
  ```

- Alpine Linux

  ```shell
  sudo apk add nginx-agent-<agent-version>.apk
  ```

- FreeBSD

  ```shell
  sudo pkg add nginx-agent-<agent-version>.pkg
  ```

## systemd environments

To start NGINX Agent on `systemd` systems, run the following command:

```shell
sudo systemctl start nginx-agent
```

To enable NGINX Agent to start on boot, run the following command:

```shell
sudo systemctl enable nginx-agent
```

## Verify that NGINX Agent is running

Once you have installed NGINX Agent, you can verify that it is running with the following command:

```shell
sudo nginx-agent -v
```

## Enable interfaces

Once NGINX Agent is successfully running, you can enable the required interfaces, which is described in the [Enable gRPC and REST interfaces]({{< relref "/how-to/enable-interfaces.md" >}}) topic.

You may also be interested in the [Start mock control plane interface]({{< relref "/contribute/start-mock-interface.md" >}}) topic for development work.