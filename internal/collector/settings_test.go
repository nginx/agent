// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.
package collector

import (
	"path/filepath"
	"testing"

	"github.com/nginx/agent/v3/internal/config"
	"github.com/nginx/agent/v3/test/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOTelCollectorSettings(t *testing.T) {
	cfg := types.GetAgentConfig()

	settings := OTelCollectorSettings(cfg)

	assert.NotNil(t, settings.Factories, "Factories should not be nil")
	assert.Equal(t, "otel-nginx-agent", settings.BuildInfo.Command, "BuildInfo command should match")
	assert.False(t, settings.DisableGracefulShutdown, "DisableGracefulShutdown should be false")
	assert.NotNil(t, settings.ConfigProviderSettings, "ConfigProviderSettings should not be nil")
	assert.Nil(t, settings.LoggingOptions, "LoggingOptions should be nil")
	assert.False(t, settings.SkipSettingGRPCLogger, "SkipSettingGRPCLogger should be false")
}

func TestConfigProviderSettings(t *testing.T) {
	cfg := types.GetAgentConfig()
	settings := ConfigProviderSettings(cfg)

	assert.NotNil(t, settings.ResolverSettings, "ResolverSettings should not be nil")
	assert.Len(t, settings.ResolverSettings.ProviderFactories, 5, "There should be 5 provider factories")
	assert.Len(t, settings.ResolverSettings.ConverterFactories, 1, "There should be 1 converter factory")
	assert.NotEmpty(t, settings.ResolverSettings.URIs, "URIs should not be empty")
	assert.Equal(t, "/var/etc/nginx-agent/nginx-agent-otelcol.yaml", settings.ResolverSettings.URIs[0],
		"Default URI should match")
}

func TestTemplateWrite(t *testing.T) {
	cfg := types.GetAgentConfig()
	cfg.Collector.ConfigPath = filepath.Join(t.TempDir(), "nginx-agent-otelcol-test.yaml")
	// cfg.Collector.ConfigPath = "/tmp/nginx-agent-otelcol-test.yaml"

	cfg.Collector.Exporters = append(cfg.Collector.Exporters, config.Exporter{
		Type: "prometheus",
		Server: &config.ServerConfig{
			Host: "localhost",
			Port: 9876,
			Type: 0,
		},
		Auth: nil, // Auth and TLS not supported yet.
		TLS:  nil,
	}, config.Exporter{
		Type:   "debug",
		Server: nil, // not relevant to the debug exporter
		Auth:   nil,
		TLS:    nil,
	})

	cfg.Collector.Receivers = append(cfg.Collector.Receivers, config.Receiver{
		Type:   "hostmetrics",
		Server: nil, // not relevant to hostmetrics receiver.
		Auth:   nil,
		TLS:    nil,
	}, config.Receiver{
		Type: "prometheus",
		Server: &config.ServerConfig{
			Host: "192.168.200.15",
			Port: 7765,
			Type: 0,
		},
		Auth: nil, // Auth and TLS not supported yet.
		TLS:  nil,
	})

	require.NotNil(t, cfg)

	err := writeCollectorConfig(cfg.Collector)
	require.NoError(t, err)
}
