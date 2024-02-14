// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package test

import (
	"os"
	"testing"

	"github.com/nginx/agent/v3/api/grpc/instances"
)

func GetFileCache(t *testing.T, files ...*os.File) map[string]*instances.File {
	t.Helper()
	cache := make(map[string]*instances.File)
	for _, file := range files {
		cache[file.Name()] = &instances.File{
			LastModified: CreateProtoTime(t, "2024-01-08T13:22:23Z"),
			Path:         file.Name(),
			Version:      "BDEIFo9anKNvAwWm9O2LpfvNiNiGMx.c",
		}
	}

	return cache
}

func GetFiles(t *testing.T, files ...*os.File) *instances.Files {
	t.Helper()
	instanceFiles := &instances.Files{}
	for _, file := range files {
		instanceFiles.Files = append(instanceFiles.GetFiles(), &instances.File{
			LastModified: CreateProtoTime(t, "2024-01-08T13:22:23Z"),
			Path:         file.Name(),
			Version:      "BDEIFo9anKNvAwWm9O2LpfvNiNiGMx.c",
		})
	}

	return instanceFiles
}

func GetFileDownloadResponse(path, instanceID string, content []byte) *instances.FileDownloadResponse {
	return &instances.FileDownloadResponse{
		Encoded:     true,
		FilePath:    path,
		InstanceId:  instanceID,
		FileContent: content,
	}
}
