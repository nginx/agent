// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package plugin

import (
	"context"
	"log/slog"

	"github.com/nginx/agent/v3/internal/collector"
	"github.com/nginx/agent/v3/internal/command"
	"github.com/nginx/agent/v3/internal/file"
	"github.com/nginx/agent/v3/internal/grpc"
	"github.com/nginx/agent/v3/internal/resource"

	"github.com/nginx/agent/v3/internal/bus"
	"github.com/nginx/agent/v3/internal/config"
	"github.com/nginx/agent/v3/internal/watcher"
)

func LoadPlugins(ctx context.Context, agentConfig *config.Config) []bus.Plugin {
	plugins := make([]bus.Plugin, 0)

	plugins = addResourcePlugin(plugins)

	if isGrpcClientConfigured(agentConfig) {
		grpcConnection, err := grpc.NewGrpcConnection(ctx, agentConfig)
		if err != nil {
			slog.WarnContext(ctx, "Failed to create gRPC connection", "error", err)
		} else {
			commandPlugin := command.NewCommandPlugin(agentConfig, grpcConnection)
			plugins = append(plugins, commandPlugin)
			filePlugin := file.NewFilePlugin(agentConfig, grpcConnection)
			plugins = append(plugins, filePlugin)
		}
	}

	plugins = addCollector(agentConfig, plugins)
	plugins = append(plugins, watcher.NewWatcher(agentConfig))

	return plugins
}

func addResourcePlugin(plugins []bus.Plugin) []bus.Plugin {
	resourcePlugin := resource.NewResource()
	plugins = append(plugins, resourcePlugin)

	return plugins
}

func isGrpcClientConfigured(agentConfig *config.Config) bool {
	return agentConfig.Command != nil &&
		agentConfig.Command.Server != nil &&
		agentConfig.Command.Server.Type == config.Grpc
}

func addCollector(agentConfig *config.Config, plugins []bus.Plugin) []bus.Plugin {
	if agentConfig.Collector != nil {
		oTelCollector, err := collector.New(agentConfig)
		if err == nil {
			plugins = append(plugins, oTelCollector)
		} else {
			slog.Error("init collector plugin", "error", err)
		}
	}

	return plugins
}
