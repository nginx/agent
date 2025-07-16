// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package file

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"

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
	manifestFilePath := manifestDirPath + "/manifest.json"
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
	fileManagerService.agentConfig.ManifestDir = manifestDirPath
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
	manifestFilePath := manifestDirPath + "/manifest.json"

	fakeFileServiceClient.GetFileStreamReturns(fakeServerStreamingClient, nil)
	agentConfig := types.AgentConfig()
	agentConfig.AllowedDirectories = []string{tempDir}
	fileManagerService := NewFileManagerService(fakeFileServiceClient, agentConfig, &sync.RWMutex{})
	fileManagerService.agentConfig.ManifestDir = manifestDirPath
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
	fileManagerService.agentConfig.ManifestDir = manifestDirPath
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
	assert.Equal(t, fileManagerService.rollbackFileContents[tempFile.Name()], previousFileContent)
	assert.Equal(t, fileManagerService.fileActions[tempFile.Name()].File, overview.GetFiles()[0])
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
	fileManagerService.agentConfig.ManifestDir = manifestDirPath
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
	assert.Equal(t, fileManagerService.rollbackFileContents[tempFile.Name()], fileContent)
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

func TestFileManagerService_ClearCache(t *testing.T) {
	fakeFileServiceClient := &v1fakes.FakeFileServiceClient{}
	fileManagerService := NewFileManagerService(fakeFileServiceClient, types.AgentConfig(), &sync.RWMutex{})

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

	contentsCache := map[string][]byte{
		"file/path/test.conf": []byte("some test data"),
	}

	fileManagerService.fileActions = filesCache
	fileManagerService.rollbackFileContents = contentsCache
	assert.NotEmpty(t, fileManagerService.fileActions)
	assert.NotEmpty(t, fileManagerService.rollbackFileContents)

	fileManagerService.ClearCache()

	assert.Empty(t, fileManagerService.fileActions)
	assert.Empty(t, fileManagerService.rollbackFileContents)
}

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
	fileContentCache := map[string][]byte{
		deleteFilePath:    oldFileContent,
		updateFile.Name(): oldFileContent,
	}

	instanceID := protos.NginxOssInstance([]string{}).GetInstanceMeta().GetInstanceId()
	fakeFileServiceClient := &v1fakes.FakeFileServiceClient{}
	fileManagerService := NewFileManagerService(fakeFileServiceClient, types.AgentConfig(), &sync.RWMutex{})
	fileManagerService.rollbackFileContents = fileContentCache
	fileManagerService.fileActions = filesCache
	fileManagerService.agentConfig.ManifestDir = manifestDirPath
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
	tempDir := os.TempDir()

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
	}{
		{
			name: "Test 1: Add, Update & Delete Files",
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
			name: "Test 2: Files same as on disk",
			modifiedFiles: map[string]*model.FileCache{
				addTestFile.Name(): {
					File: &mpi.File{
						FileMeta: protos.FileMeta(addTestFile.Name(), files.GenerateHash(fileContent)),
					},
				},
				updateTestFile.Name(): {
					File: &mpi.File{
						FileMeta: protos.FileMeta(updateTestFile.Name(), files.GenerateHash(fileContent)),
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
					FileMeta: protos.FileMeta(updateTestFile.Name(), files.GenerateHash(fileContent)),
				},
				addTestFile.Name(): {
					FileMeta: protos.FileMeta(addTestFile.Name(), files.GenerateHash(fileContent)),
				},
			},
			expectedCache:   make(map[string]*model.FileCache),
			expectedContent: make(map[string][]byte),
			expectedError:   nil,
		},
		{
			name:          "Test 3: File being deleted already doesn't exist",
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
	}

	for _, test := range tests {
		t.Run(test.name, func(tt *testing.T) {
			// Delete manifest file if it already exists
			manifestFile := CreateTestManifestFile(t, tempDir, test.currentFiles)
			defer manifestFile.Close()
			manifestDirPath := tempDir
			manifestFilePath := manifestFile.Name()

			fakeFileServiceClient := &v1fakes.FakeFileServiceClient{}
			fileManagerService := NewFileManagerService(fakeFileServiceClient, types.AgentConfig(), &sync.RWMutex{})
			fileManagerService.agentConfig.ManifestDir = manifestDirPath
			fileManagerService.manifestFilePath = manifestFilePath

			require.NoError(tt, err)

			diff, contents, fileActionErr := fileManagerService.DetermineFileActions(
				ctx,
				test.currentFiles,
				test.modifiedFiles,
			)
			require.NoError(tt, fileActionErr)
			assert.Equal(tt, test.expectedContent, contents)
			assert.Equal(tt, test.expectedCache, diff)
		})
	}
}

func CreateTestManifestFile(t testing.TB, tempDir string, currentFiles map[string]*mpi.File) *os.File {
	t.Helper()
	fakeFileServiceClient := &v1fakes.FakeFileServiceClient{}
	fileManagerService := NewFileManagerService(fakeFileServiceClient, types.AgentConfig(), &sync.RWMutex{})
	manifestFiles := fileManagerService.convertToManifestFileMap(currentFiles, true)
	manifestJSON, err := json.MarshalIndent(manifestFiles, "", "  ")
	require.NoError(t, err)
	file, err := os.CreateTemp(tempDir, "manifest.json")
	require.NoError(t, err)

	_, err = file.Write(manifestJSON)
	require.NoError(t, err)

	return file
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
