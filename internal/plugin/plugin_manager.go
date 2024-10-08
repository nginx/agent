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

	plugins = addResourcePlugin(plugins, agentConfig)
	plugins = addCommandAndFilePlugins(ctx, plugins, agentConfig)
	plugins = addCollectorPlugin(ctx, agentConfig, plugins)
	plugins = addWatcherPlugin(plugins, agentConfig)

	return plugins
}

func addResourcePlugin(plugins []bus.Plugin, agentConfig *config.Config) []bus.Plugin {
	resourcePlugin := resource.NewResource(agentConfig)
	plugins = append(plugins, resourcePlugin)

	return plugins
}

func addCommandAndFilePlugins(ctx context.Context, plugins []bus.Plugin, agentConfig *config.Config) []bus.Plugin {
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
	} else {
		slog.InfoContext(ctx, "Agent is not connected to a management plane. "+
			"Configure a command server to establish a connection with a management plane.")
	}

	return plugins
}

func addCollectorPlugin(ctx context.Context, agentConfig *config.Config, plugins []bus.Plugin) []bus.Plugin {
	if agentConfig.Collector != nil {
		oTelCollector, err := collector.New(agentConfig)
		if err == nil {
			plugins = append(plugins, oTelCollector)
		} else {
			slog.ErrorContext(ctx, "init collector plugin", "error", err)
		}
	}

	return plugins
}

func addWatcherPlugin(plugins []bus.Plugin, agentConfig *config.Config) []bus.Plugin {
	watcherPlugin := watcher.NewWatcher(agentConfig)
	plugins = append(plugins, watcherPlugin)

	return plugins
}

func isGrpcClientConfigured(agentConfig *config.Config) bool {
	return agentConfig.Command != nil &&
		agentConfig.Command.Server != nil &&
		agentConfig.Command.Server.Host != "" &&
		agentConfig.Command.Server.Port != 0 &&
		agentConfig.Command.Server.Type == config.Grpc
}
