// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.
package config

import (
	"context"
	"fmt"
	"os"
	"path"
	"reflect"
	"testing"

	helpers "github.com/nginx/agent/v3/test"

	config2 "github.com/nginx/agent/v3/internal/config"

	"github.com/google/uuid"
	"github.com/nginx/agent/v3/api/grpc/instances"
	"github.com/nginx/agent/v3/internal/client/clientfakes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWriteConfig(t *testing.T) {
	ctx := context.TODO()
	tempDir := t.TempDir()
	tenantID, instanceID := helpers.CreateTestIDs(t)
	fileContent := []byte("location /test {\n    return 200 \"Test location\\n\";\n}")
	allowedDirs := []string{"./", "/var/"}
	agentconfig := config2.Config{
		AllowedDirectories: allowedDirs,
	}

	instanceIDDir := path.Join(tempDir, instanceID.String())
	helpers.CreateDirWithErrorCheck(t, instanceIDDir)
	defer helpers.RemoveFileWithErrorCheck(t, instanceIDDir)

	testConf := helpers.CreateFileWithErrorCheck(t, tempDir, "test.conf")
	nginxConf := helpers.CreateFileWithErrorCheck(t, tempDir, "nginx.conf")
	metricsConf := helpers.CreateFileWithErrorCheck(t, tempDir, "metrics.conf")
	cacheFile := helpers.CreateFileWithErrorCheck(t, instanceIDDir, "cache.json")

	testConfPath := testConf.Name()
	cachePath := cacheFile.Name()
	filesURL := fmt.Sprintf("/instance/%s/files/", instanceID)

	files, err := helpers.GetFiles(nginxConf, testConf, metricsConf)
	require.NoError(t, err)

	tests := []struct {
		name               string
		metaDataReturn     *instances.Files
		getFileReturn      *instances.FileDownloadResponse
		cacheShouldBeEqual bool
		fileShouldBeEqual  bool
		skipped            int
	}{
		{
			name:               "file needs updating",
			metaDataReturn:     files,
			getFileReturn:      helpers.GetFileDownloadResponse(testConf.Name(), instanceID.String(), fileContent),
			cacheShouldBeEqual: false,
			fileShouldBeEqual:  true,
			skipped:            2,
		},
		{
			name:               "file doesn't need updating",
			metaDataReturn:     files,
			getFileReturn:      helpers.GetFileDownloadResponse(testConf.Name(), instanceID.String(), fileContent),
			cacheShouldBeEqual: true,
			fileShouldBeEqual:  false,
			skipped:            3,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			fakeConfigClient := &clientfakes.FakeConfigClientInterface{}
			fakeConfigClient.GetFilesMetadataReturns(test.metaDataReturn, nil)
			fakeConfigClient.GetFileReturns(test.getFileReturn, nil)

			cacheContent, getCacheErr := helpers.GetFileCache(nginxConf, testConf, metricsConf)
			require.NoError(t, getCacheErr)

			fileCache := NewFileCache(instanceID.String())
			fileCache.SetCachePath(cachePath)
			err := fileCache.UpdateFileCache(cacheContent)
			require.NoError(t, err)

			if !test.cacheShouldBeEqual {
				modified, protoErr := helpers.CreateProtoTime("2024-01-09T13:20:26Z")
				require.NoError(t, protoErr)
				cacheContent[testConf.Name()].LastModified = modified
				err = fileCache.UpdateFileCache(cacheContent)
				require.NoError(t, err)
			}

			configWriter, err := NewConfigWriter(&agentconfig, fileCache)
			require.NoError(t, err)

			configWriter.SetConfigClient(fakeConfigClient)
			testConent := []byte("location /test {\n    return 200 \"Before Write \\n\";\n}")
			err = writeFile(testConent, testConfPath)
			require.NoError(t, err)
			assert.FileExists(t, testConfPath)

			skippedFiles, err := configWriter.Write(ctx, filesURL, tenantID.String(), instanceID.String())
			require.NoError(t, err)
			assert.Len(t, skippedFiles, test.skipped)

			res := reflect.DeepEqual(cacheContent, configWriter.currentFileCache)
			assert.Equal(t, test.cacheShouldBeEqual, res)

			assert.NotEqual(t, cacheContent, files)

			testData, readErr := os.ReadFile(test.getFileReturn.GetFilePath())
			require.NoError(t, readErr)
			res = reflect.DeepEqual(fileContent, testData)
			assert.Equal(t, test.fileShouldBeEqual, res)

			helpers.RemoveFileWithErrorCheck(t, test.getFileReturn.GetFilePath())
			assert.NoFileExists(t, test.getFileReturn.GetFilePath())
		})
	}
	helpers.RemoveFileWithErrorCheck(t, cacheFile.Name())
	helpers.RemoveFileWithErrorCheck(t, metricsConf.Name())
	helpers.RemoveFileWithErrorCheck(t, testConf.Name())
	helpers.RemoveFileWithErrorCheck(t, nginxConf.Name())
	assert.NoFileExists(t, cachePath)
}

func TestRollback(t *testing.T) {
	ctx := context.TODO()
	tempDir := t.TempDir()
	tenantID, instanceID := helpers.CreateTestIDs(t)
	allowedDirs := []string{"./", "/var/"}

	instanceIDDir := path.Join(tempDir, instanceID.String())
	helpers.CreateDirWithErrorCheck(t, instanceIDDir)
	defer helpers.RemoveFileWithErrorCheck(t, instanceIDDir)

	cacheFile := helpers.CreateFileWithErrorCheck(t, instanceIDDir, "cache.json")
	nginxConf := helpers.CreateFileWithErrorCheck(t, tempDir, "nginx.conf")
	metricsConf := helpers.CreateFileWithErrorCheck(t, tempDir, "metrics.conf")
	testConf := helpers.CreateFileWithErrorCheck(t, tempDir, "test.conf")

	cachePath := cacheFile.Name()
	filesURL := fmt.Sprintf("/instance/%s/files/", instanceID)

	fileContent := []byte("location /test {\n    return 200 \"Test location\\n\";\n}")
	err := writeFile(fileContent, testConf.Name())
	require.NoError(t, err)
	assert.FileExists(t, testConf.Name())

	files, err := helpers.GetFiles(nginxConf, testConf, metricsConf)
	require.NoError(t, err)

	cacheContent, getCacheErr := helpers.GetFileCache(nginxConf, testConf, metricsConf)
	require.NoError(t, getCacheErr)

	fakeConfigClient := &clientfakes.FakeConfigClientInterface{}
	fakeConfigClient.GetFilesMetadataReturns(files, nil)
	resp := []byte("location /test {\n    return 200 \"Test changed\\n\";\n}")
	fakeConfigClient.GetFileReturns(helpers.GetFileDownloadResponse(testConf.Name(), instanceID.String(), resp), nil)

	agentconfig := config2.Config{
		AllowedDirectories: allowedDirs,
	}

	fileCache := NewFileCache(instanceID.String())
	fileCache.SetCachePath(cachePath)
	err = fileCache.UpdateFileCache(cacheContent)
	require.NoError(t, err)

	configWriter, err := NewConfigWriter(&agentconfig, fileCache)
	require.NoError(t, err)
	configWriter.SetConfigClient(fakeConfigClient)

	fileTime1, err := helpers.CreateProtoTime("2024-01-08T14:22:21Z")
	require.NoError(t, err)

	fileTime2, err := helpers.CreateProtoTime("2024-01-08T13:22:23Z")
	require.NoError(t, err)

	skippedFiles := CacheContent{
		metricsConf.Name(): {
			LastModified: fileTime1,
			Path:         "/tmp/nginx/locations/metrics.conf",
			Version:      "ibZkRVjemE5dl.tv88ttUJaXx6UJJMTu",
		},
		nginxConf.Name(): {
			LastModified: fileTime2,
			Path:         "/tmp/nginx/test.conf",
			Version:      "Rh3phZuCRwNGANTkdst51he_0WKWy.tZ",
		},
	}

	err = configWriter.Rollback(ctx, skippedFiles, filesURL, tenantID.String(), instanceID.String())
	require.NoError(t, err)

	data, err := os.ReadFile(testConf.Name())
	require.NoError(t, err)
	assert.NotEqual(t, data, fileContent)

	helpers.RemoveFileWithErrorCheck(t, cacheFile.Name())
	helpers.RemoveFileWithErrorCheck(t, metricsConf.Name())
	helpers.RemoveFileWithErrorCheck(t, nginxConf.Name())
	helpers.RemoveFileWithErrorCheck(t, testConf.Name())
	assert.NoFileExists(t, cachePath)
}

func TestComplete(t *testing.T) {
	tempDir := t.TempDir()
	instanceID, err := uuid.Parse("aecea348-62c1-4e3d-b848-6d6cdeb1cb9c")
	require.NoError(t, err)

	instanceIDDir := fmt.Sprintf("%s/%s/", tempDir, instanceID.String())

	helpers.CreateDirWithErrorCheck(t, instanceIDDir)
	defer helpers.RemoveFileWithErrorCheck(t, instanceIDDir)

	cacheFile := helpers.CreateFileWithErrorCheck(t, instanceIDDir, "cache.json")
	cachePath := cacheFile.Name()

	allowedDirs := []string{"./"}

	fakeConfigClient := &clientfakes.FakeConfigClientInterface{}

	fileCache := NewFileCache(instanceID.String())
	agentconfig := config2.Config{
		AllowedDirectories: allowedDirs,
	}
	fileCache.SetCachePath(cachePath)

	configWriter, err := NewConfigWriter(&agentconfig, fileCache)
	require.NoError(t, err)
	configWriter.configClient = fakeConfigClient

	testConf := helpers.CreateFileWithErrorCheck(t, tempDir, "test.conf")
	nginxConf := helpers.CreateFileWithErrorCheck(t, tempDir, "nginx.conf")
	metricsConf := helpers.CreateFileWithErrorCheck(t, tempDir, "metrics.conf")

	cacheData, err := helpers.GetFileCache(testConf, nginxConf, metricsConf)
	require.NoError(t, err)

	configWriter.currentFileCache, err = helpers.GetFileCache(testConf, metricsConf)
	require.NoError(t, err)

	helpers.CreateCacheFiles(t, cachePath, cacheData)

	err = configWriter.Complete()
	require.NoError(t, err)

	data, err := configWriter.fileCache.ReadFileCache()
	require.NoError(t, err)
	assert.NotEqual(t, cacheData, data)

	helpers.RemoveFileWithErrorCheck(t, cacheFile.Name())
	helpers.RemoveFileWithErrorCheck(t, metricsConf.Name())
	helpers.RemoveFileWithErrorCheck(t, nginxConf.Name())
	helpers.RemoveFileWithErrorCheck(t, testConf.Name())

	assert.NoFileExists(t, cachePath)
}

func TestWriteFile(t *testing.T) {
	tempDir := t.TempDir()
	filePath := fmt.Sprintf("%s/nginx.conf", tempDir)
	fileContent := []byte("location /test {\n    return 200 \"Test location\\n\";\n}")

	err := writeFile(fileContent, filePath)
	require.NoError(t, err)
	assert.FileExists(t, filePath)

	data, err := os.ReadFile(filePath)
	require.NoError(t, err)
	assert.Equal(t, fileContent, data)

	helpers.RemoveFileWithErrorCheck(t, filePath)
	assert.NoFileExists(t, filePath)
}

func TestIsFilePathValid(t *testing.T) {
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
		{
			name:           "not allowed dir",
			path:           "./test/test.conf",
			expectedResult: false,
		},
	}

	fakeConfigClient := &clientfakes.FakeConfigClientInterface{}
	allowedDirs := []string{"/tmp/"}
	cachePath := fmt.Sprintf(cacheLocation, "aecea348-62c1-4e3d-b848-6d6cdeb1cb9c")

	fileCache := NewFileCache("aecea348-62c1-4e3d-b848-6d6cdeb1cb9c")
	fileCache.SetCachePath(cachePath)
	agentConfig := config2.Config{
		AllowedDirectories: allowedDirs,
	}

	configWriter, err := NewConfigWriter(&agentConfig, fileCache)
	require.NoError(t, err)
	configWriter.configClient = fakeConfigClient

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			valid := configWriter.isFilePathValid(test.path)
			assert.Equal(t, test.expectedResult, valid)
		})
	}
}

func TestDoesFileRequireUpdate(t *testing.T) {
	fileTime1, err := helpers.CreateProtoTime("2024-01-08T14:22:21Z")
	require.NoError(t, err)

	fileTime2, err := helpers.CreateProtoTime("2024-01-08T13:22:23Z")
	require.NoError(t, err)

	previousFileCache := CacheContent{
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

	updateTimeFile1, err := helpers.CreateProtoTime("2024-01-08T14:22:23Z")
	require.NoError(t, err)

	tests := []struct {
		name            string
		lastConfigApply CacheContent
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
