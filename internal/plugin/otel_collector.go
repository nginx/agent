// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.
package plugin

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/nginx/agent/v3/internal/bus"
	"github.com/nginx/agent/v3/internal/collector"
	"github.com/nginx/agent/v3/internal/config"
	"go.opentelemetry.io/collector/otelcol"
)

type (
	// Collector The OTel collector plugin start an embedded OTel collector for metrics collection in the OTel format.
	Collector struct {
		svc     *otelcol.Collector
		appDone chan struct{}
		stopped bool
		cancel  context.CancelFunc
		config  *config.Config
	}
)

// NewCollector is the constructor for the Collector plugin.
func NewCollector(conf *config.Config) (*Collector, error) {
	settings := collector.OTelCollectorSettings(conf)
	oTelCollector, err := otelcol.NewCollector(settings)
	if err != nil {
		return nil, err
	}

	return &Collector{
		config: conf,
		svc:    oTelCollector,
	}, nil
}

// Init initializes and starts the plugin. Required for the `Plugin` interface.
func (oc *Collector) Init(ctx context.Context, mp bus.MessagePipeInterface) error {
	slog.InfoContext(ctx, "Starting OTel Collector plugin")

	var runCtx context.Context
	runCtx, oc.cancel = context.WithCancel(ctx)

	go func() {
		err := oc.run(runCtx)
		if err != nil {
			slog.ErrorContext(ctx, "error", err)
		}
	}()

	return nil
}

func (oc *Collector) run(ctx context.Context) error {
	var err error
	oc.appDone = make(chan struct{})

	go func() {
		defer close(oc.appDone)
		appErr := oc.svc.Run(ctx)
		if appErr != nil {
			err = appErr
			slog.ErrorContext(ctx, "error", appErr)
		}
	}()

	for {
		state := oc.svc.GetState()

		oc.svc.GetState()
		// While waiting for collector start, an error was found. Most likely
		// an invalid custom collector configuration file.
		if err != nil {
			return err
		}

		switch state {
		case otelcol.StateStarting:
			// NoOp
		case otelcol.StateRunning:
			return nil
		case otelcol.StateClosing, otelcol.StateClosed:
		default:
			err = fmt.Errorf("unable to start, otelcol state is %d", state)
		}
	}
}

// Info about the plugin. Required for the `Plugin` interface.
func (oc *Collector) Info() *bus.Info {
	return &bus.Info{
		Name: "collector",
	}
}

// Close about the plugin. Required for the `Plugin` interface.
func (oc *Collector) Close(ctx context.Context) error {
	slog.InfoContext(ctx, "Closing OTel Collector plugin")

	if !oc.stopped {
		oc.stopped = true
		oc.svc.Shutdown()
	}
	<-oc.appDone

	return nil
}

// Process an incoming Message Bus message in the plugin. Required for the `Plugin` interface.
func (oc *Collector) Process(_ context.Context, msg *bus.Message) {
}

// Subscriptions returns the list of topics the plugin is subscribed to. Required for the `Plugin` interface.
func (oc *Collector) Subscriptions() []string {
	return []string{}
}
