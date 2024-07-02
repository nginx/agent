// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package file

import (
	"context"
	"os"
	"path/filepath"
	"testing"

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

func TestFileManagerService_UpdateOverview(t *testing.T) {
	ctx := context.Background()

	fileMeta := protos.FileMeta("/etc/nginx/nginx.conf", "")

	fakeFileServiceClient := &v1fakes.FakeFileServiceClient{}
	fileManagerService := NewFileManagerService(fakeFileServiceClient, types.AgentConfig())
	fileManagerService.SetIsConnected(true)

	err := fileManagerService.UpdateOverview(ctx, "123", []*mpi.File{
		{
			FileMeta: fileMeta,
		},
	})

	require.NoError(t, err)
	assert.Equal(t, 1, fakeFileServiceClient.UpdateOverviewCallCount())
}

func TestFileManagerService_UpdateFile(t *testing.T) {
	ctx := context.Background()

	tempDir := os.TempDir()
	testFile := helpers.CreateFileWithErrorCheck(t, tempDir, "nginx.conf")
	defer helpers.RemoveFileWithErrorCheck(t, testFile.Name())

	fileMeta := protos.FileMeta(testFile.Name(), "")

	fakeFileServiceClient := &v1fakes.FakeFileServiceClient{}
	fileManagerService := NewFileManagerService(fakeFileServiceClient, types.AgentConfig())
	fileManagerService.SetIsConnected(true)

	err := fileManagerService.UpdateFile(ctx, "123", &mpi.File{FileMeta: fileMeta})

	require.NoError(t, err)
	assert.Equal(t, 1, fakeFileServiceClient.UpdateFileCallCount())
}

func TestFileManagerService_ConfigApply_Add(t *testing.T) {
	ctx := context.Background()
	tempDir := t.TempDir()
	addAction := mpi.File_FILE_ACTION_ADD

	filePath := filepath.Join(tempDir, "nginx.conf")
	fileContent := []byte("location /test {\n    return 200 \"Test location\\n\";\n}")
	fileHash := files.GenerateHash(fileContent)
	defer helpers.RemoveFileWithErrorCheck(t, filePath)

	overview := protos.FileOverview(filePath, fileHash, &addAction)

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
	fileManagerService := NewFileManagerService(fakeFileServiceClient, agentConfig)

	request := protos.CreateConfigApplyRequest(overview)
	err := fileManagerService.ConfigApply(ctx, request)
	require.NoError(t, err)
	data, readErr := os.ReadFile(filePath)
	require.NoError(t, readErr)
	assert.Equal(t, fileContent, data)
	assert.Equal(t, fileManagerService.filesCache[filePath], overview.GetFiles()[0])
}

func TestFileManagerService_ConfigApply_Update(t *testing.T) {
	ctx := context.Background()
	tempDir := t.TempDir()
	updateAction := mpi.File_FILE_ACTION_UPDATE

	fileContent := []byte("location /test {\n    return 200 \"Test location\\n\";\n}")
	previousFileContent := []byte("some test data")
	fileHash := files.GenerateHash(fileContent)
	tempFile := helpers.CreateFileWithErrorCheck(t, tempDir, "nginx.conf")
	_, writeErr := tempFile.Write(previousFileContent)
	require.NoError(t, writeErr)
	defer helpers.RemoveFileWithErrorCheck(t, tempFile.Name())

	overview := protos.FileOverview(tempFile.Name(), fileHash, &updateAction)

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
	fileManagerService := NewFileManagerService(fakeFileServiceClient, agentConfig)

	request := protos.CreateConfigApplyRequest(overview)

	err := fileManagerService.ConfigApply(ctx, request)
	require.NoError(t, err)
	data, readErr := os.ReadFile(tempFile.Name())
	require.NoError(t, readErr)
	assert.Equal(t, fileContent, data)
	assert.Equal(t, fileManagerService.fileContentsCache[tempFile.Name()], previousFileContent)
	assert.Equal(t, fileManagerService.filesCache[tempFile.Name()], overview.GetFiles()[0])
}

func TestFileManagerService_ConfigApply_Delete(t *testing.T) {
	ctx := context.Background()
	tempDir := t.TempDir()
	deleteAction := mpi.File_FILE_ACTION_DELETE

	fileContent := []byte("location /test {\n return 200 \"Test location\\n\";\n}")
	tempFile := helpers.CreateFileWithErrorCheck(t, tempDir, "nginx.conf")
	_, writeErr := tempFile.Write(fileContent)
	require.NoError(t, writeErr)

	overview := protos.FileOverview(tempFile.Name(), files.GenerateHash(fileContent), &deleteAction)

	fakeFileServiceClient := &v1fakes.FakeFileServiceClient{}
	agentConfig := types.AgentConfig()
	agentConfig.AllowedDirectories = []string{tempDir}
	fileManagerService := NewFileManagerService(fakeFileServiceClient, agentConfig)

	request := protos.CreateConfigApplyRequest(overview)

	err := fileManagerService.ConfigApply(ctx, request)
	require.NoError(t, err)
	assert.NoFileExists(t, tempFile.Name())
	assert.Equal(t, fileManagerService.fileContentsCache[tempFile.Name()], fileContent)
	assert.Equal(t, fileManagerService.filesCache[tempFile.Name()], overview.GetFiles()[0])
}

func TestFileManagerService_checkAllowedDirectory(t *testing.T) {
	fakeFileServiceClient := &v1fakes.FakeFileServiceClient{}
	fileManagerService := NewFileManagerService(fakeFileServiceClient, types.AgentConfig())

	allowedFiles := []*mpi.File{
		{
			FileMeta: &mpi.FileMeta{
				Name:         "/tmp/local/etc/nginx/allowedDirPath",
				Hash:         "",
				ModifiedTime: nil,
				Permissions:  "",
				Size:         0,
			},
			Action: nil,
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
			Action: nil,
		},
	}

	err := fileManagerService.checkAllowedDirectory(allowedFiles)
	require.NoError(t, err)
	err = fileManagerService.checkAllowedDirectory(notAllowed)
	require.Error(t, err)
}

func TestFileManagerService_ClearCache(t *testing.T) {
	fakeFileServiceClient := &v1fakes.FakeFileServiceClient{}
	fileManagerService := NewFileManagerService(fakeFileServiceClient, types.AgentConfig())

	filesCache := map[string]*mpi.File{
		"file/path/test.conf": {
			FileMeta: &mpi.FileMeta{
				Name:         "file/path/test.conf",
				Hash:         "",
				ModifiedTime: nil,
				Permissions:  "",
				Size:         0,
			},
		},
	}

	contentsCache := map[string][]byte{
		"file/path/test.conf": []byte("some test data"),
	}

	fileManagerService.filesCache = filesCache
	fileManagerService.fileContentsCache = contentsCache
	assert.NotEmpty(t, fileManagerService.filesCache)
	assert.NotEmpty(t, fileManagerService.fileContentsCache)

	fileManagerService.ClearCache()

	assert.Empty(t, fileManagerService.filesCache)
	assert.Empty(t, fileManagerService.fileContentsCache)
}

func TestFileManagerService_Rollback(t *testing.T) {
	ctx := context.Background()
	tempDir := t.TempDir()

	addAction := mpi.File_FILE_ACTION_ADD
	deleteAction := mpi.File_FILE_ACTION_DELETE
	updateAction := mpi.File_FILE_ACTION_UPDATE
	unspecifiedAction := mpi.File_FILE_ACTION_UNSPECIFIED

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

	filesCache := map[string]*mpi.File{
		addFile.Name(): {
			FileMeta: &mpi.FileMeta{
				Name:         addFile.Name(),
				Hash:         fileHash,
				ModifiedTime: timestamppb.Now(),
				Permissions:  "0777",
				Size:         0,
			},
			Action: &addAction,
		},
		updateFile.Name(): {
			FileMeta: &mpi.FileMeta{
				Name:         updateFile.Name(),
				Hash:         fileHash,
				ModifiedTime: timestamppb.Now(),
				Permissions:  "0777",
				Size:         0,
			},
			Action: &updateAction,
		},
		deleteFilePath: {
			FileMeta: &mpi.FileMeta{
				Name:         deleteFilePath,
				Hash:         "",
				ModifiedTime: timestamppb.Now(),
				Permissions:  "0777",
				Size:         0,
			},
			Action: &deleteAction,
		},
		"unspecified/file/test.conf": {
			FileMeta: &mpi.FileMeta{
				Name:         "unspecified/file/test.conf",
				Hash:         "",
				ModifiedTime: timestamppb.Now(),
				Permissions:  "0777",
				Size:         0,
			},
			Action: &unspecifiedAction,
		},
	}

	fileContentCache := map[string][]byte{
		deleteFilePath:    oldFileContent,
		updateFile.Name(): oldFileContent,
	}

	instanceID := protos.GetNginxOssInstance([]string{}).GetInstanceMeta().GetInstanceId()
	fakeFileServiceClient := &v1fakes.FakeFileServiceClient{}
	fileManagerService := NewFileManagerService(fakeFileServiceClient, types.AgentConfig())
	fileManagerService.fileContentsCache = fileContentCache
	fileManagerService.filesCache = filesCache

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

func TestFileManagerService_fileActions(t *testing.T) {
	ctx := context.Background()
	tempDir := t.TempDir()

	addAction := mpi.File_FILE_ACTION_ADD
	deleteAction := mpi.File_FILE_ACTION_DELETE
	updateAction := mpi.File_FILE_ACTION_UPDATE
	unspecifiedAction := mpi.File_FILE_ACTION_UNSPECIFIED

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

	filesCache := map[string]*mpi.File{
		addFilePath: {
			FileMeta: &mpi.FileMeta{
				Name:         addFilePath,
				Hash:         fileHash,
				ModifiedTime: timestamppb.Now(),
				Permissions:  "0777",
				Size:         0,
			},
			Action: &addAction,
		},
		updateFile.Name(): {
			FileMeta: &mpi.FileMeta{
				Name:         updateFile.Name(),
				Hash:         fileHash,
				ModifiedTime: timestamppb.Now(),
				Permissions:  "0777",
				Size:         0,
			},
			Action: &updateAction,
		},
		deleteFile.Name(): {
			FileMeta: &mpi.FileMeta{
				Name:         deleteFile.Name(),
				Hash:         "",
				ModifiedTime: timestamppb.Now(),
				Permissions:  "0777",
				Size:         0,
			},
			Action: &deleteAction,
		},
		unspecifiedFilePath: {
			FileMeta: &mpi.FileMeta{
				Name:         unspecifiedFilePath,
				Hash:         "",
				ModifiedTime: timestamppb.Now(),
				Permissions:  "0777",
				Size:         0,
			},
			Action: &unspecifiedAction,
		},
	}

	fakeFileServiceClient := &v1fakes.FakeFileServiceClient{}
	fakeFileServiceClient.GetFileReturns(&mpi.GetFileResponse{
		Contents: &mpi.FileContents{
			Contents: newFileContent,
		},
	}, nil)
	fileManagerService := NewFileManagerService(fakeFileServiceClient, types.AgentConfig())

	fileManagerService.filesCache = filesCache

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
