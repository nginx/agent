---
title: "How to configure NGINX Agent"
draft: false
weight: 100
toc: true
tags: [ "docs" ]
docs: "DOCS-1229"
categories: ["configuration"]
doctypes: ["task"]
---

The following sections explain how to configure NGINX Agent using configuration files, CLI flags, and environment variables.

{{<note>}}

- NGINX Agent interprets configuration values set by configuration files, CLI flags, and environment variables in the following priorities:

  1. CLI flags overwrite configuration files and environment variable values.
  2. Environment variables overwrite configuration file values.
  3. Config files are the lowest priority and config settings are superseded if either of the other options is used.

- You must open any required firewall ports or add SELinux/AppArmor rules for the ports and IPs you want to use.

{{</note>}}

## Configure with Config Files

The default locations of configuration files for NGINX Agent are `/etc/nginx-agent/nginx-agent.conf` and `/var/lib/nginx-agent/agent-dynamic.conf`. The `agent-dynamic.conf` file default location is different for FreeBSD which is located `/var/db/nginx-agent/agent-dynamic.conf`. These files have comments at the top indicating their purpose.

Examples of the configuration files are provided below:

<details open>
    <summary>example nginx-agent.conf</summary>

{{<note>}}
In the following example `nginx-agent.conf` file, you can change the `server.host` and `server.grpcPort` to connect to the control plane.
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
    # poll interval for dataplane status - the frequency the NGINX Agent will query the dataplane for changes
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

# Internal queue size
queue_size: 100

extensions:
  - nginx-app-protect

# Enable reporting NGINX App Protect details to the control plane.
nginx_app_protect:
  # Report interval for NGINX App Protect details - the frequency NGINX Agent checks NGINX App Protect for changes.
  report_interval: 15s
  # Enable precompiled publication from the NGINX Management Suite (true) or perform compilation on the data plane host (false).
  precompiled_publication: true
```

</details>


<details open>
    <summary>example dynamic-agent.conf</summary>

{{<note>}}
Default location in Linux environments: `/var/lib/nginx-agent/agent-dynamic.conf`

Default location in FreeBSD environments: `/var/db/nginx-agent/agent-dynamic.conf`
{{</note>}}

```yaml
# Dynamic configuration file for NGINX Agent.
#
# The purpose of this file is to track agent configuration
# values that can be dynamically changed via the API and the agent install script.
# You may edit this file, but API calls that modify the tags on this system will
# overwrite the tag values in this file.
#
# The agent configuration values that API calls can modify are as follows:
# tags:
#   - dev
#   - qa
#
# The agent configuration value that the agent install script can modify are as follows:
# instance_group: my-instance-group

instance_group: my-instance-group
tags:
  - dev
  - qa
```

</details>

## CLI Flags & Environment Variables

This section details the CLI flags and corresponding environment variables used to configure the NGINX Agent.

### Usage

#### CLI Flags

```sh
nginx-agent [flags]
```

#### Environment Variables

```sh
export ENV_VARIABLE_NAME="value"
nginx-agent
```

### CLI Flags and Environment Variables

{{< warning >}} 

Before version 2.35.0, the environment variables were prefixed with `NMS_` instead of `NGINX_AGENT_`. 

If you are upgrading from an older version, update your configuration accordingly.

{{< /warning >}}

{{<bootstrap-table "table table-responsive table-bordered">}}
| CLI flag                                    | Environment variable                 | Description                                                                 |
|---------------------------------------------|--------------------------------------|-----------------------------------------------------------------------------|
| `--api-cert`                                | `NGINX_AGENT_API_CERT`                       | Specifies the certificate used by the Agent API.                            |
| `--api-host`                                | `NGINX_AGENT_API_HOST`                       | Sets the host used by the Agent API. Default: *127.0.0.1*                   |
| `--api-key`                                 | `NGINX_AGENT_API_KEY`                        | Specifies the key used by the Agent API.                                    |
| `--api-port`                                | `NGINX_AGENT_API_PORT`                       | Sets the port for exposing nginx-agent to HTTP traffic.                     |
| `--config-dirs`                             | `NGINX_AGENT_CONFIG_DIRS`                    | Defines directories NGINX Agent can read/write. Default: *"/etc/nginx:/usr/local/etc/nginx:/usr/share/nginx/modules:/etc/nms"* |
| `--dataplane-report-interval`               | `NGINX_AGENT_DATAPLANE_REPORT_INTERVAL`      | Sets the interval for dataplane reporting. Default: *24h0m0s*               |
| `--dataplane-status-poll-interval`          | `NGINX_AGENT_DATAPLANE_STATUS_POLL_INTERVAL` | Sets the interval for polling dataplane status. Default: *30s*              |
| `--display-name`                            | `NGINX_AGENT_DISPLAY_NAME`                   | Sets the instance's display name.                                           |
| `--dynamic-config-path`                     | `NGINX_AGENT_DYNAMIC_CONFIG_PATH`            | Specifies the path of the Agent dynamic config file. Default: *"/var/lib/nginx-agent/agent-dynamic.conf"* |
| `--features`                                | `NGINX_AGENT_FEATURES`                       | Specifies a comma-separated list of features enabled for the agent. Default: *[registration, nginx-config-async, nginx-ssl-config, nginx-counting, metrics, dataplane-status, process-watcher, file-watcher, activity-events, agent-api]* |
| `--ignore-directives`                       |                                      | Specifies a comma-separated list of directives to ignore for sensitive info.|
| `--instance-group`                          | `NGINX_AGENT_INSTANCE_GROUP`                 | Sets the instance's group value.                                            |
| `--log-level`                               | `NGINX_AGENT_LOG_LEVEL`                      | Sets the logging level (e.g., panic, fatal, error, info, debug, trace). Default: *info* |
| `--log-path`                                | `NGINX_AGENT_LOG_PATH`                       | Specifies the path to output log messages.                                  |
| `--metrics-bulk-size`                       | `NGINX_AGENT_METRICS_BULK_SIZE`              | Specifies the number of metrics reports collected before sending data. Default: *20* |
| `--metrics-collection-interval`             | `NGINX_AGENT_METRICS_COLLECTION_INTERVAL`    | Sets the interval for metrics collection. Default: *15s*                    |
| `--metrics-mode`                            | `NGINX_AGENT_METRICS_MODE`                   | Sets the metrics collection mode: streaming or aggregation. Default: *aggregated* |
| `--metrics-report-interval`                 | `NGINX_AGENT_METRICS_REPORT_INTERVAL`        | Sets the interval for reporting collected metrics. Default: *1m0s*          |
| `--nginx-config-reload-monitoring-period`   |                                      | Sets the duration to monitor error logs after an NGINX reload. Default: *10s* |
| `--nginx-exclude-logs`                      | `NGINX_AGENT_NGINX_EXCLUDE_LOGS`             | Specifies paths of NGINX access logs to exclude from metrics collection.    |
| `--nginx-socket`                            | `NGINX_AGENT_NGINX_SOCKET`                   | Specifies the location of the NGINX Plus counting Unix socket. Default: *unix:/var/run/nginx-agent/nginx.sock* |
| `--nginx-treat-warnings-as-errors`          | `NGINX_AGENT_NGINX_TREAT_WARNINGS_AS_ERRORS` | Treats warnings as failures on configuration application.                   |
| `--queue-size`                              | `NGINX_AGENT_QUEUE_SIZE`                     | Specifies the size of the NGINX Agent internal queue.                       |
| `--server-command`                          |                                      | Specifies the name of the command server sent in the TLS configuration.     |
| `--server-grpcport`                         | `NGINX_AGENT_SERVER_GRPCPORT`                | Sets the desired GRPC port for NGINX Agent traffic.                         |
| `--server-host`                             | `NGINX_AGENT_SERVER_HOST`                    | Specifies the IP address of the server host.                                |
| `--server-metrics`                          |                                      | Specifies the name of the metrics server sent in the TLS configuration.     |
| `--server-token`                            | `NGINX_AGENT_SERVER_TOKEN`                   | Sets the authentication token for accessing the commander and metrics services. Default: *e202f883-54c6-4702-be15-3ba6e507879a* |
| `--tags`                                    | `NGINX_AGENT_TAGS`                           | Specifies a comma-separated list of tags for the instance or machine.       |
| `--tls-ca`                                  | `NGINX_AGENT_TLS_CA`                         | Specifies the path to the CA certificate file for TLS.                      |
| `--tls-cert`                                | `NGINX_AGENT_TLS_CERT`                       | Specifies the path to the certificate file for TLS.                         |
| `--tls-enable`                              | `NGINX_AGENT_TLS_ENABLE`                     | Enables TLS for secure communications.                                      |
| `--tls-key`                                 | `NGINX_AGENT_TLS_KEY`                        | Specifies the path to the certificate key file for TLS.                     |
| `--tls-skip-verify`                         | `NGINX_AGENT_TLS_SKIP_VERIFY`                | Insecurely skips verification for gRPC TLS credentials.                     |
{{</bootstrap-table>}}

<br>

{{<note>}}
Use the `--config-dirs` command-line option, or the `config_dirs` key in the `nginx-agent.conf` file, to identify the directories NGINX Agent can read from or write to. This setting also defines the location to which you can upload config files when using a control plane. 

NGINX Agent cannot write to directories outside the specified location when updating a config and cannot upload files to directories outside of the configured location. 

NGINX Agent follows NGINX configuration directives to file paths outside the designated directories and reads certificates' metadata. NGINX Agent uses the following directives:

- [`ssl_certificate`](https://nginx.org/en/docs/http/ngx_http_ssl_module.html#ssl_certificate)

{{</note>}}

{{<note>}} Use the `--dynamic-config-path` command-line option to set the location of the dynamic config file. This setting also requires you to move your dynamic config to the new path, or create a new dynamic config file at the specified location.

Default location in Linux environments: `/var/lib/nginx-agent/agent-dynamic.conf`

Default location in FreeBSD environments: `/var/db/nginx-agent/agent-dynamic.conf`

{{</note>}}

## Log Rotation

By default, NGINX Agent rotates logs daily using logrotate with the following configuration:

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

If you need to change the default configuration, update the file at `/etc/logrotate.d/nginx-agent`.

For more details on logrotate configuration, see [Logrotate Configuration Options](https://linux.die.net/man/8/logrotate).