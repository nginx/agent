// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package protos

import (
	mpi "github.com/nginx/agent/v3/api/grpc/mpi/v1"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func GetFileMeta(fileName, fileHash string) *mpi.FileMeta {
	lastModified, _ := CreateProtoTime("2024-01-09T13:22:21Z")

	return &mpi.FileMeta{
		ModifiedTime: lastModified,
		Name:         fileName,
		Hash:         fileHash,
		Permissions:  "0600",
	}
}

func FileOverview(filePath, fileHash string, action *mpi.File_FileAction) *mpi.FileOverview {
	return &mpi.FileOverview{
		Files: []*mpi.File{
			{
				FileMeta: &mpi.FileMeta{
					Name:         filePath,
					Hash:         fileHash,
					ModifiedTime: timestamppb.Now(),
					Permissions:  "0640",
					Size:         0,
				},
				Action: action,
			},
		},
		ConfigVersion: CreateConfigVersion(),
	}
}

func GetFileContents(content []byte) *mpi.FileContents {
	return &mpi.FileContents{
		Contents: content,
	}
}
