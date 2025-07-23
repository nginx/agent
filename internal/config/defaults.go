// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.
package config

import (
	"time"

	pkg "github.com/nginx/agent/v3/pkg/config"
)

const (
	DefGracefulShutdownPeriod      = 5 * time.Second
	DefNginxReloadMonitoringPeriod = 10 * time.Second
	DefTreatErrorsAsWarnings       = false
	DefNginxApiTlsCa               = ""

	DefCommandServerHostKey    = ""
	DefCommandServerPortKey    = 0
	DefCommandServerTypeKey    = "grpc"
	DefCommandAuthTokenKey     = ""
	DefCommandAuthTokenPathKey = ""
	DefCommandTLSCertKey       = ""
	DefCommandTLSKeyKey        = ""
	DefCommandTLSCaKey         = ""
	DefCommandTLSSkipVerifyKey = false
	DefCommandTLServerNameKey  = ""

	DefAuxiliaryCommandServerHostKey    = ""
	DefAuxiliaryCommandServerPortKey    = 0
	DefAuxiliaryCommandServerTypeKey    = "grpc"
	DefAuxiliaryCommandAuthTokenKey     = ""
	DefAuxiliaryCommandAuthTokenPathKey = ""
	DefAuxiliaryCommandTLSCertKey       = ""
	DefAuxiliaryCommandTLSKeyKey        = ""
	DefAuxiliaryCommandTLSCaKey         = ""
	DefAuxiliaryCommandTLSSkipVerifyKey = false
	DefAuxiliaryCommandTLServerNameKey  = ""

	// Client GRPC Settings
	DefMaxMessageSize               = 0       // 0 = unset
	DefMaxMessageRecieveSize        = 4194304 // default 4 MB
	DefMaxMessageSendSize           = 4194304 // default 4 MB
	DefMaxFileSize           uint32 = 1048576 // 1MB
	DefFileChunkSize         uint32 = 524288  // 0.5MB

	// Client HTTP Settings
	DefHTTPTimeout = 10 * time.Second

	// Client GRPC Keep Alive Settings
	DefGRPCKeepAliveTimeout             = 10 * time.Second
	DefGRPCKeepAliveTime                = 20 * time.Second
	DefGRPCKeepAlivePermitWithoutStream = true

	// Client Backoff defaults
	DefBackoffInitialInterval     = 1 * time.Second
	DefBackoffRandomizationFactor = 0.5 // the value is 0 <= and < 1
	DefBackoffMultiplier          = 3
	DefBackoffMaxInterval         = 20 * time.Second
	DefBackoffMaxElapsedTime      = 1 * time.Minute

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

	DefCollectorMetricsBatchProcessorSendBatchSize    = 1000
	DefCollectorMetricsBatchProcessorSendBatchMaxSize = 1000
	DefCollectorMetricsBatchProcessorTimeout          = 30 * time.Second
	DefCollectorLogsBatchProcessorSendBatchSize       = 100
	DefCollectorLogsBatchProcessorSendBatchMaxSize    = 100
	DefCollectorLogsBatchProcessorTimeout             = 60 * time.Second

	DefCollectorExtensionsHealthServerHost      = "localhost"
	DefCollectorExtensionsHealthServerPort      = 13133
	DefCollectorExtensionsHealthPath            = "/"
	DefCollectorExtensionsHealthTLSCertPath     = ""
	DefCollectorExtensionsHealthTLSKeyPath      = ""
	DefCollectorExtensionsHealthTLSCAPath       = ""
	DefCollectorExtensionsHealthTLSSkipVerify   = false
	DefCollectorExtensionsHealthTLServerNameKey = ""

	// File defaults
	DefManifestDir = "/var/lib/nginx-agent"
)

func DefaultFeatures() []string {
	return []string{
		pkg.FeatureConfiguration,
		pkg.FeatureCertificates,
		pkg.FeatureMetrics,
		pkg.FeatureFileWatcher,
		pkg.FeatureLogsNap,
	}
}

func DefaultAllowedDirectories() []string {
	return []string{
		"/etc/nginx",
		"/usr/local/etc/nginx",
		"/usr/share/nginx/modules",
		"/var/run/nginx",
		"/var/log/nginx",
		"/etc/app_protect",
	}
}

func DefaultExcludedFiles() []string {
	return []string{
		"^.*(\\.log|.swx|~|.swp)$",
	}
}

func DefaultLabels() map[string]string {
	return make(map[string]string)
}
