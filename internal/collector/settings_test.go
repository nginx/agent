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
	tmpDir := t.TempDir()

	cfg := types.AgentConfig()
	actualConfPath := filepath.Join(tmpDir, "nginx-agent-otelcol-test.yaml")
	cfg.Collector.ConfigPath = actualConfPath
	cfg.Collector.Processors.Resource = map[string]*config.Resource{
		"default": {
			Attributes: []config.ResourceAttribute{
				{
					Key:    "resource.id",
					Action: "add",
					Value:  "12345",
				},
			},
		},
	}

	cfg.Collector.Exporters.PrometheusExporter = &config.PrometheusExporter{
		Server: &config.ServerConfig{
			Host: "localhost",
			Port: 9876,
			Type: config.Grpc,
		},
		TLS: nil,
	}

	cfg.Collector.Exporters.Debug = &config.DebugExporter{}

	cfg.Collector.Receivers.ContainerMetrics = &config.ContainerMetricsReceiver{
		CollectionInterval: time.Second,
	}

	cfg.Collector.Receivers.HostMetrics = &config.HostMetrics{
		CollectionInterval: time.Minute,
		InitialDelay:       time.Second,
		Scrapers: &config.HostMetricsScrapers{
			CPU:        &config.CPUScraper{},
			Disk:       &config.DiskScraper{},
			Filesystem: &config.FilesystemScraper{},
			Memory:     &config.MemoryScraper{},
			Network:    &config.NetworkScraper{},
		},
	}
	cfg.Collector.Receivers.NginxReceivers = append(cfg.Collector.Receivers.NginxReceivers, config.NginxReceiver{
		InstanceID: "123",
		StubStatus: config.APIDetails{
			URL:      "http://localhost:80/status",
			Location: "",
			Listen:   "",
		},
		CollectionInterval: 30 * time.Second,
		AccessLogs: []config.AccessLog{
			{
				LogFormat: accessLogFormat,
				FilePath:  "/var/log/nginx/access-custom.conf",
			},
		},
	})
	// Clear default config and test collector with TLS enabled
	cfg.Collector.Receivers.OtlpReceivers["default"] = &config.OtlpReceiver{
		Server: &config.ServerConfig{
			Host: "localhost",
			Port: 4317,
			Type: config.Grpc,
		},
		OtlpTLSConfig: &config.OtlpTLSConfig{
			Cert: "/tmp/cert.pem",
			Key:  "/tmp/key.pem",
			Ca:   "/tmp/ca.pem",
		},
	}

	cfg.Collector.Receivers.TcplogReceivers = map[string]*config.TcplogReceiver{
		"default": {
			ListenAddress: "localhost:151",
			Operators: []config.Operator{
				{
					Type: "add",
					Fields: map[string]string{
						"field": "body",
						"value": `EXPR(split(body, ",")[0])`,
					},
				},
				{
					Type: "remove",
					Fields: map[string]string{
						"field": "attributes.message",
					},
				},
			},
		},
	}

	cfg.Collector.Extensions.HeadersSetter = &config.HeadersSetter{
		Headers: []config.Header{
			{
				Action: "insert",
				Key:    "authorization",
				Value:  "key1",
			}, {
				Action: "upsert",
				Key:    "uuid",
				Value:  "1234",
			},
		},
	}

	cfg.Collector.Exporters.OtlpExporters["default"].Authenticator = "headers_setter"
	// nolint: lll
	cfg.Collector.Exporters.OtlpExporters["default"].Compression = types.AgentConfig().Collector.Exporters.OtlpExporters["default"].Compression
	cfg.Collector.Exporters.OtlpExporters["default"].Server.Port = 1234
	cfg.Collector.Receivers.OtlpReceivers["default"].Server.Port = 4317
	cfg.Collector.Extensions.Health.Server.Port = 1337

	cfg.Collector.Pipelines.Metrics = make(map[string]*config.Pipeline)
	cfg.Collector.Pipelines.Metrics["default"] = &config.Pipeline{
		Receivers:  []string{"hostmetrics", "containermetrics", "otlp/default", "nginx/123"},
		Processors: []string{"resource/default", "batch/default"},
		Exporters:  []string{"otlp/default", "prometheus", "debug"},
	}
	cfg.Collector.Pipelines.Logs = make(map[string]*config.Pipeline)
	cfg.Collector.Pipelines.Logs["default"] = &config.Pipeline{
		Receivers:  []string{"tcplog/default"},
		Processors: []string{"resource/default", "batch/default"},
		Exporters:  []string{"otlp/default", "debug"},
	}

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

func TestFilePermissions(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := types.AgentConfig()
	actualConfPath := filepath.Join(tmpDir, "nginx-agent-otelcol-test.yaml")
	cfg.Collector.ConfigPath = actualConfPath

	err := writeCollectorConfig(cfg.Collector)
	require.NoError(t, err)

	// Check file permissions are 600
	fileInfo, err := os.Stat(actualConfPath)
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0o600), fileInfo.Mode())
}
