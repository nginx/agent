![GitHub go.mod Go version](https://img.shields.io/github/go-mod/go-version/nginx/agent)
![GitHub License](https://img.shields.io/github/license/nginx/agent)
![Contributions Welcome](https://img.shields.io/badge/contributions-welcome-brightgreen.svg?style=flat)
[![Slack](https://img.shields.io/badge/slack-join%20us-brightgreen.svg?logo=slack)](https://nginxcommunity.slack.com/channels/nginx-agent)
![coverage](https://raw.githubusercontent.com/nginx/agent/badges/.badges/v3/coverage.svg)

# NGINX Agent

NGINX Agent is a companion daemon for your NGINX Open Source or NGINX Plus instance. It enables:

- Remote management of NGINX configurations
- Collection and reporting of real-time NGINX performance and operating system metrics
- Notifications of NGINX events

## Development Environment Setup
### Installing Prerequisite Packages
The following packages need to be installed:
 - make
 - golang (https://go.dev/doc/install)
 - protoc (https://grpc.io/docs/protoc-installation/)
 - mdatagen (There is currently an issue installing mdatagen https://github.com/open-telemetry/opentelemetry-collector/issues/9281. See instructions below for workaround.)

#### Workaround to install mdatagen
```
git clone https://github.com/open-telemetry/opentelemetry-collector.git
cd opentelemetry-collector
git checkout v0.124.0
cd cmd/mdatagen
go install
```

Before starting development on the NGINX Agent, it is important to download and install the necessary tool and dependencies required by the NGINX Agent. You can do this by running the following `make` command:
```
make install-tools
```

### Building NGINX Agent from Source Code
Build NGINX Agent deb package:
```
OSARCH=<operating system archiecture> make local-deb-package
```
Build NGINX Agent rpm package:
```
OSARCH=<operating system archiecture> make local-rpm-package
```
Build NGINX Agent apk package:
```
OSARCH=<operating system archiecture> make local-apk-package
```

### Testing NGINX Agent

#### Unit tests
To run unit tests and check that there is enough test coverage run the following
```
make unit-test coverge
```
To check for race conditions, the unit tests can also be run with a race condition detector
```
make race-condition-test
```

#### Integration tests
To run integration tests, run the following
```
make integration-test
```

#### Testing with a mock management plane
For testing command operations, there is a mock management gRPC server that can be used. See here: [mock management gRPC server](test/mock/grpc/README.md) \
For testing metrics, there is a mock management OTel collector that can be used. See here: [mock management OTel collector](test/mock/collector/README.md)


## NGINX Agent Technical Specifications

### Supported Distributions

NGINX Agent can run in most environments. For a list of supported distributions, see the [NGINX Technical Specs](https://docs.nginx.com/nginx/technical-specs/#supported-distributions) guide.

### Supported Deployment Environments

NGINX Agent can be deployed in the following environments:

- Bare Metal
- Container
- Public Cloud: AWS, Google Cloud Platform, and Microsoft Azure
- Virtual Machine

### Supported NGINX Product Versions

NGINX Agent works with all supported versions of NGINX Open Source and NGINX Plus.

### Sizing Recommendations

Minimum system sizing recommendations for NGINX Agent:
TBD

## Community

- Our [NGINX Community Forum ](https://community.nginx.org/tag/agent) is the go-to place to ask questions and share your thoughts.

- Our [GitHub issues page](https://github.com/nginx/agent/issues) offers space for a more technical discussion at your own pace.

## Contributing

Get involved with the project by contributing! Please see our [contributing guide](CONTRIBUTING.md) for details.

## Change Log

See our [release page](https://github.com/nginx/agent/releases) to keep track of updates.

## License

[Apache License, Version 2.0](LICENSE)
