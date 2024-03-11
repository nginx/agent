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

	clientTimeout   = 5 * time.Second
	connTimeout     = 10 * time.Second
	minConnTimeout  = 7 * time.Second
	backoffDelay    = 240 * time.Second
	exportInterval  = 30 * time.Second
	produceInterval = 5 * time.Second

	bufferLength     = 55
	exportRetryCount = 3
)

func GetAgentConfig() *config.Config {
	return &config.Config{
		Version: "",
		Path:    "",
		Log:     &config.Log{},
		ProcessMonitor: &config.ProcessMonitor{
			MonitoringFrequency: time.Millisecond,
		},
		DataPlaneAPI: &config.DataPlaneAPI{
			Host: "127.0.0.1",
			Port: apiPort,
		},
		Client: &config.Client{
			Timeout: clientTimeout,
		},
		ConfigDir:          "",
		AllowedDirectories: []string{},
		Metrics: &config.Metrics{
			ProduceInterval: produceInterval,
			OTelExporter: &config.OTelExporter{
				BufferLength:     bufferLength,
				ExportRetryCount: exportRetryCount,
				ExportInterval:   exportInterval,
				GRPC: &config.GRPC{
					Target:         "dummy-target",
					ConnTimeout:    connTimeout,
					MinConnTimeout: minConnTimeout,
					BackoffDelay:   backoffDelay,
				},
			},
			PrometheusSource: &config.PrometheusSource{
				Endpoints: []string{
					"https://example.com",
					"https://acme.com",
				},
			},
		},
		Command: &config.Command{
			Server: &config.ServerConfig{
				Host: "127.0.0.1",
				Port: commandPort,
				Type: "grpc",
			},
			Auth: &config.AuthConfig{
				Token: "1234",
			},
			TLS: &config.TLSConfig{
				Cert:       "some.cert",
				Key:        "some.key",
				Ca:         "some.ca",
				SkipVerify: false,
			},
		},
	}
}
