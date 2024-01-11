//go:build tools
// +build tools

package tools

import (
	_ "github.com/deepmap/oapi-codegen/v2/cmd/oapi-codegen"
	_ "github.com/evilmartians/lefthook"
	_ "github.com/golangci/golangci-lint/cmd/golangci-lint"
	_ "google.golang.org/protobuf/cmd/protoc-gen-go"
	_ "mvdan.cc/gofumpt"
)
