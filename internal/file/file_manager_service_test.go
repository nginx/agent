// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package file

import (
	"context"
	"os"
	"testing"

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

	err := fileManagerService.UpdateFile(ctx, "123", &mpi.File{FileMeta: fileMeta})

	require.NoError(t, err)
	assert.Equal(t, 1, fakeFileServiceClient.UpdateFileCallCount())
}
