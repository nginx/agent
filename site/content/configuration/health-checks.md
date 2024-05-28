---
title: "Health checks"
draft: false
weight: 400
toc: true
tags: [ "docs" ]
categories: ["configuration"]
doctypes: ["task"]
---

## Overview

The REST API includes a health endpoint to verify the status of NGINX Agent.

## Configure the REST API

To enable the REST API, add the following configuration to the NGINX Agent configuration file, `/etc/nginx-agent/nginx-agent.conf`:

```nginx
api:
  host: 127.0.0.1
  port: 8038
```

## Using health checks

After you enable the REST API, calling the `/health` endpoint returns the following JSON response:

```json
{
  "status": "OK",
  "checks": [
    {
      "name": "registration",
      "status": "OK"
    },
    {
      "name": "commandConnection",
      "status": "OK"
    },
    {
      "name": "metricsConnection",
      "status": "OK"
    }
  ]
}
```

The top-level `status` field is the overall health status of NGINX Agent. The health status can return three different states:

1. `PENDING`: NGINX Agent is still determining its health status.
2. `OK`: NGINX Agent is in a healthy state.
3. `ERROR`: NGINX Agent is in an unhealthy state.

The health checkpoint performs three checks to determine the overall health of the NGINX Agent:

1. `registration`: Checks if NGINX Agent has successfully registered with the management plane server.
2. `commandConnection`: Checks if NGINX Agent is still able to receive and send commands.
3. `metricsConnection`: Checks if NGINX Agent is still able to send metric reports.

If any of the checks are in an `ERROR` status, then the overall status of NGINX Agent will change to `ERROR` as well.

