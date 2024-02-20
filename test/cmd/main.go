// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package main

import (
	"flag"
	"log/slog"
	"net"
	"os"
	"path/filepath"

	"github.com/nginx/agent/v3/test/mock"
)

func main() {
	var configDirectory string

	currentPath, err := os.Getwd()
	if err != nil {
		slog.Error("Unable to get current directory", "error", err)
	}

	defaultDirectory := filepath.Join(currentPath, "/test/mock/config/")
	flag.StringVar(
		&configDirectory,
		"configDirectory",
		defaultDirectory,
		"set the directory where the config files are stored",
	)
	flag.Parse()

	server := mock.NewManagementServer(configDirectory)
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		slog.Error("Failed to create listener", "error", err)
		os.Exit(1)
	}
	server.StartServer(listener)
}
