/**
 * Copyright (c) F5, Inc.
 *
 * This source code is licensed under the Apache License, Version 2.0 license found in the
 * LICENSE file in the root directory of this source tree.
 */

package metric

import (
	"context"
	"testing"
	"time"

	"github.com/nginx/agent/v3/internal/config"
	"github.com/stretchr/testify/assert"
)

func TestMeterProvider_Constructor(t *testing.T) {
	producer := NewMetricsProducer()

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
	producer := NewMetricsProducer()

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
