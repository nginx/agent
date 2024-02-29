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

	"github.com/nginx/agent/v3/internal/bus"
	"github.com/nginx/agent/v3/internal/config"
	"github.com/nginx/agent/v3/internal/metrics/source/prometheus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMetrics_Init(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "")
	})

	// Create a test server using the handler
	fakePrometheus := httptest.NewServer(handler)
	defer fakePrometheus.Close()

	messagePipe := bus.NewMessagePipe(context.TODO(), 100)
	scraper := prometheus.NewScraper([]string{fakePrometheus.URL})

	metrics, err := NewMetrics(testConfig(t), WithDataSource(scraper))
	require.NoError(t, err)

	err = messagePipe.Register(100, []bus.Plugin{metrics})
	require.NoError(t, err)
	go messagePipe.Run()

	time.Sleep(10 * time.Millisecond)

	require.NoError(t, err)

	err = metrics.Close()
	require.NoError(t, err)
}

func TestMetrics_Info(t *testing.T) {
	metrics, err := NewMetrics(testConfig(t))
	require.NoError(t, err)

	i := metrics.Info()
	assert.NotNil(t, i)

	assert.Equal(t, "metrics", i.Name)
}

func TestMetrics_Subscriptions(t *testing.T) {
	metrics, err := NewMetrics(testConfig(t))
	require.NoError(t, err)

	subscriptions := metrics.Subscriptions()
	assert.Equal(t, []string{bus.OsProcessesTopic, bus.MetricsTopic}, subscriptions)
}

func TestMetrics_Process(t *testing.T) {
	metrics, err := NewMetrics(testConfig(t))
	require.NoError(t, err)

	// Payload is ignored.
	metrics.Process(&bus.Message{Topic: bus.OsProcessesTopic, Data: struct {
		valueOne string
		valueTwo string
	}{"one", "two"}})

	// Currently doesn't do anything.
	require.NoError(t, err)
}

func TestMetrics_Errors(t *testing.T) {
	testCases := []struct {
		name        string
		confModFunc func(config.Config) config.Config
		isErr       bool
		expErr      string
	}{
		{
			name: "nil-metrics-configuration",
			confModFunc: func(c config.Config) config.Config {
				c.Metrics = nil

				return c
			},
			isErr:  true,
			expErr: "metrics configuration cannot be nil",
		},
		{
			name: "negative-produce-interval",
			confModFunc: func(c config.Config) config.Config {
				c.Metrics.ProduceInterval = -1

				return c
			},
			isErr: false,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(tt *testing.T) {
			c := test.confModFunc(testConfig(tt))

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

	metrics, err := NewMetrics(testConfig(t))
	require.NoError(t, err)

	// Payload is ignored.
	metrics.Process(&bus.Message{Topic: bus.OsProcessesTopic, Data: struct {
		valueOne string
		valueTwo string
	}{"one", "two"}})

	// Currently doesn't do anything.
	require.NoError(t, err)
}

func testConfig(t *testing.T) config.Config {
	t.Helper()
	return config.Config{
		Version: "0.1",
		Metrics: &config.Metrics{
			ProduceInterval: 5 * time.Second,
			OTelExporter: &config.OTelExporter{
				BufferLength:     10,
				ExportRetryCount: 3,
				ExportInterval:   5 * time.Second,
				GRPC: &config.GRPC{
					Target:         "dummy-target",
					ConnTimeout:    10 * time.Second,
					MinConnTimeout: 7 * time.Second,
					BackoffDelay:   240 * time.Second,
				},
			},
			PrometheusSource: &config.PrometheusSource{
				Endpoints: []string{},
			},
		},
	}
}
