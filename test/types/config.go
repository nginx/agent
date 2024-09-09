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
)

const (
	apiPort     = 8980
	commandPort = 8981
	metricsPort = 8982

	clientPermitWithoutStream = true
	clientTime                = 50 * time.Second
	clientTimeout             = 5 * time.Second

	commonInitialInterval     = 100 * time.Microsecond
	commonMaxInterval         = 1000 * time.Microsecond
	commonMaxElapsedTime      = 10 * time.Millisecond
	commonRandomizationFactor = 0.1
	commonMultiplier          = 0.2

	reloadMonitoringPeriod = 400 * time.Millisecond

	randomPort1 = 1234
	randomPort2 = 4321
	randomPort3 = 1337
)

// Produces a populated Agent Config for testing usage.
func AgentConfig() *config.Config {
	return &config.Config{
		Version: "test-version",
		UUID:    "75442486-0878-440c-9db1-a7006c25a39f",
		Path:    "/etc/nginx-agent",
		Log:     &config.Log{},
		Client: &config.Client{
			Timeout:             clientTimeout,
			Time:                clientTime,
			PermitWithoutStream: clientPermitWithoutStream,
		},
		ConfigDir:          "",
		AllowedDirectories: []string{"/tmp/"},
		Collector: &config.Collector{
			ConfigPath: "/etc/nginx-agent/nginx-agent-otelcol.yaml",
			Exporters: []config.Exporter{
				{
					Type: "otlp",
					Server: &config.ServerConfig{
						Host: "127.0.0.1",
						Port: randomPort1,
						Type: 0,
					},
					Auth: &config.AuthConfig{
						Token: "super-secret-token",
					},
				},
			},
			Processors: []config.Processor{
				{
					Type: "batch",
				},
			},
			Receivers: config.Receivers{
				OtlpReceivers: OtlpReceivers(),
				HostMetrics: config.HostMetrics{
					CollectionInterval: time.Minute,
					InitialDelay:       time.Second,
				},
			},
			Health: &config.ServerConfig{
				Host: "localhost",
				Port: randomPort3,
				Type: 0,
			},
		},
		Command: &config.Command{
			Server: &config.ServerConfig{
				Host: "127.0.0.1",
				Port: commandPort,
				Type: config.Grpc,
			},
			Auth: &config.AuthConfig{
				Token: "1234",
			},
			TLS: &config.TLSConfig{
				Cert:       "cert.pem",
				Key:        "key.pem",
				Ca:         "ca.pem",
				SkipVerify: true,
				ServerName: "test-server",
			},
		},
		File: &config.File{},
		Common: &config.CommonSettings{
			InitialInterval:     commonInitialInterval,
			MaxInterval:         commonMaxInterval,
			MaxElapsedTime:      commonMaxElapsedTime,
			RandomizationFactor: commonRandomizationFactor,
			Multiplier:          commonMultiplier,
		},
		DataPlaneConfig: &config.DataPlaneConfig{
			Nginx: &config.NginxDataPlaneConfig{
				TreatWarningsAsError:   true,
				ReloadMonitoringPeriod: reloadMonitoringPeriod,
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
			},
		},
	}
}

// Produces a populated Agent Config with a temp Collector config path for testing usage.
func OTelConfig(t *testing.T) *config.Config {
	t.Helper()

	ac := AgentConfig()
	ac.Collector.ConfigPath = filepath.Join(t.TempDir(), "otel-collector-config.yaml")

	return ac
}

func OtlpReceivers() []config.OtlpReceiver {
	return []config.OtlpReceiver{
		{
			Server: &config.ServerConfig{
				Host: "localhost",
				Port: randomPort2,
				Type: 0,
			},
			Auth: &config.AuthConfig{
				Token: "even-secreter-token",
			},
		},
	}
}
