// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package test

import (
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/nginx/agent/v3/api/grpc/instances"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func GetFileCache(files ...*os.File) map[string]*instances.File {
	cache := make(map[string]*instances.File)
	for _, file := range files {
		cache[file.Name()] = &instances.File{
			LastModified: CreateProtoTime("2024-01-08T13:22:23Z"),
			Path:         file.Name(),
			Version:      "BDEIFo9anKNvAwWm9O2LpfvNiNiGMx.c",
		}
	}

	return cache
}

func GetFiles(files ...*os.File) *instances.Files {
	instanceFiles := &instances.Files{}
	for _, file := range files {
		instanceFiles.Files = append(instanceFiles.GetFiles(), &instances.File{
			LastModified: CreateProtoTime("2024-01-08T13:22:23Z"),
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

func CreateProtoTime(timeString string) *timestamppb.Timestamp {
	newTime, err := time.Parse(time.RFC3339, timeString)
	if err != nil {
		slog.Error("failed to parse time")
	}

	protoTime := timestamppb.New(newTime)
	if err != nil {
		slog.Error(fmt.Sprintf("failed on creating timestamp %s", protoTime))
	}

	return protoTime
}
