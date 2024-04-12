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
	instanceID    = "aecea348-62c1-4e3d-b848-6d6cdeb1cb9c"
	correlationID = "dfsbhj6-bc92-30c1-a9c9-85591422068e"
)

func GetNginxOssInstance() *v1.Instance {
	return &v1.Instance{
		InstanceMeta: &v1.InstanceMeta{
			InstanceId:   "123",
			InstanceType: v1.InstanceMeta_INSTANCE_TYPE_NGINX,
		},
		InstanceRuntime: &v1.InstanceRuntime{
			ProcessId:  1234,
			BinaryPath: "/var/run/nginx",
			ConfigPath: "/etc/nginx",
			Details: &v1.InstanceRuntime_NginxRuntimeInfo{
				NginxRuntimeInfo: &v1.NGINXRuntimeInfo{
					StubStatus:      "/stub",
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
			InstanceId:   "123",
			InstanceType: v1.InstanceMeta_INSTANCE_TYPE_NGINX,
		},
		InstanceRuntime: &v1.InstanceRuntime{
			ProcessId:  1234,
			BinaryPath: "/var/run/nginx",
			ConfigPath: "/etc/nginx",
			Details: &v1.InstanceRuntime_NginxPlusRuntimeInfo{
				NginxPlusRuntimeInfo: &v1.NGINXPlusRuntimeInfo{
					StubStatus:      "/stub",
					AccessLogs:      []string{},
					ErrorLogs:       []string{},
					LoadableModules: []string{},
					DynamicModules:  []string{},
					PlusApi:         "/api",
				},
			},
		},
	}
}

func CreateInProgressStatus() *instances.ConfigurationStatus {
	return &instances.ConfigurationStatus{
		InstanceId:    instanceID,
		CorrelationId: correlationID,
		Status:        instances.Status_IN_PROGRESS,
		Message:       "Instance configuration update in progress",
		Timestamp:     timestamppb.Now(),
	}
}

func CreateSuccessStatus() *instances.ConfigurationStatus {
	return &instances.ConfigurationStatus{
		InstanceId:    instanceID,
		CorrelationId: correlationID,
		Status:        instances.Status_SUCCESS,
		Message:       "Config applied successfully",
		Timestamp:     timestamppb.Now(),
	}
}

func CreateFailStatus(err string) *instances.ConfigurationStatus {
	return &instances.ConfigurationStatus{
		InstanceId:    instanceID,
		CorrelationId: correlationID,
		Status:        instances.Status_FAILED,
		Message:       err,
	}
}

func CreateRollbackSuccessStatus() *instances.ConfigurationStatus {
	return &instances.ConfigurationStatus{
		InstanceId:    instanceID,
		CorrelationId: correlationID,
		Status:        instances.Status_ROLLBACK_SUCCESS,
		Timestamp:     timestamppb.Now(),
		Message:       "Rollback successful",
	}
}

func CreateRollbackFailStatus(err string) *instances.ConfigurationStatus {
	return &instances.ConfigurationStatus{
		InstanceId:    instanceID,
		CorrelationId: correlationID,
		Status:        instances.Status_ROLLBACK_FAILED,
		Timestamp:     timestamppb.Now(),
		Message:       err,
	}
}

func CreateRollbackInProgressStatus() *instances.ConfigurationStatus {
	return &instances.ConfigurationStatus{
		InstanceId:    instanceID,
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
	instanceFiles := &v1.FileOverview{}

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
