//go:build tools
// +build tools

package tools

import (
	_ "github.com/alvaroloes/enumer"
	_ "github.com/gogo/protobuf/protoc-gen-gogo"
	_ "github.com/gogo/protobuf/protoc-gen-gogofast"
	_ "github.com/golang/mock/mockgen"
	_ "github.com/maxbrunsfeld/counterfeiter/v6"
	_ "github.com/mwitkow/go-proto-validators/protoc-gen-govalidators"
	_ "google.golang.org/grpc/cmd/protoc-gen-go-grpc"
	_ "github.com/bufbuild/buf/cmd/buf"
)
