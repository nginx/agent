// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.
package config

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path"
	"testing"
	"time"

	helpers "github.com/nginx/agent/v3/test"

	"google.golang.org/protobuf/proto"

	config2 "github.com/nginx/agent/v3/internal/config"

	"github.com/google/uuid"
	"github.com/nginx/agent/v3/api/grpc/instances"
	"github.com/nginx/agent/v3/internal/client/clientfakes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/timestamppb"
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

			configWriter := NewConfigWriter(&agentconfig, fileCache)

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

	err = os.Remove(cachePath)
	require.NoError(t, err)
	assert.NoFileExists(t, cachePath)
}

func TestComplete(t *testing.T) {
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

	allowedDirs := []string{"./"}

	fakeConfigClient := &clientfakes.FakeConfigClientInterface{}

	fileCache := NewFileCache(instanceID.String())
	agentconfig := config2.Config{
		AllowedDirectories: allowedDirs,
	}

	configWriter := NewConfigWriter(&agentconfig, fileCache)
	configWriter.configClient = fakeConfigClient

	testConf, err := os.CreateTemp(".", "test.conf")
	require.NoError(t, err)
	defer os.Remove(testConf.Name())

	nginxConf, err := os.CreateTemp("./", "nginx.conf")
	require.NoError(t, err)
	defer os.Remove(nginxConf.Name())

	metricsConf, err := os.CreateTemp("./", "metrics.conf")
	require.NoError(t, err)
	defer os.Remove(metricsConf.Name())

	fileTime1, err := createProtoTime("2024-01-08T13:22:23Z")
	require.NoError(t, err)

	fileTime2, err := createProtoTime("2024-01-08T13:22:25Z")
	require.NoError(t, err)

	fileTime3, err := createProtoTime("2024-01-08T13:22:21Z")
	require.NoError(t, err)

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

	configWriter.currentFileCache = CacheContent{
		metricsConf.Name(): {
			LastModified: fileTime1,
			Path:         metricsConf.Name(),
			Version:      "ibZkRVjemE5dl.tv88ttUJaXx6UJJMTu",
		},
		testConf.Name(): {
			LastModified: fileTime2,
			Path:         testConf.Name(),
			Version:      "Rh3phZuCRwNGANTkdst51he_0WKWy.tZ",
		},
	}
	configWriter.fileCache.SetCachePath(cachePath)

	err = configWriter.Complete()
	require.NoError(t, err)

	data, err := configWriter.fileCache.ReadFileCache()
	require.NoError(t, err)
	assert.NotEqual(t, cacheData, data)

	err = os.Remove(cachePath)
	require.NoError(t, err)
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

	err = os.Remove(filePath)
	require.NoError(t, err)
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

	configWriter := NewConfigWriter(&agentConfig, fileCache)
	configWriter.configClient = fakeConfigClient

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			valid := configWriter.isFilePathValid(test.path)
			assert.Equal(t, test.expectedResult, valid)
		})
	}
}

func TestDoesFileRequireUpdate(t *testing.T) {
	fileTime1, err := createProtoTime("2024-01-08T14:22:21Z")
	require.NoError(t, err)

	fileTime2, err := createProtoTime("2024-01-08T13:22:23Z")
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

	updateTimeFile1, err := createProtoTime("2024-01-08T14:22:23Z")
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

func createProtoTime(timeString string) (*timestamppb.Timestamp, error) {
	newTime, err := time.Parse(time.RFC3339, timeString)
	protoTime := timestamppb.New(newTime)

	return protoTime, err
}

func createCacheFile(cachePath string, cacheData CacheContent) error {
	cache, err := json.MarshalIndent(cacheData, "", "  ")
	if err != nil {
		return fmt.Errorf("error marshaling cache, error: %w", err)
	}

	err = os.MkdirAll(path.Dir(cachePath), 0o750)
	if err != nil {
		return fmt.Errorf("error creating cache directory, error: %w", err)
	}

	err = os.WriteFile(cachePath, cache, 0o600)
	if err != nil {
		return fmt.Errorf("error writing to file, error: %w", err)
	}

	return err
}

func checkProtoEquality(t *testing.T, expected, actual proto.Message, shouldBeEqual bool) {
	t.Helper()
	res := proto.Equal(expected, actual)
	assert.Equal(t, shouldBeEqual, res, "Expected %v, \nActual   %v", expected, actual)
}
