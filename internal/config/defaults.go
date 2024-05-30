// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.
package config

import (
	"time"
)

var DefCollectorReceivers = []string{"otlp"}

const (
	DefGracefulShutdownPeriod = 5 * time.Second

	DefCollectorConfigPath = "/var/run/nginx-agent-otelcol.yaml"
	DefOTLPExportURL       = ""
	DefOTLPReceiverURL     = "localhost:4317"

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
)
