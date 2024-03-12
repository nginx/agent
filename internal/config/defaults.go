// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.
package config

import "time"

const (
	DefGracefulShutdownPeriod = 5 * time.Second

	DefMetricsProduceInterval       = 30 * time.Second
	DefOTelExporterBufferLength     = 100
	DefOTelExporterExportRetryCount = 3
	DefOTelExporterExportInterval   = 20 * time.Second
	DefOTelGRPCConnTimeout          = 10 * time.Second
	DefOTelGRPCMinConnTimeout       = 5 * time.Second
	DefOTelGRPCMBackoffDelay        = 240 * time.Second

	DefCommandServerHostKey    = ""
	DefCommandServerPortKey    = 0
	DefCommandServerTypeKey    = "grpc"
	DefCommandAuthTokenKey     = ""
	DefCommandTLSCertKey       = ""
	DefCommandTLSKeyKey        = ""
	DefCommandTLSCaKey         = ""
	DefCommandTLSSkipVerifyKey = false
)
