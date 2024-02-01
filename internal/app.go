// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package internal

import (
	"context"
	"log/slog"

	"github.com/nginx/agent/v3/internal/bus"
	"github.com/nginx/agent/v3/internal/config"
	"github.com/nginx/agent/v3/internal/logger"
	"github.com/nginx/agent/v3/internal/plugin"
	"github.com/spf13/cobra"
)

const (
	defaultMessagePipeChannelSize = 100
	defaultQueueSize              = 100
)

type App struct {
	commit  string
	version string
}

func NewApp(commit, version string) *App {
	return &App{commit, version}
}

func (a *App) Run() error {
	config.Init(a.version, a.commit)

	config.RegisterRunner(func(_ *cobra.Command, _ []string) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		err := config.RegisterConfigFile()
		if err != nil {
			slog.Error("Failed to load configuration file", "error", err)
			return
		}

		agentConfig := config.GetConfig()

		slogger := logger.New(agentConfig.Log)
		slog.SetDefault(slogger)

		slog.Info("Starting NGINX Agent")

		messagePipe := bus.NewMessagePipe(ctx, defaultMessagePipeChannelSize)
		err = messagePipe.Register(defaultQueueSize, plugin.LoadPlugins(agentConfig, slogger))
		if err != nil {
			slog.Error("Failed to register plugins", "error", err)
			return
		}

		messagePipe.Run()
	})
	err := config.Execute()
	if err != nil {
		return err
	}

	return nil
}
