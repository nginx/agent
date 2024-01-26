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
	"github.com/nginx/agent/v3/internal/metrics"
	"github.com/nginx/agent/v3/internal/metrics/prometheus"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
)

const otelServiceName = "nginx-agent"

type (
	shutdownFunc func()

	DataSourceOption func(DataSource) error

	DataSource interface {
		Register(reportCallBack func(metricdata.Metrics))
		// Starts the data source and begins data collection at a regular interval. Metrics should be pushed to the
		// MessagePipe using the metrics topic, as they will be handled in the metrics plugin.
		Start(ctx context.Context, updateInterval time.Duration) error
		// Stops the data source from collecting data.
		Stop()
		// Getter for the type of data source.
		Type() string
	}

	// Tuple for a data source and its associated OTel Meter.
	MetricsSource struct {
		dataSource DataSource
		meter      metric.Meter
	}

	// The Metrics plugin. Discovers and owns the data sources that produce metrics for the Agent.
	Metrics struct {
		Sources       map[string]*MetricsSource // key = DataSource.Type
		conf          config.Config
		pipe          bus.MessagePipeInterface
		prod          *metrics.MetricsProducer
		shutdownFuncs []shutdownFunc
	}

	// The Metrics plugin can be configured with functional options.
	MetricsOption func(*Metrics) error
)

// Constructor for the Metrics plugin.
func NewMetrics(c config.Config, options ...MetricsOption) (*Metrics, error) {
	ctx, cancel := context.WithCancel(context.Background())

	prod := metrics.NewMetricsProducer(c.Version)
	meterProvider, err := metrics.NewMeterProvider(ctx, otelServiceName, *c.Metrics, prod)
	if err != nil {
		log.Printf("failed to create a meterProvider: %v", err)
	}

	// Sets the global meter provider.
	otel.SetMeterProvider(meterProvider)

	m := Metrics{
		Sources:       make(map[string]*MetricsSource, 0), // Key = data source type
		conf:          c,
		pipe:          nil,
		prod:          prod,
		shutdownFuncs: []shutdownFunc{shutdownFunc(cancel)},
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
	go m.prod.StartListen(mp.Context())

	err := m.discoverSources()
	if err != nil {
		return fmt.Errorf("data source discovery failed: %w", err)
	}

	return m.startDataSources()
}

// Stops the metrics plugin. Required for the `Plugin` interface.
func (m *Metrics) Close() error {
	slog.Info("stopping all metrics data sources")
	for _, ms := range m.Sources {
		ms.dataSource.Stop()
	}

	for _, shutdown := range m.shutdownFuncs {
		shutdown()
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
	}
}

// Dynamically populates the `[]*MetricsSource` of `*Metrics`. This is currently a one-to-one mapping of each data type
// to its data source, can later be changed to a one-to-many.
func (m *Metrics) discoverSources() error {
	// Always initialized currently, should dynamically determine if we have a Prometheus source or not.
	// Data sources can also be passed in as options on plugin init.
	if _, ok := m.Sources[prometheus.DataSourceType]; !ok {
		prom := prometheus.NewScraper()
		prom.Register(m.prod.RecordMetrics)
		m.Sources[prometheus.DataSourceType] = &MetricsSource{
			dataSource: prom,
			meter:      otel.GetMeterProvider().Meter(prometheus.DataSourceType, metric.WithInstrumentationVersion(m.conf.Version)),
		}
	}

	return nil
}

// Calls Start() on each data source.
func (m *Metrics) startDataSources() error {
	for _, metricsSrc := range m.Sources {
		go func(ds DataSource) {
			err := ds.Start(m.pipe.Context(), m.conf.Metrics.ReportInterval)
			if err != nil {
				// Probably need to figure out how to handle these errors better (e.g. with a `chan error`).
				slog.Error("failed to start metrics source", "error", err)
			}
		}(metricsSrc.dataSource)
	}
	return nil
}

//##########################
//## Begin MetricsOptions ##
//##########################

// Appends a Metrics data source that will automatically collect metrics.
func WithDataSource(ds DataSource) MetricsOption {
	return func(m *Metrics) error {
		m.Sources[ds.Type()] = &MetricsSource{
			dataSource: ds,
			meter:      otel.GetMeterProvider().Meter(prometheus.DataSourceType, metric.WithInstrumentationVersion(m.conf.Version)),
		}
		return nil
	}
}

//########################
//## End MetricsOptions ##
//########################
