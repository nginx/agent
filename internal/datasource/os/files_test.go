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

	cacheData, err := createCacheFile(cachePath)
	assert.NoError(t, err)

	lastConfigApply, err := ReadInstanceCache(cachePath)
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

	cacheData, err := createCacheFile(cachePath)
	assert.NoError(t, err)

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

	data, err := ReadInstanceCache(cachePath)
	assert.NoError(t, err)
	assert.NotEqual(t, cacheData, data)

	err = os.Remove(cachePath)
	assert.NoError(t, err)
	assert.NoFileExists(t, cachePath)
}

func createProtoTime(timeString string) (*timestamppb.Timestamp, error) {
	time, err := time.Parse(time.RFC3339, timeString)
	protoTime := timestamppb.New(time)

	return protoTime, err
}

func createCacheFile(cachePath string) (map[string]*instances.File, error) {
	timeFile1, err := createProtoTime("2024-01-08T13:22:23Z")
	if err != nil {
		return nil, fmt.Errorf("error creating time, error: %v", err)
	}
	timeFile2, err := createProtoTime("2024-01-08T13:22:25Z")
	if err != nil {
		return nil, fmt.Errorf("error creating time, error: %v", err)
	}

	timeFile3, err := createProtoTime("2024-01-08T13:22:21Z")
	if err != nil {
		return nil, fmt.Errorf("error creating time, error: %v", err)
	}

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
	if err != nil {
		return nil, fmt.Errorf("error marshaling cache, error: %v", err)
	}

	err = os.MkdirAll(path.Dir(cachePath), 0o750)
	if err != nil {
		return nil, fmt.Errorf("error creating cache directory, error: %v", err)
	}

	err = os.WriteFile(cachePath, cache, 0o644)
	if err != nil {
		return nil, fmt.Errorf("error writing to file, error: %v", err)
	}

	return cacheData, err
}
