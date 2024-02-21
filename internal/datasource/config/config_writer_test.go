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
	"testing"

	helpers "github.com/nginx/agent/v3/test"

	"google.golang.org/protobuf/proto"

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
	testConf := helpers.CreateFileWithErrorCheck(t, tempDir, "test.conf")

	testConfPath := testConf.Name()
	fileContent := []byte("location /test {\n    return 200 \"Test location\\n\";\n}")

	tenantID, instanceID, err := helpers.CreateTestIDs()
	require.NoError(t, err)

	instanceIDDir := path.Join(tempDir, instanceID.String())
	helpers.CreateDirWithErrorCheck(t, instanceIDDir)
	defer helpers.RemoveFileWithErrorCheck(t, instanceIDDir)

	cacheFile := helpers.CreateFileWithErrorCheck(t, instanceIDDir, "cache.json")
	cachePath := cacheFile.Name()

	nginxConf := helpers.CreateFileWithErrorCheck(t, tempDir, "nginx.conf")
	metricsConf := helpers.CreateFileWithErrorCheck(t, tempDir, "metrics.conf")

	filesURL := fmt.Sprintf("/instance/%s/files/", instanceID)

	files, err := helpers.GetFiles(nginxConf, testConf, metricsConf)
	require.NoError(t, err)

	tests := []struct {
		name           string
		metaDataReturn *instances.Files
		getFileReturn  *instances.FileDownloadResponse
		shouldBeEqual  bool
	}{
		{
			name:           "file needs updating",
			metaDataReturn: files,
			getFileReturn:  helpers.GetFileDownloadResponse(testConf.Name(), instanceID.String(), fileContent),
			shouldBeEqual:  false,
		},
		{
			name:           "file doesn't need updating",
			metaDataReturn: files,
			getFileReturn:  helpers.GetFileDownloadResponse(testConf.Name(), instanceID.String(), fileContent),
			shouldBeEqual:  true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			cacheContent, getCacheErr := helpers.GetFileCache(nginxConf, testConf, metricsConf)
			require.NoError(t, getCacheErr)
			fakeConfigClient := &clientfakes.FakeConfigClientInterface{}
			fakeConfigClient.GetFilesMetadataReturns(test.metaDataReturn, nil)
			fakeConfigClient.GetFileReturns(test.getFileReturn, nil)
			allowedDirs := []string{"./", "/var/"}

			fileCache := NewFileCache(instanceID.String())
			fileCache.SetCachePath(cachePath)
			err = fileCache.UpdateFileCache(cacheContent)
			require.NoError(t, err)
			agentconfig := config2.Config{
				AllowedDirectories: allowedDirs,
			}

			if !test.shouldBeEqual {
				modified, protoErr := helpers.CreateProtoTime("2024-01-09T13:22:26Z")
				require.NoError(t, protoErr)
				cacheContent[nginxConf.Name()].LastModified = modified
			}

			configWriter, err := NewConfigWriter(&agentconfig, fileCache)
			require.NoError(t, err)

			configWriter.SetConfigClient(fakeConfigClient)
			err = writeFile(fileContent, testConfPath)
			require.NoError(t, err)
			assert.FileExists(t, testConfPath)

			_, err = configWriter.Write(ctx, filesURL, tenantID.String(), instanceID.String())
			require.NoError(t, err)

			// Will expand on this test in future PR to add more test scenarios (every file is updated etc)
			checkProtoEquality(t, cacheContent[nginxConf.Name()], configWriter.currentFileCache[nginxConf.Name()],
				test.shouldBeEqual)

			testData, readErr := os.ReadFile(test.getFileReturn.GetFilePath())
			require.NoError(t, readErr)

			slog.Warn("", "file Content", fileContent)
			slog.Warn("", "file Content", testData)
			assert.Equal(t, fileContent, testData)

			helpers.RemoveFileWithErrorCheck(t, test.getFileReturn.GetFilePath())
			assert.NoFileExists(t, test.getFileReturn.GetFilePath())
		})
	}

	helpers.RemoveFileWithErrorCheck(t, cacheFile.Name())
	helpers.RemoveFileWithErrorCheck(t, metricsConf.Name())
	helpers.RemoveFileWithErrorCheck(t, nginxConf.Name())
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
	filePath := "/tmp/test.conf"
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

func checkProtoEquality(t *testing.T, expected, actual proto.Message, shouldBeEqual bool) {
	t.Helper()
	res := proto.Equal(expected, actual)
	assert.Equal(t, shouldBeEqual, res, "Expected %v, \nActual   %v", expected, actual)
}
