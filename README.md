# NGINX Agent

This is an Agent for various NGINX data plane services. It communicates via GRPC and protobuf with a Management Plane via HTTP.

## Prerequisites
* Install [Goreleaser](https://goreleaser.com/install)
* Install [nFPM](https://nfpm.goreleaser.com/install/)
* Install [Protoc](https://grpc.io/docs/protoc-installation/)
* Install tools: `make install-tools`
* Install [golangci-lint](https://golangci-lint.run/usage/install/)

## Running the Agent Locally
To simply get going, run the following make target (without building a binary):
```
make run
```

## Building the Agent Locally
To build the agent locally, from the root directory, run:
```
make build
```
These commands output the agent binaries into `build/dist`.  You can run the executable with `./build/nginx-agent`

## Packaging The Agent Locally
If you want to build the operating system packages (not signed), take a look at the "Local Packaging" section of the Makefile:

## Docker With The Agent
#### Prerequisites
1. Place your nginx-repo.key and nginx-repo.crt in ./build

Running the docker make target will run agent and NGINX Plus within an appropriate Operating System.
The list of available Operating Systems are as follows:

 https://docs.nginx.com/nginx/releases/																		  
These images are based on https://github.com/nginxinc/docker-nginx and are NOT recommended for production 
| OS_RELEASE       | OS_VERSION                 | NOTES                                                          |
| ---------------- | -------------------------- | -------------------------------------------------------------- |
| amazonlinux      | 2                          |                                                                |
| ubuntu           | 18.04, 20.04, 22.04        |                                                                |
| debian           | bullseye-slim, buster-slim |                                                                |
| centos           | 7                          | centos 7 (below 7.4) uses plus-pkgs.nginx.com as PACKAGES_REPO |
| redhatenterprise | 7, 8, 9                    |                                                                |
| alpine           | 3.13, 3.14, 3.15, 3.16     |                                                                |
| oraclelinux      | 7, 8                       |                                                                |
| suse             | sles12sp5, sle15           |                                                                |
| freebsd          |                            | Not supported                                                  |

Edit the Makefile variables, to choose the appropriate Operating System for you (defaults to Ubuntu 22.04) e.g:
```
OS_RELEASE:=redhatenterprise
OS_VERSION:=9
```

### Build Container Image
Run the ```make build-docker``` command from the agent root directory

### Run Container Image
Run the ```make run-docker``` command from the agent root directory

## Local Development with the Agent
If you want to run a debug session with the agent through vscode, here is an example debug configuration you can use (gives verbose output for GRPC):

```
{
    // Use IntelliSense to learn about possible attributes.
    // Hover to view descriptions of existing attributes.
    // For more information, visit: https://go.microsoft.com/fwlink/?linkid=830387
    "version": "0.2.0",
    "configurations": [
        {
            "name": "Launch file",
            "type": "go",
            "request": "launch",
            "mode": "debug",
            "program": "${workspaceFolder}/main.go",
            "env": {
                "GRPC_VERBOSITY": "debug",
                "GRPC_TRACE": "all",
                "GODEBUG": "http2debug=2",
                "GRPC_GO_LOG_SEVERITY_LEVEL": "debug",
                "GRPC_GO_LOG_VERBOSITY_LEVEL": "99",
                "CGO_ENABLED": "0",
            }
        }
    ]
}
```

## Config Options
### TLS
For testing TLS, generate test certs by running:
```
make certs
```
Then update the nginx agent configuration with the following:
```
tls:
  # enable tls in the nginx-agent setup for grpcs
  enable: true
  # path to certificate
  cert: /path/to/client.crt
  # path to certificate key
  key: /path/to/client.key
  # path to CA cert
  ca: /path/to/ca.crt
  # skip cert verification (insecure)
  skip_verify: false
```
### Config Directories
The directories the agent will be only allowed read / write files to / from
```
config_dirs: "/etc/nginx:/usr/local/etc/nginx:/usr/share/nginx/modules:/etc/nms"
```

## Testing
### Unit Tests
`make unit-test`

### Install Tests
`make test-install`

### Performance Tests
To performance tests require a docker container to run. To build the docker container run

```make build-benchmark-docker```

Then run the following to run the performance tests

```make performance-test```
