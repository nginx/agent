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
	"time"

	"github.com/nginx/agent/v3/internal/config"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.17.0"
)

const (
	// tenantID is static for now.
	tenantID         = "7332d596-d2e6-4d1e-9e75-70f91ef9bd0e"
	serviceNamespace = "nginx"
	readInterval     = 10 * time.Second
)

// Constructs an OTel MeterProvider that generates metrics from the given `producer` every 10 seconds and exports
// them via gRPC to an OTel Collector.
func NewMeterProvider(ctx context.Context, serviceName string, c config.Metrics, producer *MetricsProducer) (*metric.MeterProvider, error) {
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
