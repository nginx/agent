/**
 * Copyright (c) F5, Inc.
 *
 * This source code is licensed under the Apache License, Version 2.0 license found in the
 * LICENSE file in the root directory of this source tree.
 */
package metrics

import (
	"context"
	"testing"
	"time"

	"github.com/nginx/agent/v3/internal/config"
	"github.com/stretchr/testify/assert"
)

func TestGRPCExporter_Constructor(t *testing.T) {
	conf := config.Metrics{
		OTelExporterTarget: "dummy-target",
		ReportInterval:     1 * time.Second,
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	_, err := NewGRPCExporter(ctx, conf)
	assert.NoError(t, err)
}

func TestHTTPExporter_Constructor(t *testing.T) {
	conf := config.Metrics{
		OTelExporterTarget: "dummy-target",
		ReportInterval:     1 * time.Second,
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	_, err := NewHTTPExporter(ctx, conf)
	assert.NoError(t, err)
}

func TestMeterProvider_Constructor(t *testing.T) {
	producer := NewMetricsProducer(agentVersion)

	conf := config.Metrics{
		OTelExporterTarget: "dummy-target",
		ReportInterval:     1 * time.Second,
	}

	serviceName := "test-service"
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	_, err := NewMeterProvider(ctx, serviceName, conf, producer)
	assert.NoError(t, err)
}

func TestMeterProvider_BadGRPCTarget(t *testing.T) {
	producer := NewMetricsProducer(agentVersion)

	conf := config.Metrics{
		OTelExporterTarget: "",
		ReportInterval:     1 * time.Second,
	}

	serviceName := "test-service"
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	_, err := NewMeterProvider(ctx, serviceName, conf, producer)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create GRPC Exporter")
}
