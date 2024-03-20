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

	"github.com/nginx/agent/v3/internal"
)

var (
	// set at buildtime
	commit  = ""
	version = ""
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	app := internal.NewApp(commit, version)

	err := app.Run(ctx)
	if err != nil {
		slog.Error("NGINX Agent exiting due to error", "error", err)
	}
}
