// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/nginx/agent/v3/internal"
	"github.com/nginx/agent/v3/internal/config"
)

var (
	// set at buildtime
	commit  = ""
	version = ""
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		select {
		case <-sigChan:
			slog.WarnContext(ctx, "NGINX Agent exiting")
			cancel()

			time.Sleep(config.DefGracefulShutdownPeriod)
			slog.Error(
				fmt.Sprintf(
					"Failed to gracefully shutdown within timeout of %v. Exiting",
					config.DefGracefulShutdownPeriod,
				),
			)
			os.Exit(1)
		case <-ctx.Done():
		}
	}()

	app := internal.NewApp(commit, version)

	err := app.Run(ctx)
	if err != nil {
		slog.ErrorContext(ctx, "NGINX Agent exiting due to error", "error", err)
	}
}
