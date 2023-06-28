//go:build tools
// +build tools

// https://www.jvt.me/posts/2022/06/15/go-tools-dependency-management/

package tools

import (
	_ "github.com/alvaroloes/enumer"
	_ "github.com/bufbuild/buf/cmd/buf"
	_ "github.com/evilmartians/lefthook"
	_ "github.com/go-swagger/go-swagger/cmd/swagger"
	_ "github.com/gogo/protobuf/protoc-gen-gogo"
	_ "github.com/gogo/protobuf/protoc-gen-gogofast"
	_ "github.com/golang/mock/mockgen"
	_ "github.com/golangci/golangci-lint/cmd/golangci-lint"
	_ "github.com/goreleaser/nfpm/v2/cmd/nfpm"
	_ "github.com/maxbrunsfeld/counterfeiter/v6"
	_ "github.com/mwitkow/go-proto-validators/protoc-gen-govalidators"
	_ "github.com/pseudomuto/protoc-gen-doc/cmd/protoc-gen-doc"
	_ "google.golang.org/grpc/cmd/protoc-gen-go-grpc"
)
