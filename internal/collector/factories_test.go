// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.
package collector

import (
	"github.com/stretchr/testify/require"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestOTelComponentFactoriesDefault(t *testing.T) {
	factories, err := OTelComponentFactories()

	require.NoError(t, err, "OTelComponentFactories should not return an error")
	assert.NotNil(t, factories, "factories should not be nil")

	assert.Len(t, factories.Receivers, 4)
	assert.Len(t, factories.Processors, 20)
	assert.Len(t, factories.Exporters, 4)
	assert.Len(t, factories.Extensions, 3)
	assert.Empty(t, factories.Connectors)
}

func TestOTelComponentFactories(t *testing.T) {
	tests := []struct {
		name           string
		receiverCount  int
		processorCount int
		exporterCount  int
		extensionCount int
		connectorCount int
	}{
		{
			name:           "Test 1: Defaults",
			receiverCount:  4,
			processorCount: 20,
			exporterCount:  4,
			extensionCount: 3,
			connectorCount: 0,
		},
		{
			name:           "Test 2: All 0",
			receiverCount:  0,
			processorCount: 0,
			exporterCount:  0,
			extensionCount: 0,
			connectorCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			factories, err := OTelComponentFactories()

			require.NoError(t, err, "OTelComponentFactories should not return an error")
			assert.NotNil(t, factories, "factories should not be nil")

			if tt.receiverCount == 0 {
				factories.Receivers = nil
			}
			if tt.processorCount == 0 {
				factories.Processors = nil
			}
			if tt.exporterCount == 0 {
				factories.Exporters = nil
			}
			if tt.extensionCount == 0 {
				factories.Extensions = nil
			}
			if tt.connectorCount == 0 {
				factories.Connectors = nil
			}

			assert.Len(t, factories.Receivers, tt.receiverCount)
			assert.Len(t, factories.Processors, tt.processorCount)
			assert.Len(t, factories.Exporters, tt.exporterCount)
			assert.Len(t, factories.Extensions, tt.extensionCount)
			assert.Len(t, factories.Connectors, tt.connectorCount)
		})
	}
}
