---
title: "Health Checks"
draft: false
weight: 400
toc: true
tags: [ "docs" ]
docs: "DOCS-XXX"
categories: ["configuration"]
doctypes: ["task"]
---

## Overview
To monitor the state of the NGINX Agent, there is a health endpoint in the REST API that can be used.

## Configure REST API
To enable the REST API, the following configuration is needed:
```nginx configuration
api:
  host: 127.0.0.1
  port: 8038
```

## Using health checks
Once the REST API is enabled, calling the `/health` endpoint will return the following json response:
```
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

The top level `status` field is the overall health status of the NGINX Agent. There are 3 states that the health status can be in.
1. `PENDING`: The NGINX Agent is still determining the it's health status.
2. `OK`: The NGINX Agent is in a healthy state.
3. `ERROR`: The NGINX Agent is in an unhealthy state.

To determine the overall health of the NGINX Agent, there are 3 checks that are performed.
1. `registration`: This checks if the NGINX Agent has successfully registered with the management plane server.
2. `commandConnection`: This checks if the NGINX Agent is still able to receive and send commands.
3. `metricsConnection`: This checks if the NGINX Agent is still able to send metric reports.

If any of these checks have an `ERROR` status, then the overall status of the NGINX Agent will change to `ERROR` as well.
