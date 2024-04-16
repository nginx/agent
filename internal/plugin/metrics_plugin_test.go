// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.
package plugin

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/nginx/agent/v3/internal/model/modelfakes"
	"github.com/nginx/agent/v3/test/types"

	"github.com/nginx/agent/v3/internal/model"

	"github.com/nginx/agent/v3/internal/bus"
	"github.com/nginx/agent/v3/internal/config"
	"github.com/nginx/agent/v3/internal/metrics/source/prometheus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMetrics_Init(t *testing.T) {
	ctx := context.Background()
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "")
	})

	// Create a test server using the handler
	fakePrometheus := httptest.NewServer(handler)
	defer fakePrometheus.Close()

	messagePipe := bus.NewMessagePipe(100)
	scraper := prometheus.NewScraper([]string{fakePrometheus.URL})

	metrics, err := NewMetrics(types.GetAgentConfig(), WithDataSource(scraper))
	require.NoError(t, err)

	err = messagePipe.Register(ctx, 100, []bus.Plugin{metrics})
	require.NoError(t, err)
	go messagePipe.Run(ctx)

	time.Sleep(10 * time.Millisecond)

	require.NoError(t, err)

	err = metrics.Close(ctx)
	require.NoError(t, err)
}

func TestMetrics_Info(t *testing.T) {
	metrics, err := NewMetrics(types.GetAgentConfig())
	require.NoError(t, err)

	i := metrics.Info()
	assert.NotNil(t, i)

	assert.Equal(t, "metrics", i.Name)
}

func TestMetrics_Subscriptions(t *testing.T) {
	metrics, err := NewMetrics(types.GetAgentConfig())
	require.NoError(t, err)

	subscriptions := metrics.Subscriptions()
	assert.Equal(t, []string{bus.OsProcessesTopic, bus.MetricsTopic}, subscriptions)
}

func TestMetrics_ProcessMessage(t *testing.T) {
	metrics, err := NewMetrics(types.GetAgentConfig())
	require.NoError(t, err)

	dataPoint := model.DataPoint{
		Name:   "value1",
		Labels: make(map[string]string),
		Value:  2,
	}

	invalidData := struct {
		valueOne string
		valueTwo string
	}{
		"one", "two",
	}

	dataEntry := model.DataEntry{
		Name:        "Test1",
		Type:        model.Counter,
		SourceType:  model.Prometheus,
		Description: "testing",
		Values: []model.DataPoint{
			dataPoint,
		},
	}

	tests := []struct {
		name        string
		topic       string
		data        bus.Payload
		expectError error
	}{
		{
			name:  "Test 1: Can not cast data",
			topic: bus.MetricsTopic,
			data:  invalidData,
			expectError: fmt.Errorf("metrics plugin received metrics event but could not cast it to correct "+
				"type: %v", invalidData),
		},
		{
			name:  "Test 2: No exporter",
			topic: bus.MetricsTopic,
			data:  dataEntry,
			expectError: fmt.Errorf("metrics plugin received metrics event but source type had no exporter"+
				": %v", dataEntry.SourceType),
		},
		{
			name:        "exporter",
			topic:       bus.MetricsTopic,
			data:        dataEntry,
			expectError: nil,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			exporter := modelfakes.FakeExporter{}
			exporter.ExportReturns(nil)
			if test.name == "exporter" {
				metrics.exporters[model.OTel] = &exporter
			}

			err = metrics.processMessage(&bus.Message{Topic: test.topic, Data: test.data})

			assert.Equal(t, test.expectError, err)
		})
	}
}

func TestMetrics_CallProduce(t *testing.T) {
	dataPoint := model.DataPoint{
		Name:   "value1",
		Labels: make(map[string]string),
		Value:  2,
	}
	dataEntry := model.DataEntry{
		Name:        "Test1",
		Type:        model.Counter,
		SourceType:  model.Prometheus,
		Description: "testing",
		Values: []model.DataPoint{
			dataPoint,
		},
	}

	tests := []struct {
		name                   string
		entries                []model.DataEntry
		expectedFailedAttempts int
		expectedNumMessage     int
		expectedProduceError   error
	}{
		{
			name:                   "Test 1: Failed to call producer",
			entries:                nil,
			expectedFailedAttempts: 1,
			expectedNumMessage:     0,
			expectedProduceError:   fmt.Errorf("produce error"),
		},
		{
			name: "Test 2: Successfully called producer",
			entries: []model.DataEntry{
				dataEntry,
			},
			expectedProduceError:   nil,
			expectedNumMessage:     1,
			expectedFailedAttempts: 0,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			messagePipe := bus.NewFakeMessagePipe()
			metrics, err := NewMetrics(types.GetAgentConfig())
			metrics.pipe = messagePipe
			require.NoError(t, err)

			producer := modelfakes.FakeMetricsProducer{}

			producer.ProduceReturns(test.entries, test.expectedProduceError)
			failedAttempts := metrics.callProduce(context.Background(), &producer, 0)

			assert.Len(t, messagePipe.GetMessages(), test.expectedNumMessage)
			assert.Equal(t, test.expectedFailedAttempts, failedAttempts)
		})
	}
}

func TestMetrics_Errors(t *testing.T) {
	ctx := context.Background()

	testCases := []struct {
		name        string
		confModFunc func(*config.Config) *config.Config
		isErr       bool
		expErr      string
	}{
		{
			name: "Test 1: Metrics configuration is nil",
			confModFunc: func(c *config.Config) *config.Config {
				c.Metrics = nil

				return c
			},
			isErr:  true,
			expErr: "metrics configuration cannot be nil",
		},
		{
			name: "Test 2: Produce interval is a negative value",
			confModFunc: func(c *config.Config) *config.Config {
				c.Metrics.ProduceInterval = -1

				return c
			},
			isErr: false,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(tt *testing.T) {
			c := test.confModFunc(types.GetAgentConfig())

			metrics, err := NewMetrics(c)
			if test.isErr {
				require.Error(t, err)
				require.Nil(t, metrics)
				assert.Contains(t, err.Error(), test.expErr)
			} else {
				require.NoError(tt, err)
				require.NotEmpty(tt, metrics)
			}
		})
	}

	metrics, err := NewMetrics(types.GetAgentConfig())
	require.NoError(t, err)

	// Payload is ignored.
	metrics.Process(ctx, &bus.Message{Topic: bus.OsProcessesTopic, Data: struct {
		valueOne string
		valueTwo string
	}{"one", "two"}})

	// Currently doesn't do anything.
	require.NoError(t, err)
}
