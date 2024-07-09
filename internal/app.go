// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package internal

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/nginx/agent/v3/internal/collector"
	"github.com/nginx/agent/v3/internal/command"
	"github.com/nginx/agent/v3/internal/file"
	"github.com/nginx/agent/v3/internal/grpc"
	"github.com/nginx/agent/v3/internal/resource"
	"github.com/nginx/agent/v3/internal/watcher"

	"github.com/nginx/agent/v3/internal/bus"
	"github.com/nginx/agent/v3/internal/config"
	"github.com/nginx/agent/v3/internal/logger"
	"github.com/spf13/cobra"
)

const (
	defaultMessagePipeChannelSize = 100
	defaultQueueSize              = 100
)

type App struct {
	grpcConn *grpc.GrpcConnection
	commit   string
	version  string
}

func NewApp(commit, version string) *App {
	return &App{nil, version, commit}
}

func (a *App) Run(ctx context.Context) (err error) {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		select {
		case <-sigChan:
			slog.DebugContext(ctx, "NGINX Agent App exiting")
		case <-ctx.Done():
			connCloseErr := a.grpcConn.Close(ctx)
			slog.ErrorContext(ctx, "Issue closing gRPC connection", "error", connCloseErr)
		}
	}()

	config.Init(a.version, a.commit)

	config.RegisterRunner(func(_ *cobra.Command, _ []string) {
		err = config.RegisterConfigFile()
		if err != nil {
			slog.ErrorContext(ctx, "Failed to load configuration file", "error", err)
			return
		}

		agentConfig, configErr := config.ResolveConfig()
		if configErr != nil {
			slog.ErrorContext(ctx, "Invalid config", "error", configErr)
			return
		}

		standardLogger := logger.New(*agentConfig.Log)
		slog.SetDefault(standardLogger)

		slog.InfoContext(ctx, "Starting NGINX Agent",
			slog.String("version", a.version),
			slog.String("commit", a.commit),
		)

		messagePipe := bus.NewMessagePipe(defaultMessagePipeChannelSize)
		msgPipeErr := messagePipe.Register(defaultQueueSize, a.loadPlugins(agentConfig))
		if msgPipeErr != nil {
			slog.ErrorContext(ctx, "Failed to register plugins", "error", msgPipeErr)
			return
		}

		messagePipe.Run(ctx)
	})
	err = config.Execute(ctx)
	if err != nil {
		return err
	}

	return nil
}

func (a *App) loadPlugins(agentConfig *config.Config) []bus.Plugin {
	plugins := make([]bus.Plugin, 0)

	plugins = a.addResourcePlugin(plugins, agentConfig)

	if a.isGrpcClientConfigured(agentConfig) {
		commandPlugin := command.NewCommandPlugin(agentConfig, a.grpcConn)
		plugins = append(plugins, commandPlugin)
		filePlugin := file.NewFilePlugin(agentConfig, a.grpcConn)
		plugins = append(plugins, filePlugin)
	}

	plugins = a.addCollector(agentConfig, plugins)
	plugins = append(plugins, watcher.NewWatcher(agentConfig))

	return plugins
}

func (a *App) addResourcePlugin(plugins []bus.Plugin, agentConfig *config.Config) []bus.Plugin {
	resourcePlugin := resource.NewResource(agentConfig)
	plugins = append(plugins, resourcePlugin)

	return plugins
}

func (a *App) isGrpcClientConfigured(agentConfig *config.Config) bool {
	return agentConfig.Command != nil &&
		agentConfig.Command.Server != nil &&
		agentConfig.Command.Server.Type == config.Grpc
}

func (a *App) addCollector(agentConfig *config.Config, plugins []bus.Plugin) []bus.Plugin {
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
