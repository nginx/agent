// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package model

import "github.com/nginx/agent/v3/api/grpc/mpi/v1"

type NginxConfigContext struct {
	StubStatus string
	PlusAPI    string
	InstanceID string
	Files      []*v1.File
	AccessLogs []*AccessLog
	ErrorLogs  []*ErrorLog
}

type ConfigApplyMessage struct {
	Error         error
	CorrelationID string
	InstanceID    string
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

type (
	WriteStatus int
)

const (
	RollbackRequired WriteStatus = iota + 1
	NoChange
	Error
	OK
)
