// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.
package export

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/nginx/agent/v3/internal/config"
	"github.com/nginx/agent/v3/internal/model"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/sdk/instrumentation"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.17.0"
	"google.golang.org/grpc"
	"google.golang.org/grpc/backoff"
	"google.golang.org/grpc/credentials/insecure"
)

const (
	agentScope       = "github.com/agent/v3"
	serviceNamespace = "nginx"
)

type OTelExporter struct {
	conf           *config.Config
	intExp         *otlpmetricgrpc.Exporter
	convert        model.Converter[metricdata.Metrics]
	bufferMutex    *sync.Mutex
	sink           chan model.DataEntry
	buffer         []metricdata.Metrics
	bufferLen      int
	retryCount     int
	exportInterval time.Duration
	res            *resource.Resource
	scope          instrumentation.Scope
}

// NewGRPCExporter returns a OTel export that transmits via gRPC.
func NewGRPCExporter(ctx context.Context, conf *config.GRPC) (*otlpmetricgrpc.Exporter, error) {
	ctx, cancel := context.WithTimeout(ctx, conf.ConnTimeout)
	defer cancel()
	// Exponential back-off strategy.
	backoffConf := backoff.DefaultConfig
	// You can also change the base delay, multiplier, and jitter here.
	backoffConf.MaxDelay = conf.BackoffDelay

	conn, err := grpc.DialContext(
		ctx,
		conf.Target,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithConnectParams(grpc.ConnectParams{
			Backoff: backoffConf,
			// Connection timeout.
			MinConnectTimeout: conf.MinConnTimeout,
		}),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create grpc connection: %w", err)
	}

	return otlpmetricgrpc.New(
		ctx, otlpmetricgrpc.WithGRPCConn(conn), otlpmetricgrpc.WithTimeout(conf.ConnTimeout),
	)
}

func NewOTelExporter(ctx context.Context, agentConf *config.Config, serviceName, id string,
	c model.Converter[metricdata.Metrics],
) (*OTelExporter, error) {
	exp, err := initGPRCExporter(ctx, agentConf.Metrics)
	if err != nil {
		return nil, fmt.Errorf("failed to instantiate gRPC exporter for OTel collector: %w", err)
	}

	scope := instrumentation.Scope{
		Name:    agentScope,
		Version: agentConf.Version,
	}

	res, err := resource.New(ctx,
		// Keep the default detectors
		resource.WithTelemetrySDK(),
		// Add your own custom attributes to identify your application
		resource.WithAttributes(
			semconv.ServiceNamespaceKey.String(serviceNamespace),
			semconv.ServiceNameKey.String(serviceName),
			semconv.ServiceInstanceIDKey.String(id),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to instantiate a new OTel resource: %w", err)
	}

	bufferLength := agentConf.Metrics.OTelExporter.BufferLength
	if bufferLength < 1 {
		slog.Warn("Invalid OTel exporter buffer length configured: using default",
			"configured", bufferLength, "default", config.DefOTelExporterBufferLength,
		)
		bufferLength = config.DefOTelExporterBufferLength
	}

	retryCount := agentConf.Metrics.OTelExporter.ExportRetryCount
	if retryCount < 1 {
		slog.Warn("Invalid OTel exporter export retry count configured: using default",
			"configured", retryCount, "default", config.DefOTelExporterExportRetryCount,
		)
		retryCount = config.DefOTelExporterExportRetryCount
	}

	exportInterval := agentConf.Metrics.OTelExporter.ExportInterval
	if exportInterval <= 0 {
		slog.Warn("Invalid OTel exporter export interval configured: using default",
			"configured", exportInterval, "default", config.DefOTelExporterExportInterval,
		)
		exportInterval = config.DefOTelExporterExportInterval
	}

	return &OTelExporter{
		conf:           agentConf,
		intExp:         exp,
		convert:        c,
		bufferMutex:    &sync.Mutex{},
		sink:           make(chan model.DataEntry),
		buffer:         make([]metricdata.Metrics, 0, bufferLength),
		bufferLen:      bufferLength,
		retryCount:     retryCount,
		exportInterval: exportInterval,
		res:            res,
		scope:          scope,
	}, nil
}

func (oe *OTelExporter) StartSink(ctx context.Context) {
	ttlTicker := time.NewTicker(oe.exportInterval)

	for {
		select {
		case <-ctx.Done():
			slog.Info("OTelExporter shutting down")
			return
		case <-ttlTicker.C:
			err := oe.sendBuffer(ctx)
			if err != nil {
				slog.Error("Failed to send buffer")
			}
		case entry := <-oe.sink:
			oe.processEntry(ctx, entry)
		}
	}
}

// Export implements the `Exporter` interface.
// nolint: unparam
func (oe *OTelExporter) Export(entry model.DataEntry) error {
	oe.sink <- entry
	return nil
}

func (oe *OTelExporter) Type() model.ExporterType {
	return model.OTel
}

func (oe *OTelExporter) sendBuffer(ctx context.Context) error {
	scopeMetrics := []metricdata.ScopeMetrics{
		{
			Scope:   oe.scope,
			Metrics: oe.buffer,
		},
	}
	rm := &metricdata.ResourceMetrics{
		Resource:     oe.res,
		ScopeMetrics: scopeMetrics,
	}

	slog.Debug("Exporting metrics", "metrics_count", len(oe.buffer))
	i := 0
	for err := oe.intExp.Export(ctx, rm); i < oe.retryCount && err != nil; i++ {
		slog.Debug("Retrying OTel export after failure", "failed_attempts", i+1, "error", err)
		err = oe.intExp.Export(ctx, rm)
	}

	if i == oe.retryCount {
		return fmt.Errorf("exporting OTel metrics failed after %d attempts", oe.retryCount)
	}

	slog.Debug("Emptying OTel export buffer")
	oe.bufferMutex.Lock()
	oe.buffer = make([]metricdata.Metrics, 0, oe.bufferLen)
	oe.bufferMutex.Unlock()

	return nil
}

// Exports contents of the export's buffer and flushes if the buffer is full and then adds the given data entry to
// the buffer.
func (oe *OTelExporter) processEntry(ctx context.Context, de model.DataEntry) {
	if len(oe.buffer) == oe.bufferLen {
		err := oe.sendBuffer(ctx)
		if err != nil {
			slog.Error("Failed to send buffer, one metric entry dropped!", "error", err)

			return
		}
	}

	metric, convErr := oe.convert(de)
	if convErr != nil {
		slog.Warn("Failed to convert internal data entry to OTel metric",
			"data_entry", de, "error", convErr,
		)

		return
	}

	oe.bufferMutex.Lock()
	oe.buffer = append(oe.buffer, metric)
	oe.bufferMutex.Unlock()
}

func (oe *OTelExporter) getBuffer() []metricdata.Metrics {
	oe.bufferMutex.Lock()
	defer oe.bufferMutex.Unlock()

	return oe.buffer
}

func initGPRCExporter(ctx context.Context, conf *config.Metrics) (*otlpmetricgrpc.Exporter, error) {
	if conf == nil {
		return nil, fmt.Errorf("metrics configuration missing")
	}

	if conf.OTelExporter == nil {
		return nil, fmt.Errorf("OTel Exporter configuration missing")
	}

	if conf.OTelExporter.GRPC == nil {
		return nil, fmt.Errorf("gRPC configuration missing")
	}

	return NewGRPCExporter(ctx, conf.OTelExporter.GRPC)
}
