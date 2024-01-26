/**
 * Copyright (c) F5, Inc.
 *
 * This source code is licensed under the Apache License, Version 2.0 license found in the
 * LICENSE file in the root directory of this source tree.
 */

package model

type ConfigContext interface {
	NginxConfigContext | NginxGatewayFabricConfigContext
}

type NginxConfigContext struct {
	AccessLogs []*AccessLog
	ErrorLogs  []*ErrorLog
}

type AccessLog struct {
	Name        string
	Format      string
	Permissions string
	Readable    bool
}

type ErrorLog struct {
	Name        string
	LogLevel    string
	Permissions string
	Readable    bool
}

type NginxGatewayFabricConfigContext struct {
	PrometheusEndpoint string
}
