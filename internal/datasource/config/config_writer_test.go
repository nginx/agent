// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.
package config

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path"
	"reflect"
	"testing"

	"github.com/nginx/agent/v3/api/grpc/mpi/v1"

	"github.com/nginx/agent/v3/test/helpers"
	"github.com/nginx/agent/v3/test/protos"
	"github.com/nginx/agent/v3/test/types"

	"github.com/google/uuid"
	"github.com/nginx/agent/v3/internal/client/clientfakes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWriteConfig(t *testing.T) {
	ctx := context.Background()
	tempDir := t.TempDir()
	tenantID, instanceID := helpers.CreateTestIDs(t)
	fileContent := []byte("location /test {\n    return 200 \"Test location\\n\";\n}")
	allowedDirs := []string{tempDir}
	agentConfig := types.GetAgentConfig()
	agentConfig.AllowedDirectories = allowedDirs

	instanceIDDir := path.Join(tempDir, instanceID.String())
	helpers.CreateDirWithErrorCheck(t, instanceIDDir)
	defer helpers.RemoveFileWithErrorCheck(t, instanceIDDir)

	testConf := helpers.CreateFileWithErrorCheck(t, tempDir, "test.conf")
	defer helpers.RemoveFileWithErrorCheck(t, testConf.Name())
	nginxConf := helpers.CreateFileWithErrorCheck(t, tempDir, "nginx.conf")
	defer helpers.RemoveFileWithErrorCheck(t, nginxConf.Name())
	metricsConf := helpers.CreateFileWithErrorCheck(t, tempDir, "metrics.conf")
	defer helpers.RemoveFileWithErrorCheck(t, metricsConf.Name())

	testConfPath := testConf.Name()
	filesURL := fmt.Sprintf("/instance/%s/files/", instanceID)

	files, err := protos.GetFiles(nginxConf, testConf, metricsConf)
	require.NoError(t, err)

	tests := []struct {
		name               string
		metaDataReturn     *v1.FileOverview
		getFileReturn      *v1.FileContents
		cacheShouldBeEqual bool
		fileShouldBeEqual  bool
		expSkippedCount    int
	}{
		{
			name:               "Test 1: File needs updating",
			metaDataReturn:     files,
			getFileReturn:      protos.GetFileDownloadResponse(fileContent),
			cacheShouldBeEqual: false,
			fileShouldBeEqual:  true,
			expSkippedCount:    2,
		},
		{
			name:               "Test 2: File doesn't need updating",
			metaDataReturn:     files,
			getFileReturn:      protos.GetFileDownloadResponse(fileContent),
			cacheShouldBeEqual: true,
			fileShouldBeEqual:  false,
			expSkippedCount:    3,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			cacheFile := helpers.CreateFileWithErrorCheck(t, instanceIDDir, "cache.json")
			defer helpers.RemoveFileWithErrorCheck(t, cacheFile.Name())
			cachePath := cacheFile.Name()
			fakeConfigClient := &clientfakes.FakeConfigClientInterface{}
			fakeConfigClient.GetFilesMetadataReturns(test.metaDataReturn, nil)
			fakeConfigClient.GetFileReturns(test.getFileReturn, nil)

			cacheContent, getCacheErr := protos.GetFileCache(nginxConf, testConf, metricsConf)
			require.NoError(t, getCacheErr)

			fileCache := NewFileCache(instanceID.String())
			fileCache.SetCachePath(cachePath)
			err = fileCache.UpdateFileCache(ctx, cacheContent)
			require.NoError(t, err)

			if !test.cacheShouldBeEqual {
				modified, protoErr := protos.CreateProtoTime("2024-01-09T13:20:26Z")
				require.NoError(t, protoErr)
				cacheContent[testConf.Name()].ModifiedTime = modified
				err = fileCache.UpdateFileCache(ctx, cacheContent)
				require.NoError(t, err)
			}

			configWriter, cwErr := NewConfigWriter(agentConfig, fileCache)
			require.NoError(t, cwErr)

			configWriter.SetConfigClient(fakeConfigClient)
			testContent := []byte("location /test {\n    return 200 \"Before Write \\n\";\n}")
			err = writeFile(ctx, testContent, testConfPath)
			require.NoError(t, err)
			assert.FileExists(t, testConfPath)

			skippedFiles, cwErr := configWriter.Write(ctx, filesURL, tenantID.String(), instanceID.String())
			require.NoError(t, cwErr)
			slog.Info("Skipped Files: ", "", skippedFiles)
			assert.Len(t, skippedFiles, test.expSkippedCount)

			res := reflect.DeepEqual(cacheContent, configWriter.currentFileCache)
			assert.Equal(t, test.cacheShouldBeEqual, res)

			assert.NotEqual(t, cacheContent, files)

			testData, readErr := os.ReadFile(testConfPath)
			require.NoError(t, readErr)
			res = reflect.DeepEqual(fileContent, testData)
			assert.Equal(t, test.fileShouldBeEqual, res)
		})
	}
}

func TestDeleteFile(t *testing.T) {
	ctx := context.Background()

	tempDir := t.TempDir()
	_, instanceID := helpers.CreateTestIDs(t)

	agentconfig := types.GetAgentConfig()

	instanceIDDir := path.Join(tempDir, instanceID.String())
	helpers.CreateDirWithErrorCheck(t, instanceIDDir)
	defer helpers.RemoveFileWithErrorCheck(t, instanceIDDir)

	testConf := helpers.CreateFileWithErrorCheck(t, tempDir, "test.conf")
	nginxConf := helpers.CreateFileWithErrorCheck(t, tempDir, "nginx.conf")
	defer helpers.RemoveFileWithErrorCheck(t, nginxConf.Name())
	metricsConf := helpers.CreateFileWithErrorCheck(t, tempDir, "metrics.conf")
	defer helpers.RemoveFileWithErrorCheck(t, metricsConf.Name())

	fileCacheContent, getCacheErr := protos.GetFileCache(nginxConf, testConf, metricsConf)
	require.NoError(t, getCacheErr)

	currentFileCache, getCacheErr := protos.GetFileCache(nginxConf, metricsConf)
	require.NoError(t, getCacheErr)

	tests := []struct {
		name             string
		fileCache        CacheContent
		currentFileCache CacheContent
		fileDeleted      bool
	}{
		{
			name:             "Test 1: File doesn't need deleting",
			fileCache:        fileCacheContent,
			currentFileCache: fileCacheContent,
			fileDeleted:      false,
		},
		{
			name:             "Test 2: File needs deleting",
			fileCache:        fileCacheContent,
			currentFileCache: currentFileCache,
			fileDeleted:      true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			cacheFile := helpers.CreateFileWithErrorCheck(t, instanceIDDir, "cache.json")
			defer helpers.RemoveFileWithErrorCheck(t, cacheFile.Name())
			cachePath := cacheFile.Name()
			fileCache := NewFileCache(instanceID.String())
			fileCache.SetCachePath(cachePath)
			err := fileCache.UpdateFileCache(ctx, test.fileCache)
			require.NoError(t, err)
			slog.Info("", "", &agentconfig)
			configWriter, err := NewConfigWriter(agentconfig, fileCache)
			require.NoError(t, err)

			err = configWriter.removeFiles(ctx, test.currentFileCache, test.fileCache)
			require.NoError(t, err)
			if test.fileDeleted {
				assert.NoFileExists(t, testConf.Name())
			} else {
				assert.FileExists(t, testConf.Name())
			}
		})
	}
}

func TestRollback(t *testing.T) {
	ctx := context.Background()
	tempDir := t.TempDir()
	tenantID, instanceID := helpers.CreateTestIDs(t)
	allowedDirs := []string{tempDir}

	instanceIDDir := path.Join(tempDir, instanceID.String())
	helpers.CreateDirWithErrorCheck(t, instanceIDDir)
	defer helpers.RemoveFileWithErrorCheck(t, instanceIDDir)

	cacheFile := helpers.CreateFileWithErrorCheck(t, instanceIDDir, "cache.json")
	defer helpers.RemoveFileWithErrorCheck(t, cacheFile.Name())
	nginxConf := helpers.CreateFileWithErrorCheck(t, tempDir, "nginx.conf")
	defer helpers.RemoveFileWithErrorCheck(t, nginxConf.Name())
	metricsConf := helpers.CreateFileWithErrorCheck(t, tempDir, "metrics.conf")
	defer helpers.RemoveFileWithErrorCheck(t, metricsConf.Name())
	testConf := helpers.CreateFileWithErrorCheck(t, tempDir, "test.conf")
	defer helpers.RemoveFileWithErrorCheck(t, testConf.Name())

	cachePath := cacheFile.Name()
	filesURL := fmt.Sprintf("/instance/%s/files/", instanceID)

	fileContent := []byte("location /test {\n    return 200 \"Test location\\n\";\n}")
	err := writeFile(ctx, fileContent, testConf.Name())
	require.NoError(t, err)
	assert.FileExists(t, testConf.Name())

	files, err := protos.GetFiles(nginxConf, testConf, metricsConf)
	require.NoError(t, err)

	cacheContent, getCacheErr := protos.GetFileCache(nginxConf, testConf, metricsConf)
	require.NoError(t, getCacheErr)

	fakeConfigClient := &clientfakes.FakeConfigClientInterface{}
	fakeConfigClient.GetFilesMetadataReturns(files, nil)
	resp := []byte("location /test {\n    return 200 \"Test changed\\n\";\n}")
	fakeConfigClient.GetFileReturns(protos.GetFileDownloadResponse(resp), nil)

	agentConfig := types.GetAgentConfig()
	agentConfig.AllowedDirectories = allowedDirs

	fileCache := NewFileCache(instanceID.String())
	fileCache.SetCachePath(cachePath)
	err = fileCache.UpdateFileCache(ctx, cacheContent)
	require.NoError(t, err)

	configWriter, err := NewConfigWriter(agentConfig, fileCache)
	require.NoError(t, err)
	configWriter.SetConfigClient(fakeConfigClient)

	fileTime1, err := protos.CreateProtoTime("2024-01-08T14:22:21Z")
	require.NoError(t, err)

	fileTime2, err := protos.CreateProtoTime("2024-01-08T13:22:23Z")
	require.NoError(t, err)

	skippedFiles := CacheContent{
		metricsConf.Name(): {
			ModifiedTime: fileTime1,
			Name:         "/tmp/nginx/locations/metrics.conf",
			Hash:         "ibZkRVjemE5dl.tv88ttUJaXx6UJJMTu",
		},
		nginxConf.Name(): {
			ModifiedTime: fileTime2,
			Name:         "/tmp/nginx/test.conf",
			Hash:         "Rh3phZuCRwNGANTkdst51he_0WKWy.tZ",
		},
	}

	err = configWriter.Rollback(ctx, skippedFiles, filesURL, tenantID.String(), instanceID.String())
	require.NoError(t, err)

	data, err := os.ReadFile(testConf.Name())
	require.NoError(t, err)
	assert.NotEqual(t, data, fileContent)
}

func TestComplete(t *testing.T) {
	ctx := context.Background()

	tempDir := t.TempDir()
	instanceID, err := uuid.Parse("aecea348-62c1-4e3d-b848-6d6cdeb1cb9c")
	require.NoError(t, err)

	instanceIDDir := fmt.Sprintf("%s/%s/", tempDir, instanceID.String())

	helpers.CreateDirWithErrorCheck(t, instanceIDDir)
	defer helpers.RemoveFileWithErrorCheck(t, instanceIDDir)

	cacheFile := helpers.CreateFileWithErrorCheck(t, instanceIDDir, "cache.json")
	defer helpers.RemoveFileWithErrorCheck(t, cacheFile.Name())
	cachePath := cacheFile.Name()

	allowedDirs := []string{tempDir}
	fakeConfigClient := &clientfakes.FakeConfigClientInterface{}

	fileCache := NewFileCache(instanceID.String())
	agentConfig := types.GetAgentConfig()
	agentConfig.AllowedDirectories = allowedDirs
	fileCache.SetCachePath(cachePath)

	configWriter, err := NewConfigWriter(agentConfig, fileCache)
	require.NoError(t, err)
	configWriter.configClient = fakeConfigClient

	testConf := helpers.CreateFileWithErrorCheck(t, tempDir, "test.conf")
	defer helpers.RemoveFileWithErrorCheck(t, testConf.Name())
	nginxConf := helpers.CreateFileWithErrorCheck(t, tempDir, "nginx.conf")
	defer helpers.RemoveFileWithErrorCheck(t, nginxConf.Name())
	metricsConf := helpers.CreateFileWithErrorCheck(t, tempDir, "metrics.conf")
	defer helpers.RemoveFileWithErrorCheck(t, metricsConf.Name())

	cacheData, err := protos.GetFileCache(testConf, nginxConf, metricsConf)
	require.NoError(t, err)

	configWriter.currentFileCache, err = protos.GetFileCache(testConf, metricsConf)
	require.NoError(t, err)

	helpers.CreateCacheFiles(t, cachePath, cacheData)

	err = configWriter.Complete(ctx)
	require.NoError(t, err)

	data, err := configWriter.fileCache.ReadFileCache(ctx)
	require.NoError(t, err)
	assert.NotEqual(t, cacheData, data)
}

func TestWriteFile(t *testing.T) {
	ctx := context.Background()

	tempDir := t.TempDir()
	filePath := fmt.Sprintf("%s/nginx.conf", tempDir)
	fileContent := []byte("location /test {\n    return 200 \"Test location\\n\";\n}")

	err := writeFile(ctx, fileContent, filePath)
	require.NoError(t, err)
	assert.FileExists(t, filePath)

	data, err := os.ReadFile(filePath)
	require.NoError(t, err)
	assert.Equal(t, fileContent, data)

	helpers.RemoveFileWithErrorCheck(t, filePath)
	assert.NoFileExists(t, filePath)
}

func TestIsFilePathValid(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name           string
		path           string
		expectedResult bool
	}{
		{
			name:           "Test 1: Valid path",
			path:           "/tmp/test.conf",
			expectedResult: true,
		},
		{
			name:           "Test 2: Directory path",
			path:           "/tmp/",
			expectedResult: false,
		},
		{
			name:           "Test 3: Invalid path",
			path:           "/",
			expectedResult: false,
		},
		{
			name:           "Test 4: Empty path",
			path:           "",
			expectedResult: false,
		},
		{
			name:           "Test 5: Path not allowed directory",
			path:           "./test/test.conf",
			expectedResult: false,
		},
	}

	fakeConfigClient := &clientfakes.FakeConfigClientInterface{}
	cachePath := fmt.Sprintf(cacheLocation, "aecea348-62c1-4e3d-b848-6d6cdeb1cb9c")

	fileCache := NewFileCache("aecea348-62c1-4e3d-b848-6d6cdeb1cb9c")
	fileCache.SetCachePath(cachePath)
	agentConfig := types.GetAgentConfig()

	configWriter, err := NewConfigWriter(agentConfig, fileCache)
	require.NoError(t, err)
	configWriter.configClient = fakeConfigClient

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			valid := configWriter.isFilePathValid(ctx, test.path)
			assert.Equal(t, test.expectedResult, valid)
		})
	}
}

func TestDoesFileRequireUpdate(t *testing.T) {
	fileTime1, err := protos.CreateProtoTime("2024-01-08T14:22:21Z")
	require.NoError(t, err)

	fileTime2, err := protos.CreateProtoTime("2024-01-08T13:22:23Z")
	require.NoError(t, err)

	previousFileCache := CacheContent{
		"/tmp/nginx/locations/metrics.conf": {
			ModifiedTime: fileTime1,
			Name:         "/tmp/nginx/locations/metrics.conf",
			Hash:         "ibZkRVjemE5dl.tv88ttUJaXx6UJJMTu",
		},
		"/tmp/nginx/test.conf": {
			ModifiedTime: fileTime2,
			Name:         "/tmp/nginx/test.conf",
			Hash:         "Rh3phZuCRwNGANTkdst51he_0WKWy.tZ",
		},
	}

	updateTimeFile1, err := protos.CreateProtoTime("2024-01-08T14:22:23Z")
	require.NoError(t, err)

	tests := []struct {
		name            string
		lastConfigApply CacheContent
		fileData        *v1.FileMeta
		expectedResult  bool
	}{
		{
			name:            "Test 1: File is latest version",
			lastConfigApply: previousFileCache,
			fileData: &v1.FileMeta{
				ModifiedTime: fileTime1,
				Name:         "/tmp/nginx/locations/metrics.conf",
				Hash:         "ibZkRVjemE5dl.tv88ttUJaXx6UJJMTu",
			},
			expectedResult: false,
		},
		{
			name:            "Test 2: File needs updating",
			lastConfigApply: previousFileCache,
			fileData: &v1.FileMeta{
				ModifiedTime: updateTimeFile1,
				Name:         "/tmp/nginx/test.conf",
				Hash:         "ibZkRVjemE5dl.tv88ttUJaXx6UJJMTu",
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
