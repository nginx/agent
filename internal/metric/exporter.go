/**
 * Copyright (c) F5, Inc.
 *
 * This source code is licensed under the Apache License, Version 2.0 license found in the
 * LICENSE file in the root directory of this source tree.
 */

package metric

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/nginx/agent/v3/internal/config"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.17.0"
	"google.golang.org/grpc"
	"google.golang.org/grpc/backoff"
	"google.golang.org/grpc/credentials/insecure"
)

const (
	// tenantID is static for now.
	tenantID         = "7332d596-d2e6-4d1e-9e75-70f91ef9bd0e"
	serviceNamespace = "nginx"
)

// Returns a OTel exporter that transmits via gRPC.
func NewGRPCExporter(ctx context.Context, conf config.Metrics) (*otlpmetricgrpc.Exporter, error) {
	ctx, cancel := context.WithTimeout(ctx, time.Second*10)
	defer cancel()
	// Exponential back-off strategy.
	backoffConf := backoff.DefaultConfig
	// You can also change the base delay, multiplier, and jitter here.
	backoffConf.MaxDelay = 240 * time.Second

	conn, err := grpc.DialContext(
		ctx,
		conf.OTelExporterTarget,
		// TODO: Add TLS support
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithConnectParams(grpc.ConnectParams{
			Backoff: backoffConf,
			// Connection timeout.
			MinConnectTimeout: 5 * time.Second,
		}),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create grpc connection: %v", err)
	}

	return otlpmetricgrpc.New(ctx, otlpmetricgrpc.WithGRPCConn(conn), otlpmetricgrpc.WithTimeout(7*time.Second))
}

// Returns a OTel exporter that transmits via HTTP.
func NewHTTPExporter(ctx context.Context, conf config.Metrics) (*otlpmetrichttp.Exporter, error) {
	target := "0.0.0.0:4317"
	if value, ok := os.LookupEnv("HTTP_TARGET"); ok {
		target = value
	}

	return otlpmetrichttp.New(
		ctx,
		// TODO: Add TLS support
		otlpmetrichttp.WithInsecure(),
		otlpmetrichttp.WithEndpoint(target),
		// WithTimeout sets the max amount of time the Exporter will attempt an
		// export.
		otlpmetrichttp.WithTimeout(7*time.Second),
		otlpmetrichttp.WithRetry(otlpmetrichttp.RetryConfig{
			// Enabled indicates whether to not retry sending batches in case
			// of export failure.
			Enabled: true,
			// InitialInterval the time to wait after the first failure before
			// retrying.
			InitialInterval: 1 * time.Second,
			// MaxInterval is the upper bound on backoff interval. Once this
			// value is reached the delay between consecutive retries will
			// always be `MaxInterval`.
			MaxInterval: 10 * time.Second,
			// MaxElapsedTime is the maximum amount of time (including retries)
			// spent trying to send a request/batch. Once this value is
			// reached, the data is discarded.
			MaxElapsedTime: 240 * time.Second,
		}),
	)
}

// Constructs a OTel MeterProvider that generates metrics from the given `producer` every 10 seconds and exports
// them via gRPC to an OTel Collector.
func NewMeterProvider(
	ctx context.Context, serviceName string, c config.Metrics, producer *MetricsProducer,
) (*metric.MeterProvider, error) {
	exp, err := NewGRPCExporter(ctx, c)
	if err != nil {
		return nil, fmt.Errorf("failed to create GRPC Exporter: %w", err)
	}

	res, err := resource.New(ctx,
		// Keep the default detectors
		resource.WithTelemetrySDK(),
		// Add your own custom attributes to identify your application
		resource.WithAttributes(
			semconv.ServiceNamespaceKey.String(serviceNamespace),
			semconv.ServiceNameKey.String(serviceName),
			semconv.ServiceInstanceIDKey.String(tenantID),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create new resource: %w", err)
	}

	// Override the default 60 second read interval.
	reader := metric.NewPeriodicReader(exp, metric.WithInterval(c.ReportInterval), metric.WithProducer(producer))

	meterProvider := metric.NewMeterProvider(
		metric.WithResource(res),
		metric.WithReader(reader),
	)

	return meterProvider, nil
}
