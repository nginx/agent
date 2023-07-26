---
title: "Configuration Overview"
draft: false
description: "Learn the overview of configuration"
weight: 500
toc: true
tags: [ "docs" ]
docs: "DOCS-1229"
categories: ["configuration"]
doctypes: ["task"]
---

## Overview

The following sections explain how to configure the NGINX Agent using configuration files, CLI flags, and environment variables.

{{<note>}}

- The NGINX Agent interprets configuration values set by configuration files, CLI flags, and environment variables in the following priorities:

  1. CLI flags overwrite configuration files and environment variable values.
  2. Environment variables overwrite configuration file values.
  3. Config files are the lowest priority and config settings are superseded if either of the other options is used.

- You must open any required firewall ports or add SELinux/AppArmor rules for the ports and IPs you want to use.

{{</note>}}

### Configure with Config Files

The configuration files for the NGINX Agent are `/etc/nginx-agent/nginx-agent.conf` and `/var/lib/nginx-agent/agent-dynamic.conf`. The `agent-dynamic.conf` file location is different for FreeBSD which is located `/var/db/nginx-agent/agent-dynamic.conf`. These files have comments at the top indicating their purpose.

Examples of the configuration files are provided below:

<details open>
    <summary>example nginx-agent.conf</summary>

{{<note>}}
In the following example `nginx-agent.conf` file, you can change the `server.host` and `server.grpcPort` to connect to the control/management plane.
{{</note>}}

```nginx {hl_lines=[13]}
#
# /etc/nginx-agent/nginx-agent.conf
#
# Configuration file for NGINX Agent.
#
# This file tracks agent configuration values that are meant to be statically set. There  
# are additional NGINX Agent configuration values that are set via the API and agent install script
# which can be found in /etc/nginx-agent/agent-dynamic.conf. 

# specify the server grpc port to connect to
server:
  # host of the control plane
  host: <FQDN>
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
  # Set to true when NGINX configuration should contain no warnings when performing a configuration apply (nginx -t is used to carry out this check)
  treat_warnings_as_errors: false # Default is false
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

extensions:
  - nginx-app-protect

# Enable reporting NGINX App Protect details to the control plane.
nginx_app_protect:
  # Report interval for NGINX App Protect details - the frequency the NGINX Agent checks NGINX App Protect for changes.
  report_interval: 15s
  # Enable precompiled publication from the NGINX Management Suite (true) or perform compilation on the data plane host (false).
  precompiled_publication: true
```

</details>


<details open>
    <summary>example dynamic-agent.conf</summary>

```yaml
#
# /var/lib/nginx-agent/agent-dynamic.conf
# On FreeBSD /var/db/nginx-agent/agent-dynamic.conf
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


### NGINX Agent CLI Flags & Usage {#nginx-agent-cli-flags-usage}

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
      --api-cert string                                  The cert used by the Agent API.
      --api-host string                                  The host used by the Agent API. (default "127.0.0.1")
      --api-key string                                   The key used by the Agent API.
      --api-port int                                     The desired port to use for nginx-agent to expose for HTTP traffic.
      --config-dirs string                               Defines the paths that you want to grant nginx-agent read/write access to. This key is formatted as a string and follows Unix PATH format. (default "/etc/nginx:/usr/local/etc/nginx:/usr/share/nginx/modules:/etc/nms")
      --dataplane-report-interval duration               The amount of time the agent will report on the dataplane. After this period of time it will send a snapshot of the dataplane information. (default 24h0m0s)
      --dataplane-status-poll-interval duration          The frequency the agent will check the dataplane for changes. Used as a "heartbeat" to keep the gRPC connections alive. (default 30s)
      --display-name string                              The instance's 'name' value.
      --features strings                                 A comma-separated list of features enabled for the agent. (default [registration,nginx-config-async,nginx-ssl-config,nginx-counting,metrics,metrics-throttle,dataplane-status,process-watcher,file-watcher,activity-events,agent-api])
  -h, --help                                             help for nginx-agent
      --instance-group string                            The instance's 'group' value.
      --log-level string                                 The desired verbosity level for logging messages from nginx-agent. Available options, in order of severity from highest to lowest, are: panic, fatal, error, info, debug, and trace. (default "info")
      --log-path string                                  The path to output log messages to. If the default path doesn't exist, log messages are output to stdout/stderr. (default "/var/log/nginx-agent")
      --metrics-bulk-size int                            The amount of metrics reports collected before sending the data back to the server. (default 20)
      --metrics-collection-interval duration             Sets the interval, in seconds, at which metrics are collected. (default 15s)
      --metrics-mode string                              Sets the desired metrics collection mode: streaming or aggregation. (default "aggregated")
      --metrics-report-interval duration                 The polling period specified for a single set of metrics being collected. (default 1m0s)
      --nginx-config-reload-monitoring-period duration   The duration the NGINX Agent will monitor error logs after a NGINX reload (default 10s)
      --nginx-exclude-logs string                        One or more NGINX access log paths that you want to exclude from metrics collection. This key is formatted as a string and multiple values should be provided as a comma-separated list.
      --nginx-socket string                              The NGINX Plus counting unix socket location. (default "unix:/var/run/nginx-agent/nginx.sock")
      --nginx-treat-warnings-as-errors                   On nginx -t, treat warnings as failures on configuration application.
      --server-command string                            The name of the command server sent in the tls configuration.
      --server-grpcport int                              The desired GRPC port to use for nginx-agent traffic.
      --server-host string                               The IP address of the server host. IPv4 addresses and hostnames are supported.
      --server-metrics string                            The name of the metrics server sent in the tls configuration.
      --server-token string                              An authentication token that grants nginx-agent access to the commander and metrics services. Auto-generated by default. (default "750d0148-c4b2-499a-9011-ca5a8c752d52")
      --tags strings                                     A comma-separated list of tags to add to the current instance or machine, to be used for inventory purposes.
      --tls-ca string                                    The path to the CA certificate file to use for TLS.
      --tls-cert string                                  The path to the certificate file to use for TLS.
      --tls-enable                                       Enables TLS for secure communications.
      --tls-key string                                   The path to the certificate key file to use for TLS.
      --tls-skip-verify                                  Only intended for demonstration, sets InsecureSkipVerify for gRPC TLS credentials
  -v, --version                                          version for nginx-agent

Use "nginx-agent [command] --help" for more information about a command.
```

</details>

#### NGINX Agent Config Dirs Option

Use the `--config-dirs` command-line option, or the `config_dirs` key in the `nginx-agent.conf` file, to identify the directories the NGINX Agent can read from or write to. This setting also defines the location to which you can upload config files when using a control/management plane. The NGINX Agent cannot write to directories outside the specified location when updating a config and cannot upload files to directories outside of the configured location.
The NGINX Agent follows NGINX configuration directives to file paths outside the designated directories and reads certificates' metadata. The NGINX Agent uses the following directives:

- [`ssl_certificate`](https://nginx.org/en/docs/http/ngx_http_ssl_module.html#ssl_certificate)

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
- NMS_NGINX_TREAT_WARNINGS_AS_ERRORS
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
```
</details>

### NGINX Agent Log Rotation

Logs are rotated daily by default in NGINX Agent using logrotate, with the following configuration: 

<details open>
  <summary>NGINX Agent Logrotate Configuration</summary>

``` yaml
/var/log/nginx-agent/*.log
{
   # log files are rotated every day
   daily
   # log files are rotated if they grow bigger than 5M
   size 5M
   # truncate the original log file after creating a copy
   copytruncate
   # remove rotated logs older than 10 days
   maxage 10
   # log files are rotated 10 times before being removed
   rotate 10
   # old log files are compressed
   compress
   # if the log file is missing it will go on to the next one without issuing an error message
   missingok 
   # do not rotate the log if it is empty
   notifempty
}
```
</details>

If you need to make changes to the default configuration you can update the file here `/etc/logrotate.d/nginx-agent`

For more detail on logrotate configuration see [Logrotate Configuration Options](https://linux.die.net/man/8/logrotate)