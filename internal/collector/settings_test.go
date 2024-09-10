// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.
package collector

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/nginx/agent/v3/internal/config"
	"github.com/nginx/agent/v3/test/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	expectedTemplatePath = "../../test/config/collector/test-opentelemetry-collector-agent.yaml"
	// The log format's double quotes must be escaped so that valid YAML is produced when executing the template.
	accessLogFormat = `$remote_addr - $remote_user [$time_local] \"$request\" $status $body_bytes_sent ` +
		`\"$http_referer\" \"$http_user_agent\" \"$http_x_forwarded_for\"\"$upstream_cache_status\"`
)

func TestOTelCollectorSettings(t *testing.T) {
	cfg := types.AgentConfig()

	settings := OTelCollectorSettings(cfg)

	assert.NotNil(t, settings.Factories, "Factories should not be nil")
	assert.Equal(t, "otel-nginx-agent", settings.BuildInfo.Command, "BuildInfo command should match")
	assert.False(t, settings.DisableGracefulShutdown, "DisableGracefulShutdown should be false")
	assert.NotNil(t, settings.ConfigProviderSettings, "ConfigProviderSettings should not be nil")
	assert.Nil(t, settings.LoggingOptions, "LoggingOptions should be nil")
	assert.False(t, settings.SkipSettingGRPCLogger, "SkipSettingGRPCLogger should be false")
}

func TestConfigProviderSettings(t *testing.T) {
	cfg := types.AgentConfig()
	settings := ConfigProviderSettings(cfg)

	assert.NotNil(t, settings.ResolverSettings, "ResolverSettings should not be nil")
	assert.Len(t, settings.ResolverSettings.ProviderFactories, 5, "There should be 5 provider factories")
	assert.Empty(t, settings.ResolverSettings.ConverterFactories, "There should be 0 converter factory")
	assert.NotEmpty(t, settings.ResolverSettings.URIs, "URIs should not be empty")
	assert.Equal(t, "/etc/nginx-agent/nginx-agent-otelcol.yaml", settings.ResolverSettings.URIs[0],
		"Default URI should match")
}

func TestTemplateWrite(t *testing.T) {
	cfg := types.AgentConfig()
	actualConfPath := filepath.Join("/tmp/", "nginx-agent-otelcol-test.yaml")
	cfg.Collector.ConfigPath = actualConfPath

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

	cfg.Collector.Receivers.HostMetrics = config.HostMetrics{
		CollectionInterval: time.Minute,
		InitialDelay:       time.Second,
		Scrapers: map[string]struct{}{
			"cpu":        {},
			"disk":       {},
			"filesystem": {},
			"memory":     {},
			"network":    {},
		},
	}
	cfg.Collector.Receivers.NginxReceivers = append(cfg.Collector.Receivers.NginxReceivers, config.NginxReceiver{
		InstanceID: "123",
		StubStatus: "http://localhost:80/status",
		AccessLogs: []config.AccessLog{
			{
				LogFormat: accessLogFormat,
				FilePath:  "/var/log/nginx/access-custom.conf",
			},
		},
	})

	require.NotNil(t, cfg)

	err := writeCollectorConfig(cfg.Collector)
	require.NoError(t, err)

	expected, err := os.ReadFile(expectedTemplatePath)
	require.NoError(t, err)

	actual, err := os.ReadFile(actualConfPath)
	require.NoError(t, err)

	// Convert to string for human readable error messages.
	assert.Equal(t, string(expected), string(actual))
}
