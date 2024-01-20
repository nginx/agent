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
	"github.com/nginx/agent/v3/internal/datasource/metric"
	"github.com/nginx/agent/v3/internal/datasource/prometheus"
	"go.opentelemetry.io/otel"
)

const otelServiceName = "prometheus"

type (
	MetricsSource interface {
		// Starts the data source and begins data collection.
		Start(ctx context.Context, updateInterval time.Duration) error
		// Stops the data source from collecting data.
		Stop()
	}

	MetricsPluginConf struct {
		reportRate time.Duration
	}

	// The Metrics plugin. Discovers and owns the data sources that produce metrics for the Agent.
	Metrics struct {
		Sources []MetricsSource
		conf    MetricsPluginConf
		pipe    bus.MessagePipeInterface
		prod    *metric.MetricsProducer
	}

	// The Metrics plugin is configured with functional options.
	MetricsOption func(*Metrics) error
)

// Constructor for the Metrics plugin.
func NewMetrics(options ...MetricsOption) (*Metrics, error) {
	defaultConfig := MetricsPluginConf{
		reportRate: 30 * time.Second,
	}

	m := Metrics{
		Sources: make([]MetricsSource, 0),
		conf:    defaultConfig,
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

	m.prod = metric.NewMetricsProducer()
	go m.prod.StartListen()

	meterProvider, err := metric.NewMeterProvider(m.pipe.Context(), otelServiceName, m.prod)
	if err != nil {
		log.Printf("Failed to create meterProvider: %v", err)
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
func (m *Metrics) Process(msg *bus.Message) error {
	switch msg.Topic {
	case bus.OS_PROCESSES_TOPIC:
		slog.Debug("OS Processes have been updated")
		m.Sources = make([]MetricsSource, 0)
		err := m.discoverSources()
		if err != nil {
			return fmt.Errorf("data source discovery failed: %w", err)
		}

		return m.startDataSources()
	}

	return nil
}

// Returns the list of topics the plugin is subscribed to. Required for the `Plugin` interface.
func (m *Metrics) Subscriptions() []string {
	return []string{
		bus.METRICS_TOPIC,
	}
}

// Dynamically populates the `[]*MetricsSource` of `*Metrics`.
func (m *Metrics) discoverSources() error {
	// Always initialized currently, should dynamically determine if we have a Prometheus source or not.
	prometheusSource := prometheus.NewScraper(otel.GetMeterProvider(), m.prod)

	m.Sources = append(m.Sources, prometheusSource)
	return nil
}

// Calls Start() on each data source.
func (m *Metrics) startDataSources() error {
	for _, datasrc := range m.Sources {
		go func(ds MetricsSource) {
			err := ds.Start(m.pipe.Context(), m.conf.reportRate)
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
		m.Sources = append(m.Sources, ds)
		return nil
	}
}

// Sets a full config for the Metrics plugin.
func WithMetricsConfig(mpc MetricsPluginConf) MetricsOption {
	return func(m *Metrics) error {
		m.conf = mpc
		return nil
	}
}

//########################
//## End MetricsOptions ##
//########################
