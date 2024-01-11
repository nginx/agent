/**
 * Copyright (c) F5, Inc.
 *
 * This source code is licensed under the Apache License, Version 2.0 license found in the
 * LICENSE file in the root directory of this source tree.
 */

package internal

import (
	"context"
	"log/slog"
	"os"
	"time"

	"github.com/nginx/agent/v3/internal/bus"
	"github.com/nginx/agent/v3/internal/plugin"
)

type App struct{}

func NewApp() *App {
	return &App{}
}

func (*App) Run() {
	ctx := context.Background()

	handler := slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug})
	logger := slog.New(handler)
	slog.SetDefault(logger)

	slog.Info("starting NGINX Agent")

	processMonitor := plugin.NewProcessMonitor(&plugin.ProcessMonitorParameters{
		MonitoringFrequency: 1 * time.Minute,
	})

	dataplaneServer := plugin.NewDataplaneServer(&plugin.DataplaneServerParameters{
		Address: "0.0.0.0:8091",
		Logger:  logger,
	})

	messagePipe := bus.NewMessagePipe(ctx, 100)
	err := messagePipe.Register(100, []bus.Plugin{processMonitor, dataplaneServer})
	if err != nil {
		slog.Error("failed to register plugins", "error", err)
		os.Exit(0)
	}

	messagePipe.Run()
}
