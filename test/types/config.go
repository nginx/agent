// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package types

import (
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
)

func GetAgentConfig() *config.Config {
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
		Metrics: &config.Metrics{
			CollectorEnabled:    false,
			OTLPExportURL:       "localhost:3000",
			OTLPReceiverURL:     "localhost:1234",
			CollectorConfigPath: "/var/etc/nginx-agent/nginx-agent-otelcol.yaml",
			CollectorReceivers:  []config.OTelReceiver{config.HostMetrics},
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
		},
	}
}
