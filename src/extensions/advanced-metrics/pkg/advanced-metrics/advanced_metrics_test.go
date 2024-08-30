/**
 * Copyright (c) F5, Inc.
 *
 * This source code is licensed under the Apache License, Version 2.0 license found in the
 * LICENSE file in the root directory of this source tree.
 */
package advanced_metrics

import (
	"context"
	"testing"
	"time"

	"github.com/nginx/agent/v2/src/extensions/advanced-metrics/tables/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewAdvancedMetrics(t *testing.T) {
	config := Config{
		Address: "test_address",
		AggregatorConfig: AggregatorConfig{
			AggregationPeriod: 1 * time.Minute,
			PublishingPeriod:  2 * time.Minute,
		},
		TableSizesLimits: TableSizesLimits{
			StagingTableMaxSize:    1000,
			StagingTableThreshold:  500,
			PriorityTableMaxSize:   1000,
			PriorityTableThreshold: 500,
		},
	}

	schema := &schema.Schema{}

	advancedMetrics, err := NewAdvancedMetrics(config, schema)
	require.NoError(t, err, "Failed to create AdvancedMetrics instance")
	assert.NotNil(t, advancedMetrics, "AdvancedMetrics instance should not be nil")

	assert.Equal(t, config, advancedMetrics.config, "Config should match")
	assert.NotNil(t, advancedMetrics.metricsChannel, "metricsChannel should not be nil")
	assert.NotNil(t, advancedMetrics.publisher, "publisher should not be nil")
	assert.NotNil(t, advancedMetrics.reader, "reader should not be nil")
	assert.NotNil(t, advancedMetrics.ingester, "ingester should not be nil")
	assert.NotNil(t, advancedMetrics.aggregator, "aggregator should not be nil")
}

func TestNewAdvancedMetrics_Failure(t *testing.T) {
	// Invalid TableSizesLimits to trigger error
	config := Config{
		Address: "test_address",
		AggregatorConfig: AggregatorConfig{
			AggregationPeriod: 1 * time.Minute,
			PublishingPeriod:  2 * time.Minute,
		},
		TableSizesLimits: TableSizesLimits{
			StagingTableMaxSize:    -1, // Invalid size
			StagingTableThreshold:  500,
			PriorityTableMaxSize:   1000,
			PriorityTableThreshold: 500,
		},
	}

	schema := &schema.Schema{}

	advancedMetrics, err := NewAdvancedMetrics(config, schema)
	require.Error(t, err, "Expected error due to invalid table sizes limits")
	assert.Nil(t, advancedMetrics, "AdvancedMetrics instance should be nil")
}

func TestAdvancedMetrics_OutChannel(t *testing.T) {
	config := Config{
		Address: "test_address",
		AggregatorConfig: AggregatorConfig{
			AggregationPeriod: 1 * time.Minute,
			PublishingPeriod:  2 * time.Minute,
		},
		TableSizesLimits: TableSizesLimits{
			StagingTableMaxSize:    1000,
			StagingTableThreshold:  500,
			PriorityTableMaxSize:   1000,
			PriorityTableThreshold: 500,
		},
	}

	schema := &schema.Schema{}

	advancedMetrics, err := NewAdvancedMetrics(config, schema)
	require.NoError(t, err, "Failed to create AdvancedMetrics instance")

	outChannel := advancedMetrics.OutChannel()
	assert.NotNil(t, outChannel, "OutChannel should not be nil")
}

func TestAdvancedMetrics_Run(t *testing.T) {
	config := Config{
		Address: "test_address",
		AggregatorConfig: AggregatorConfig{
			AggregationPeriod: 1 * time.Second,
			PublishingPeriod:  2 * time.Second,
		},
		TableSizesLimits: TableSizesLimits{
			StagingTableMaxSize:    1000,
			StagingTableThreshold:  500,
			PriorityTableMaxSize:   1000,
			PriorityTableThreshold: 500,
		},
	}

	schema := &schema.Schema{}

	advancedMetrics, err := NewAdvancedMetrics(config, schema)
	require.NoError(t, err, "Failed to create AdvancedMetrics instance")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Run AdvancedMetrics in a separate goroutine to avoid blocking
	go func() {
		err := advancedMetrics.Run(ctx)
		assert.NoError(t, err, "Run should not return an error")
	}()

	// Allow some time for goroutines to start
	time.Sleep(500 * time.Millisecond)

	// After short delay, cancel the context to stop the run
	cancel()

	// Wait for all goroutines to finish
	time.Sleep(500 * time.Millisecond)
}
