# Mock Management gRPC Server

To start the server, run the following command
```
make run-mock-management-grpc-server
```

By default, this will start a HTTP and gRPC server on a random port. \
It will also sabe config files to the `/tmp` by default. \
To override these behaviours the following environment variables can be set to override them
```
MOCK_MANAGEMENT_PLANE_GRPC_ADDRESS=127.0.0.1:9091
MOCK_MANAGEMENT_PLANE_API_ADDRESS=127.0.0.1:9092
MOCK_MANAGEMENT_PLANE_CONFIG_DIRECTORY=/tmp/
```

Before starting the NGINX Agent, update the agent configuration with the following command config block
```
command:
  server:
    host: localhost
    port: 9091
```

To interact with the mock management gRPC server, it also starts a HTTP server with the following endpoints
```
GET http://127.0.0.1:9092/api/v1/connection

GET http://127.0.0.1:9092/api/v1/status

GET http://127.0.0.1:9092/api/v1/health

GET http://127.0.0.1:9092/api/v1/responses

POST http://127.0.0.1:9092/api/v1/requests

POST http://127.0.0.1:9092/api/v1/instance/<instance id>/config/apply
```

# Endpoints
## GET /api/v1/connection
Used to check if the NGINX Agent successfully created connection with the management plane. \
Example response:
```
{
    "messageMeta": {
        "correlationId": "0d777e07-bbdf-4ce9-a8d2-22b1e8383984",
        "messageId": "4300d60e-0e4e-4761-a934-d93623e07b90",
        "timestamp": "2024-10-25T13:14:36.028654Z"
    },
    "resource": {
        "hostInfo": {
            "hostId": "14bb13db-347d-33b4-92f6-0e3d33e0d840",
            "hostname": "example.com",
            "releaseInfo": {
                "codename": "darwin",
                "id": "darwin",
                "name": "Standalone Workstation",
                "version": "23.6.0",
                "versionId": "14.6.1"
            }
        },
        "instances": [
            {
                "instanceConfig": {
                    "agentConfig": {
                        "command": {},
                        "features": [
                            "configuration",
                            "connection",
                            "metrics",
                            "file-watcher"
                        ],
                        "file": {},
                        "metrics": {}
                    }
                },
                "instanceMeta": {
                    "instanceId": "9da17ae7-aadc-3dff-8299-fbce82ec0175",
                    "instanceType": "INSTANCE_TYPE_AGENT"
                },
                "instanceRuntime": {
                    "binaryPath": "/var/usr/bin/nginx-agent",
                    "configPath": "/etc/nginx-agent/nginx-agent.conf",
                    "processId": 31877
                }
            },
            {
                "instanceMeta": {
                    "instanceId": "6cb1a2bc-7552-33b1-9e7c-cb6658b82ebb",
                    "instanceType": "INSTANCE_TYPE_NGINX",
                    "version": "1.27.0"
                },
                "instanceRuntime": {
                    "binaryPath": "/usr/local/bin/nginx",
                    "configPath": "/usr/local/etc/nginx/nginx.conf",
                    "instanceChildren": [
                        {
                            "processId": 703
                        }
                    ],
                    "nginxRuntimeInfo": {
                        "accessLogs": [
                            "/usr/local/var/log/nginx/access.log"
                        ],
                        "dynamicModules": [
                            "http_addition_module",
                            "http_auth_request_module",
                            "http_dav_module",
                            "http_degradation_module",
                            "http_flv_module",
                            "http_gunzip_module",
                            "http_gzip_static_module",
                            "http_mp4_module",
                            "http_random_index_module",
                            "http_realip_module",
                            "http_secure_link_module",
                            "http_slice_module",
                            "http_ssl_module",
                            "http_stub_status_module",
                            "http_sub_module",
                            "http_v2_module",
                            "http_v3_module",
                            "mail_ssl_module",
                            "stream_realip_module",
                            "stream_ssl_module",
                            "stream_ssl_preread_module"
                        ],
                        "errorLogs": [
                            "/usr/local/var/log/nginx/error.log"
                        ],
                        "stubStatus": "http://127.0.0.1:8084/apitesst"
                    },
                    "processId": 595
                }
            }
        ],
        "resourceId": "14bb13db-347d-33b4-92f6-0e3d33e0d840"
    }
}
```
## GET /api/v1/status
Used to check if the NGINX Agent successfully sent a data plane status update to the management plane. \
Example response:
```
{
    "messageMeta": {
        "correlationId": "0d777e07-bbdf-4ce9-a8d2-22b1e8383984",
        "messageId": "4300d60e-0e4e-4761-a934-d93623e07b90",
        "timestamp": "2024-10-25T13:14:36.028654Z"
    },
    "resource": {
        "hostInfo": {
            "hostId": "14bb13db-347d-33b4-92f6-0e3d33e0d840",
            "hostname": "example.com",
            "releaseInfo": {
                "codename": "darwin",
                "id": "darwin",
                "name": "Standalone Workstation",
                "version": "23.6.0",
                "versionId": "14.6.1"
            }
        },
        "instances": [
            {
                "instanceConfig": {
                    "agentConfig": {
                        "command": {},
                        "features": [
                            "configuration",
                            "connection",
                            "metrics",
                            "file-watcher"
                        ],
                        "file": {},
                        "metrics": {}
                    }
                },
                "instanceMeta": {
                    "instanceId": "9da17ae7-aadc-3dff-8299-fbce82ec0175",
                    "instanceType": "INSTANCE_TYPE_AGENT"
                },
                "instanceRuntime": {
                    "binaryPath": "/var/usr/bin/nginx-agent",
                    "configPath": "/etc/nginx-agent/nginx-agent.conf",
                    "processId": 31877
                }
            },
            {
                "instanceMeta": {
                    "instanceId": "6cb1a2bc-7552-33b1-9e7c-cb6658b82ebb",
                    "instanceType": "INSTANCE_TYPE_NGINX",
                    "version": "1.27.0"
                },
                "instanceRuntime": {
                    "binaryPath": "/usr/local/bin/nginx",
                    "configPath": "/usr/local/etc/nginx/nginx.conf",
                    "instanceChildren": [
                        {
                            "processId": 703
                        }
                    ],
                    "nginxRuntimeInfo": {
                        "accessLogs": [
                            "/usr/local/var/log/nginx/access.log"
                        ],
                        "dynamicModules": [
                            "http_addition_module",
                            "http_auth_request_module",
                            "http_dav_module",
                            "http_degradation_module",
                            "http_flv_module",
                            "http_gunzip_module",
                            "http_gzip_static_module",
                            "http_mp4_module",
                            "http_random_index_module",
                            "http_realip_module",
                            "http_secure_link_module",
                            "http_slice_module",
                            "http_ssl_module",
                            "http_stub_status_module",
                            "http_sub_module",
                            "http_v2_module",
                            "http_v3_module",
                            "mail_ssl_module",
                            "stream_realip_module",
                            "stream_ssl_module",
                            "stream_ssl_preread_module"
                        ],
                        "errorLogs": [
                            "/usr/local/var/log/nginx/error.log"
                        ],
                        "stubStatus": "http://127.0.0.1:8084/apitesst"
                    },
                    "processId": 595
                }
            }
        ],
        "resourceId": "14bb13db-347d-33b4-92f6-0e3d33e0d840"
    }
}
```
## GET /api/v1/health
Used to check if the NGINX Agent successfully sent a data plane health update to the management plane. \
Example response:
```
{
    "instanceHealths": [
        {
            "instanceHealthStatus": "INSTANCE_HEALTH_STATUS_HEALTHY",
            "instanceId": "6cb1a2bc-7552-33b1-9e7c-cb6658b82ebb"
        }
    ],
    "messageMeta": {
        "correlationId": "ba95cf3b-793a-4761-a843-c85c7562faca",
        "messageId": "7cd63160-c408-4ab6-8639-5909453c8574",
        "timestamp": "2024-10-25T13:14:37.894919Z"
    }
}
```
## GET /api/v1/responses
Used to check if what responses the management plane received from the NGINX Agent on the Subscribe rpc stream.
Example response:
```
[
    {
        "message_meta": {
            "message_id": "0971692c-c0bf-4b67-a03b-ef13b3c3ea9b",
            "correlation_id": "87e3c12f-eaf7-4f54-a27c-c91fe75d44a6",
            "timestamp": {
                "seconds": 1729862076,
                "nanos": 98825000
            }
        },
        "command_response": {
            "status": 1,
            "message": "Successfully updated all files"
        }
    }
]
```
## POST /api/v1/requests
Used to send management plane requests over the Subscribe rpc stream to the NGINX Agent. \
Example request body:
```
{
    "message_meta": {
    "message_id": "e2254df9-8edd-4900-91ce-88782473bcb9",
    "correlation_id": "9673f3b4-bf33-4d98-ade1-ded9266f6818",
    "timestamp": "2023-01-15T01:30:15.01Z"
    },
    "health_request": {}
}
```
## POST /api/v1/instance/\<instance id\>/config/apply
Used to send management plane config apply request over the Subscribe rpc stream to the NGINX Agent for a particular data plane instance.

The config files that you need to change to perform a config apply are located here `/tmp/config/<instance id>/<location of nginx files>`.

The `<instance id>` and `<location of nginx files>` can be determined from the response of the `/api/v1/connection` endpoint. \
In the example above the `instance id` would be `6cb1a2bc-7552-33b1-9e7c-cb6658b82ebb` and the `<location of nginx files>` would be `/etc/nginx-agent/nginx-agent.conf`. \
So the full path to the file used by the mock management plane would be `/tmp/config/6cb1a2bc-7552-33b1-9e7c-cb6658b82ebb/etc/nginx-agent/nginx.conf`.

Simply edit this file and then perform a POST request against the `/api/v1/instance/<instance id>/config/apply` endpoint to execute a config apply request.

