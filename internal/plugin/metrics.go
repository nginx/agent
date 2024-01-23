/**
 * Copyright (c) F5, Inc.
 *
 * This source code is licensed under the Apache License, Version 2.0 license found in the
 * LICENSE file in the root directory of this source tree.
 */

package plugin

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"time"

	"github.com/nginx/agent/v3/internal/bus"
	"github.com/nginx/agent/v3/internal/config"
	"github.com/nginx/agent/v3/internal/metric"
	"github.com/nginx/agent/v3/internal/metric/prometheus"
	"go.opentelemetry.io/otel"
)

const otelServiceName = "nginx-agent"

type (
	MetricsSource interface {
		// Starts the data source and begins data collection.
		Start(ctx context.Context, updateInterval time.Duration) error
		// Stops the data source from collecting data.
		Stop()
		// The type of data source.
		Type() string
	}

	// The Metrics plugin. Discovers and owns the data sources that produce metrics for the Agent.
	Metrics struct {
		Sources map[string]MetricsSource // key = MetricsSource type
		conf    config.Config
		pipe    bus.MessagePipeInterface
		prod    *metric.MetricsProducer
	}

	// The Metrics plugin is configured with functional options.
	MetricsOption func(*Metrics) error
)

// Constructor for the Metrics plugin.
func NewMetrics(c config.Config, options ...MetricsOption) (*Metrics, error) {
	m := Metrics{
		Sources: make(map[string]MetricsSource, 0),
		conf:    c,
		pipe:    nil,
	}

	for _, opt := range options {
		err := opt(&m)
		if err != nil {
			return nil, fmt.Errorf("failed to apply metrics plugin option: %w", err)
		}
	}

	return &m, nil
}

// Initializes and starts the plugin. Required for the `Plugin` interface.
func (m *Metrics) Init(mp bus.MessagePipeInterface) error {
	m.pipe = mp

	m.prod = metric.NewMetricsProducer(m.conf.Version)
	go m.prod.StartListen(mp.Context())

	meterProvider, err := metric.NewMeterProvider(m.pipe.Context(), otelServiceName, *m.conf.Metrics, m.prod)
	if err != nil {
		log.Printf("failed to create a meterProvider: %v", err)
	}

	// Sets the global meter provider.
	otel.SetMeterProvider(meterProvider)

	err = m.discoverSources()
	if err != nil {
		return fmt.Errorf("data source discovery failed: %w", err)
	}

	return m.startDataSources()
}

// Stops the metrics plugin. Required for the `Plugin` interface.
func (m *Metrics) Close() error {
	slog.Info("stopping all metrics data sources")
	for _, datasrc := range m.Sources {
		datasrc.Stop()
	}

	return nil
}

// Returns info about the plugin. Required for the `Plugin` interface.
func (m *Metrics) Info() *bus.Info {
	return &bus.Info{
		Name: "metrics",
	}
}

// Processes an incoming Message Bus message in the plugin. Required for the `Plugin` interface.
func (m *Metrics) Process(msg *bus.Message) {
	switch msg.Topic {
	case bus.OS_PROCESSES_TOPIC:
		slog.Debug("OS Processes have been updated")
		// TODO: We would need to add rediscovery logic here where any new data sources are added to the plugin's
		// sources slice.
	}
}

// Returns the list of topics the plugin is subscribed to. Required for the `Plugin` interface.
func (m *Metrics) Subscriptions() []string {
	return []string{
		bus.OS_PROCESSES_TOPIC,
		bus.METRICS_TOPIC,
	}
}

// Dynamically populates the `[]*MetricsSource` of `*Metrics`.
func (m *Metrics) discoverSources() error {
	// Always initialized currently, should dynamically determine if we have a Prometheus source or not.
	// Data sources can also be passed in as options on plugin init.
	if _, ok := m.Sources[prometheus.DataSourceType]; !ok {
		m.Sources[prometheus.DataSourceType] = prometheus.NewScraper(otel.GetMeterProvider(), m.prod)
	}

	return nil
}

// Calls Start() on each data source.
func (m *Metrics) startDataSources() error {
	for _, datasrc := range m.Sources {
		go func(ds MetricsSource) {
			err := ds.Start(m.pipe.Context(), m.conf.Metrics.ReportInterval)
			if err != nil {
				// Probably need to figure out how to handle these errors better (e.g. with a `chan error`).
				slog.Error("failed to start metrics source", "error", err)
			}
		}(datasrc)
	}
	return nil
}

//##########################
//## Begin MetricsOptions ##
//##########################

// Appends a Metrics data source that will automatically collect metrics.
func WithDataSource(ds MetricsSource) MetricsOption {
	return func(m *Metrics) error {
		m.Sources[ds.Type()] = ds
		return nil
	}
}

//########################
//## End MetricsOptions ##
//########################
