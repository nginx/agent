// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package protos

import (
	"os"

	"github.com/nginx/agent/v3/api/grpc/instances"
)

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

func GetFileDownloadResponse(path, instanceID string, content []byte) *instances.FileDownloadResponse {
	return &instances.FileDownloadResponse{
		Encoded:     true,
		FilePath:    path,
		InstanceId:  instanceID,
		FileContent: content,
	}
}
