// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package plugin

import (
	"context"
	"log/slog"
	"sync"

	"github.com/nginx/agent/v3/internal/logger"
	"github.com/nginx/agent/v3/internal/model"

	pkg "github.com/nginx/agent/v3/pkg/config"

	"github.com/nginx/agent/v3/internal/collector"
	"github.com/nginx/agent/v3/internal/command"
	"github.com/nginx/agent/v3/internal/grpc"
	"github.com/nginx/agent/v3/internal/resource"

	"github.com/nginx/agent/v3/internal/bus"
	"github.com/nginx/agent/v3/internal/config"
	"github.com/nginx/agent/v3/internal/watcher"
)

func LoadPlugins(ctx context.Context, agentConfig *config.Config) []bus.Plugin {
	plugins := make([]bus.Plugin, 0)

	manifestLock := &sync.RWMutex{}

	plugins = addCommandAndResourcePlugins(ctx, plugins, agentConfig, manifestLock)
	plugins = addAuxiliaryCommandAndResourcePlugins(ctx, plugins, agentConfig, manifestLock)
	plugins = addCollectorPlugin(ctx, agentConfig, plugins)
	plugins = addWatcherPlugin(plugins, agentConfig)

	return plugins
}

func addCommandAndResourcePlugins(ctx context.Context, plugins []bus.Plugin, agentConfig *config.Config,
	manifestLock *sync.RWMutex,
) []bus.Plugin {
	if agentConfig.IsCommandGrpcClientConfigured() {
		newCtx := context.WithValue(
			ctx,
			logger.ServerTypeContextKey, slog.Any(logger.ServerTypeKey, model.Command),
		)

		grpcConnection, err := grpc.NewGrpcConnection(newCtx, agentConfig, agentConfig.Command)
		if err != nil {
			slog.WarnContext(newCtx, "Failed to create gRPC connection for command server", "error", err)
		} else {
			commandPlugin := command.NewCommandPlugin(agentConfig, grpcConnection, model.Command)
			plugins = append(plugins, commandPlugin)
			resourcePlugin := resource.NewResource(agentConfig, grpcConnection, model.Command, manifestLock)
			plugins = append(plugins, resourcePlugin)
		}
	} else {
		slog.InfoContext(ctx, "Agent is not connected to a management plane. "+
			"Configure a command server to establish a connection with a management plane.")
	}

	return plugins
}

func addAuxiliaryCommandAndResourcePlugins(ctx context.Context, plugins []bus.Plugin,
	agentConfig *config.Config, manifestLock *sync.RWMutex,
) []bus.Plugin {
	if agentConfig.IsAuxiliaryCommandGrpcClientConfigured() {
		newCtx := context.WithValue(
			ctx,
			logger.ServerTypeContextKey, slog.Any(logger.ServerTypeKey, model.Auxiliary),
		)

		auxGRPCConnection, err := grpc.NewGrpcConnection(newCtx, agentConfig, agentConfig.AuxiliaryCommand)
		if err != nil {
			slog.WarnContext(newCtx, "Failed to create gRPC connection for auxiliary command server", "error", err)
		} else {
			auxCommandPlugin := command.NewCommandPlugin(agentConfig, auxGRPCConnection, model.Auxiliary)
			plugins = append(plugins, auxCommandPlugin)
			resourceReadPlugin := resource.NewResource(agentConfig, auxGRPCConnection, model.Auxiliary, manifestLock)
			plugins = append(plugins, resourceReadPlugin)
		}
	} else {
		slog.DebugContext(ctx, "Agent is not connected to an auxiliary management plane. "+
			"Configure a auxiliary command server to establish a connection.")
	}

	return plugins
}

func addCollectorPlugin(ctx context.Context, agentConfig *config.Config, plugins []bus.Plugin) []bus.Plugin {
	if !agentConfig.IsFeatureEnabled(pkg.FeatureMetrics) {
		slog.WarnContext(ctx, "Metrics feature disabled, no metrics will be collected",
			"enabled_features", agentConfig.Features)

		return plugins
	}
	if agentConfig.IsACollectorExporterConfigured() {
		oTelCollector, err := collector.NewCollector(agentConfig)
		if err == nil {
			plugins = append(plugins, oTelCollector)
		} else {
			slog.ErrorContext(ctx, "Failed to initialize collector plugin", "error", err)
		}
	} else {
		slog.InfoContext(ctx, "Agent OTel collector isn't started. "+
			"Configure a collector to begin collecting metrics.")
	}

	return plugins
}

func addWatcherPlugin(plugins []bus.Plugin, agentConfig *config.Config) []bus.Plugin {
	watcherPlugin := watcher.NewWatcher(agentConfig)
	plugins = append(plugins, watcherPlugin)

	return plugins
}
