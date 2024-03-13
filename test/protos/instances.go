// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package protos

import (
	"os"

	"github.com/nginx/agent/v3/api/grpc/instances"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const (
	instanceID    = "aecea348-62c1-4e3d-b848-6d6cdeb1cb9c"
	correlationID = "dfsbhj6-bc92-30c1-a9c9-85591422068e"
)

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

func GetFileCache(files ...*os.File) (map[string]*instances.File, error) {
	cache := make(map[string]*instances.File)
	for _, file := range files {
		lastModified, err := CreateProtoTime("2024-01-09T13:22:21Z")
		if err != nil {
			return nil, err
		}

		cache[file.Name()] = &instances.File{
			LastModified: lastModified,
			Path:         file.Name(),
			Version:      "BDEIFo9anKNvAwWm9O2LpfvNiNiGMx.c",
		}
	}

	return cache, nil
}

func GetFiles(files ...*os.File) (*instances.Files, error) {
	instanceFiles := &instances.Files{}

	for _, file := range files {
		lastModified, err := CreateProtoTime("2024-01-09T13:22:21Z")
		if err != nil {
			return nil, err
		}
		instanceFiles.Files = append(instanceFiles.GetFiles(), &instances.File{
			LastModified: lastModified,
			Path:         file.Name(),
			Version:      "BDEIFo9anKNvAwWm9O2LpfvNiNiGMx.c",
		})
	}

	return instanceFiles, nil
}

func GetFileDownloadResponse(filePath, instanceID string, content []byte) *instances.FileDownloadResponse {
	return &instances.FileDownloadResponse{
		Encoded:     true,
		FilePath:    filePath,
		InstanceId:  instanceID,
		FileContent: content,
	}
}
