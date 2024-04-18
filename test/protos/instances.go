// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package protos

import (
	"os"

	"github.com/nginx/agent/v3/api/grpc/instances"
	"github.com/nginx/agent/v3/api/grpc/mpi/v1"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const (
	ossInstanceID  = "e1374cb1-462d-3b6c-9f3b-f28332b5f10c"
	plusInstanceID = "40f9dda0-e45f-34cf-bba7-f173700f50a2"
	correlationID  = "dfsbhj6-bc92-30c1-a9c9-85591422068e"
	processID      = 1234
)

func GetNginxOssInstance() *v1.Instance {
	return &v1.Instance{
		InstanceMeta: &v1.InstanceMeta{
			InstanceId:   ossInstanceID,
			InstanceType: v1.InstanceMeta_INSTANCE_TYPE_NGINX,
			Version:      "1.25.3",
		},
		InstanceRuntime: &v1.InstanceRuntime{
			ProcessId:  processID,
			BinaryPath: "/usr/local/Cellar/nginx/1.25.3/bin/nginx",
			ConfigPath: "/usr/local/etc/nginx/nginx.conf",
			Details: &v1.InstanceRuntime_NginxRuntimeInfo{
				NginxRuntimeInfo: &v1.NGINXRuntimeInfo{
					StubStatus:      "",
					AccessLogs:      []string{},
					ErrorLogs:       []string{},
					LoadableModules: []string{},
					DynamicModules:  []string{},
				},
			},
		},
	}
}

func GetNginxPlusInstance() *v1.Instance {
	return &v1.Instance{
		InstanceMeta: &v1.InstanceMeta{
			InstanceId:   plusInstanceID,
			InstanceType: v1.InstanceMeta_INSTANCE_TYPE_NGINX_PLUS,
			Version:      "nginx-plus-r31-p1",
		},
		InstanceRuntime: &v1.InstanceRuntime{
			ProcessId:  processID,
			BinaryPath: "/usr/local/Cellar/nginx/1.25.3/bin/nginx",
			ConfigPath: "/etc/nginx/nginx.conf",
			Details: &v1.InstanceRuntime_NginxPlusRuntimeInfo{
				NginxPlusRuntimeInfo: &v1.NGINXPlusRuntimeInfo{
					StubStatus:      "",
					AccessLogs:      []string{},
					ErrorLogs:       []string{},
					LoadableModules: []string{},
					DynamicModules:  []string{},
					PlusApi:         "",
				},
			},
		},
	}
}

func CreateInProgressStatus() *instances.ConfigurationStatus {
	return &instances.ConfigurationStatus{
		InstanceId:    ossInstanceID,
		CorrelationId: correlationID,
		Status:        instances.Status_IN_PROGRESS,
		Message:       "Instance configuration update in progress",
		Timestamp:     timestamppb.Now(),
	}
}

func CreateSuccessStatus() *instances.ConfigurationStatus {
	return &instances.ConfigurationStatus{
		InstanceId:    ossInstanceID,
		CorrelationId: correlationID,
		Status:        instances.Status_SUCCESS,
		Message:       "Config applied successfully",
		Timestamp:     timestamppb.Now(),
	}
}

func CreateFailStatus(err string) *instances.ConfigurationStatus {
	return &instances.ConfigurationStatus{
		InstanceId:    ossInstanceID,
		CorrelationId: correlationID,
		Status:        instances.Status_FAILED,
		Message:       err,
	}
}

func CreateRollbackSuccessStatus() *instances.ConfigurationStatus {
	return &instances.ConfigurationStatus{
		InstanceId:    ossInstanceID,
		CorrelationId: correlationID,
		Status:        instances.Status_ROLLBACK_SUCCESS,
		Timestamp:     timestamppb.Now(),
		Message:       "Rollback successful",
	}
}

func CreateRollbackFailStatus(err string) *instances.ConfigurationStatus {
	return &instances.ConfigurationStatus{
		InstanceId:    ossInstanceID,
		CorrelationId: correlationID,
		Status:        instances.Status_ROLLBACK_FAILED,
		Timestamp:     timestamppb.Now(),
		Message:       err,
	}
}

func CreateRollbackInProgressStatus() *instances.ConfigurationStatus {
	return &instances.ConfigurationStatus{
		InstanceId:    ossInstanceID,
		CorrelationId: correlationID,
		Status:        instances.Status_ROLLBACK_IN_PROGRESS,
		Timestamp:     timestamppb.Now(),
		Message:       "Rollback in progress",
	}
}

func GetFileCache(files ...*os.File) (map[string]*v1.FileMeta, error) {
	cache := make(map[string]*v1.FileMeta)
	for _, file := range files {
		lastModified, err := CreateProtoTime("2024-01-09T13:22:21Z")
		if err != nil {
			return nil, err
		}

		cache[file.Name()] = &v1.FileMeta{
			ModifiedTime: lastModified,
			Name:         file.Name(),
			Hash:         "BDEIFo9anKNvAwWm9O2LpfvNiNiGMx.c",
		}
	}

	return cache, nil
}

func GetFiles(files ...*os.File) (*v1.FileOverview, error) {
	instance := GetNginxOssInstance()
	instanceFiles := &v1.FileOverview{
		ConfigVersion: &v1.ConfigVersion{
			Version:    "f9a31750-566c-31b3-a763-b9fb5982547b",
			InstanceId: instance.GetInstanceMeta().InstanceId,
		},
	}

	for _, file := range files {
		lastModified, err := CreateProtoTime("2024-01-09T13:22:21Z")
		if err != nil {
			return nil, err
		}
		instanceFiles.Files = append(instanceFiles.GetFiles(), &v1.File{
			FileMeta: &v1.FileMeta{
				ModifiedTime: lastModified,
				Name:         file.Name(),
				Hash:         "BDEIFo9anKNvAwWm9O2LpfvNiNiGMx.c",
			},
		})
	}

	return instanceFiles, nil
}

func GetFileDownloadResponse(content []byte) *v1.FileContents {
	return &v1.FileContents{
		Contents: content,
	}
}
