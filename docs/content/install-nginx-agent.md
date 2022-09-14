---
title: Install and Configure NGINX Agent
description: 'Follow the instructions in this guide to install and configure the NGINX Agent on your data plane systems.'
categories:
- installation
date: "2021-12-21T12:00:00-07:00"
doctypes:
- tutorial
draft: false
journeys:
- getting started
- using
personas:
- devops
- netops
- secops
- support
tags:
- docs
toc: true
versions: []
weight: 100
docs: "DOCS-800"
aliases:
- /getting-started/installation/install-nginx-agent/
---

<style>
    h2 {
        margin-top: 30px;
        margin-bottom: 10px;
    }
    h3 {
        margin-top: 30px;
        margin-bottom: 10px;
        font-weight: 300;
        font-size: 1.75em;
    }
    hr {
        margin-top: 40px;
        margin-bottom: 20px;
    }
</style>

## Prerequisites

This section lists the prerequisites for installing and configuring NGINX Agent. Follow the steps below to complete the requirements:

1. [Instance Manager is installed and configured on a server](http://docs.nginx.com/nginx-management-suite/admin-guides/getting-started/prerequisites/).

    {{<note>}} When installing and configuring Instance Manager, take note of the fully qualified domain name (FQDN) and gRPC port number. You'll need this information to properly configure the NGINX Agent to communicate with Instance Manager.
    {{</note>}}

2. Make sure NGINX is running on your instance:

    ```bash
    ps aux | grep nginx
    ```

3. If a previous version of NGINX Agent was installed, you must stop the current NGINX Agent process before running the NGINX Agent install script. To check if any NGINX Agent processes are running, run the following command:

    ```bash
    ps aux | grep nginx-agent
    ```

4. If a previous version of NGINX Agent was installed, make sure to uninstall `nginx-agent-selinux` before running the NGINX Agent install script.
To see if `nginx_agent_selinux` is installed, run the following command:

    {{<tabs name="install_repo">}}
    {{%tab name="CentOS, RHEL, and RPM-Based"%}}

  ```bash
  rpm -qa | grep nginx_agent_selinux
  ```

   {{%/tab%}}
   {{%tab name="Debian, Ubuntu, and Deb-Based"%}}

  ```bash
  dpkg -s nginx_agent_selinux
  ```

    {{%/tab%}}
    {{</tabs>}}

5. Review the [NGINX Management  Suite Technical Specifications](https://docs.nginx.com/nginx-management-suite/tech-specs/) guide for system requirements.

---

## Install NGINX Agent

You can choose one of the following two methods to install the NGINX Agent on your data plane host:

- Install via the Instance Manager API Gateway
- Install from packages downloaded from [MyF5 Customer Portal](https://account.f5.com/myf5) or from your NGINX/F5 sales team.

### Install using the API

You can install the NGINX Agent using `curl`, `wget`, or any command-line tool for transferring data with URLs. If the NGINX Management Suite is not set up with valid TLS certificates, you can use the available insecure flags of those tools. See the following examples:

{{<tabs name="install-agent-api">}}

{{%tab name="curl"%}}

- Secure:

  ```bash
  curl https://<NMS-FQDN>/install/nginx-agent | sudo sh -s -- --skip-verify false
  ```

- Insecure:

  ```bash
  curl -k https://<NMS-FQDN>/install/nginx-agent | sudo sh
  ```

  When installing the NGINX Agent with the install script, you can optionally set the `instance-group` with the following flag:

  - `--instance-group`

  The following example shows how to download and run the script with the optional flag:

  ```bash
  curl https://<NMS-FQDN>/install/nginx-agent > install.sh
  sudo sh ./install.sh --instance-group my-instance-group
  ```

  By default, the install script attempts to use a secure connection when downloading packages. If, however, the script cannot create a secure connection, it uses an insecure connection instead and logs the following warning message:

  ``` text
  Warning: An insecure connection will be used during this nginx-agent installation
  ```

  To require a secure connection, you can set the optional flag `skip-verify` to `false`.

  The following xample shows how to download and run the script with an enforced secure connection:

  ```bash
  curl https://<NMS-FQDN>/install/nginx-agent > install.sh
  sudo sh ./install.sh --skip-verify false
  ```

{{%/tab%}}

{{%tab name="wget"%}}

- Secure:

  ```bash
  wget https://<NMS-FQDN>/install/nginx-agent -O - | sudo sh -s -- --skip-verify false
  ```

- Insecure:

  ```bash
  wget --no-check-certificate https://<NMS-FQDN>/install/nginx-agent -O - | sudo sh
  ```

{{%/tab%}}
{{</tabs>}}

### Install from Package Files

To install directly from package files, you'll need to download the files from the [MyF5 Customer Portal](https://account.f5.com/myf5) or use the package files provided by your NGINX Sales Team. The file you need is `agent-install.sh`.

To install from package files, take the following steps:

1. Copy `agent-install.sh` to the data plane host where you want to install the NGINX Agent.
2. Run the following command to install OR update NGINX Agent:

    ```bash
    sudo PACKAGE_HOST=<NMS-FQDN> ./agent-install.sh 
    ```

   {{<note>}}
   In the example above, `<NMS-FQDN>` is the the fully qualified domain name or IP address for the Instance Manager API gateway.
   {{</note>}}

---

## Start and Enable NGINX Agent

To start the NGINX Agent on systemd systems, run the following command:

```bash
sudo systemctl start nginx-agent
```

To enable the NGINX Agent to start on boot, run the following command:

```bash
sudo systemctl enable nginx-agent
```

---

## Verifying NGINX Agent is Running and Registered

Run the following command on your data plane to verify that the NGINX Agent process is running:

```bash
ps aux | grep nginx-agent
```

You should see output that looks similar to the following example:

```text
root      293850  109  1.1 1240056 23536 ?       Ssl  22:00   0:07 /usr/local/bin/nginx-agent
vagrant   293866  0.0  0.0   8160   736 pts/0    S+   22:00   0:00 grep --color=auto nginx-agent
```

Once you've verified the NGINX Agent is running on your data plane, you should confirm it's registered with Instance Manager. You can do this two ways:

{{<tabs name="verify-nginx">}}

{{%tab name="API"%}}

Send an API request similar to the following example to get the inventory list. Your instance should be listed.

  ```bash
  curl -u <user>:<password> https://<NMS-FQDN>/api/platform/v1/systems | jq
  ```

{{%/tab%}}

{{%tab name="WEBUI"%}}

Open the NGINX Management Suite web interface and log in. The registered instance is shown in the **Instances** list.

  {{< img src="/getting-started/install/registered-instance.png" alt="Registered instances" >}}

{{%/tab%}}

{{</tabs>}}

<br>

Once you've verified the NGINX Agent instance is registered with Instance Manager, no additional action is required for monitoring the instance.

{{<note>}}
If you need to remove the instance, ensure that the NGINX Agent service is stopped first. Then you can remove the instance from the inventory.
{{</note>}}

---

## Configuring the NGINX Agent

The following sections explain how to configure the NGINX Agent using configuration files, CLI flags, and environment variables.

{{<note>}}

- The NGINX Agent interprets configuration values set by configuration files, CLI flags, and environment variables in the following priorities:

  1. CLI flags overwrite configuration files and environment variable values.
  2. Environment variables overwrite configuration file values.
  3. Config files are the lowest priority and config settings are superseded if either of the other options is used.

- The NGINX Agent is configured by default to connect to the NGINX Management Suite on port 443 based on the address used to download the install script. If this setting doesn't work, you can change the `server` fields in the `nginx-agent.conf` file. Instructions are provided in the following sections.

- Open any required firewall ports or SELinux/AppArmor rules for the ports and IPs you want to use.

{{</note>}}

### Configure with Config Files

The configuration files for the NGINX Agent are `/etc/nginx-agent/nginx-agent.conf` and `/etc/nginx-agent/agent-dynamic.conf`. These files have comments at the top indicating their purpose.

Examples of the configuration files are provided below:

<details open>
    <summary>example nginx-agent.conf</summary>

{{<note>}}
In the following example `nginx-agent.conf` file, you can change the `server.host` and `server.grpcPort` to connect to the NGINX Management Suite.
{{</note>}}

```nginx {hl_lines=[13]}
#
# /etc/nginx-agent/nginx-agent.conf
#
# Configuration file for NGINX Agent.
#
# This file tracks agent configuration values that are meant to be statically set. There  
# are additional agent configuration values that are set via the API and agent install script
# which can be found in /etc/nginx-agent/agent-dynamic.conf. 

# specify the server grpc port to connect to
server:
  # host of the control plane
  host: <NMS-FQDN>
  grpcPort: 443
# tls options
tls:
  # enable tls in the nginx-agent setup for grpcs
  # default to enable to connect with secure connection but without client cert for mtls
  enable: true
  # controls whether the server certificate chain and host name are verified.
  # for production use, see instructions for configuring TLS
  skip_verify: false
log:
  # set log level (panic, fatal, error, info, debug, trace; default "info")
  level: info
  # set log path. if empty, don't log to file.
  path: /var/log/nginx-agent/
nginx:
  # path of NGINX logs to exclude
  exclude_logs: ""
# data plane status message / 'heartbeat'
dataplane:
  status:
    # poll interval for dataplane status - the frequency the agent will query the dataplane for changes
    poll_interval: 30s
    # report interval for dataplane status - the maximum duration to wait before syncing dataplane information if no updates have been observed
    report_interval: 24h
metrics:
  # specify the size of a buffer to build before sending metrics
  bulk_size: 20
  # specify metrics poll interval
  report_interval: 1m
  collection_interval: 15s
  mode: aggregated

# OSS NGINX default config path
# path to aux file dirs can also be added
config_dirs: "/etc/nginx:/usr/local/etc/nginx"

# Enable reporting NGINX App Protect details to the control plane.
nginx_app_protect:
  # Report interval for NGINX App Protect details - the frequency the NGINX Agent checks NGINX App Protect for changes.
  report_interval: 15s
```

</details>


<details open>
    <summary>example dynamic-agent.conf</summary>

```nginx
#
# /etc/nginx-agent/dynamic-agent.conf
#
# Dynamic configuration file for NGINX Agent.
#
# The purpose of this file is to track agent configuration
# values that can be dynamically changed via the API and the agent install script.
# You may edit this file, but API calls that modify the tags on this system will
# overwrite the tag values in this file.
#
# The agent configuration values that API calls can modify are as follows:
#    - tags
#
# The agent configuration value that the agent install script can modify are as follows:
#    - instance_group

instance_group: devenv-group 
tags:
  - devenv
  - test
```

</details>


### NGINX Agent CLI Flags & Usage

This section displays the configurable options for the NGINX Agent that can be set with CLI flags. See the CLI flags and their uses in the figure below:

<details open>
  <summary>NGINX Agent CLI flags & usage</summary>

```text
Usage:
  nginx-agent [flags]
  nginx-agent [command]

Available Commands:
  completion  Generate completion script.
  help        Help about any command

Flags:
      --advanced-metrics-aggregation-period duration                      Sets the interval, in seconds, at which advanced metrics are collected. (default 10s)
      --advanced-metrics-publishing-period duration                       The polling period specified for a single set of advanced metrics being collected. (default 30s)
      --advanced-metrics-socket-path string                               The advanced metrics socket location (default "/tmp/acm.sock")
      --advanced-metrics-table-sizes-limits-priority-table-max-size int   Default Maximum Size of the Priority Table. (default 1000)
      --advanced-metrics-table-sizes-limits-priority-table-threshold int  Default Threshold of the Priority Table - normally a value which is a percentage of the corresponding Default Maximum Size of the Priority Table (<100%, but its value is not an actual percentage, i.e 88%, rather 88%*AdvancedMetricsTableSizesLimitsPTMS). (default 1000}
      -advanced-metrics-table-sizes-limits--staging-table-max-size int    Default Maximum Size of the Staging Table. (default 1000)
      --advanced-metrics-table-sizes-limits-staging-table-threshold int   AdvancedMetricsTableSizesLimitsSTT - Default Threshold of the Staging Table - normally a value which is a percentage of the corresponding Default Maximum Size of the Staging Table (<100%, but its value is not an actual percentage, i.e 88%, rather 88%+AdvancedMetricsTableSizesLimitsSTMS). (default 1000)
      --config-dirs string	                   Defines the paths that you want to grant nginx-agent read/write access to. This key is formatted as a string and follows Unix PATH format. (default "/etc/nginx:/usr/local/etc/nginx:/usr/share/nginx/modules:/etc/nms")
      --dataplane-report-interval duration     The amount of time the agent will report on the dataplane. After this period of time it will send a snapshot of the dataplane information. (default 24h0m0s)}
      --dataplane-status-poll-interval duration        The frequnecy the agent will check the dataplane for changes. Used as a "heartbeat" to keep the grpc connections alive. (default 30s)
      --display-name string	                   The instance's 'group' value.
      --features strings                       A comma-separated list of features enabled for the agent.
  -h, --help                                   help for nginx-agent
      --instance-group string                  The instance's 'group' value.
      --log-level string                       The desired verbosity level for logging messages from nginx-agent. Available options, in order of severity from highest to lowest, are: panic, fatal, error, info, debug, and trace. (default "info")
      --log-path string                        The path to output log messages to. If the default path doesn't exist, log messages are output to stdout/stderr. (default "/var/log/nginx-agent")
      --metrics-bulk-size int                  The amount of metrics reports collected before sending the data back to the server. (default 20)
      --metrics-collection-interval duration           Sets the interval, in seconds, at which metrics are collected. (default 15s)
      --metrics-mode string                    Sets the desired metrics collection mode: streaming or aggregation. (default "aggregation")
      --metrics-report-interval duration       The polling period specified for a single set of metrics being collected. (default 1m0s)
      -nginx-app-protect-report-interval duration      The period of time the agent will check for App Protect software changes on the dataplane
      --nginx-exclude-logs string              One or more NGINX access log paths that you want to exclude from metrics collection. This key is formatted as a string and multiple values should be provided as a comma-separated list.
      --nginx-socket string                    The NGINX plus counting unix socket location. (default "unix:/var/run/nginx-agent/nginx.sock")
      --server-command string                  The name of the command server sent in the tls configuration.
      --server-grpcport int                    The desired GRPC port to use for nginx-agent traffic. (default 443)
      --server-host string                     The IP address of the server host. IPv4 addresses and hostnames are supported. (default "127.0.0.1")
      --server-metrics string                  The name of the metrics server sent in the tls configuration.
      --server-token string                    An authentication token that grants nginx-agent access to the commander and metrics services. Auto-generated by default. (default "658d6d1a-c868-487b-8be4-96b36b3a536")
      --tags strings                           A comma-separated list of tags to add to the current instance or machine, to be used for inventory purposes.
      --tls-ca string                          The path to the CA certificate file to use for TLS.
      --tls-cert string                        The path to the certificate file to use for TLS.
      --tls-enable                             Enables TLS for secure communications.
      --tls-key string                         The path to the certificate key file to use for TLS.
      --tls-skip-verify                        For demo purposes only not recommended for production, sets InsecureSkipVerify for grpc tls credentials.

  -v, --version                                version for nginx-agent

Use "nginx-agent [command] --help" for more information about a command.
```

{{< note >}}
The following commands were deprecated In Instance Manager v2.1:

- `--instance-name`
- `--location`

{{< /note >}}

</details>

### NGINX Agent Environment Variables

This section displays the configurable options for the NGINX Agent that can be set with environment variables. A list of the configurable environment variables can be seen below:

<details open>
  <summary>NGINX Agent Environment Variables</summary>

```text
- NMS_INSTANCE_GROUP
- NMS_DISPLAY_NAME
- NMS_FEATURES
- NMS_LOG_LEVEL
- NMS_LOG_PATH
- NMS_PATH
- NMS_METRICS_COLLECTION_INTERVAL
- NMS_METRICS_MODE
- NMS_METRICS_BULK_SIZE
- NMS_METRICS_REPORT_INTERVAL
- NMS_NGINX_EXCLUDE_LOGS
- NMS_NGINX_SOCKET
- NMS_SERVER_GRPCPORT
- NMS_SERVER_HOST
- NMS_SERVER_TOKEN
- NMS_SERVER_COMMAND
- NMS_SERVER_METRICS
- NMS_TAGS
- NMS_TLS_CA
- NMS_TLS_CERT
- NMS_TLS_ENABLE
- NMS_TLS_KEY
- NMS_TLS_SKIP_VERIFY
- NMS_CONFIG_DIRS
- NMS_DATAPLANE_REPORT_INTERVAL
- NMS_DATAPLANE_STATUS_POLL_INTERVAL
- NMS_NGINX_APP_PROTECT_REPORT_INTERVAL
- NMS_ADVANCED_METRICS_AGGREGATION_PERIOD
- NMS_ADVANCED_METRICS_PUBLISHING-PERIOD
- NMS_ADVANCED_METRICS_SOCKET_PATH
- NMS_ADVANCED_METRICS_TABLE_SIZES_LIMITS_PRIORITY_TABLE_MAX_SIZE
- NMS_ADVANCED_METRICS_TABLE_SIZES_LIMITS_PRIORITY_TABLE_THRESHOLD
- NMS_ADVANCED_METRICS_TABLE_SIZES_LIMITS_STAGING_TABLE_MAX_SIZE
- NMS_ADVANCED_METRICS_TABLE_SIZES_LIMITS_STAGING_TABLE_THRESHOLD
```

</details>

### Configure NGINX Agent to enable NGINX App Protect WAF Status reporting

You can configure NGINX Agent to report the following NGINX App Protect WAF installation information to NGINX Management Suite:

- the current version of NGINX App Protect WAF
- the current status of NGINX App Protect WAF (active or inactive)
- the [Attack Signatures package](https://docs.nginx.com/nginx-app-protect/admin-guide/install/#attack-signatures-package) version 
- the Threat Campaigns package version

To enable NGINX App Protect WAF reporting, edit the `/etc/nginx-agent/nginx-agent.conf` to add the following directives:

```text
# Enable reporting NGINX App Protect details to the control plane.
nginx_app_protect:
  # Report interval for NGINX App Protect details - the frequency the NGINX Agent checks NGINX App Protect for changes.
  report_interval: 15s
```

---

## SELinux for NGINX Agent

This section explains how to install and configure the SELinux policy for the NGINX Agent.

### Installing NGINX Agent SELinux Policy Module

The NGINX Agent package includes the following SELinux files:

- `/usr/share/man/man8/nginx_agent_selinux.8.gz`
- `/usr/share/selinux/devel/include/contrib/nginx_agent.if`
- `/usr/share/selinux/packages/nginx_agent.pp`

To install and load the policy, run the following commands:

```bash
sudo semodule -n -i /usr/share/selinux/packages/nginx_agent.pp
sudo /usr/sbin/load_policy
sudo restorecon -R /usr/bin/nginx-agent;
sudo restorecon -R /var/log/nginx-agent;
sudo restorecon -R /etc/nginx-agent;
```

### Adding Ports for NGINX Agent SELinux Context

You can modify the NGINX Agent to comply with SELinux. You should add external ports to the firewall exception.

The following example shows how to allow external ports outside the HTTPD context. You may need to enable NGINX to connect to these ports.

```bash
sudo setsebool -P httpd_can_network_connect 1
```

For additional information on using NGINX with SELinux, refer to the guide [Using NGINX and NGINX Plus with SELinux](https://www.nginx.com/blog/using-nginx-plus-with-selinux/).

---

## Secure the NGINX Agent with mTLS

{{< important >}}By default, communication between the NGINX Agent and Instance Manager is unsecured.{{< /important >}}

For instructions on how configure mTLS to secure communication between the NGINX Agent and Instance Manager, see [NGINX Agent TLS Settings]({{< relref "/encrypt-nginx-agent-comms.md" >}}).

---

## NGINX Metrics

After you register an NGINX instance with Instance Manager, the NGINX Agent will collect and report metrics. For more information about the metrics that are reported, see [Overview: Instance Metrics](https://docs.nginx.com/nginx-management-suite/nim/about/overview-metrics/).
