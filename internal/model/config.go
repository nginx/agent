// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package model

import (
	"reflect"

	"github.com/nginx/agent/v3/api/grpc/mpi/v1"
)

type NginxConfigContext struct {
	StubStatus string
	PlusAPI    string
	InstanceID string
	Files      []*v1.File
	AccessLogs []*AccessLog
	ErrorLogs  []*ErrorLog
}

func (ncc *NginxConfigContext) Equal(otherNginxConfigContext *NginxConfigContext) bool {
	if ncc.StubStatus != otherNginxConfigContext.StubStatus {
		return false
	}

	if ncc.PlusAPI != otherNginxConfigContext.PlusAPI {
		return false
	}

	if ncc.InstanceID != otherNginxConfigContext.InstanceID {
		return false
	}

	if !ncc.areFileEqual(otherNginxConfigContext.Files) {
		return false
	}

	if !reflect.DeepEqual(ncc.AccessLogs, otherNginxConfigContext.AccessLogs) {
		return false
	}

	if !reflect.DeepEqual(ncc.ErrorLogs, otherNginxConfigContext.ErrorLogs) {
		return false
	}

	return true
}

func (ncc *NginxConfigContext) areFileEqual(files []*v1.File) bool {
	if len(ncc.Files) != len(files) {
		return false
	}

	for _, file := range ncc.Files {
		for _, otherFile := range files {
			if file.GetFileMeta().GetName() == otherFile.GetFileMeta().GetName() &&
				file.GetFileMeta().GetHash() != otherFile.GetFileMeta().GetHash() {
				return false
			}
		}
	}

	return true
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
