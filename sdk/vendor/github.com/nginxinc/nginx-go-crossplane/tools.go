//go:build tools
// +build tools

// This file just exists to ensure we download the tools we need for building
// See https://github.com/golang/go/wiki/Modules#how-can-i-track-tool-dependencies-for-a-module

package crossplane

import (
	_ "github.com/jstemmer/go-junit-report/parser"
	_ "github.com/maxbrunsfeld/counterfeiter/v6/generator"
	_ "golang.org/x/tools/imports"
)
