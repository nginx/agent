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

	// "net/http"
	// "net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/nginx/agent/v3/api/grpc/instances"
	// "github.com/nginx/agent/v3/internal/client"

	// "github.com/nginx/agent/v3/internal/client"
	"github.com/stretchr/testify/assert"

	"google.golang.org/protobuf/types/known/timestamppb"
)

func createCacheFile(t *testing.T, cachePath string) map[string]*instances.File {
	timeFile1, err := time.Parse(time.RFC3339, "2024-01-08T13:22:23Z")
	protoTimeFile1 := timestamppb.New(timeFile1)
	assert.NoError(t, err)

	timeFile2, err := time.Parse(time.RFC3339, "2024-01-08T13:22:25Z")
	protoTimeFile2 := timestamppb.New(timeFile2)
	assert.NoError(t, err)

	timeFile3, err := time.Parse(time.RFC3339, "2024-01-08T13:22:21Z")
	protoTimeFile3 := timestamppb.New(timeFile3)
	assert.NoError(t, err)

	cacheData := map[string]*instances.File{
		"/usr/local/etc/nginx/locations/metrics.conf": {
			LastModified: protoTimeFile1,
			Path:         "/usr/local/etc/nginx/locations/metrics.conf",
			Version:      "ibZkRVjemE5dl.tv88ttUJaXx6UJJMTu",
		},
		"/usr/local/etc/nginx/locations/test.conf": {
			LastModified: protoTimeFile2,
			Path:         "/usr/local/etc/nginx/locations/test.conf",
			Version:      "Rh3phZuCRwNGANTkdst51he_0WKWy.tZ",
		},
		"/usr/local/etc/nginx/nginx.conf": {
			LastModified: protoTimeFile3,
			Path:         "/usr/local/etc/nginx/nginx.conf",
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

	timeFile1, err := time.Parse(time.RFC3339, "2024-01-08T13:22:23Z")
	protoTimeFile1 := timestamppb.New(timeFile1)
	assert.NoError(t, err)

	timeFile2, err := time.Parse(time.RFC3339, "2024-01-08T13:22:25Z")
	protoTimeFile2 := timestamppb.New(timeFile2)
	assert.NoError(t, err)

	updateCacheData := map[string]*instances.File{
		"/usr/local/etc/nginx/locations/metrics.conf": {
			LastModified: protoTimeFile1,
			Path:         "/usr/local/etc/nginx/locations/metrics.conf",
			Version:      "ibZkRVjemE5dl.tv88ttUJaXx6UJJMTu",
		},
		"/usr/local/etc/nginx/test.conf": {
			LastModified: protoTimeFile2,
			Path:         "/usr/local/etc/nginx/test.conf",
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

// func TestUpdateInstanceConfig(t *testing.T) {
// 	configDownloader := client.FakeHttpConfigDownloaderInterface{}

// 	configDownloader.
// }
