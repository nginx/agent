/**
 * Copyright (c) F5, Inc.
 *
 * This source code is licensed under the Apache License, Version 2.0 license found in the
 * LICENSE file in the root directory of this source tree.
 */

package os

import (
	"encoding/json"
	"fmt"
	"os"
	"path"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/nginx/agent/v3/api/grpc/instances"
	"github.com/nginx/agent/v3/internal/client"
	"github.com/stretchr/testify/assert"

	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestWriteFile(t *testing.T) {
	filePath := "/tmp/test.conf"
	fileContent := []byte("location /test {\n    return 200 \"Test location\\n\";\n}")

	err := WriteFile(fileContent, filePath)
	assert.NoError(t, err)
	assert.FileExists(t, filePath)

	data, err := os.ReadFile(filePath)
	assert.NoError(t, err)
	assert.Equal(t, fileContent, data)

	err = os.Remove(filePath)
	assert.NoError(t, err)
	assert.NoFileExists(t, filePath)
}

func TestReadCache(t *testing.T) {
	instanceId, err := uuid.Parse("aecea348-62c1-4e3d-b848-6d6cdeb1cb9c")
	assert.NoError(t, err)
	cachePath := fmt.Sprintf("/tmp/%v/cache.json", instanceId.String())

	cacheData := createCacheFile(t, cachePath)

	lastConfigApply, err := ReadCache(cachePath)
	assert.NoError(t, err)
	assert.Equal(t, cacheData, lastConfigApply)

	err = os.Remove(cachePath)
	assert.NoError(t, err)
	assert.NoFileExists(t, cachePath)
}

func TestUpdateCache(t *testing.T) {
	instanceId, err := uuid.Parse("aecea348-62c1-4e3d-b848-6d6cdeb1cb9c")
	assert.NoError(t, err)
	cachePath := fmt.Sprintf("/tmp/%v/cache.json", instanceId.String())

	cacheData := createCacheFile(t, cachePath)

	timeFile1, err := createProtoTime("2024-01-08T13:22:23Z")
	assert.NoError(t, err)

	timeFile2, err := createProtoTime("2024-01-08T13:22:23Z")
	assert.NoError(t, err)

	updateCacheData := map[string]*instances.File{
		"/tmp/nginx/locations/metrics.conf": {
			LastModified: timeFile1,
			Path:         "/tmp/nginx/locations/metrics.conf",
			Version:      "ibZkRVjemE5dl.tv88ttUJaXx6UJJMTu",
		},
		"/tmp/nginx/test.conf": {
			LastModified: timeFile2,
			Path:         "/tmp/nginx/test.conf",
			Version:      "Rh3phZuCRwNGANTkdst51he_0WKWy.tZ",
		},
	}

	err = UpdateCache(updateCacheData, cachePath)
	assert.NoError(t, err)

	data, err := ReadCache(cachePath)
	assert.NoError(t, err)
	assert.NotEqual(t, cacheData, data)

	err = os.Remove(cachePath)
	assert.NoError(t, err)
	assert.NoFileExists(t, cachePath)
}

func TestUpdateInstanceConfig(t *testing.T) {
	filePath := "/tmp/test.conf"
	fileContent := []byte("location /test {\n    return 200 \"Test location\\n\";\n}")

	tenantId, instanceId, err := createTestIds()
	assert.NoError(t, err)
	filesUrl := fmt.Sprintf("/instance/%s/files/", instanceId)

	err = WriteFile(fileContent, filePath)
	assert.NoError(t, err)
	assert.FileExists(t, filePath)

	metaDataTime1, err := createProtoTime("2024-01-08T14:22:21Z")
	assert.NoError(t, err)

	metaDataTime2, err := createProtoTime("2024-01-08T13:22:25Z")
	assert.NoError(t, err)

	metaDataReturn := &instances.Files{
		Files: []*instances.File{
			{
				LastModified: metaDataTime1,
				Path:         "/tmp/nginx/nginx.conf",
				Version:      "BDEIFo9anKNvAwWm9O2LpfvNiNiGMx.c",
			},
			{
				LastModified: metaDataTime2,
				Path:         "/tmp/test.conf",
				Version:      "Rh3phZuCRwNGANTkdst51he_0WKWy.tZ",
			},
		},
	}

	getFileReturn := &instances.FileDownloadResponse{
		Encoded:     true,
		FilePath:    "/tmp/test.conf",
		InstanceId:  instanceId.String(),
		FileContent: []byte("location /test {\n    return 200 \"Test location\\n\";\n}"),
	}

	fakeConfigDownloader := &client.FakeHttpConfigDownloaderInterface{}
	fakeConfigDownloader.GetFilesMetadataReturns(metaDataReturn, nil)
	fakeConfigDownloader.GetFileReturns(getFileReturn, nil)

	fileSource := NewFileSource(&FileSourceParameters{
		configDownloader: fakeConfigDownloader,
	})

	cacheTime1, err := createProtoTime("2024-01-08T14:22:21Z")
	assert.NoError(t, err)

	cacheTime2, err := createProtoTime("2024-01-08T12:22:21Z")
	assert.NoError(t, err)

	lastConfigApply := map[string]*instances.File{
		"/tmp/nginx/nginx.conf": {
			LastModified: cacheTime1,
			Path:         "/tmp/nginx/nginx.conf",
			Version:      "BDEIFo9anKNvAwWm9O2LpfvNiNiGMx.c",
		},
		"/tmp/test.conf": {
			LastModified: cacheTime2,
			Path:         "/tmp/test.conf",
			Version:      "Rh3phZuCRwNGANTkdst51he_0WKWy.tZ",
		},
	}

	currentCache, skippedFiles, err := fileSource.UpdateInstanceConfig(lastConfigApply, filesUrl, tenantId)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(skippedFiles))
	assert.NotEqual(t, currentCache, lastConfigApply)

	path := "/tmp/test.conf"
	err = os.Remove(path)
	assert.NoError(t, err)
	assert.NoFileExists(t, path)
}

func createTestIds() (uuid.UUID, uuid.UUID, error) {
	tenantId, err := uuid.Parse("7332d596-d2e6-4d1e-9e75-70f91ef9bd0e")
	if err != nil {
		fmt.Printf("Error creating tenantId: %v", err)
	}

	instanceId, err := uuid.Parse("aecea348-62c1-4e3d-b848-6d6cdeb1cb9c")
	if err != nil {
		fmt.Printf("Error creating instanceId: %v", err)
	}

	return tenantId, instanceId, err
}

func createProtoTime(timeString string) (*timestamppb.Timestamp, error) {
	time, err := time.Parse(time.RFC3339, timeString)
	protoTime := timestamppb.New(time)

	return protoTime, err
}

func createCacheFile(t *testing.T, cachePath string) map[string]*instances.File {
	timeFile1, err := createProtoTime("2024-01-08T13:22:23Z")
	assert.NoError(t, err)

	timeFile2, err := createProtoTime("2024-01-08T13:22:25Z")
	assert.NoError(t, err)

	timeFile3, err := createProtoTime("2024-01-08T13:22:21Z")
	assert.NoError(t, err)

	cacheData := map[string]*instances.File{
		"/tmp/nginx/locations/metrics.conf": {
			LastModified: timeFile1,
			Path:         "/tmp/nginx/locations/metrics.conf",
			Version:      "ibZkRVjemE5dl.tv88ttUJaXx6UJJMTu",
		},
		"/tmp/nginx/locations/test.conf": {
			LastModified: timeFile2,
			Path:         "/tmp/nginx/locations/test.conf",
			Version:      "Rh3phZuCRwNGANTkdst51he_0WKWy.tZ",
		},
		"/tmp/nginx/nginx.conf": {
			LastModified: timeFile3,
			Path:         "/tmp/nginx/nginx.conf",
			Version:      "BDEIFo9anKNvAwWm9O2LpfvNiNiGMx.c",
		},
	}

	cache, err := json.MarshalIndent(cacheData, "", "  ")
	assert.NoError(t, err)

	err = os.MkdirAll(path.Dir(cachePath), 0o750)
	assert.NoError(t, err)

	err = os.WriteFile(cachePath, cache, 0o644)
	assert.FileExists(t, cachePath)
	assert.NoError(t, err)

	return cacheData
}
