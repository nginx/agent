// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.
package config

import (
	"math"
	"time"

	pkg "github.com/nginx/agent/v3/pkg/config"
)

const (
	DefConfigDirectories           = "/etc/nginx:/usr/local/etc/nginx:/usr/share/nginx/modules"
	DefGracefulShutdownPeriod      = 5 * time.Second
	DefNginxReloadMonitoringPeriod = 10 * time.Second
	DefTreatErrorsAsWarnings       = true

	// Command defaults
	DefCommandServerHostKey    = "127.0.0.1"
	DefCommandServerPortKey    = 8080
	DefCommandServerTypeKey    = "grpc"
	DefCommandAuthTokenKey     = ""
	DefCommandTLSCertKey       = ""
	DefCommandTLSKeyKey        = ""
	DefCommandTLSCaKey         = ""
	DefCommandTLSSkipVerifyKey = false
	DefCommandTLServerNameKey  = ""

	DefMaxMessageSize        = 0       // 0 = unset
	DefMaxMessageRecieveSize = 4194304 // default 4 MB
	DefMaxMessageSendSize    = math.MaxInt32

	// Backoff defaults
	DefBackoffInitialInterval     = 50 * time.Millisecond
	DefBackoffRandomizationFactor = 0.1 // the value is 0 <= and < 1
	DefBackoffMultiplier          = 1.5
	DefBackoffMaxInterval         = 200 * time.Millisecond
	DefBackoffMaxElapsedTime      = 3 * time.Second

	// Watcher defaults
	DefInstanceWatcherMonitoringFrequency       = 5 * time.Second
	DefInstanceHealthWatcherMonitoringFrequency = 5 * time.Second
	DefFileWatcherMonitoringFrequency           = 5 * time.Second

	// Collector defaults
	DefCollectorConfigPath  = "/etc/nginx-agent/opentelemetry-collector-agent.yaml"
	DefCollectorLogLevel    = "INFO"
	DefCollectorLogPath     = "/var/log/nginx-agent/opentelemetry-collector-agent.log"
	DefCollectorTLSCertPath = "/var/lib/nginx-agent/cert.pem"
	DefCollectorTLSKeyPath  = "/var/lib/nginx-agent/key.pem"
	DefCollectorTLSCAPath   = "/var/lib/nginx-agent/ca.pem"
	DefCollectorTLSSANNames = "127.0.0.1,::1,localhost"

	DefCollectorBatchProcessorSendBatchSize    = 8192
	DefCollectorBatchProcessorSendBatchMaxSize = 0
	DefCollectorBatchProcessorTimeout          = 200 * time.Millisecond

	DefCollectorExtensionsHealthServerHost      = "localhost"
	DefCollectorExtensionsHealthServerPort      = 13133
	DefCollectorExtensionsHealthPath            = "/"
	DefCollectorExtensionsHealthTLSCertPath     = ""
	DefCollectorExtensionsHealthTLSKeyPath      = ""
	DefCollectorExtensionsHealthTLSCAPath       = ""
	DefCollectorExtensionsHealthTLSSkipVerify   = false
	DefCollectorExtensionsHealthTLServerNameKey = ""

	DefCollectorPrometheusExporterServerHost      = ""
	DefCollectorPrometheusExporterServerPort      = 0
	DefCollectorPrometheusExporterTLSCertPath     = ""
	DefCollectorPrometheusExporterTLSKeyPath      = ""
	DefCollectorPrometheusExporterTLSCAPath       = ""
	DefCollectorPrometheusExporterTLSSkipVerify   = false
	DefCollectorPrometheusExporterTLServerNameKey = ""

	DefCollectorOtlpExporterServerHostKey = "127.0.0.1"
	DefCollectorOtlpExporterServerPortKey = 5643
)

func GetDefaultFeatures() []string {
	return []string{
		pkg.FeatureConfiguration,
		pkg.FeatureConnection,
		pkg.FeatureMetrics,
		pkg.FeatureFileWatcher,
	}
}
