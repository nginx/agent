// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package config

import (
	"fmt"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	instanceID = "aecea348-62c1-4e3d-b848-6d6cdeb1cb9c"
)

func TestUpdateCache(t *testing.T) {
	instanceIDDir := fmt.Sprintf("./%s/", instanceID)

	err := os.Mkdir(instanceIDDir, 0o755)
	require.NoError(t, err)
	defer os.Remove(instanceIDDir)
	cacheFile, err := os.CreateTemp(instanceIDDir, "cache.json")
	require.NoError(t, err)
	defer os.Remove(cacheFile.Name())

	cacheData := CacheContent{}

	err = createCacheFile(cacheFile.Name(), cacheData)
	require.NoError(t, err)

	fileTime1, err := createProtoTime("2024-01-08T13:22:23Z")
	require.NoError(t, err)

	fileTime2, err := createProtoTime("2024-01-08T13:22:25Z")
	require.NoError(t, err)

	fileTime3, err := createProtoTime("2024-01-08T13:22:21Z")
	require.NoError(t, err)

	testConf, err := os.CreateTemp(".", "test.conf")
	require.NoError(t, err)
	defer os.Remove(testConf.Name())

	nginxConf, err := os.CreateTemp("./", "nginx.conf")
	require.NoError(t, err)
	defer os.Remove(nginxConf.Name())

	metricsConf, err := os.CreateTemp("./", "metrics.conf")
	require.NoError(t, err)
	defer os.Remove(metricsConf.Name())

	expected := CacheContent{
		nginxConf.Name(): {
			LastModified: fileTime1,
			Path:         nginxConf.Name(),
			Version:      "BDEIFo9anKNvAwWm9O2LpfvNiNiGMx.c",
		},
		testConf.Name(): {
			LastModified: fileTime2,
			Path:         testConf.Name(),
			Version:      "Rh3phZuCRwNGANTkdst51he_0WKWy.tZ",
		},
		metricsConf.Name(): {
			LastModified: fileTime3,
			Path:         metricsConf.Name(),
			Version:      "ibZkRVjemE5dl.tv88ttUJaXx6UJJMTu",
		},
	}

	fileCache := NewFileCache(instanceID)
	fileCache.SetCachePath(cacheFile.Name())

	err = fileCache.UpdateFileCache(expected)
	require.NoError(t, err)

	assert.Equal(t, expected, fileCache.cacheContent)
}

func TestSetCachePath(t *testing.T) {
	fileCache := NewFileCache(instanceID)
	expected := fmt.Sprintf(cacheLocation, instanceID)

	fileCache.SetCachePath(expected)

	assert.Equal(t, expected, fileCache.cachePath)
}

func TestGetCachePath(t *testing.T) {
	fileCache := NewFileCache(instanceID)
	expected := fmt.Sprintf(cacheLocation, instanceID)

	fileCache.SetCachePath(expected)

	result := fileCache.GetCachePath()

	assert.Equal(t, expected, result)
}

func TestReadCache(t *testing.T) {
	instanceID, err := uuid.Parse("aecea348-62c1-4e3d-b848-6d6cdeb1cb9c")
	require.NoError(t, err)

	instanceIDDir := fmt.Sprintf("./%s/", instanceID.String())

	err = os.Mkdir(instanceIDDir, 0o755)
	require.NoError(t, err)
	defer os.Remove(instanceIDDir)

	cacheFile, err := os.CreateTemp(instanceIDDir, "cache.json")
	require.NoError(t, err)
	defer os.Remove(cacheFile.Name())

	cachePath := cacheFile.Name()

	fileTime1, err := createProtoTime("2024-01-08T13:22:23Z")
	require.NoError(t, err)

	fileTime2, err := createProtoTime("2024-01-08T13:22:25Z")
	require.NoError(t, err)

	fileTime3, err := createProtoTime("2024-01-08T13:22:21Z")
	require.NoError(t, err)

	testConf, err := os.CreateTemp(".", "test.conf")
	require.NoError(t, err)
	defer os.Remove(testConf.Name())

	nginxConf, err := os.CreateTemp("./", "nginx.conf")
	require.NoError(t, err)
	defer os.Remove(nginxConf.Name())

	metricsConf, err := os.CreateTemp("./", "metrics.conf")
	require.NoError(t, err)
	defer os.Remove(metricsConf.Name())

	cacheData := CacheContent{
		nginxConf.Name(): {
			LastModified: fileTime1,
			Path:         nginxConf.Name(),
			Version:      "BDEIFo9anKNvAwWm9O2LpfvNiNiGMx.c",
		},
		testConf.Name(): {
			LastModified: fileTime2,
			Path:         testConf.Name(),
			Version:      "Rh3phZuCRwNGANTkdst51he_0WKWy.tZ",
		},
		metricsConf.Name(): {
			LastModified: fileTime3,
			Path:         metricsConf.Name(),
			Version:      "ibZkRVjemE5dl.tv88ttUJaXx6UJJMTu",
		},
	}

	err = createCacheFile(cachePath, cacheData)
	require.NoError(t, err)

	tests := []struct {
		name            string
		path            string
		shouldHaveError bool
	}{
		{
			name:            "cache file exists",
			path:            cachePath,
			shouldHaveError: false,
		},
		{
			name:            "cache file doesn't exist",
			path:            "/tmp/cache.json",
			shouldHaveError: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			fileCache := NewFileCache("aecea348-62c1-4e3d-b848-6d6cdeb1cb9c")
			fileCache.SetCachePath(test.path)
			cacheFile, readErr := fileCache.ReadFileCache()

			if test.shouldHaveError {
				require.Error(t, readErr)
				assert.NotEqual(t, cacheData, cacheFile)
			} else {
				require.NoError(t, readErr)
				assert.Equal(t, cacheData, cacheFile)
			}
		})
	}

	err = os.Remove(cachePath)
	require.NoError(t, err)
	assert.NoFileExists(t, cachePath)
}
