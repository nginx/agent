// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.
package collector

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/nginx/agent/v3/internal/bus"
	"github.com/nginx/agent/v3/internal/config"
)

func TestNewCollector(t *testing.T) {
	conf := setupOTelConfig(t)
	_, err := New(conf)
	require.NoError(t, err, "NewCollector should not return an error with valid config")
}

func TestInitAndClose(t *testing.T) {
	conf := setupOTelConfig(t)
	collector, err := New(conf)
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

func setupOTelConfig(t *testing.T) *config.Config {
	t.Helper()
	return &config.Config{
		Collector: &config.Collector{
			ConfigPath: filepath.Join(t.TempDir(), "otel-collector-config.yaml"),
			Exporters: []config.Exporter{
				{
					Type: "otlp",
					Server: &config.ServerConfig{
						Host: "127.0.0.1",
						Port: 1234,
						Type: 0,
					},
				},
			},
			Receivers: []config.Receiver{
				{
					Type: "otlp",
					Server: &config.ServerConfig{
						Host: "localhost",
						Port: 4321,
						Type: 0,
					},
				},
			},
			Health: &config.ServerConfig{
				Host: "localhost",
				Port: 1337,
				Type: 0,
			},
		},
	}
}
