// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/nginx/agent/v3/internal"
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
			slog.Warn("NGINX Agent exiting")
			cancel()
			os.Exit(1)
		case <-ctx.Done():
		}
	}()

	app := internal.NewApp(commit, version)

	err := app.Run(ctx)
	if err != nil {
		slog.Error("NGINX Agent exiting due to error", "error", err)
		cancel()
		os.Exit(1)
	}
}
