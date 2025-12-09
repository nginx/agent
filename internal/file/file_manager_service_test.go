// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package file

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/nginx/agent/v3/internal/config"
	"github.com/nginx/agent/v3/internal/model"

	"github.com/nginx/agent/v3/pkg/files"
	"google.golang.org/protobuf/types/known/timestamppb"

	mpi "github.com/nginx/agent/v3/api/grpc/mpi/v1"
	"github.com/nginx/agent/v3/api/grpc/mpi/v1/v1fakes"
	"github.com/nginx/agent/v3/test/helpers"
	"github.com/nginx/agent/v3/test/protos"
	"github.com/nginx/agent/v3/test/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFileManagerService_ConfigApply_Add(t *testing.T) {
	ctx := context.Background()
	tempDir := t.TempDir()

	filePath := filepath.Join(tempDir, "nginx.conf")

	fileContent := []byte("location /test {\n    return 200 \"Test location\\n\";\n}")
	fileHash := files.GenerateHash(fileContent)
	defer helpers.RemoveFileWithErrorCheck(t, filePath)

	overview := protos.FileOverview(filePath, fileHash)

	manifestDirPath := tempDir
	manifestFilePath := filepath.Join(manifestDirPath, "manifest.json")
	helpers.CreateFileWithErrorCheck(t, manifestDirPath, "manifest.json")

	fakeFileServiceClient := &v1fakes.FakeFileServiceClient{}
	fakeFileServiceClient.GetOverviewReturns(&mpi.GetOverviewResponse{
		Overview: overview,
	}, nil)
	fakeFileServiceClient.GetFileReturns(&mpi.GetFileResponse{
		Contents: &mpi.FileContents{
			Contents: fileContent,
		},
	}, nil)
	agentConfig := types.AgentConfig()
	agentConfig.AllowedDirectories = []string{tempDir}

	fileManagerService := NewFileManagerService(fakeFileServiceClient, agentConfig, &sync.RWMutex{})
	fileManagerService.agentConfig.LibDir = manifestDirPath
	fileManagerService.manifestFilePath = manifestFilePath

	request := protos.CreateConfigApplyRequest(overview)
	writeStatus, err := fileManagerService.ConfigApply(ctx, request)
	require.NoError(t, err)
	assert.Equal(t, model.OK, writeStatus)
	data, readErr := os.ReadFile(filePath)
	require.NoError(t, readErr)
	assert.Equal(t, fileContent, data)
	assert.Equal(t, fileManagerService.fileActions[filePath].File, overview.GetFiles()[0])
	assert.Equal(t, 1, fakeFileServiceClient.GetFileCallCount())
	assert.True(t, fileManagerService.rollbackManifest)
}

func TestFileManagerService_ConfigApply_Add_LargeFile(t *testing.T) {
	ctx := context.Background()
	tempDir := t.TempDir()

	filePath := filepath.Join(tempDir, "nginx.conf")
	fileContent := []byte("location /test {\n    return 200 \"Test location\\n\";\n}")
	fileHash := files.GenerateHash(fileContent)
	defer helpers.RemoveFileWithErrorCheck(t, filePath)

	overview := protos.FileOverviewLargeFile(filePath, fileHash)

	fakeFileServiceClient := &v1fakes.FakeFileServiceClient{}
	fakeFileServiceClient.GetOverviewReturns(&mpi.GetOverviewResponse{
		Overview: overview,
	}, nil)

	fakeServerStreamingClient := &FakeServerStreamingClient{
		chunks:         make(map[uint32][]byte),
		currentChunkID: 0,
		fileName:       filePath,
	}

	for i := range fileContent {
		fakeServerStreamingClient.chunks[uint32(i)] = []byte{fileContent[i]}
	}

	manifestDirPath := tempDir
	manifestFilePath := filepath.Join(manifestDirPath, "manifest.json")

	fakeFileServiceClient.GetFileStreamReturns(fakeServerStreamingClient, nil)
	agentConfig := types.AgentConfig()
	agentConfig.AllowedDirectories = []string{tempDir}
	fileManagerService := NewFileManagerService(fakeFileServiceClient, agentConfig, &sync.RWMutex{})
	fileManagerService.agentConfig.LibDir = manifestDirPath
	fileManagerService.manifestFilePath = manifestFilePath

	request := protos.CreateConfigApplyRequest(overview)
	writeStatus, err := fileManagerService.ConfigApply(ctx, request)
	require.NoError(t, err)
	assert.Equal(t, model.OK, writeStatus)
	data, readErr := os.ReadFile(filePath)
	require.NoError(t, readErr)
	assert.Equal(t, fileContent, data)
	assert.Equal(t, fileManagerService.fileActions[filePath].File, overview.GetFiles()[0])
	assert.Equal(t, 0, fakeFileServiceClient.GetFileCallCount())
	assert.Equal(t, 53, int(fakeServerStreamingClient.currentChunkID))
	assert.True(t, fileManagerService.rollbackManifest)
}

func TestFileManagerService_ConfigApply_Update(t *testing.T) {
	ctx := context.Background()
	tempDir := t.TempDir()

	fileContent := []byte("location /test {\n    return 200 \"Test location\\n\";\n}")
	previousFileContent := []byte("some test data")
	previousFileHash := files.GenerateHash(previousFileContent)
	fileHash := files.GenerateHash(fileContent)
	tempFile := helpers.CreateFileWithErrorCheck(t, tempDir, "nginx.conf")
	_, writeErr := tempFile.Write(previousFileContent)
	require.NoError(t, writeErr)
	defer helpers.RemoveFileWithErrorCheck(t, tempFile.Name())

	filesOnDisk := map[string]*mpi.File{
		tempFile.Name(): {
			FileMeta: &mpi.FileMeta{
				Name:         tempFile.Name(),
				Hash:         previousFileHash,
				ModifiedTime: timestamppb.Now(),
				Permissions:  "0640",
				Size:         0,
			},
		},
	}

	manifestDirPath := tempDir
	manifestFilePath := manifestDirPath + "/manifest.json"
	helpers.CreateFileWithErrorCheck(t, manifestDirPath, "manifest.json")

	overview := protos.FileOverview(tempFile.Name(), fileHash)

	fakeFileServiceClient := &v1fakes.FakeFileServiceClient{}
	fakeFileServiceClient.GetOverviewReturns(&mpi.GetOverviewResponse{
		Overview: overview,
	}, nil)
	fakeFileServiceClient.GetFileReturns(&mpi.GetFileResponse{
		Contents: &mpi.FileContents{
			Contents: fileContent,
		},
	}, nil)
	agentConfig := types.AgentConfig()
	agentConfig.AllowedDirectories = []string{tempDir}

	fileManagerService := NewFileManagerService(fakeFileServiceClient, agentConfig, &sync.RWMutex{})
	fileManagerService.agentConfig.LibDir = manifestDirPath
	fileManagerService.manifestFilePath = manifestFilePath
	err := fileManagerService.UpdateCurrentFilesOnDisk(ctx, filesOnDisk, false)
	require.NoError(t, err)

	request := protos.CreateConfigApplyRequest(overview)
	writeStatus, err := fileManagerService.ConfigApply(ctx, request)
	require.NoError(t, err)
	assert.Equal(t, model.OK, writeStatus)
	data, readErr := os.ReadFile(tempFile.Name())
	require.NoError(t, readErr)
	assert.Equal(t, fileContent, data)

	content, err := os.ReadFile(tempBackupFilePath(tempFile.Name()))
	require.NoError(t, err)
	assert.Equal(t, previousFileContent, content)

	assert.Equal(t, fileManagerService.fileActions[tempFile.Name()].File, overview.GetFiles()[0])
	assert.True(t, fileManagerService.rollbackManifest)
}

func TestFileManagerService_ConfigApply_Delete(t *testing.T) {
	ctx := context.Background()
	tempDir := t.TempDir()

	fileContent := []byte("location /test {\n return 200 \"Test location\\n\";\n}")
	tempFile := helpers.CreateFileWithErrorCheck(t, tempDir, "nginx.conf")
	_, writeErr := tempFile.Write(fileContent)
	require.NoError(t, writeErr)

	tempFile2 := helpers.CreateFileWithErrorCheck(t, tempDir, "test.conf")
	overview := protos.FileOverview(tempFile2.Name(), files.GenerateHash(fileContent))

	filesOnDisk := map[string]*mpi.File{
		tempFile.Name(): {
			FileMeta: &mpi.FileMeta{
				Name:         tempFile.Name(),
				Hash:         files.GenerateHash(fileContent),
				ModifiedTime: timestamppb.Now(),
				Permissions:  "0640",
				Size:         0,
			},
		},
	}

	manifestDirPath := tempDir
	manifestFilePath := manifestDirPath + "/manifest.json"
	helpers.CreateFileWithErrorCheck(t, manifestDirPath, "manifest.json")

	fakeFileServiceClient := &v1fakes.FakeFileServiceClient{}
	agentConfig := types.AgentConfig()
	agentConfig.AllowedDirectories = []string{tempDir}

	fileManagerService := NewFileManagerService(fakeFileServiceClient, agentConfig, &sync.RWMutex{})
	fileManagerService.agentConfig.LibDir = manifestDirPath
	fileManagerService.manifestFilePath = manifestFilePath
	err := fileManagerService.UpdateCurrentFilesOnDisk(ctx, filesOnDisk, false)
	require.NoError(t, err)

	request := protos.CreateConfigApplyRequest(overview)

	fakeFileServiceClient.GetOverviewReturns(&mpi.GetOverviewResponse{
		Overview: overview,
	}, nil)
	fakeFileServiceClient.GetFileReturns(&mpi.GetFileResponse{
		Contents: &mpi.FileContents{
			Contents: fileContent,
		},
	}, nil)

	writeStatus, err := fileManagerService.ConfigApply(ctx, request)
	require.NoError(t, err)
	assert.NoFileExists(t, tempFile.Name())

	content, err := os.ReadFile(tempBackupFilePath(tempFile.Name()))
	require.NoError(t, err)
	assert.Equal(t, fileContent, content)

	assert.Equal(t,
		fileManagerService.fileActions[tempFile.Name()].File.GetFileMeta().GetName(),
		filesOnDisk[tempFile.Name()].GetFileMeta().GetName(),
	)
	assert.Equal(t,
		fileManagerService.fileActions[tempFile.Name()].File.GetFileMeta().GetHash(),
		filesOnDisk[tempFile.Name()].GetFileMeta().GetHash(),
	)
	assert.Equal(t,
		fileManagerService.fileActions[tempFile.Name()].File.GetFileMeta().GetSize(),
		filesOnDisk[tempFile.Name()].GetFileMeta().GetSize(),
	)
	assert.Equal(t, model.OK, writeStatus)
	assert.True(t, fileManagerService.rollbackManifest)
}

func TestFileManagerService_ConfigApply_Failed(t *testing.T) {
	ctx := t.Context()
	tempDir := t.TempDir()

	filePath := filepath.Join(tempDir, "nginx.conf")
	fileContent := []byte("# this is going to fail")
	fileHash := files.GenerateHash(fileContent)

	overview := protos.FileOverview(filePath, fileHash)

	manifestDirPath := tempDir
	manifestFilePath := manifestDirPath + "/manifest.json"
	helpers.CreateFileWithErrorCheck(t, manifestDirPath, "manifest.json")

	fakeFileServiceClient := &v1fakes.FakeFileServiceClient{}
	fakeFileServiceClient.GetOverviewReturns(&mpi.GetOverviewResponse{
		Overview: overview,
	}, nil)
	fakeFileServiceClient.GetFileReturns(nil, errors.New("file not found"))

	agentConfig := types.AgentConfig()
	agentConfig.AllowedDirectories = []string{tempDir}

	fileManagerService := NewFileManagerService(fakeFileServiceClient, agentConfig, &sync.RWMutex{})
	fileManagerService.agentConfig.LibDir = manifestDirPath
	fileManagerService.manifestFilePath = manifestFilePath

	request := protos.CreateConfigApplyRequest(overview)
	writeStatus, err := fileManagerService.ConfigApply(ctx, request)

	require.Error(t, err)
	assert.Equal(t, model.RollbackRequired, writeStatus)
	assert.False(t, fileManagerService.rollbackManifest)
}

func TestFileManagerService_ConfigApply_FileWithExecutePermissions(t *testing.T) {
	ctx := context.Background()
	tempDir := t.TempDir()

	filePath := filepath.Join(tempDir, "nginx.conf")

	fileContent := []byte("location /test {\n    return 200 \"Test location\\n\";\n}")
	fileHash := files.GenerateHash(fileContent)
	defer helpers.RemoveFileWithErrorCheck(t, filePath)

	overview := protos.FileOverview(filePath, fileHash)

	overview.GetFiles()[0].GetFileMeta().Permissions = "0755"

	manifestDirPath := tempDir
	manifestFilePath := filepath.Join(manifestDirPath, "manifest.json")
	helpers.CreateFileWithErrorCheck(t, manifestDirPath, "manifest.json")

	fakeFileServiceClient := &v1fakes.FakeFileServiceClient{}
	fakeFileServiceClient.GetOverviewReturns(&mpi.GetOverviewResponse{
		Overview: overview,
	}, nil)
	fakeFileServiceClient.GetFileReturns(&mpi.GetFileResponse{
		Contents: &mpi.FileContents{
			Contents: fileContent,
		},
	}, nil)
	agentConfig := types.AgentConfig()
	agentConfig.AllowedDirectories = []string{tempDir}

	fileManagerService := NewFileManagerService(fakeFileServiceClient, agentConfig, &sync.RWMutex{})
	fileManagerService.agentConfig.LibDir = manifestDirPath
	fileManagerService.manifestFilePath = manifestFilePath

	request := protos.CreateConfigApplyRequest(overview)
	writeStatus, err := fileManagerService.ConfigApply(ctx, request)
	require.NoError(t, err)
	assert.Equal(t, model.OK, writeStatus)
	assert.Equal(t, "0644", fileManagerService.fileActions[filePath].File.GetFileMeta().GetPermissions())
	data, readErr := os.ReadFile(filePath)
	require.NoError(t, readErr)
	assert.Equal(t, fileContent, data)
	assert.Equal(t, fileManagerService.fileActions[filePath].File, overview.GetFiles()[0])
	assert.Equal(t, 1, fakeFileServiceClient.GetFileCallCount())
	assert.True(t, fileManagerService.rollbackManifest)
}

func TestFileManagerService_checkAllowedDirectory(t *testing.T) {
	fakeFileServiceClient := &v1fakes.FakeFileServiceClient{}
	fileManagerService := NewFileManagerService(fakeFileServiceClient, types.AgentConfig(), &sync.RWMutex{})

	allowedFiles := []*mpi.File{
		{
			FileMeta: &mpi.FileMeta{
				Name:         "/tmp/local/etc/nginx/allowedDirPath",
				Hash:         "",
				ModifiedTime: nil,
				Permissions:  "",
				Size:         0,
			},
		},
	}

	notAllowed := []*mpi.File{
		{
			FileMeta: &mpi.FileMeta{
				Name:         "/not/allowed/dir/path",
				Hash:         "",
				ModifiedTime: nil,
				Permissions:  "",
				Size:         0,
			},
		},
	}

	err := fileManagerService.checkAllowedDirectory(allowedFiles)
	require.NoError(t, err)
	err = fileManagerService.checkAllowedDirectory(notAllowed)
	require.Error(t, err)
}

func TestFileManagerService_validateAndUpdateFilePermissions(t *testing.T) {
	ctx := context.Background()
	fileManagerService := NewFileManagerService(nil, types.AgentConfig(), &sync.RWMutex{})

	testFiles := []*mpi.File{
		{
			FileMeta: &mpi.FileMeta{
				Name:        "exec.conf",
				Permissions: "0700",
			},
		},
		{
			FileMeta: &mpi.FileMeta{
				Name:        "normal.conf",
				Permissions: "0620",
			},
		},
	}

	err := fileManagerService.validateAndUpdateFilePermissions(ctx, testFiles)
	require.NoError(t, err)
	assert.Equal(t, "0600", testFiles[0].GetFileMeta().GetPermissions())
	assert.Equal(t, "0620", testFiles[1].GetFileMeta().GetPermissions())
}

func TestFileManagerService_areExecuteFilePermissionsSet(t *testing.T) {
	fileManagerService := NewFileManagerService(nil, types.AgentConfig(), &sync.RWMutex{})

	tests := []struct {
		name        string
		permissions string
		expectBool  bool
	}{
		{
			name:        "Test 1: File with read and write permissions for owner",
			permissions: "0600",
			expectBool:  false,
		},
		{
			name:        "Test 2: File with read/write and execute permissions for owner",
			permissions: "0700",
			expectBool:  true,
		},
		{
			name:        "Test 3: File with read/write and execute permissions for owner and group",
			permissions: "0770",
			expectBool:  true,
		},
		{
			name:        "Test 4: File with read and execute permissions for everyone",
			permissions: "0555",
			expectBool:  true,
		},
		{
			name:        "Test 5: File with malformed permissions",
			permissions: "abcde",
			expectBool:  false,
		},
		{
			name:        "Test 6: File with invalid permissions",
			permissions: "000070",
			expectBool:  false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			file := &mpi.File{
				FileMeta: &mpi.FileMeta{
					Name:        "test.conf",
					Permissions: test.permissions,
				},
			}

			got := fileManagerService.areExecuteFilePermissionsSet(file)
			assert.Equal(t, test.expectBool, got)
		})
	}
}

func TestFileManagerService_removeExecuteFilePermissions(t *testing.T) {
	fileManagerService := NewFileManagerService(nil, types.AgentConfig(), &sync.RWMutex{})

	tests := []struct {
		name              string
		permissions       string
		errorMsg          string
		expectPermissions string
		expectError       bool
	}{
		{
			name:              "Test 1: File with execute permissions for owner and others",
			permissions:       "0703",
			expectError:       false,
			expectPermissions: "0602",
		},
		{
			name:        "Test 2: File with malformed permissions",
			permissions: "abcde",
			expectError: true,
			errorMsg:    "falied to parse file permissions",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			file := &mpi.File{
				FileMeta: &mpi.FileMeta{
					Name:        "test.conf",
					Permissions: test.permissions,
				},
			}

			parseErr := fileManagerService.removeExecuteFilePermissions(t.Context(), file)

			if test.expectError {
				require.Error(t, parseErr)
				assert.Contains(t, parseErr.Error(), test.errorMsg)
			} else {
				require.NoError(t, parseErr)
				assert.Equal(t, test.expectPermissions, file.GetFileMeta().GetPermissions())
			}
		})
	}
}

//nolint:usetesting // need to use MkDirTemp instead of t.tempDir for rollback as t.tempDir does not accept a pattern
func TestFileManagerService_ClearCache(t *testing.T) {
	tempDir := t.TempDir()
	agentConfig := types.AgentConfig()
	tempPath := fmt.Sprintf("%s.agent_%s", tempDir, agentConfig.UUID)
	err := os.Mkdir(tempPath, dirPerm)
	require.NoError(t, err)

	fakeFileServiceClient := &v1fakes.FakeFileServiceClient{}
	fileManagerService := NewFileManagerService(fakeFileServiceClient, agentConfig, &sync.RWMutex{})

	filesCache := map[string]*model.FileCache{
		"file/path/test.conf": {
			File: &mpi.File{
				FileMeta: &mpi.FileMeta{
					Name:         "file/path/test.conf",
					Hash:         "",
					ModifiedTime: nil,
					Permissions:  "",
					Size:         0,
				},
			},
		},
	}

	fileManagerService.fileActions = filesCache
	assert.NotEmpty(t, fileManagerService.fileActions)

	fileManagerService.ClearCache()

	assert.Empty(t, fileManagerService.fileActions)
}

//nolint:usetesting // need to use MkDirTemp instead of t.tempDir for rollback as t.tempDir does not accept a pattern
func TestFileManagerService_Rollback(t *testing.T) {
	ctx := context.Background()
	tempDir := t.TempDir()

	deleteFilePath := filepath.Join(tempDir, "nginx_delete.conf")

	newFileContent := []byte("location /test {\n    return 200 \"This config needs to be rolled back\\n\";\n}")
	oldFileContent := []byte("location /test {\n    return 200 \"This is the saved config\\n\";\n}")
	fileHash := files.GenerateHash(newFileContent)

	addFile := helpers.CreateFileWithErrorCheck(t, tempDir, "nginx_add.conf")
	_, writeErr := addFile.Write(newFileContent)
	require.NoError(t, writeErr)

	updateFile := helpers.CreateFileWithErrorCheck(t, tempDir, "nginx_update.conf")
	_, writeErr = updateFile.Write(newFileContent)
	require.NoError(t, writeErr)

	tempAddFile, createErr := os.Create(tempBackupFilePath(addFile.Name()))
	require.NoError(t, createErr)
	_, writeErr = tempAddFile.Write(oldFileContent)
	require.NoError(t, writeErr)

	tempUpdateFile, createErr := os.Create(tempBackupFilePath(updateFile.Name()))
	require.NoError(t, createErr)
	_, writeErr = tempUpdateFile.Write(oldFileContent)
	require.NoError(t, writeErr)
	t.Log(tempUpdateFile.Name())

	tempDeleteFile, createErr := os.Create(tempBackupFilePath(tempDir + "/nginx_delete.conf"))
	require.NoError(t, createErr)
	_, writeErr = tempDeleteFile.Write(oldFileContent)
	require.NoError(t, writeErr)
	t.Log(tempDeleteFile.Name())

	manifestDirPath := tempDir
	manifestFilePath := manifestDirPath + "/manifest.json"
	helpers.CreateFileWithErrorCheck(t, manifestDirPath, "manifest.json")

	filesCache := map[string]*model.FileCache{
		addFile.Name(): {
			File: &mpi.File{
				FileMeta: &mpi.FileMeta{
					Name:         addFile.Name(),
					Hash:         fileHash,
					ModifiedTime: timestamppb.Now(),
					Permissions:  "0777",
					Size:         0,
				},
				Unmanaged: false,
			},
			Action: model.Add,
		},
		updateFile.Name(): {
			File: &mpi.File{
				FileMeta: &mpi.FileMeta{
					Name:         updateFile.Name(),
					Hash:         fileHash,
					ModifiedTime: timestamppb.Now(),
					Permissions:  "0777",
					Size:         0,
				},
				Unmanaged: false,
			},
			Action: model.Update,
		},
		deleteFilePath: {
			File: &mpi.File{
				FileMeta: &mpi.FileMeta{
					Name:         deleteFilePath,
					Hash:         "",
					ModifiedTime: timestamppb.Now(),
					Permissions:  "0777",
					Size:         0,
				},
				Unmanaged: false,
			},
			Action: model.Delete,
		},
		"unspecified/file/test.conf": {
			File: &mpi.File{
				FileMeta: &mpi.FileMeta{
					Name:         "unspecified/file/test.conf",
					Hash:         "",
					ModifiedTime: timestamppb.Now(),
					Permissions:  "0777",
					Size:         0,
				},
				Unmanaged: false,
			},
		},
	}

	instanceID := protos.NginxOssInstance([]string{}).GetInstanceMeta().GetInstanceId()
	fakeFileServiceClient := &v1fakes.FakeFileServiceClient{}
	fileManagerService := NewFileManagerService(fakeFileServiceClient, types.AgentConfig(), &sync.RWMutex{})
	fileManagerService.fileActions = filesCache
	fileManagerService.agentConfig.LibDir = manifestDirPath
	fileManagerService.manifestFilePath = manifestFilePath

	err := fileManagerService.Rollback(ctx, instanceID)
	require.NoError(t, err)

	assert.NoFileExists(t, addFile.Name())
	assert.FileExists(t, deleteFilePath)
	updateData, readUpdateErr := os.ReadFile(updateFile.Name())
	require.NoError(t, readUpdateErr)
	assert.Equal(t, oldFileContent, updateData)

	deleteData, readDeleteErr := os.ReadFile(deleteFilePath)
	require.NoError(t, readDeleteErr)
	assert.Equal(t, oldFileContent, deleteData)

	defer helpers.RemoveFileWithErrorCheck(t, updateFile.Name())
	defer helpers.RemoveFileWithErrorCheck(t, deleteFilePath)
}

func TestFileManagerService_DetermineFileActions(t *testing.T) {
	ctx := context.Background()
	tempDir := filepath.Clean(os.TempDir())

	deleteTestFile := helpers.CreateFileWithErrorCheck(t, tempDir, "nginx_delete.conf")
	defer helpers.RemoveFileWithErrorCheck(t, deleteTestFile.Name())
	fileContent, readErr := os.ReadFile("../../test/config/nginx/nginx.conf")
	require.NoError(t, readErr)
	err := os.WriteFile(deleteTestFile.Name(), fileContent, 0o600)
	require.NoError(t, err)

	updateTestFile := helpers.CreateFileWithErrorCheck(t, tempDir, "nginx_update.conf")
	defer helpers.RemoveFileWithErrorCheck(t, updateTestFile.Name())
	updatedFileContent := []byte("test update file")
	updateErr := os.WriteFile(updateTestFile.Name(), updatedFileContent, 0o600)
	require.NoError(t, updateErr)

	addTestFileName := tempDir + "nginx_add.conf"

	unmanagedFile := helpers.CreateFileWithErrorCheck(t, tempDir, "nginx_unmanaged.conf")
	defer helpers.RemoveFileWithErrorCheck(t, unmanagedFile.Name())
	unmanagedFileContent := []byte("test unmanaged file")
	unmanagedErr := os.WriteFile(unmanagedFile.Name(), unmanagedFileContent, 0o600)
	require.NoError(t, unmanagedErr)

	addTestFile := helpers.CreateFileWithErrorCheck(t, tempDir, "nginx_add.conf")
	defer helpers.RemoveFileWithErrorCheck(t, addTestFile.Name())
	addFileContent := []byte("test add file")
	addErr := os.WriteFile(addTestFile.Name(), addFileContent, 0o600)
	require.NoError(t, addErr)

	tests := []struct {
		expectedError   error
		modifiedFiles   map[string]*model.FileCache
		currentFiles    map[string]*mpi.File
		expectedCache   map[string]*model.FileCache
		expectedContent map[string][]byte
		name            string
		allowedDirs     []string
	}{
		{
			name:        "Test 1: Add, Update & Delete Files",
			allowedDirs: []string{tempDir},
			modifiedFiles: map[string]*model.FileCache{
				addTestFileName: {
					File: &mpi.File{
						FileMeta:  protos.FileMeta(addTestFileName, files.GenerateHash(fileContent)),
						Unmanaged: false,
					},
				},
				updateTestFile.Name(): {
					File: &mpi.File{
						FileMeta:  protos.FileMeta(updateTestFile.Name(), files.GenerateHash(updatedFileContent)),
						Unmanaged: false,
					},
				},
				unmanagedFile.Name(): {
					File: &mpi.File{
						FileMeta:  protos.FileMeta(unmanagedFile.Name(), files.GenerateHash(unmanagedFileContent)),
						Unmanaged: true,
					},
				},
			},
			currentFiles: map[string]*mpi.File{
				deleteTestFile.Name(): {
					FileMeta: protos.FileMeta(deleteTestFile.Name(), files.GenerateHash(fileContent)),
				},
				updateTestFile.Name(): {
					FileMeta: protos.FileMeta(updateTestFile.Name(), files.GenerateHash(fileContent)),
				},
				unmanagedFile.Name(): {
					FileMeta:  protos.FileMeta(unmanagedFile.Name(), files.GenerateHash(fileContent)),
					Unmanaged: true,
				},
			},
			expectedCache: map[string]*model.FileCache{
				deleteTestFile.Name(): {
					File: &mpi.File{
						FileMeta:  protos.ManifestFileMeta(deleteTestFile.Name(), files.GenerateHash(fileContent)),
						Unmanaged: false,
					},
					Action: model.Delete,
				},
				updateTestFile.Name(): {
					File: &mpi.File{
						FileMeta:  protos.FileMeta(updateTestFile.Name(), files.GenerateHash(updatedFileContent)),
						Unmanaged: false,
					},
					Action: model.Update,
				},
				addTestFileName: {
					File: &mpi.File{
						FileMeta:  protos.FileMeta(addTestFileName, files.GenerateHash(fileContent)),
						Unmanaged: false,
					},
					Action: model.Add,
				},
			},
			expectedContent: map[string][]byte{
				deleteTestFile.Name(): fileContent,
				updateTestFile.Name(): updatedFileContent,
			},
			expectedError: nil,
		},
		{
			name:        "Test 2: Files same as on disk",
			allowedDirs: []string{tempDir},
			modifiedFiles: map[string]*model.FileCache{
				addTestFile.Name(): {
					File: &mpi.File{
						FileMeta: protos.FileMeta(addTestFile.Name(), files.GenerateHash(addFileContent)),
					},
				},
				updateTestFile.Name(): {
					File: &mpi.File{
						FileMeta: protos.FileMeta(updateTestFile.Name(), files.GenerateHash(updatedFileContent)),
					},
				},
				deleteTestFile.Name(): {
					File: &mpi.File{
						FileMeta: protos.FileMeta(deleteTestFile.Name(), files.GenerateHash(fileContent)),
					},
				},
			},
			currentFiles: map[string]*mpi.File{
				deleteTestFile.Name(): {
					FileMeta: protos.FileMeta(deleteTestFile.Name(), files.GenerateHash(fileContent)),
				},
				updateTestFile.Name(): {
					FileMeta: protos.FileMeta(updateTestFile.Name(), files.GenerateHash(updatedFileContent)),
				},
				addTestFile.Name(): {
					FileMeta: protos.FileMeta(addTestFile.Name(), files.GenerateHash(addFileContent)),
				},
			},
			expectedCache:   make(map[string]*model.FileCache),
			expectedContent: make(map[string][]byte),
			expectedError:   nil,
		},
		{
			name:          "Test 3: File being deleted already doesn't exist",
			allowedDirs:   []string{tempDir, "/unknown"},
			modifiedFiles: make(map[string]*model.FileCache),
			currentFiles: map[string]*mpi.File{
				"/unknown/file.conf": {
					FileMeta: protos.FileMeta("/unknown/file.conf", files.GenerateHash(fileContent)),
				},
			},
			expectedCache:   make(map[string]*model.FileCache),
			expectedContent: make(map[string][]byte),
			expectedError:   nil,
		},
		{
			name:        "Test 4: File is actually a directory",
			allowedDirs: []string{tempDir},
			modifiedFiles: map[string]*model.FileCache{
				tempDir: {
					File: &mpi.File{
						FileMeta: protos.FileMeta(tempDir, files.GenerateHash(fileContent)),
					},
				},
			},
			currentFiles:    make(map[string]*mpi.File),
			expectedCache:   map[string]*model.FileCache(nil),
			expectedContent: make(map[string][]byte),
			expectedError: fmt.Errorf(
				"unable to create file %s since a directory with the same name already exists",
				tempDir,
			),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(tt *testing.T) {
			// Delete manifest file if it already exists
			manifestFile := CreateTestManifestFile(t, tempDir, test.currentFiles, true)
			defer manifestFile.Close()
			manifestDirPath := tempDir
			manifestFilePath := manifestFile.Name()

			fakeFileServiceClient := &v1fakes.FakeFileServiceClient{}
			fileManagerService := NewFileManagerService(fakeFileServiceClient, types.AgentConfig(), &sync.RWMutex{})
			fileManagerService.agentConfig.AllowedDirectories = test.allowedDirs
			fileManagerService.agentConfig.LibDir = manifestDirPath
			fileManagerService.manifestFilePath = manifestFilePath

			require.NoError(tt, err)

			diff, fileActionErr := fileManagerService.DetermineFileActions(
				ctx,
				test.currentFiles,
				test.modifiedFiles,
			)

			if test.expectedError != nil {
				require.EqualError(tt, fileActionErr, test.expectedError.Error())
			} else {
				require.NoError(tt, fileActionErr)
			}

			assert.Equal(tt, test.expectedCache, diff)
		})
	}
}

func CreateTestManifestFile(t testing.TB, tempDir string, currentFiles map[string]*mpi.File, refrenced bool) *os.File {
	t.Helper()
	fakeFileServiceClient := &v1fakes.FakeFileServiceClient{}
	fileManagerService := NewFileManagerService(fakeFileServiceClient, types.AgentConfig(), &sync.RWMutex{})
	manifestFiles := fileManagerService.convertToManifestFileMap(currentFiles, refrenced)
	manifestJSON, err := json.MarshalIndent(manifestFiles, "", "  ")
	require.NoError(t, err)
	file, err := os.CreateTemp(tempDir, "manifest.json")
	require.NoError(t, err)

	_, err = file.Write(manifestJSON)
	require.NoError(t, err)

	return file
}

func TestFileManagerService_UpdateManifestFile(t *testing.T) {
	ctx := t.Context()
	fileContent := []byte("location /test {\n    return 200 \"Test location\\n\";\n}")
	fileHash := files.GenerateHash(fileContent)

	tests := []struct {
		currentFiles         map[string]*mpi.File
		currentManifestFiles map[string]*model.ManifestFile
		expectedFiles        map[string]*model.ManifestFile
		name                 string
		referenced           bool
		previousReferenced   bool
	}{
		{
			name: "Test 1: Manifest file empty",
			currentFiles: map[string]*mpi.File{
				"/etc/nginx/nginx.conf": {
					FileMeta: protos.FileMeta("/etc/nginx/nginx.conf", fileHash),
				},
			},
			expectedFiles: map[string]*model.ManifestFile{
				"/etc/nginx/nginx.conf": {
					ManifestFileMeta: &model.ManifestFileMeta{
						Name:       "/etc/nginx/nginx.conf",
						Hash:       fileHash,
						Size:       0,
						Referenced: true,
					},
				},
			},
			currentManifestFiles: make(map[string]*model.ManifestFile),
			referenced:           true,
			previousReferenced:   true,
		},
		{
			name: "Test 2: Manifest file populated - unreferenced",
			currentFiles: map[string]*mpi.File{
				"/etc/nginx/nginx.conf": {
					FileMeta: protos.FileMeta("/etc/nginx/nginx.conf", fileHash),
				},
				"/etc/nginx/unref.conf": {
					FileMeta: protos.FileMeta("/etc/nginx/unref.conf", fileHash),
				},
			},
			expectedFiles: map[string]*model.ManifestFile{
				"/etc/nginx/nginx.conf": {
					ManifestFileMeta: &model.ManifestFileMeta{
						Name:       "/etc/nginx/nginx.conf",
						Hash:       fileHash,
						Size:       0,
						Referenced: false,
					},
				},
				"/etc/nginx/unref.conf": {
					ManifestFileMeta: &model.ManifestFileMeta{
						Name:       "/etc/nginx/unref.conf",
						Hash:       fileHash,
						Size:       0,
						Referenced: false,
					},
				},
			},
			currentManifestFiles: map[string]*model.ManifestFile{
				"/etc/nginx/nginx.conf": {
					ManifestFileMeta: &model.ManifestFileMeta{
						Name:       "/etc/nginx/nginx.conf",
						Hash:       fileHash,
						Size:       0,
						Referenced: true,
					},
				},
			},
			referenced:         false,
			previousReferenced: true,
		},
		{
			name: "Test 3: Manifest file populated - referenced",
			currentFiles: map[string]*mpi.File{
				"/etc/nginx/nginx.conf": {
					FileMeta: protos.FileMeta("/etc/nginx/nginx.conf", fileHash),
				},
				"/etc/nginx/test.conf": {
					FileMeta: protos.FileMeta("/etc/nginx/test.conf", fileHash),
				},
			},
			expectedFiles: map[string]*model.ManifestFile{
				"/etc/nginx/nginx.conf": {
					ManifestFileMeta: &model.ManifestFileMeta{
						Name:       "/etc/nginx/nginx.conf",
						Hash:       fileHash,
						Size:       0,
						Referenced: true,
					},
				},
				"/etc/nginx/test.conf": {
					ManifestFileMeta: &model.ManifestFileMeta{
						Name:       "/etc/nginx/test.conf",
						Hash:       fileHash,
						Size:       0,
						Referenced: true,
					},
				},
				"/etc/nginx/unref.conf": {
					ManifestFileMeta: &model.ManifestFileMeta{
						Name:       "/etc/nginx/unref.conf",
						Hash:       fileHash,
						Size:       0,
						Referenced: false,
					},
				},
			},
			currentManifestFiles: map[string]*model.ManifestFile{
				"/etc/nginx/nginx.conf": {
					ManifestFileMeta: &model.ManifestFileMeta{
						Name:       "/etc/nginx/nginx.conf",
						Hash:       fileHash,
						Size:       0,
						Referenced: false,
					},
				},
				"/etc/nginx/unref.conf": {
					ManifestFileMeta: &model.ManifestFileMeta{
						Name:       "/etc/nginx/unref.conf",
						Hash:       fileHash,
						Size:       0,
						Referenced: false,
					},
				},
			},
			referenced:         true,
			previousReferenced: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(tt *testing.T) {
			manifestDirPath := t.TempDir()
			file := helpers.CreateFileWithErrorCheck(t, manifestDirPath, "manifest.json")

			fakeFileServiceClient := &v1fakes.FakeFileServiceClient{}
			fileManagerService := NewFileManagerService(fakeFileServiceClient, types.AgentConfig(), &sync.RWMutex{})
			fileManagerService.agentConfig.AllowedDirectories = []string{"manifestDirPath"}
			fileManagerService.agentConfig.LibDir = manifestDirPath
			fileManagerService.manifestFilePath = file.Name()

			manifestJSON, err := json.MarshalIndent(test.currentManifestFiles, "", "  ")
			require.NoError(t, err)

			_, err = file.Write(manifestJSON)
			require.NoError(t, err)

			updateErr := fileManagerService.UpdateManifestFile(ctx, test.currentFiles, test.referenced)
			require.NoError(tt, updateErr)

			manifestFiles, _, manifestErr := fileManagerService.manifestFile()
			require.NoError(tt, manifestErr)
			assert.Equal(tt, test.expectedFiles, manifestFiles)
		})
	}
}

func TestFileManagerService_fileActions(t *testing.T) {
	ctx := context.Background()
	tempDir := t.TempDir()

	addFilePath := filepath.Join(tempDir, "nginx_add.conf")
	unspecifiedFilePath := "unspecified/file/test.conf"

	newFileContent := []byte("location /test {\n    return 200 \"This config needs to be rolled back\\n\";\n}")
	oldFileContent := []byte("location /test {\n    return 200 \"This is the saved config\\n\";\n}")
	fileHash := files.GenerateHash(newFileContent)

	deleteFile := helpers.CreateFileWithErrorCheck(t, tempDir, "nginx_delete.conf")
	_, writeErr := deleteFile.Write(oldFileContent)
	require.NoError(t, writeErr)

	updateFile := helpers.CreateFileWithErrorCheck(t, tempDir, "nginx_update.conf")
	_, writeErr = updateFile.Write(oldFileContent)
	require.NoError(t, writeErr)

	filesCache := map[string]*model.FileCache{
		addFilePath: {
			File: &mpi.File{
				FileMeta: &mpi.FileMeta{
					Name:         addFilePath,
					Hash:         fileHash,
					ModifiedTime: timestamppb.Now(),
					Permissions:  "0777",
					Size:         0,
				},
			},
			Action: model.Add,
		},
		updateFile.Name(): {
			File: &mpi.File{
				FileMeta: &mpi.FileMeta{
					Name:         updateFile.Name(),
					Hash:         fileHash,
					ModifiedTime: timestamppb.Now(),
					Permissions:  "0777",
					Size:         0,
				},
			},
			Action: model.Update,
		},
		deleteFile.Name(): {
			File: &mpi.File{
				FileMeta: &mpi.FileMeta{
					Name:         deleteFile.Name(),
					Hash:         "",
					ModifiedTime: timestamppb.Now(),
					Permissions:  "0777",
					Size:         0,
				},
			},
			Action: model.Delete,
		},
		unspecifiedFilePath: {
			File: &mpi.File{
				FileMeta: &mpi.FileMeta{
					Name:         unspecifiedFilePath,
					Hash:         "",
					ModifiedTime: timestamppb.Now(),
					Permissions:  "0777",
					Size:         0,
				},
			},
		},
	}

	fakeFileServiceClient := &v1fakes.FakeFileServiceClient{}
	fakeFileServiceClient.GetFileReturns(&mpi.GetFileResponse{
		Contents: &mpi.FileContents{
			Contents: newFileContent,
		},
	}, nil)
	fileManagerService := NewFileManagerService(fakeFileServiceClient, types.AgentConfig(), &sync.RWMutex{})

	fileManagerService.fileActions = filesCache

	actionErr := fileManagerService.executeFileActions(ctx)
	require.NoError(t, actionErr)

	assert.FileExists(t, addFilePath)
	assert.NoFileExists(t, deleteFile.Name())
	assert.NoFileExists(t, unspecifiedFilePath)
	updateData, readUpdateErr := os.ReadFile(updateFile.Name())
	require.NoError(t, readUpdateErr)
	assert.Equal(t, newFileContent, updateData)

	defer helpers.RemoveFileWithErrorCheck(t, updateFile.Name())
	defer helpers.RemoveFileWithErrorCheck(t, addFilePath)
}

func TestParseX509Certificates(t *testing.T) {
	tests := []struct {
		certName       string
		certContent    string
		name           string
		expectedSerial string
	}{
		{
			name:           "Test 1: generated cert",
			certName:       "public_cert",
			certContent:    "",
			expectedSerial: "123123",
		},
		{
			name:     "Test 2: open ssl cert",
			certName: "open_ssl_cert",
			certContent: `-----BEGIN CERTIFICATE-----
MIIDazCCAlOgAwIBAgIUR+YGgRHhYwotFyBOvSc1KD9d45kwDQYJKoZIhvcNAQEL
BQAwRTELMAkGA1UEBhMCQVUxEzARBgNVBAgMClNvbWUtU3RhdGUxITAfBgNVBAoM
GEludGVybmV0IFdpZGdpdHMgUHR5IEx0ZDAeFw0yNDExMjcxNTM0MDZaFw0yNDEy
MjcxNTM0MDZaMEUxCzAJBgNVBAYTAkFVMRMwEQYDVQQIDApTb21lLVN0YXRlMSEw
HwYDVQQKDBhJbnRlcm5ldCBXaWRnaXRzIFB0eSBMdGQwggEiMA0GCSqGSIb3DQEB
AQUAA4IBDwAwggEKAoIBAQDnDDVGflbZ3dmuQJj+8QuJIQ8lWjVGYhlsFI4AGFTX
9VfYOqJEPyuMRuSj2eN7C/mR4yTJSggnv0kFtjmeGh2keNdmb4R/0CjYWZVl/Na6
cAfldB8v2+sm0LZ/OD9F9CbnYB95takPOZq3AP5kUA+qlFYzroqXsxJKvZF6dUuI
+kTOn5pWD+eFmueFedOz1aucOvblUJLueVZnvAbIrBoyaulw3f2kjk0J1266nFMb
s72AvjyYbOXbyur3BhPThCaOeqMGggDmFslZ4pBgQFWUeFvmqJMFzf1atKTWlbj7
Mj+bNKNs4xvUuNhqd/F99Pz2Fe0afKbTHK83hqgSHKbtAgMBAAGjUzBRMB0GA1Ud
DgQWBBQq0Bzde0bl9CFb81LrvFfdWlY7hzAfBgNVHSMEGDAWgBQq0Bzde0bl9CFb
81LrvFfdWlY7hzAPBgNVHRMBAf8EBTADAQH/MA0GCSqGSIb3DQEBCwUAA4IBAQAo
8GXvwRa0M0D4x4Lrj2K57FxH4ECNBnAqWlh3Ce9LEioL2CYaQQw6I2/FsnTk8TYY
WgGgXMEyA6OeOXvwxWjSllK9+D2ueTMhNRO0tYMUi0kDJqd9EpmnEcSWIL2G2SNo
BWQjqEoEKFjvrgx6h13AtsFlpdURoVtodrtnUrXp1r4wJvljC2qexoNfslhpbqsT
X/vYrzgKRoKSUWUt1ejKTntrVuaJK4NMxANOTTjIXgxyoV3YcgEmL9KzribCqILi
p79Nno9d+kovtX5VKsJ5FCcPw9mEATgZDOQ4nLTk/HHG6bwtpubp6Zb7H1AjzBkz
rQHX6DP4w6IwZY8JB8LS
-----END CERTIFICATE-----`,
			expectedSerial: "410468082718062724391949173062901619571168240537",
		},
	}

	tempDir := os.TempDir()

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var certBytes []byte
			var certPath string

			if test.certContent == "" {
				_, certBytes = helpers.GenerateSelfSignedCert(t)
				certContents := helpers.Cert{
					Name:     test.certName + ".pem",
					Type:     "CERTIFICATE",
					Contents: certBytes,
				}
				certPath = helpers.WriteCertFiles(t, tempDir, certContents)
			} else {
				certPath = fmt.Sprintf("%s%c%s", tempDir, os.PathSeparator, test.certName)
				err := os.WriteFile(certPath, []byte(test.certContent), 0o600)
				require.NoError(t, err)
			}

			certFileMeta, certFileMetaErr := files.FileMetaWithCertificate(certPath)
			require.NoError(t, certFileMetaErr)

			assert.Equal(t, test.expectedSerial, certFileMeta.GetCertificateMeta().GetSerialNumber())
		})
	}
}

func TestFileManagerService_DetermineFileActions_ExternalFile(t *testing.T) {
	ctx := context.Background()
	tempDir := t.TempDir()
	fileName := filepath.Join(tempDir, "external.conf")

	modifiedFiles := map[string]*model.FileCache{
		fileName: {
			File: &mpi.File{
				FileMeta: &mpi.FileMeta{
					Name: fileName,
				},
				ExternalDataSource: &mpi.ExternalDataSource{Location: "http://example.com/file"},
			},
		},
	}

	fakeFileServiceClient := &v1fakes.FakeFileServiceClient{}
	fileManagerService := NewFileManagerService(fakeFileServiceClient, types.AgentConfig(), &sync.RWMutex{})
	fileManagerService.agentConfig.AllowedDirectories = []string{tempDir}

	diff, err := fileManagerService.DetermineFileActions(ctx, make(map[string]*mpi.File), modifiedFiles)
	require.NoError(t, err)

	fc, ok := diff[fileName]
	require.True(t, ok, "expected file to be present in diff")
	assert.Equal(t, model.ExternalFile, fc.Action)
}

//nolint:gocognit,revive,govet // cognitive complexity is 22
func TestFileManagerService_downloadExternalFiles(t *testing.T) {
	type tc struct {
		allowedDomains      []string
		expectContent       []byte
		name                string
		expectHeaderETag    string
		expectHeaderLastMod string
		expectErrContains   string
		handler             http.HandlerFunc
		maxBytes            int
		expectError         bool
		expectTempFile      bool
	}

	tests := []tc{
		{
			name: "Test 1: Success",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("ETag", "test-etag")
				w.Header().Set("Last-Modified", time.RFC1123)
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte("external file content"))
			},
			allowedDomains:      nil,
			maxBytes:            0,
			expectError:         false,
			expectTempFile:      true,
			expectContent:       []byte("external file content"),
			expectHeaderETag:    "test-etag",
			expectHeaderLastMod: time.RFC1123,
		},
		{
			name: "Test 2: NotModified",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusNotModified)
			},
			allowedDomains:      nil,
			maxBytes:            0,
			expectError:         false,
			expectTempFile:      false,
			expectContent:       nil,
			expectHeaderETag:    "",
			expectHeaderLastMod: "",
		},
		{
			name: "Test 3: NotAllowedDomain",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte("external file content"))
			},
			allowedDomains:    []string{"not-the-host"},
			maxBytes:          0,
			expectError:       true,
			expectErrContains: "not in the allowed domains",
			expectTempFile:    false,
		},
		{
			name: "Test 4: NotFound",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusNotFound)
			},
			allowedDomains:    nil,
			maxBytes:          0,
			expectError:       true,
			expectErrContains: "status code 404",
			expectTempFile:    false,
		},
		{
			name: "Test 5: ProxyWithConditionalHeaders",
			handler: func(w http.ResponseWriter, r *http.Request) {
				// verify conditional headers from manifest are added
				if r.Header.Get("If-None-Match") != "manifest-test-etag" {
					http.Error(w, "missing If-None-Match", http.StatusBadRequest)
					return
				}
				if r.Header.Get("If-Modified-Since") != time.RFC1123 {
					http.Error(w, "missing If-Modified-Since", http.StatusBadRequest)
					return
				}
				w.Header().Set("ETag", "resp-etag")
				w.Header().Set("Last-Modified", time.RFC1123)
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte("external file via proxy"))
			},
			allowedDomains:      nil,
			maxBytes:            0,
			expectError:         false,
			expectTempFile:      true,
			expectContent:       []byte("external file via proxy"),
			expectHeaderETag:    "resp-etag",
			expectHeaderLastMod: time.RFC1123,
			expectErrContains:   "",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx := context.Background()
			tempDir := t.TempDir()
			fileName := filepath.Join(tempDir, "external.conf")

			ts := httptest.NewServer(test.handler)
			defer ts.Close()

			u, err := url.Parse(ts.URL)
			require.NoError(t, err)
			host := u.Hostname()

			fakeFileServiceClient := &v1fakes.FakeFileServiceClient{}
			fileManagerService := NewFileManagerService(fakeFileServiceClient, types.AgentConfig(), &sync.RWMutex{})

			eds := &config.ExternalDataSource{
				ProxyURL:       config.ProxyURL{URL: ""},
				AllowedDomains: []string{host},
				MaxBytes:       int64(test.maxBytes),
			}

			if test.allowedDomains != nil {
				eds.AllowedDomains = test.allowedDomains
			}

			if test.name == "Test 5: ProxyWithConditionalHeaders" {
				manifestFiles := map[string]*model.ManifestFile{
					fileName: {
						ManifestFileMeta: &model.ManifestFileMeta{
							Name:         fileName,
							ETag:         "manifest-test-etag",
							LastModified: time.RFC1123,
						},
					},
				}
				manifestJSON, mErr := json.MarshalIndent(manifestFiles, "", "  ")
				require.NoError(t, mErr)

				manifestFile, mErr := os.CreateTemp(tempDir, "manifest.json")
				require.NoError(t, mErr)
				_, mErr = manifestFile.Write(manifestJSON)
				require.NoError(t, mErr)
				_ = manifestFile.Close()

				fileManagerService.agentConfig.LibDir = tempDir
				fileManagerService.manifestFilePath = manifestFile.Name()

				eds.ProxyURL = config.ProxyURL{URL: ts.URL}
			}

			fileManagerService.agentConfig.ExternalDataSource = eds

			fileManagerService.fileActions = map[string]*model.FileCache{
				fileName: {
					File: &mpi.File{
						FileMeta:           &mpi.FileMeta{Name: fileName},
						ExternalDataSource: &mpi.ExternalDataSource{Location: ts.URL},
					},
					Action: model.ExternalFile,
				},
			}

			err = fileManagerService.downloadUpdatedFilesToTempLocation(ctx)

			if test.expectError {
				require.Error(t, err)
				if test.expectErrContains != "" {
					assert.Contains(t, err.Error(), test.expectErrContains)
				}
				_, statErr := os.Stat(tempFilePath(fileName))
				assert.True(t, os.IsNotExist(statErr))

				return
			}

			require.NoError(t, err)

			if test.expectTempFile {
				b, readErr := os.ReadFile(tempFilePath(fileName))
				require.NoError(t, readErr)
				assert.Equal(t, test.expectContent, b)

				h, ok := fileManagerService.externalFileHeaders[fileName]
				require.True(t, ok)
				assert.Equal(t, test.expectHeaderETag, h.ETag)
				assert.Equal(t, test.expectHeaderLastMod, h.LastModified)

				_ = os.Remove(tempFilePath(fileName))
			} else {
				_, statErr := os.Stat(tempFilePath(fileName))
				assert.True(t, os.IsNotExist(statErr))
			}
		})
	}
}

func TestFileManagerService_DownloadFileContent_MaxBytesLimit(t *testing.T) {
	ctx := context.Background()
	fms := NewFileManagerService(nil, types.AgentConfig(), &sync.RWMutex{})

	// test server returns 10 bytes, we set MaxBytes to 4 and expect only 4 bytes returned
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("ETag", "etag-1")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("0123456789"))
	}))
	defer ts.Close()

	u, err := url.Parse(ts.URL)
	require.NoError(t, err)

	fms.agentConfig.ExternalDataSource = &config.ExternalDataSource{
		AllowedDomains: []string{u.Hostname()},
		MaxBytes:       4,
	}

	fileName := filepath.Join(t.TempDir(), "external.conf")
	file := &mpi.File{
		FileMeta:           &mpi.FileMeta{Name: fileName},
		ExternalDataSource: &mpi.ExternalDataSource{Location: ts.URL},
	}

	content, headers, err := fms.downloadFileContent(ctx, file)
	require.NoError(t, err)
	assert.Len(t, content, 4)
	assert.Equal(t, "etag-1", headers.ETag)
}

func TestFileManagerService_TestDownloadFileContent_InvalidProxyURL(t *testing.T) {
	ctx := context.Background()
	fms := NewFileManagerService(nil, types.AgentConfig(), &sync.RWMutex{})

	downURL := "http://example.com/file"
	fms.agentConfig.ExternalDataSource = &config.ExternalDataSource{
		AllowedDomains: []string{"example.com"},
		ProxyURL:       config.ProxyURL{URL: "http://:"},
	}

	file := &mpi.File{
		FileMeta:           &mpi.FileMeta{Name: "/tmp/file"},
		ExternalDataSource: &mpi.ExternalDataSource{Location: downURL},
	}

	_, _, err := fms.downloadFileContent(ctx, file)
	require.Error(t, err)
	if !strings.Contains(err.Error(), "invalid proxy URL configured") &&
		!strings.Contains(err.Error(), "failed to execute download request") &&
		!strings.Contains(err.Error(), "proxyconnect") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestFileManagerService_IsDomainAllowed(t *testing.T) {
	type testCase struct {
		name           string
		url            string
		allowedDomains []string
		expected       bool
	}

	tests := []testCase{
		{
			name:           "Invalid URL (Percent)",
			url:            "http://%",
			allowedDomains: []string{"example.com"},
			expected:       false,
		},
		{
			name:           "Invalid URL (Empty Host)",
			url:            "http://",
			allowedDomains: []string{"example.com"},
			expected:       false,
		},
		{
			name:           "Empty Allowed List",
			url:            "http://example.com/path",
			allowedDomains: []string{""},
			expected:       false,
		},
		{
			name:           "Basic Match",
			url:            "http://example.com/path",
			allowedDomains: []string{"example.com"},
			expected:       true,
		},
		{
			name:           "Wildcard Subdomain Match",
			url:            "http://sub.example.com/path",
			allowedDomains: []string{"*.example.com"},
			expected:       true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			actual := isDomainAllowed(tc.url, tc.allowedDomains)
			assert.Equal(t, tc.expected, actual, "for URL: %s and domains: %v", tc.url, tc.allowedDomains)
		})
	}
}

func TestFileManagerService_IsMatchesWildcardDomain(t *testing.T) {
	type testCase struct {
		name     string
		hostname string
		pattern  string
		expected bool
	}

	tests := []testCase{
		{
			name:     "True Match - Subdomain",
			hostname: "sub.example.com",
			pattern:  "*.example.com",
			expected: true,
		},
		{
			name:     "True Match - Exact Base Domain",
			hostname: "example.com",
			pattern:  "*.example.com",
			expected: true,
		},
		{
			name:     "False Match - Bad Domain Suffix",
			hostname: "badexample.com",
			pattern:  "*.example.com",
			expected: false,
		},
		{
			name:     "False Match - No Wildcard Prefix",
			hostname: "test.com",
			pattern:  "google.com",
			expected: false,
		},
		{
			name:     "False Match - Different Suffix",
			hostname: "sub.anotherexample.com",
			pattern:  "*.example.com",
			expected: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			actual := isMatchesWildcardDomain(tc.hostname, tc.pattern)
			assert.Equal(t, tc.expected, actual, "Hostname: %s, Pattern: %s", tc.hostname, tc.pattern)
		})
	}
}
