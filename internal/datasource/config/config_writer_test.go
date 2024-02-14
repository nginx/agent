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

	"github.com/google/uuid"
	"github.com/nginx/agent/v3/api/grpc/instances"
	"github.com/nginx/agent/v3/internal/client/clientfakes"
	"github.com/nginx/agent/v3/internal/service/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWriteConfig(t *testing.T) {
	ctx := context.TODO()
	tempDir := t.TempDir()
	testConf := helpers.CreateFileWithErrorCheck(t, tempDir, "test.conf")

	testConfPath := testConf.Name()
	fileContent := []byte("location /test {\n    return 200 \"Test location\\n\";\n}")
	tenantID, instanceID, err := createTestIDs()
	require.NoError(t, err)

	instanceIDDir := path.Join(tempDir, instanceID.String())
	helpers.CreateDirWithErrorCheck(t, instanceIDDir)

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
			previousFileCache, err := helpers.GetFileCache(nginxConf, testConf, metricsConf)
			require.NoError(t, err)

			fakeConfigClient := &clientfakes.FakeConfigClientInterface{}
			fakeConfigClient.GetFilesMetadataReturns(test.metaDataReturn, nil)
			fakeConfigClient.GetFileReturns(test.getFileReturn, nil)
			allowedDirs := []string{tempDir}

			configWriter := NewConfigWriter(fakeConfigClient, allowedDirs, instanceID.String())
			assert.NotNil(t, configWriter)
			configWriter.cachePath = cachePath
			configWriter.previousFileCache = previousFileCache

			err = writeFile(fileContent, testConfPath)
			require.NoError(t, err)
			assert.FileExists(t, testConfPath)

			if !test.shouldBeEqual {
				modified, protoErr := helpers.CreateProtoTime("2024-01-09T13:22:21Z")
				require.NoError(t, protoErr)
				previousFileCache[nginxConf.Name()].LastModified = modified
			}

			err = configWriter.Write(ctx, filesURL, tenantID)
			require.NoError(t, err)

			defaults, err := helpers.GetFileCache(nginxConf, testConf, metricsConf)
			require.NoError(t, err)

			equalityCheck := reflect.DeepEqual(defaults, configWriter.currentFileCache)
			assert.Equal(t, test.shouldBeEqual, equalityCheck)

			testData, readErr := os.ReadFile(test.getFileReturn.GetFilePath())
			require.NoError(t, readErr)
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

	cacheFile := helpers.CreateFileWithErrorCheck(t, instanceIDDir, "cache.json")
	cachePath := cacheFile.Name()

	allowedDirs := []string{"./"}

	fakeConfigClient := &clientfakes.FakeConfigClientInterface{}
	configWriter := NewConfigWriter(fakeConfigClient, allowedDirs, instanceID.String())
	configWriter.cachePath = cachePath

	testConf := helpers.CreateFileWithErrorCheck(t, tempDir, "test.conf")
	nginxConf := helpers.CreateFileWithErrorCheck(t, tempDir, "nginx.conf")
	metricsConf := helpers.CreateFileWithErrorCheck(t, tempDir, "metrics.conf")

	cacheData, err := helpers.GetFileCache(testConf, nginxConf, metricsConf)
	require.NoError(t, err)

	configWriter.currentFileCache, err = helpers.GetFileCache(testConf, metricsConf)
	require.NoError(t, err)

	configWriter.cachePath = cachePath

	err = configWriter.Complete()
	require.NoError(t, err)

	data, err := readInstanceCache(cachePath)
	require.NoError(t, err)
	assert.NotEqual(t, cacheData, data)

	helpers.RemoveFileWithErrorCheck(t, cacheFile.Name())
	helpers.RemoveFileWithErrorCheck(t, metricsConf.Name())
	helpers.RemoveFileWithErrorCheck(t, nginxConf.Name())
	helpers.RemoveFileWithErrorCheck(t, testConf.Name())

	assert.NoFileExists(t, cachePath)
}

func TestDataPlaneConfig(t *testing.T) {
	fakeConfigClient := &clientfakes.FakeConfigClientInterface{}
	tempDir := t.TempDir()
	allowedDirs := []string{"./"}
	cachePath := fmt.Sprintf("%s/%s", tempDir, "aecea348-62c1-4e3d-b848-6d6cdeb1cb9c")
	configWriter := NewConfigWriter(fakeConfigClient, allowedDirs, "aecea348-62c1-4e3d-b848-6d6cdeb1cb9c")
	assert.NotNil(t, configWriter)

	configWriter.cachePath = cachePath
	nginxConfig := config.NewNginx()

	configWriter.SetDataPlaneConfig(nginxConfig)

	assert.Equal(t, configWriter.dataPlaneConfig, nginxConfig)
	require.NoError(t, os.Remove(tempDir))
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

func TestReadCache(t *testing.T) {
	instanceID, err := uuid.Parse("aecea348-62c1-4e3d-b848-6d6cdeb1cb9c")
	require.NoError(t, err)

	tempDir := t.TempDir()
	instanceIDDir := fmt.Sprintf("%s/%s/", tempDir, instanceID.String())
	helpers.CreateDirWithErrorCheck(t, instanceIDDir)
	defer helpers.RemoveFileWithErrorCheck(t, instanceIDDir)

	cacheFile := helpers.CreateFileWithErrorCheck(t, instanceIDDir, "cache.json")
	require.NoError(t, err)

	cachePath := cacheFile.Name()

	testConf := helpers.CreateFileWithErrorCheck(t, tempDir, "test.conf")
	nginxConf := helpers.CreateFileWithErrorCheck(t, tempDir, "nginx.conf")
	metricsConf := helpers.CreateFileWithErrorCheck(t, tempDir, "metrics.conf")

	cacheData, err := helpers.GetFileCache(testConf, nginxConf, metricsConf)
	require.NoError(t, err)

	helpers.CreateCacheFiles(t, cachePath, cacheData)

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
			previousFileCache, readErr := readInstanceCache(test.path)

			if test.shouldHaveError {
				require.Error(t, readErr)
				assert.NotEqual(t, cacheData, previousFileCache)
			} else {
				require.NoError(t, readErr)
				assert.Equal(t, cacheData, previousFileCache)
			}
		})
	}

	helpers.RemoveFileWithErrorCheck(t, cachePath)
	helpers.RemoveFileWithErrorCheck(t, testConf.Name())
	helpers.RemoveFileWithErrorCheck(t, nginxConf.Name())
	helpers.RemoveFileWithErrorCheck(t, metricsConf.Name())
	assert.NoFileExists(t, cachePath)
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
	configWriter := NewConfigWriter(fakeConfigClient, allowedDirs, "aecea348-62c1-4e3d-b848-6d6cdeb1cb9c")
	assert.NotNil(t, configWriter)
	configWriter.cachePath = cachePath

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

	updateTimeFile1, err := helpers.CreateProtoTime("2024-01-08T14:22:23Z")
	require.NoError(t, err)

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

	configWriter := NewConfigWriter(nil, nil, "")

	for _, test := range tests {
		configWriter.previousFileCache = test.lastConfigApply
		t.Run(test.name, func(t *testing.T) {
			valid := configWriter.doesFileRequireUpdate(test.fileData)
			assert.Equal(t, test.expectedResult, valid)
		})
	}
}

func createTestIDs() (uuid.UUID, uuid.UUID, error) {
	tenantID, err := uuid.Parse("7332d596-d2e6-4d1e-9e75-70f91ef9bd0e")
	if err != nil {
		fmt.Printf("Error creating tenantID: %v", err)
	}

	instanceID, err := uuid.Parse("aecea348-62c1-4e3d-b848-6d6cdeb1cb9c")
	if err != nil {
		fmt.Printf("Error creating instanceID: %v", err)
	}

	return tenantID, instanceID, err
}
