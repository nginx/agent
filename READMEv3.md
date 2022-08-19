# Agent

Whether you are using NGINX OSS or NGINX Plus for App Delivery, Content Caching, API Gateway, App Security, ... Agent enhances your deployment experience and enables you to monitor your system and app performance. To do so, Agent provides an administrative entry point to remotely manage, configure and collect metrics and events on the datapath instances.

Simply put, Agent optimizes the operationalization of your application stack.

## How it works 

<Oliver/Chris, please provide a paragraph with technical explanation on how Agent communicates/interact with NGINX OSS and how a user can interact with Agent>

<Drawing is ideal here >

## Installation

### Running NGINX Agent locally
<!-- ### Install local software Go minium and version-->
Make sure you have Go installed ([download](https://go.dev/dl/)). Version 1.18 or higher is required.

### Building the Agent Locally
* Install [Goreleaser](https://goreleaser.com/install)
* Install [nFPM](https://nfpm.goreleaser.com/install/)
* Install [Protoc](https://grpc.io/docs/protoc-installation/)
* Install [pre-commit](https://pre-commit.com/#install)
* Install tools: `make install-tools`
* Install git hooks: `pre-commit install`
* Install [golangci-lint](https://golangci-lint.run/usage/install/)



## Agent in action in 2 minutes

At the end of the following steps, you will locally have an NGINX instance running, Agent running, and a mock management plane to which the Agent reports.

Ensure you have Go installed ([download](https://go.dev/dl/)). Version 1.18 or higher is required.

Ensure an NGINX instance is running. See [Prebuilt Packages for Linux and BSD](https://www.nginx.com/resources/wiki/start/topics/tutorials/install/) or if your running on Mac see [brew nginx](https://formulae.brew.sh/formula/nginx)

Next, start the mock control plane using the below command
```
go run sdk/examples/server.go

# Command Output
INFO[0000] http listening at 54790 # mock control plane port
INFO[0000] grpc listening at 54789 # grpc control plane port which NGINX Agent will report to
```

Next change the nginx-agent.conf file within the root directory
### Agent Settings
```yaml
server:
  # host of the control plane
  host: 127.0.0.1
  grpcPort: 54789 # control plane grpc port
# tls options - NOT RECOMMENDED FOR PRODUCTION
tls:
  enable: false
  skip_verify: true
```

Next, open up another terminal and start the NGINX Agent
```
go run main.go

# Command Output snippet
WARN[0000] Log level is info                            
INFO[0000] setting displayName to XXX            
INFO[0000] NGINX Agent at with pid 12345, clientID=XXXXXX-XXXXXX-XXXXXX-XXXXXX-XXXXXX name=XXX
INFO[0000] NginxBinary initializing                     
INFO[0000] Commander initializing                       
INFO[0000] Comms initializing                           
INFO[0000] OneTimeRegistration initializing             
INFO[0000] Registering XXXXXX-XXXXXX-XXXXXX-XXXXXX-XXXXXX 
INFO[0000] Metrics initializing                         
INFO[0000] MetricsThrottle initializing                 
INFO[0000] DataPlaneStatus initializing                 
INFO[0000] MetricsThrottle waiting for report ready     
INFO[0000] Metrics waiting for handshake to be completed 
INFO[0000] ProcessWatcher initializing                  
INFO[0000] Extensions initializing                      
INFO[0000] FileWatcher initializing                     
INFO[0000] FileWatchThrottle initializing
INFO[0001] Events initializing                          
INFO[0001] OneTimeRegistration completed
```

Next, open up a web browser to view the mock control plane [http://localhost:54790](http://localhost:54790). 6 links will be presented on the control plane

- registred - Shows
- nginxes - Shows
- configs - Shows
- configs/chunked - Shows
- configs/raw - Shows
- metrics - Shows 


For more use-cases of Agent, refer to https://github.com/nginx/agent/tree/main/sdk/examples


## Community

- The go-to place to start asking questions and share your thoughts is our [Slack channel](https://nginxcommunity.slack.com/).

- Our [GitHub issues page](https://github.com/nginx/agent/issues) offers space for a more technical discussion at your own pace.

- The [project map](https://github.com/orgs/nginx/projects/2) on GitHub sheds some light on our current work and plans for the future.

- Get involved with the project by contributing! See the contributing guide for details.

To reach the team directly, subscribe to the [mailing list](https://mailman.nginx.org/mailman/listinfo/agent).

For security issues, [email us](https://github.com/nginx/unit/blob/master/security-alert@nginx.org), mentioning NGINX Agent in the subject and following the [CVSS v3.1](https://www.first.org/cvss/v3.1/specification-document) spec.