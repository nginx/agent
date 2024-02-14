// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package test

import (
	//"github.com/google/uuid"
	"github.com/nginx/agent/v3/api/grpc/instances"
	//"github.com/stretchr/testify/require"
	"os"
	"testing"
)

//
//func CreateFileCache(t *testing.T, files ...*os.File) {
//	fileTime1 := CreateProtoTime(t, "2024-01-08T13:22:23Z")
//	fileTime2 := CreateProtoTime(t, "2024-01-08T13:22:25Z")
//	fileTime3 := CreateProtoTime(t, "2024-01-08T13:22:21Z")
//
//	//cacheData := FileCache{
//	//	nginxConf.Name(): {
//	//		LastModified: fileTime1,
//	//		Path:         nginxConf.Name(),
//	//		Version:      "BDEIFo9anKNvAwWm9O2LpfvNiNiGMx.c",
//	//	},
//	//	testConf.Name(): {
//	//		LastModified: fileTime2,
//	//		Path:         testConf.Name(),
//	//		Version:      "Rh3phZuCRwNGANTkdst51he_0WKWy.tZ",
//	//	},
//	//	metricsConf.Name(): {
//	//		LastModified: fileTime3,
//	//		Path:         metricsConf.Name(),
//	//		Version:      "ibZkRVjemE5dl.tv88ttUJaXx6UJJMTu",
//	//	},
//	//}
//
//	err = createCacheFile(cachePath, cacheData)
//	require.NoError(t, err)
//}

func GetFileCache(t *testing.T, files ...*os.File) map[string]*instances.File {
	t.Helper()
	cache := map[string]*instances.File{}
	for _, file := range files {
		cache[file.Name()] = &instances.File{
			LastModified: CreateProtoTime(t, "2024-01-08T13:22:23Z"),
			Path:         file.Name(),
			Version:      "BDEIFo9anKNvAwWm9O2LpfvNiNiGMx.c",
		}
	}
	return cache
	//return map[string]*instances.File{
	//	nginxConf.Name(): {
	//		LastModified: time1,
	//		Path:         nginxConf.Name(),
	//		Version:      "BDEIFo9anKNvAwWm9O2LpfvNiNiGMx.c",
	//	},
	//	testConf.Name(): {
	//		LastModified: test2Time2,
	//		Path:         testConf.Name(),
	//		Version:      "Rh3phZuCRwNGANTkdst51he_0WKWy.tZ",
	//	},
	//	metricsConf.Name(): {
	//		LastModified: time3,
	//		Path:         metricsConf.Name(),
	//		Version:      "ibZkRVjemE5dl.tv88ttUJaXx6UJJMTu",
	//	},
	//}
}

func GetFiles(t *testing.T, files ...*os.File) *instances.Files {
	instanceFiles := &instances.Files{}
	for _, file := range files {
		//version, err := uuid.NewUUID()
		//require.NoError(t, err)
		instanceFiles.Files = append(instanceFiles.Files, &instances.File{
			LastModified: CreateProtoTime(t, "2024-01-08T13:22:23Z"),
			Path:         file.Name(),
			Version:      "BDEIFo9anKNvAwWm9O2LpfvNiNiGMx.c",
		})
	}
	return instanceFiles
	//time1 := CreateProtoTime(t, "2024-01-08T13:22:23Z")
	//test1Time2 := CreateProtoTime(t, "2024-01-08T14:22:20Z")
	//test2Time2 := CreateProtoTime(t, "2024-01-08T13:22:25Z")
	//time3 := CreateProtoTime(t, "2024-01-08T13:22:21Z")
	//return &instances.Files{
	//	Files: []*instances.File{
	//		{
	//			LastModified: CreateProtoTime(t, "2024-01-08T13:22:23Z"),
	//			Path:         nginxConf.Name(),
	//			Version:      "BDEIFo9anKNvAwWm9O2LpfvNiNiGMx.c",
	//		},
	//		{
	//			LastModified: CreateProtoTime(t, "2024-01-08T14:22:20Z"),
	//			Path:         testConf.Name(),
	//			Version:      "Rh3phZuCRwNGANTkdst51he_0WKWy.tZ",
	//		},
	//		{
	//			LastModified: CreateProtoTime(t, "2024-01-08T13:22:21Z")
	//			Path:         metricsConf.Name(),
	//			Version:      "ibZkRVjemE5dl.tv88ttUJaXx6UJJMTu",
	//		},
	//	},
	//}
}

func GetFileDownloadResponse(path, instanceId string, content []byte) *instances.FileDownloadResponse {
	return &instances.FileDownloadResponse{
		Encoded:     true,
		FilePath:    path,
		InstanceId:  instanceId,
		FileContent: content,
	}
}
