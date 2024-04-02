// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package config

import (
	"context"
	"fmt"
	"path"
	"testing"

	"github.com/nginx/agent/v3/test/helpers"
	"github.com/nginx/agent/v3/test/protos"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUpdateCache(t *testing.T) {
	ctx := context.Background()

	_, instanceID := helpers.CreateTestIDs(t)
	tempDir := t.TempDir()
	instanceIDDir := path.Join(tempDir, instanceID.String())
	helpers.CreateDirWithErrorCheck(t, instanceIDDir)
	defer helpers.RemoveFileWithErrorCheck(t, instanceIDDir)

	cacheFile := helpers.CreateFileWithErrorCheck(t, instanceIDDir, "cache.json")

	cacheData := CacheContent{}

	helpers.CreateCacheFiles(t, cacheFile.Name(), cacheData)

	nginxConf := helpers.CreateFileWithErrorCheck(t, tempDir, "nginx.conf")
	metricsConf := helpers.CreateFileWithErrorCheck(t, tempDir, "metrics.conf")
	testConf := helpers.CreateFileWithErrorCheck(t, tempDir, "test.conf")

	expected, getCacheErr := protos.GetFileCache(nginxConf, testConf, metricsConf)
	require.NoError(t, getCacheErr)

	fileCache := NewFileCache(instanceID.String())
	fileCache.SetCachePath(cacheFile.Name())

	err := fileCache.UpdateFileCache(ctx, expected)
	require.NoError(t, err)

	helpers.RemoveFileWithErrorCheck(t, metricsConf.Name())
	helpers.RemoveFileWithErrorCheck(t, nginxConf.Name())
	helpers.RemoveFileWithErrorCheck(t, cacheFile.Name())
	assert.Equal(t, expected, fileCache.cacheContent)
}

func TestSetCachePath(t *testing.T) {
	_, instanceID := helpers.CreateTestIDs(t)
	fileCache := NewFileCache(instanceID.String())
	expected := fmt.Sprintf(cacheLocation, instanceID)

	fileCache.SetCachePath(expected)

	assert.Equal(t, expected, fileCache.CachePath)
}

func TestReadCache(t *testing.T) {
	ctx := context.Background()
	_, instanceID := helpers.CreateTestIDs(t)

	tempDir := t.TempDir()
	instanceIDDir := fmt.Sprintf("%s/%s/", tempDir, instanceID.String())
	helpers.CreateDirWithErrorCheck(t, instanceIDDir)
	defer helpers.RemoveFileWithErrorCheck(t, instanceIDDir)

	cacheFile := helpers.CreateFileWithErrorCheck(t, instanceIDDir, "cache.json")
	cachePath := cacheFile.Name()

	testConf := helpers.CreateFileWithErrorCheck(t, tempDir, "test.conf")
	nginxConf := helpers.CreateFileWithErrorCheck(t, tempDir, "nginx.conf")
	metricsConf := helpers.CreateFileWithErrorCheck(t, tempDir, "metrics.conf")

	cacheData, err := protos.GetFileCache(testConf, nginxConf, metricsConf)
	require.NoError(t, err)

	helpers.CreateCacheFiles(t, cachePath, cacheData)

	tests := []struct {
		name            string
		path            string
		shouldHaveError bool
	}{
		{
			name:            "Test 1: Cache file exists",
			path:            cachePath,
			shouldHaveError: false,
		},
		{
			name:            "Test 2: Cache file doesn't exist",
			path:            "/tmp/cache.json",
			shouldHaveError: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			fileCache := NewFileCache(instanceID.String())
			fileCache.SetCachePath(test.path)
			cacheFile, readErr := fileCache.ReadFileCache(ctx)

			if test.shouldHaveError {
				require.Error(t, readErr)
				assert.NotEqual(t, cacheData, cacheFile)
			} else {
				require.NoError(t, readErr)
				assert.Equal(t, cacheData, cacheFile)
			}
		})
	}

	helpers.RemoveFileWithErrorCheck(t, cacheFile.Name())
	helpers.RemoveFileWithErrorCheck(t, metricsConf.Name())
	helpers.RemoveFileWithErrorCheck(t, nginxConf.Name())
	assert.NoFileExists(t, cachePath)
}
