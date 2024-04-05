// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package main

import (
	"flag"
	"io"
	"log/slog"
	"net"
	"os"
	"path/filepath"

	"github.com/nginx/agent/v3/api/grpc/mpi/v1"
	"github.com/nginx/agent/v3/internal/logger"
	mockGrpc "github.com/nginx/agent/v3/test/mock/grpc"
	"google.golang.org/grpc"
)

const (
	directoryPermissions = 0o700
)

func main() {
	var configDirectory string
	var grpcAddress string
	var apiAddress string
	var address string
	var logLevel string

	flag.StringVar(
		&configDirectory,
		"configDirectory",
		"",
		"set the directory where the config files are stored",
	)

	flag.StringVar(
		&address,
		"address",
		"127.0.0.1:0",
		"set the address to run the server on",
	)

	flag.StringVar(
		&grpcAddress,
		"grpcAddress",
		"127.0.0.1:0",
		"set the gRPC address to run the server on",
	)

	flag.StringVar(
		&apiAddress,
		"apiAddress",
		"127.0.0.1:0",
		"set the API address to run the server on",
	)

	flag.StringVar(
		&logLevel,
		"logLevel",
		"INFO",
		"set the log level",
	)

	flag.Parse()

	newLogger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: logger.GetLogLevel(logLevel),
	}))
	slog.SetDefault(newLogger)

	if configDirectory == "" {
		defaultConfigDirectory, err := generateDefaultConfigDirectory()
		configDirectory = defaultConfigDirectory
		if err != nil {
			slog.Error("Failed to create default config directory", "error", err)
			os.Exit(1)
		}
	}

	commandServer := mockGrpc.NewManagementGrpcServer()

	go func() {
		listener, listenError := net.Listen("tcp", apiAddress)
		if listenError != nil {
			slog.Error("Failed to create listener", "error", listenError)
			os.Exit(1)
		}

		commandServer.StartServer(listener)
	}()

	fileServer, err := mockGrpc.NewManagementGrpcFileServer(configDirectory)
	if err != nil {
		slog.Error("Failed to create file server", "error", err)
		os.Exit(1)
	}

	listener, err := net.Listen("tcp", grpcAddress)
	if err != nil {
		slog.Error("Failed to listen", "error", err)
		os.Exit(1)
	}
	var opts []grpc.ServerOption

	grpcServer := grpc.NewServer(opts...)
	v1.RegisterCommandServiceServer(grpcServer, commandServer)
	v1.RegisterFileServiceServer(grpcServer, fileServer)

	slog.Info("Starting mock management plane gRPC server", "address", listener.Addr().String())

	err = grpcServer.Serve(listener)
	if err != nil {
		slog.Error("Failed to serve mock management plane gRPC server", "error", err)
		os.Exit(1)
	}
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
	defer source.Close()

	destination, err := os.Create(filepath.Join(tempDirectory, "config/1/etc/nginx/nginx.conf"))
	if err != nil {
		slog.Error("Failed to create nginx.conf", "error", err)
		return "", err
	}
	defer destination.Close()

	_, err = io.Copy(destination, source)
	if err != nil {
		slog.Error("Failed to copy nginx.conf", "error", err)
		return "", err
	}

	return filepath.Join(tempDirectory, "config"), nil
}
