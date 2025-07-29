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
	StubStatus       *APIDetails
	PlusAPI          *APIDetails
	InstanceID       string
	Files            []*v1.File
	AccessLogs       []*AccessLog
	ErrorLogs        []*ErrorLog
	NAPSysLogServers []string
	Includes         []string
}

type APIDetails struct {
	URL      string
	Listen   string
	Location string
	Ca       string
}

type ManifestFile struct {
	ManifestFileMeta *ManifestFileMeta `json:"manifest_file_meta"`
}

type ManifestFileMeta struct {
	// The full path of the file
	Name string `json:"name"`
	// The hash of the file contents sha256, hex encoded
	Hash string `json:"hash"`
	// The size of the file in bytes
	Size int64 `json:"size"`
	// File referenced in the NGINX config
	Referenced bool `json:"referenced"`
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

type ConfigApplySuccess struct {
	ConfigContext     *NginxConfigContext
	DataPlaneResponse *v1.DataPlaneResponse
}

// Complexity is 11, allowed is 10
// nolint: revive, cyclop
func (ncc *NginxConfigContext) Equal(otherNginxConfigContext *NginxConfigContext) bool {
	if ncc.StubStatus != nil && otherNginxConfigContext.StubStatus != nil {
		if ncc.StubStatus.URL != otherNginxConfigContext.StubStatus.URL || ncc.StubStatus.Listen !=
			otherNginxConfigContext.StubStatus.Listen || ncc.StubStatus.Location !=
			otherNginxConfigContext.StubStatus.Location {
			return false
		}
	}

	if ncc.PlusAPI != nil && otherNginxConfigContext.PlusAPI != nil {
		if ncc.PlusAPI.URL != otherNginxConfigContext.PlusAPI.URL || ncc.PlusAPI.Listen !=
			otherNginxConfigContext.PlusAPI.Listen || ncc.PlusAPI.Location !=
			otherNginxConfigContext.PlusAPI.Location {
			return false
		}
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

	if !reflect.DeepEqual(ncc.NAPSysLogServers, otherNginxConfigContext.NAPSysLogServers) {
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

func ConvertAccessLogs(accessLogs []*AccessLog) (logs []string) {
	for _, log := range accessLogs {
		logs = append(logs, log.Name)
	}

	return logs
}

func ConvertErrorLogs(errorLogs []*ErrorLog) (logs []string) {
	for _, log := range errorLogs {
		logs = append(logs, log.Name)
	}

	return logs
}
