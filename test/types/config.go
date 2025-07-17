// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package types

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/nginx/agent/v3/internal/config"
	"github.com/nginx/agent/v3/test/helpers"
	"github.com/stretchr/testify/require"
)

const (
	clientPermitWithoutStream = true
	clientTime                = 50 * time.Second
	clientTimeout             = 5 * time.Second

	clientHTTPTimeout = 5 * time.Second

	commonInitialInterval     = 100 * time.Microsecond
	commonMaxInterval         = 1000 * time.Microsecond
	commonMaxElapsedTime      = 10 * time.Millisecond
	commonRandomizationFactor = 0.1
	commonMultiplier          = 0.2

	reloadMonitoringPeriod = 400 * time.Millisecond
)

// Produces a populated Agent Config for testing usage.
func AgentConfig() *config.Config {
	return &config.Config{
		Version: "test-version",
		UUID:    "75442486-0878-440c-9db1-a7006c25a39f",
		Path:    "/etc/nginx-agent",
		Log:     &config.Log{},
		Client: &config.Client{
			HTTP: &config.HTTP{
				Timeout: clientHTTPTimeout,
			},
			Grpc: &config.GRPC{
				KeepAlive: &config.KeepAlive{
					Timeout:             clientTimeout,
					Time:                clientTime,
					PermitWithoutStream: clientPermitWithoutStream,
				},
				MaxMessageReceiveSize: 1,
				MaxMessageSendSize:    1,
				MaxFileSize:           1,
				FileChunkSize:         1,
			},
			Backoff: &config.BackOff{
				InitialInterval:     commonInitialInterval,
				MaxInterval:         commonMaxInterval,
				MaxElapsedTime:      commonMaxElapsedTime,
				RandomizationFactor: commonRandomizationFactor,
				Multiplier:          commonMultiplier,
			},
		},
		AllowedDirectories: []string{"/tmp/"},
		Collector: &config.Collector{
			ConfigPath: "/etc/nginx-agent/nginx-agent-otelcol.yaml",
			Exporters: config.Exporters{
				OtlpExporters: map[string]*config.OtlpExporter{
					"default": {
						Server: &config.ServerConfig{
							Host: "127.0.0.1",
							Port: 0,
						},
						Compression: "none",
					},
				},
			},
			Processors: config.Processors{
				Batch: map[string]*config.Batch{
					"default": {
						SendBatchSize:    config.DefCollectorMetricsBatchProcessorSendBatchMaxSize,
						SendBatchMaxSize: config.DefCollectorMetricsBatchProcessorSendBatchMaxSize,
						Timeout:          config.DefCollectorMetricsBatchProcessorTimeout,
					},
				},
			},
			Receivers: config.Receivers{
				OtlpReceivers: map[string]*config.OtlpReceiver{
					"default": {
						Server: &config.ServerConfig{
							Host: "127.0.0.1",
							Port: 0,
							Type: config.Grpc,
						},
						Auth: &config.AuthConfig{
							Token: "even-secreter-token",
						},
					},
				},
				HostMetrics: &config.HostMetrics{
					CollectionInterval: time.Minute,
					InitialDelay:       time.Second,
					Scrapers: &config.HostMetricsScrapers{
						CPU:        &config.CPUScraper{},
						Disk:       &config.DiskScraper{},
						Filesystem: &config.FilesystemScraper{},
						Memory:     &config.MemoryScraper{},
						Network:    &config.NetworkScraper{},
					},
				},
			},
			Extensions: config.Extensions{
				Health: &config.Health{
					Server: &config.ServerConfig{
						Host: "127.0.0.1",
						Port: 0,
					},
				},
				HeadersSetter: &config.HeadersSetter{
					Headers: []config.Header{
						{
							Action: "insert",
							Key:    "authorization",
							Value:  "fake-authorization",
						},
					},
				},
			},
			Log: &config.Log{
				Level: "INFO",
				Path:  "/var/log/nginx-agent/opentelemetry-collector-agent.log",
			},
			Pipelines: config.Pipelines{
				Metrics: map[string]*config.Pipeline{
					"default": {
						Receivers:  []string{"host_metrics"},
						Processors: []string{"batch/default"},
						Exporters:  []string{"otlp/default"},
					},
				},
			},
		},
		Command: &config.Command{
			Server: &config.ServerConfig{
				Host: "127.0.0.1",
				Port: 0,
				Type: config.Grpc,
			},
			Auth: &config.AuthConfig{
				Token:     "1234",
				TokenPath: "/tmp/token",
			},
			TLS: &config.TLSConfig{
				Cert:       "cert.pem",
				Key:        "key.pem",
				Ca:         "ca.pem",
				SkipVerify: true,
				ServerName: "test-server",
			},
		},
		DataPlaneConfig: &config.DataPlaneConfig{
			Nginx: &config.NginxDataPlaneConfig{
				TreatWarningsAsErrors:  true,
				ReloadMonitoringPeriod: reloadMonitoringPeriod,
				ExcludeLogs:            []string{},
			},
		},
		Watchers: &config.Watchers{
			InstanceWatcher: config.InstanceWatcher{
				MonitoringFrequency: config.DefInstanceWatcherMonitoringFrequency,
			},
			InstanceHealthWatcher: config.InstanceHealthWatcher{
				MonitoringFrequency: config.DefInstanceWatcherMonitoringFrequency,
			},
			FileWatcher: config.FileWatcher{
				MonitoringFrequency: config.DefFileWatcherMonitoringFrequency,
				ExcludeFiles:        config.DefaultExcludedFiles(),
			},
		},
		Features: config.DefaultFeatures(),
	}
}

// Produces a populated Agent Config with a temp Collector config path for testing usage.
func OTelConfig(t *testing.T) *config.Config {
	t.Helper()

	ac := AgentConfig()
	ac.Collector.ConfigPath = filepath.Join(t.TempDir(), "otel-collector-config.yaml")

	exporterPort, expErr := helpers.RandomPort(t)
	require.NoError(t, expErr)
	ac.Collector.Exporters.OtlpExporters["default"].Server.Port = exporterPort

	receiverPort, recErr := helpers.RandomPort(t)
	require.NoError(t, recErr)
	ac.Collector.Receivers.OtlpReceivers["default"].Server.Port = receiverPort

	healthPort, healthErr := helpers.RandomPort(t)
	require.NoError(t, healthErr)
	ac.Collector.Extensions.Health.Server.Port = healthPort

	commandPort, commandErr := helpers.RandomPort(t)
	require.NoError(t, commandErr)
	ac.Command.Server.Port = commandPort

	return ac
}
