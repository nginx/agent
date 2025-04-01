// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package model

import mpi "github.com/nginx/agent/v3/api/grpc/mpi/v1"

type FileCache struct {
	File   *mpi.File  `json:"file"`
	Action FileAction `json:"action"`
}

type FileAction int

const (
	Add FileAction = iota + 1
	Update
	Delete
	Unchanged
)

// ConvertToMapOfFiles converts a list of files to a map of file caches (file and action) with the file name as the key
func ConvertToMapOfFileCache(convertFiles []*mpi.File) map[string]*FileCache {
	filesMap := make(map[string]*FileCache)
	for _, convertFile := range convertFiles {
		filesMap[convertFile.GetFileMeta().GetName()] = &FileCache{
			File: convertFile,
		}
	}

	return filesMap
}
