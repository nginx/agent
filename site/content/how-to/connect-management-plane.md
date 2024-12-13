---
title: "Connect to management plane"
toc: true
weight: 600
docs: DOCS-000
---

## Overview

To monitor and manage all your F5 NGINX Agent instances from a central management plane server, you first need to connect your instances and the server. You can configure the connection by making the required changes to the NGINX Agent configuration file.

There are three types of connections you can establish between the NGINX Agent and the management plane server:

- [Mutual Transport Layer Security (mTLS) connection](#mtls-connection)
- [Transport Layer Security (TLS) connection](#tls-connection)
- [Insecure connection](#insecure-connection)

## mTLS connection

To establish a mTLS connection between the NGINX Agent and the management plane server, follow these steps:

 1. Edit the `/etc/nginx-agent/nginx-agent.conf` file to enable mTLS for NGINX Agent. Replace the example values with your own:

   ```yaml
   command:
    server:
        # the server host to connect to in order to send
        # and receive commands e.g. config apply instructions
        host: example.com
        # the server port to connect to in order to send and receive commands
        # e.g. config apply instructions
        port: 443
        # the type of connection. Currently only "grpc" is supported.
        type: grpc
    auth:
        # the token to be used in the authorization header
        # for the Agent initiated requests
        token: ...
    tls:
        # The client key to be used in the TLS/mTLS connection
        key: /etc/ssl/certs/key.pem
        # The client certificate to be used in the TLS/mTLS connection
        cert: /etc/ssl/certs/cert.pem
        # The certificate authority certificate to be used in the mTLS connection
        ca: /etc/ssl/certs/ca.pem
        # controls whether the server certificate chain and host name are verified
        skip_verify: false
        # A hostname value specified in the Subject Alternative Name extension
        server_name: example.com
   ```
2. Restart the NGINX Agent service:

   ```shell
   sudo systemctl restart nginx-agent
   ```

## TLS connection

To establish a TLS connection between the NGINX Agent and the management plane server, follow these steps:

1. Edit the `/etc/nginx-agent/nginx-agent.conf` file to enable TLS for NGINX Agent. Replace the example values with your own:

   ```yaml
   command:
      server:
         # the server host to connect to in order to send and receive commands
         # e.g. config apply instructions
         host: example.com
         # the server port to connect to in order to send and receive commands
         # e.g. config apply instructions
         port: 443
         # the type of connection. Currently only "grpc" is supported.
         type: grpc
      auth:
         # the token to be used in the authorization header for the
         # Agent initiated requests
         token: ...
      tls:
         # controls whether the server certificate chain and host name are verified
         skip_verify: false
   ```

   {{< note >}}To enable server-side TLS with a self-signed certificate, you must have TLS enabled and set `skip_verify` to `true`, which disables hostname validation. Setting `skip_verify` can be done only by updating the configuration file. **This is not recommended for production environments**.{{< /note >}}

2. Restart the NGINX Agent service:

   ```shell
   sudo systemctl restart nginx-agent
   ```
   
## Insecure connection

{{< warning >}}Insecure connections are not recommended for production environments.{{< /warning >}}

To establish an insecure connection between the NGINX Agent and the management plane server, follow these steps:

1. Edit the `/etc/nginx-agent/nginx-agent.conf` file to enable an insecure connection for NGINX Agent. Replace the example values with your own:

   ```yaml
   command:
      server:
         # the server host to connect to in order to send and receive commands e.g. config apply instructions
         host: example.com
         # the server port to connect to in order to send and receive commands e.g. config apply instructions
         port: 443
         # the type of connection. Currently only "grpc" is supported.
         type: grpc
   ```

2. Restart the NGINX Agent service:

   ```shell
   sudo systemctl restart nginx-agent
   ```
