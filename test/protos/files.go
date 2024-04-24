// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package protos

import (
	"github.com/nginx/agent/v3/api/grpc/mpi/v1"
)

func GetFileMeta(fileName string) (*v1.FileMeta, error) {
	lastModified, err := CreateProtoTime("2024-01-09T13:22:21Z")
	if err != nil {
		return nil, err
	}

	return &v1.FileMeta{
		ModifiedTime: lastModified,
		Name:         fileName,
		Hash:         "BDEIFo9anKNvAwWm9O2LpfvNiNiGMx.c",
	}, err
}

func GetFileOverview(files ...string) (*v1.FileOverview, error) {
	fileOverview := &v1.FileOverview{
		ConfigVersion: CreateConfigVersion(),
	}
	for _, file := range files {
		fileMeta, err := GetFileMeta(file)
		if err != nil {
			return nil, err
		}
		fileOverview.Files = append(fileOverview.GetFiles(), &v1.File{
			FileMeta: fileMeta,
		})
	}

	return fileOverview, nil
}

func GetFileContents(content []byte) *v1.FileContents {
	return &v1.FileContents{
		Contents: content,
	}
}

func CreateGetOverviewRequest() *v1.GetOverviewRequest {
	return &v1.GetOverviewRequest{
		MessageMeta:   CreateMessageMeta(),
		ConfigVersion: CreateConfigVersion(),
	}
}

func CreateGetFileRequest(fileName string) (*v1.GetFileRequest, error) {
	fileMeta, err := GetFileMeta(fileName)
	return &v1.GetFileRequest{
		MessageMeta: CreateMessageMeta(),
		FileMeta:    fileMeta,
	}, err
}

func CreateGetOverviewResponse() (*v1.GetOverviewResponse, error) {
	fileOverview, err := GetFileOverview("nginx.conf")
	return &v1.GetOverviewResponse{
		Overview: fileOverview,
	}, err
}

func CreateGetFileResponse(content []byte) *v1.GetFileResponse {
	return &v1.GetFileResponse{
		Contents: GetFileContents(content),
	}
}
