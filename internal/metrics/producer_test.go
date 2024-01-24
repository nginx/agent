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

	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/otel/sdk/instrumentation"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
)

const agentVersion = "v0.1"

func TestMetricsProducer_Constructor(t *testing.T) {
	mp := NewMetricsProducer(agentVersion)

	assert.Equal(t, []metricdata.Metrics{}, mp.metrics)
}

func TestMetricsProducer_Produce(t *testing.T) {
	mp := NewMetricsProducer(agentVersion)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go mp.StartListen(ctx)

	testdata := metricdata.Metrics{
		Name:        "test-metric",
		Description: "test-description",
		Unit:        "test-unit",
		Data:        nil,
	}

	mp.RecordMetrics(testdata)

	assert.Len(t, mp.metrics, 1)
	assert.Equal(t, testdata, mp.metrics[0])

	sm, err := mp.Produce(ctx)
	assert.NoError(t, err)

	assert.Equal(t, []metricdata.ScopeMetrics{
		{
			Scope: instrumentation.Scope{
				Name:    "github.com/agent/v3",
				Version: "v0.1",
			},
			Metrics: []metricdata.Metrics{testdata},
		},
	}, sm)
}
