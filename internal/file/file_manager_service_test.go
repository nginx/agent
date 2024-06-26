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

	fileMeta, fileMetaError := protos.GetFileMeta("/etc/nginx/nginx.conf")
	require.NoError(t, fileMetaError)

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

	fileMeta, fileMetaError := protos.GetFileMeta(testFile.Name())
	require.NoError(t, fileMetaError)

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

	overview := mpi.FileOverview{
		Files: []*mpi.File{
			{
				FileMeta: &mpi.FileMeta{
					Name:         filePath,
					Hash:         fileHash,
					ModifiedTime: timestamppb.Now(),
					Permissions:  "0640",
					Size:         0,
				},
				Action: &addAction,
			},
		},
		ConfigVersion: protos.CreateConfigVersion(),
	}

	fakeFileServiceClient := &v1fakes.FakeFileServiceClient{}
	fakeFileServiceClient.GetOverviewReturns(&mpi.GetOverviewResponse{
		Overview: &overview,
	}, nil)
	fakeFileServiceClient.GetFileReturns(&mpi.GetFileResponse{
		Contents: &mpi.FileContents{
			Contents: fileContent,
		},
	}, nil)
	agentConfig := types.AgentConfig()
	agentConfig.AllowedDirectories = []string{tempDir}
	fileManagerService := NewFileManagerService(fakeFileServiceClient, agentConfig)

	request := &mpi.ConfigApplyRequest{
		Overview:      &overview,
		ConfigVersion: protos.CreateConfigVersion(),
	}

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

	overview := mpi.FileOverview{
		Files: []*mpi.File{
			{
				FileMeta: &mpi.FileMeta{
					Name:         tempFile.Name(),
					Hash:         fileHash,
					ModifiedTime: timestamppb.Now(),
					Permissions:  "0640",
					Size:         0,
				},
				Action: &updateAction,
			},
		},
		ConfigVersion: protos.CreateConfigVersion(),
	}

	fakeFileServiceClient := &v1fakes.FakeFileServiceClient{}
	fakeFileServiceClient.GetOverviewReturns(&mpi.GetOverviewResponse{
		Overview: &overview,
	}, nil)
	fakeFileServiceClient.GetFileReturns(&mpi.GetFileResponse{
		Contents: &mpi.FileContents{
			Contents: fileContent,
		},
	}, nil)
	agentConfig := types.AgentConfig()
	agentConfig.AllowedDirectories = []string{tempDir}
	fileManagerService := NewFileManagerService(fakeFileServiceClient, agentConfig)

	request := &mpi.ConfigApplyRequest{
		Overview:      &overview,
		ConfigVersion: protos.CreateConfigVersion(),
	}

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

	fileContent := []byte("some test data")
	fileHash := files.GenerateHash(fileContent)
	tempFile := helpers.CreateFileWithErrorCheck(t, tempDir, "nginx.conf")
	_, writeErr := tempFile.Write(fileContent)
	require.NoError(t, writeErr)

	overview := mpi.FileOverview{
		Files: []*mpi.File{
			{
				FileMeta: &mpi.FileMeta{
					Name:         tempFile.Name(),
					Hash:         fileHash,
					ModifiedTime: timestamppb.Now(),
					Permissions:  "0640",
					Size:         0,
				},
				Action: &deleteAction,
			},
		},
		ConfigVersion: protos.CreateConfigVersion(),
	}

	fakeFileServiceClient := &v1fakes.FakeFileServiceClient{}
	agentConfig := types.AgentConfig()
	agentConfig.AllowedDirectories = []string{tempDir}
	fileManagerService := NewFileManagerService(fakeFileServiceClient, agentConfig)

	request := &mpi.ConfigApplyRequest{
		Overview:      &overview,
		ConfigVersion: protos.CreateConfigVersion(),
	}

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
