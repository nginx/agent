// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package types

import (
	"fmt"
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

	connTimeout     = 10 * time.Second
	minConnTimeout  = 7 * time.Second
	backoffDelay    = 240 * time.Second
	exportInterval  = 30 * time.Second
	produceInterval = 5 * time.Second

	commonInitialInterval     = 100 * time.Microsecond
	commonMaxInterval         = 1000 * time.Microsecond
	commonMaxElapsedTime      = 10 * time.Millisecond
	commonRandomizationFactor = 0.1
	commonMultiplier          = 0.2

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
			Timeout:             clientTimeout,
			Time:                clientTime,
			PermitWithoutStream: clientPermitWithoutStream,
		},
		ConfigDir:          "",
		AllowedDirectories: []string{"/tmp/"},
		Metrics: &config.Metrics{
			ProduceInterval: produceInterval,
			OTelExporter: &config.OTelExporter{
				BufferLength:     bufferLength,
				ExportRetryCount: exportRetryCount,
				ExportInterval:   exportInterval,
				GRPC: &config.GRPC{
					Target:         fmt.Sprintf("%s:%d", "dummy-target", metricsPort),
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
	}
}
