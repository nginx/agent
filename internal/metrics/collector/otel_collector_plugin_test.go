// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.
package collector

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/nginx/agent/v3/internal/bus"
	"github.com/nginx/agent/v3/internal/config"
)

// Example OpenTelemetry Collector configuration YAML embedded as a string
const otelCollectorConfig = `
receivers:
  otlp:
    protocols:
      grpc:
        endpoint: 0.0.0.0:55690

processors:
  batch:

exporters:
  debug:
    verbosity: detailed
    sampling_initial: 5
    sampling_thereafter: 200
  otlp:
    endpoint: ${ENDPOINT}:443
    compression: none
    timeout: 10s
    retry_on_failure:
      enabled: true
      initial_interval: 10s
      max_interval: 60s
      max_elapsed_time: 10m

extensions:
  health_check:
  pprof:
    endpoint: 0.0.0.0:1888

service:
  extensions: [pprof, health_check]
  pipelines:
    metrics:
      receivers: [otlp]
      processors: [batch]
      exporters: [otlp, debug]
`

func TestNewCollector(t *testing.T) {
	configFilePath, removFn, err := setupOTelConfig()
	require.NoError(t, err)
	defer func() {
		require.NoError(t, removFn(configFilePath))
	}()

	conf := &config.Config{}
	_, err = NewCollector(conf)
	require.NoError(t, err, "NewCollector should not return an error with valid config")
}

func TestInitAndClose(t *testing.T) {
	configFilePath, removFn, err := setupOTelConfig()
	require.NoError(t, err)
	defer func() {
		require.NoError(t, removFn(configFilePath))
	}()

	conf := &config.Config{}
	collector, err := NewCollector(conf)
	require.NoError(t, err, "NewCollector should not return an error with valid config")

	ctx := context.Background()
	messagePipe := bus.NewMessagePipe(10)
	err = messagePipe.Register(10, []bus.Plugin{collector})

	require.NoError(t, err)
	require.NoError(t, collector.Init(ctx, messagePipe), "Init should not return an error")

	time.Sleep(time.Second * 5)

	require.NoError(t, collector.Close(ctx), "Close should not return an error")
	select {
	case <-collector.appDone:
		t.Log("Success")
	case <-time.After(10 * time.Second):
		t.Error("Timed out waiting for app to shut down")
	}
}

func setupOTelConfig() (string, func(string) error, error) {
	// Define the path to the configuration file
	configFilePath := "/tmp/otel-collector-config.yaml"

	// Write the configuration to the file using os.WriteFile
	err := os.WriteFile(configFilePath, []byte(otelCollectorConfig), 0o600)
	// assert.NoError(t, err, "should be able to write to the configuration file")

	// Clean up the file afterwards
	return configFilePath, os.Remove, err
}
