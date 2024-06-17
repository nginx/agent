# Overview

This is a basic layout for NGINX Agent application. This document focusing only on the general layout and not what you have inside. It`s basic so that the information is quickly digestable. 

## Directories

### `/cmd`

Main applications for this project.

The directory name for each application should match the name of the executable (e.g., /cmd/agent).

### `/internal`

Private application and library code. This is the code that will not be exported and available outside of this project.

### `/pkg`

Library code to be used by external applications (e.g., /pkg/files and /pkg/uuid). Other projects will import these libraries.

This is a way of grouping public facing Go code in one place away from the root directory. The root directory may contain non-Go components and directories making it easier to run various Go tools (as mentioned in these talks: Best Practices for Industrial Programming from GopherCon EU 2018, GopherCon 2018: Kat Zien - How Do You Structure Your Go Apps and GoLab 2018 - Massimiliano Pippi - Project layout patterns in Go).

The pkg directory origins: The old Go source code used to use pkg for its packages and then various Go projects in the community started copying the pattern (see this Brad Fitzpatrick`s tweet for more context).

### `/vendor`

Application dependencies (managed manually or by your favorite dependency management tool like the new built-in Go Modules feature). The go mod vendor command will create the /vendor directory for you. Note that you might need to add the -mod=vendor flag to your go build command if you are not using Go 1.14 where it`s on by default.

Dependencies are not under source control.

### `/test`

A set of test specific configurations and helper libraries. The convention with this project is to place the testing code alongside the features. This directory also contains the mock management server which can be run from /mock/grpc/cmd/main.go. This is to similuate a mock mangement server for testing purposes. It can be used as a reference implementation for a mangement server perspective.

## Other Directories

### `/api`

A series of API definitions based on [protobuf](https://github.com/golang/protobuf) and generated [gRPC](https://grpc.io/) generated code for reuse by clients and servers.

### `/docs`

The location of the project documentation.

### `/scripts`

Scripts to perform various docker, packaging and testing operations.

Some customer 'helper' configurations may be present here, including log rotate [Log Rotate](https://linux.die.net/man/8/logrotate) configuration.

### `/build`

Transient packaging and build artifacts location.

This is not under source control.

### `/.github`

Continuous Integration workflows folder.