/**
 * Copyright (c) F5, Inc.
 *
 * This source code is licensed under the Apache License, Version 2.0 license found in the
 * LICENSE file in the root directory of this source tree.
 */
package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"github.com/nginx/agent/v3/api/grpc/instances"
	"github.com/nginx/agent/v3/internal/client"

	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestWriteConfig(t *testing.T) {
	filePath := "/tmp/test.conf"
	fileContent := []byte("location /test {\n    return 200 \"Test location\\n\";\n}")
	cachePath := "/tmp/cache.json"

	tenantId, instanceId, err := createTestIds()
	assert.NoError(t, err)
	filesUrl := fmt.Sprintf("/instance/%s/files/", instanceId)

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

	fakeConfigClient := &client.FakeHttpConfigClientInterface{}
	fakeConfigClient.GetFilesMetadataReturns(metaDataReturn, nil)
	fakeConfigClient.GetFileReturns(getFileReturn, nil)

	_, err = createCacheFile(cachePath)
	assert.NoError(t, err)

	configWriter := NewConfigWriter(&ConfigWriterParameters{
		configClient: fakeConfigClient,
		Client: Client{
			Timeout: time.Second * 10,
		},
	}, cachePath)

	err = writeFile(fileContent, filePath)
	assert.NoError(t, err)
	assert.FileExists(t, filePath)

	cacheTime1, err := createProtoTime("2024-01-08T14:22:21Z")
	assert.NoError(t, err)

	cacheTime2, err := createProtoTime("2024-01-08T12:22:21Z")
	assert.NoError(t, err)

	previouseFileCache := FileCache{
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

	skippedFiles, err := configWriter.Write(filesUrl, tenantId)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(skippedFiles))
	assert.NotEqual(t, configWriter.currentFileCache, previouseFileCache)
	path := "/tmp/test.conf"
	err = os.Remove(path)
	assert.NoError(t, err)
	assert.NoFileExists(t, path)
}

func TestIsPathValid(t *testing.T) {
	tests := []struct {
		name           string
		path           string
		expectedResult bool
	}{
		{
			name:           "valid path",
			path:           "/tmp/test.conf",
			expectedResult: true,
		},
		{
			name:           "directory path",
			path:           "/tmp/",
			expectedResult: false,
		},
		{
			name:           "invalid path",
			path:           "/",
			expectedResult: false,
		},
		{
			name:           "empty path",
			path:           "",
			expectedResult: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			valid := isFilePathValid(test.path)
			assert.Equal(t, test.expectedResult, valid)
		})
	}
}

func TestDoesFileRequireUpdate(t *testing.T) {
	fileTime1, err := createProtoTime("2024-01-08T14:22:21Z")
	assert.NoError(t, err)

	fileTime2, err := createProtoTime("2024-01-08T13:22:23Z")
	assert.NoError(t, err)

	previousFileCache := FileCache{
		"/tmp/nginx/locations/metrics.conf": {
			LastModified: fileTime1,
			Path:         "/tmp/nginx/locations/metrics.conf",
			Version:      "ibZkRVjemE5dl.tv88ttUJaXx6UJJMTu",
		},
		"/tmp/nginx/test.conf": {
			LastModified: fileTime2,
			Path:         "/tmp/nginx/test.conf",
			Version:      "Rh3phZuCRwNGANTkdst51he_0WKWy.tZ",
		},
	}

	updateTimeFile1, err := createProtoTime("2024-01-08T14:22:23Z")
	assert.NoError(t, err)

	tests := []struct {
		name            string
		lastConfigApply FileCache
		fileData        *instances.File
		expectedResult  bool
	}{
		{
			name:            "file is latest",
			lastConfigApply: previousFileCache,
			fileData: &instances.File{
				LastModified: fileTime1,
				Path:         "/tmp/nginx/locations/metrics.conf",
				Version:      "ibZkRVjemE5dl.tv88ttUJaXx6UJJMTu",
			},
			expectedResult: false,
		},
		{
			name:            "file needs updating",
			lastConfigApply: previousFileCache,
			fileData: &instances.File{
				LastModified: updateTimeFile1,
				Path:         "/tmp/nginx/test.conf",
				Version:      "ibZkRVjemE5dl.tv88ttUJaXx6UJJMTu",
			},
			expectedResult: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			valid := doesFileRequireUpdate(test.lastConfigApply, test.fileData)
			assert.Equal(t, test.expectedResult, valid)
		})
	}
}

func TestWriteFile(t *testing.T) {
	filePath := "/tmp/test.conf"
	fileContent := []byte("location /test {\n    return 200 \"Test location\\n\";\n}")

	err := writeFile(fileContent, filePath)
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
	cachePath := fmt.Sprintf("/tmp/%s/cache.json", instanceId.String())

	cacheData, err := createCacheFile(cachePath)
	assert.NoError(t, err)

	previousFileCache, err := readInstanceCache(cachePath)
	assert.NoError(t, err)
	assert.Equal(t, cacheData, previousFileCache)

	err = os.Remove(cachePath)
	assert.NoError(t, err)
	assert.NoFileExists(t, cachePath)
}

func TestUpdateCache(t *testing.T) {
	instanceId, err := uuid.Parse("aecea348-62c1-4e3d-b848-6d6cdeb1cb9c")
	assert.NoError(t, err)
	cachePath := fmt.Sprintf("/tmp/%s/cache.json", instanceId.String())

	cacheData, err := createCacheFile(cachePath)
	assert.NoError(t, err)

	fileTime1, err := createProtoTime("2024-01-08T13:22:23Z")
	assert.NoError(t, err)

	fileTime2, err := createProtoTime("2024-01-08T13:22:23Z")
	assert.NoError(t, err)

	currentFileCache := FileCache{
		"/tmp/nginx/locations/metrics.conf": {
			LastModified: fileTime1,
			Path:         "/tmp/nginx/locations/metrics.conf",
			Version:      "ibZkRVjemE5dl.tv88ttUJaXx6UJJMTu",
		},
		"/tmp/nginx/test.conf": {
			LastModified: fileTime2,
			Path:         "/tmp/nginx/test.conf",
			Version:      "Rh3phZuCRwNGANTkdst51he_0WKWy.tZ",
		},
	}

	err = updateCache(currentFileCache, cachePath)
	assert.NoError(t, err)

	data, err := readInstanceCache(cachePath)
	assert.NoError(t, err)
	assert.NotEqual(t, cacheData, data)

	err = os.Remove(cachePath)
	assert.NoError(t, err)
	assert.NoFileExists(t, cachePath)
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

func createCacheFile(cachePath string) (FileCache, error) {
	fileTime1, err := createProtoTime("2024-01-08T13:22:23Z")
	if err != nil {
		return nil, fmt.Errorf("error creating time, error: %w", err)
	}
	fileTime2, err := createProtoTime("2024-01-08T13:22:25Z")
	if err != nil {
		return nil, fmt.Errorf("error creating time, error: %w", err)
	}

	fileTime3, err := createProtoTime("2024-01-08T13:22:21Z")
	if err != nil {
		return nil, fmt.Errorf("error creating time, error: %w", err)
	}

	cacheData := FileCache{
		"/tmp/nginx/locations/metrics.conf": {
			LastModified: fileTime1,
			Path:         "/tmp/nginx/locations/metrics.conf",
			Version:      "ibZkRVjemE5dl.tv88ttUJaXx6UJJMTu",
		},
		"/tmp/nginx/locations/test.conf": {
			LastModified: fileTime2,
			Path:         "/tmp/nginx/locations/test.conf",
			Version:      "Rh3phZuCRwNGANTkdst51he_0WKWy.tZ",
		},
		"/tmp/nginx/nginx.conf": {
			LastModified: fileTime3,
			Path:         "/tmp/nginx/nginx.conf",
			Version:      "BDEIFo9anKNvAwWm9O2LpfvNiNiGMx.c",
		},
	}

	cache, err := json.MarshalIndent(cacheData, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("error marshaling cache, error: %w", err)
	}

	err = os.MkdirAll(path.Dir(cachePath), 0o750)
	if err != nil {
		return nil, fmt.Errorf("error creating cache directory, error: %w", err)
	}

	err = os.WriteFile(cachePath, cache, 0o644)
	if err != nil {
		return nil, fmt.Errorf("error writing to file, error: %w", err)
	}

	return cacheData, err
}
