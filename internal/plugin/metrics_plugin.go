// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.
package plugin

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"github.com/nginx/agent/v3/internal/bus"
	"github.com/nginx/agent/v3/internal/config"
	"github.com/nginx/agent/v3/internal/metrics/export"
	"github.com/nginx/agent/v3/internal/metrics/source/prometheus"
	"github.com/nginx/agent/v3/internal/model"
)

const (
	metricsInfo             = "metrics"
	maxProducerRetries      = 20
	fallbackCollectInterval = 20 * time.Second
)

type (
	// The Metrics plugin. Discovers and owns the data source that produce metrics for the Agent.
	Metrics struct {
		producers       map[model.MetricsSourceType]model.MetricsProducer
		exporters       map[model.ExporterType]model.Exporter
		config          *config.Config
		pipe            bus.MessagePipeInterface
		shutdownFuncs   []context.CancelFunc
		collectInterval time.Duration
		metricsTimer    *time.Ticker
	}

	// MetricsOption a functional option for `*Metrics`.
	MetricsOption func(*Metrics) error
)

// NewMetrics is the constructor for the Metrics plugin.
func NewMetrics(conf *config.Config, options ...MetricsOption) (*Metrics, error) {
	if conf.Metrics == nil {
		return nil, fmt.Errorf("metrics configuration cannot be nil")
	}

	produceInterval := conf.Metrics.ProduceInterval
	if produceInterval <= 0 {
		slog.Warn("Invalid metrics produce interval configured: using default",
			"configured", produceInterval, "default", config.DefMetricsProduceInterval,
		)
		produceInterval = config.DefMetricsProduceInterval
	}

	metrics := Metrics{
		producers:       make(map[model.MetricsSourceType]model.MetricsProducer),
		exporters:       make(map[model.ExporterType]model.Exporter),
		config:          conf,
		pipe:            nil,
		collectInterval: produceInterval,
		shutdownFuncs:   make([]context.CancelFunc, 0),
		metricsTimer:    nil,
	}

	for _, opt := range options {
		err := opt(&metrics)
		if err != nil {
			return nil, fmt.Errorf("failed to apply metrics plugin option: %w", err)
		}
	}

	return &metrics, nil
}

// Init initializes and starts the plugin. Required for the `Plugin` interface.
func (m *Metrics) Init(ctx context.Context, mp bus.MessagePipeInterface) error {
	slog.Debug("Starting metrics plugin")

	m.pipe = mp
	metricsCtx, cancel := context.WithCancel(ctx)
	m.shutdownFuncs = append(m.shutdownFuncs, cancel)

	m.discoverSources()

	err := m.createExporters(metricsCtx)
	if err != nil {
		return fmt.Errorf("could not start exporters: %w", err)
	}

	m.startExporters(metricsCtx)

	for srcType, producer := range m.producers {
		slog.Info("Starting producer", "producer_type", srcType.String())
		go m.runProducer(metricsCtx, producer)
	}

	return nil
}

// Info about the plugin. Required for the `Plugin` interface.
func (m *Metrics) Info() *bus.Info {
	return &bus.Info{
		Name: metricsInfo,
	}
}

// Close about the plugin. Required for the `Plugin` interface.
func (m *Metrics) Close(_ context.Context) error {
	slog.Debug("Closing metrics plugin")
	for _, shutdownFunc := range m.shutdownFuncs {
		shutdownFunc()
	}

	m.shutdownFuncs = nil
	m.producers = nil
	m.exporters = nil

	return nil
}

// Process an incoming Message Bus message in the plugin. Required for the `Plugin` interface.
func (m *Metrics) Process(_ context.Context, msg *bus.Message) {
	switch msg.Topic {
	case bus.MetricsTopic:
		err := m.processMessage(msg)
		if err != nil {
			slog.Debug("error processing message", "err", err)
		}
	case bus.OsProcessesTopic:
		slog.Debug("OS Processes have been updated")
	}
}

// Subscriptions returns the list of topics the plugin is subscribed to. Required for the `Plugin` interface.
func (m *Metrics) Subscriptions() []string {
	return []string{
		bus.OsProcessesTopic,
		bus.MetricsTopic,
	}
}

// Dynamically populates `MetricsProducer`s.
func (m *Metrics) discoverSources() {
	// Initialize the first time only if the Prometheus config section is configured.
	if m.config.Metrics.PrometheusSource == nil {
		slog.Debug("Prometheus metrics source not configured: skipping initialization")
		return
	}

	if _, ok := m.producers[model.Prometheus]; !ok {
		m.producers[model.Prometheus] = prometheus.NewScraper(m.config.Metrics.PrometheusSource.Endpoints)
	} else {
		slog.Debug("Prometheus metrics source already initialized")
	}
}

func (m *Metrics) createExporters(ctx context.Context) error {
	if m.config.Metrics.OTelExporter == nil {
		slog.Debug("OTel exporter not configured: skipping exporter instantiation")
		return nil
	}

	// This needs to be unique to the Agent and persistent in the future. Generating a runtime UUID for now.
	// nolint: revive
	id, err := uuid.NewV7()
	if err != nil {
		return fmt.Errorf("failed to generate UUID for OTel Exporter: %w", err)
	}

	if _, ok := m.exporters[model.OTel]; !ok {
		exporter, err := export.NewOTelExporter(
			ctx, m.config, model.Prometheus.String(), id.String(), prometheus.ConvertPrometheus,
		)
		if err != nil {
			return fmt.Errorf("failed to create OTel exporter: %w", err)
		}

		m.exporters[exporter.Type()] = exporter
	}

	return nil
}

func (m *Metrics) startExporters(ctx context.Context) {
	for expType, exp := range m.exporters {
		slog.Info("Starting export", "exporter_type", expType.String())
		go exp.StartSink(ctx)
	}
}

// Produces metrics from the MetricsProducer at the configured interval and pushes to the given channel.
func (m *Metrics) runProducer(ctx context.Context, producer model.MetricsProducer) {
	m.metricsTimer = time.NewTicker(m.collectInterval)

	// this should get moved to the individual producers destruction function
	defer m.metricsTimer.Stop()

	failedAttempts := 0
	for failedAttempts != maxProducerRetries {
		select {
		case <-m.metricsTimer.C:
			failedAttempts = m.callProduce(ctx, producer, failedAttempts)
		case <-ctx.Done():
			slog.Info("MetricsProducer stopped", "producer_type", producer.Type().String())

			return
		}
	}

	if failedAttempts == maxProducerRetries {
		slog.Error("MetricsProducer stopped after max number of retries reached",
			"producer_type", producer.Type().String(), "max_retries", maxProducerRetries)
	}
}

func (m *Metrics) processMessage(msg *bus.Message) error {
	de, ok := msg.Data.(model.DataEntry)
	if !ok {
		return fmt.Errorf("metrics plugin received metrics event but could not cast it to correct type: %v", msg.Data)
	}

	exporter, ok := m.exporters[model.OTel]
	if !ok {
		return fmt.Errorf("metrics plugin received metrics event but source type had no exporter: %v", de.SourceType)
	}
	err := exporter.Export(de)

	return err
}

func (m *Metrics) callProduce(ctx context.Context, producer model.MetricsProducer, failedAttempts int) int {
	entries, err := producer.Produce(ctx)
	if err != nil {
		slog.Debug("MetricsProducer produce call failed", "error", err)

		return failedAttempts + 1
	}
	failedAttempts = 0

	busMsgs := make([]*bus.Message, len(entries))

	for i, e := range entries {
		busMsgs[i] = e.ToBusMessage()
	}

	m.pipe.Process(busMsgs...)

	return failedAttempts
}

// WithDataSource appends a Metrics data source that will automatically collect metrics.
func WithDataSource(ds model.MetricsProducer) MetricsOption {
	return func(m *Metrics) error {
		m.producers[ds.Type()] = ds
		return nil
	}
}
