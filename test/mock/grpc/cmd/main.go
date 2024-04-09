// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package main

import (
	"context"
	"flag"
	"io"
	"log/slog"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/nginx/agent/v3/internal/logger"
	"github.com/nginx/agent/v3/test/mock/grpc"
	"github.com/nginx/agent/v3/test/types"
)

const (
	defaultSleepDuration = time.Millisecond * 100
	directoryPermissions = 0o700
)

var (
	sleepDuration   = flag.Duration("sleepDuration", defaultSleepDuration, "duration between changes in health")
	configDirectory = flag.String("configDirectory", "", "set the directory where the config files are stored")
	grpcAddress     = flag.String("grpcAddress", "127.0.0.1:0", "set the gRPC address to run the server on")
	apiAddress      = flag.String("apiAddress", "127.0.0.1:0", "set the API address to run the server on")
	logLevel        = flag.String("logLevel", "INFO", "set the log level")
)

func main() {
	flag.Parse()
	ctx := context.Background()

	agentConfig := types.GetAgentConfig()
	grpcHost, grpcPort, err := net.SplitHostPort(*grpcAddress)
	if err != nil {
		slog.Error("Failed to read host and port", "error", err)
		os.Exit(1)
	}
	portInt, err := strconv.Atoi(grpcPort)
	if err != nil {
		slog.Error("Failed to convert port", "error", err)
		os.Exit(1)
	}

	agentConfig.Command.Server.Host = grpcHost
	agentConfig.Command.Server.Port = portInt
	agentConfig.Command.Auth = nil
	agentConfig.Command.TLS = nil
	agentConfig.Common.MaxElapsedTime = *sleepDuration

	newLogger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: logger.GetLogLevel(*logLevel),
	}))
	slog.SetDefault(newLogger)

	if configDirectory == nil {
		defaultConfigDirectory, configDirErr := generateDefaultConfigDirectory()
		configDirectory = &defaultConfigDirectory
		if configDirErr != nil {
			slog.Error("Failed to create default config directory", "error", err)
			os.Exit(1)
		}
	}

	_, err = grpc.NewMockManagementServer(*apiAddress, agentConfig)
	if err != nil {
		slog.Error("Failed to start mock management server", "error", err)
		os.Exit(1)
	}
	<-ctx.Done()
}

func generateDefaultConfigDirectory() (string, error) {
	tempDirectory := os.TempDir()

	err := os.MkdirAll(filepath.Join(tempDirectory, "config/1/etc/nginx"), directoryPermissions)
	if err != nil {
		slog.Error("Failed to create directories", "error", err)
		return "", err
	}

	source, err := os.Open("test/config/nginx/nginx.conf")
	if err != nil {
		slog.Error("Failed to open nginx.conf", "error", err)
		return "", err
	}
	defer CloseFile(source)

	destination, err := os.Create(filepath.Join(tempDirectory, "config/1/etc/nginx/nginx.conf"))
	if err != nil {
		slog.Error("Failed to create nginx.conf", "error", err)
		return "", err
	}
	defer CloseFile(destination)

	_, err = io.Copy(destination, source)
	if err != nil {
		slog.Error("Failed to copy nginx.conf", "error", err)
		return "", err
	}

	return filepath.Join(tempDirectory, "config"), nil
}

func CloseFile(file *os.File) {
	err := file.Close()
	if err != nil {
		slog.Error("Failed to close file", "error", err)
	}
}
