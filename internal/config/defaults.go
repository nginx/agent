// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.
package config

import (
	"math"
	"time"
)

const (
	DefGracefulShutdownPeriod      = 5 * time.Second
	DefNginxReloadMonitoringPeriod = 10 * time.Second
	DefTreatErrorsAsWarnings       = true

	DefCollectorConfigPath           = "/var/run/nginx-agent/opentelemetry-collector-agent.yaml"
	DefCollectorTLSGenSelfSignedCert = false
	DefCollectorLogLevel             = "INFO"
	DefCollectorLogPath              = "/var/log/nginx-agent/opentelemetry-collector-agent.log"
	DefConfigDirectories             = "/etc/nginx:/usr/local/etc/nginx:/usr/share/nginx/modules"
	DefCollectorTLSCertPath          = "/var/lib/nginx-agent/cert.pem"
	DefCollectorTLSKeyPath           = "/var/lib/nginx-agent/key.pem"
	DefCollectorTLSCAPath            = "/var/lib/nginx-agent/ca.pem"
	DefCollectorTLSSANNames          = "127.0.0.1,::1,localhost"

	DefCommandServerHostKey    = ""
	DefCommandServerPortKey    = 0
	DefCommandServerTypeKey    = "grpc"
	DefCommandAuthTokenKey     = ""
	DefCommandTLSCertKey       = ""
	DefCommandTLSKeyKey        = ""
	DefCommandTLSCaKey         = ""
	DefCommandTLSSkipVerifyKey = false

	DefBackoffInitialInterval = 50 * time.Millisecond
	// the value is 0 <= and < 1
	DefBackoffRandomizationFactor = 0.1
	DefBackoffMultiplier          = 1.5
	DefBackoffMaxInterval         = 200 * time.Millisecond
	DefBackoffMaxElapsedTime      = 3 * time.Second

	DefInstanceWatcherMonitoringFrequency       = 5 * time.Second
	DefInstanceHealthWatcherMonitoringFrequency = 5 * time.Second
	DefFileWatcherMonitoringFrequency           = 5 * time.Second

	// 0 = unset
	DefMaxMessageSize = 0
	// default 4 MB
	DefMaxMessageRecieveSize = 4194304
	// math.MaxInt32
	DefMaxMessageSendSize = math.MaxInt32
)
