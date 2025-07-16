// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package file

import (
	"context"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"

	mpi "github.com/nginx/agent/v3/api/grpc/mpi/v1"
	"github.com/nginx/agent/v3/api/grpc/mpi/v1/v1fakes"
	"github.com/nginx/agent/v3/pkg/files"
	"github.com/nginx/agent/v3/test/helpers"
	"github.com/nginx/agent/v3/test/protos"
	"github.com/nginx/agent/v3/test/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFileServiceOperator_UpdateOverview(t *testing.T) {
	ctx := context.Background()

	filePath := filepath.Join(t.TempDir(), "nginx.conf")
	fileMeta := protos.FileMeta(filePath, "")

	fileContent := []byte("location /test {\n    return 200 \"Test location\\n\";\n}")
	fileHash := files.GenerateHash(fileContent)

	fileWriteErr := os.WriteFile(filePath, fileContent, 0o600)
	require.NoError(t, fileWriteErr)

	overview := protos.FileOverview(filePath, fileHash)

	fakeFileServiceClient := &v1fakes.FakeFileServiceClient{}
	fakeFileServiceClient.UpdateOverviewReturnsOnCall(0, &mpi.UpdateOverviewResponse{
		Overview: overview,
	}, nil)

	fakeFileServiceClient.UpdateOverviewReturnsOnCall(1, &mpi.UpdateOverviewResponse{}, nil)

	fakeFileServiceClient.UpdateFileReturns(&mpi.UpdateFileResponse{}, nil)

	fileServiceOperator := NewFileServiceOperator(types.AgentConfig(), fakeFileServiceClient, &sync.RWMutex{})
	fileServiceOperator.SetIsConnected(true)

	err := fileServiceOperator.UpdateOverview(ctx, "123", []*mpi.File{
		{
			FileMeta: fileMeta,
		},
	}, 0)

	require.NoError(t, err)
	assert.Equal(t, 2, fakeFileServiceClient.UpdateOverviewCallCount())
}

func TestFileServiceOperator_UpdateOverview_MaxIterations(t *testing.T) {
	ctx := context.Background()

	filePath := filepath.Join(t.TempDir(), "nginx.conf")
	fileMeta := protos.FileMeta(filePath, "")

	fileContent := []byte("location /test {\n    return 200 \"Test location\\n\";\n}")
	fileHash := files.GenerateHash(fileContent)

	fileWriteErr := os.WriteFile(filePath, fileContent, 0o600)
	require.NoError(t, fileWriteErr)

	overview := protos.FileOverview(filePath, fileHash)

	fakeFileServiceClient := &v1fakes.FakeFileServiceClient{}

	// do 5 iterations
	for i := range 6 {
		fakeFileServiceClient.UpdateOverviewReturnsOnCall(i, &mpi.UpdateOverviewResponse{
			Overview: overview,
		}, nil)
	}

	fakeFileServiceClient.UpdateFileReturns(&mpi.UpdateFileResponse{}, nil)

	fileServiceOperator := NewFileServiceOperator(types.AgentConfig(), fakeFileServiceClient, &sync.RWMutex{})
	fileServiceOperator.SetIsConnected(true)

	err := fileServiceOperator.UpdateOverview(ctx, "123", []*mpi.File{
		{
			FileMeta: fileMeta,
		},
	}, 0)

	require.Error(t, err)
	assert.Equal(t, "too many UpdateOverview attempts", err.Error())
}

func TestFileManagerService_UpdateFile(t *testing.T) {
	tests := []struct {
		name   string
		isCert bool
	}{
		{
			name:   "non-cert",
			isCert: false,
		},
		{
			name:   "cert",
			isCert: true,
		},
	}

	tempDir := os.TempDir()

	for _, test := range tests {
		ctx := context.Background()

		testFile := helpers.CreateFileWithErrorCheck(t, tempDir, "nginx.conf")

		var fileMeta *mpi.FileMeta
		if test.isCert {
			fileMeta = protos.CertMeta(testFile.Name(), "")
		} else {
			fileMeta = protos.FileMeta(testFile.Name(), "")
		}

		fakeFileServiceClient := &v1fakes.FakeFileServiceClient{}
		fileServiceOperator := NewFileServiceOperator(types.AgentConfig(), fakeFileServiceClient, &sync.RWMutex{})
		fileServiceOperator.SetIsConnected(true)

		err := fileServiceOperator.UpdateFile(ctx, "123", &mpi.File{FileMeta: fileMeta})

		require.NoError(t, err)
		assert.Equal(t, 1, fakeFileServiceClient.UpdateFileCallCount())

		helpers.RemoveFileWithErrorCheck(t, testFile.Name())
	}
}

func TestFileManagerService_UpdateFile_LargeFile(t *testing.T) {
	ctx := context.Background()
	tempDir := os.TempDir()

	testFile := helpers.CreateFileWithErrorCheck(t, tempDir, "nginx.conf")
	writeFileError := os.WriteFile(testFile.Name(), []byte("#test content"), 0o600)
	require.NoError(t, writeFileError)
	fileMeta := protos.FileMetaLargeFile(testFile.Name(), "")

	fakeFileServiceClient := &v1fakes.FakeFileServiceClient{}
	fakeClientStreamingClient := &FakeClientStreamingClient{sendCount: atomic.Int32{}}
	fakeFileServiceClient.UpdateFileStreamReturns(fakeClientStreamingClient, nil)
	fileServiceOperator := NewFileServiceOperator(types.AgentConfig(), fakeFileServiceClient, &sync.RWMutex{})

	fileServiceOperator.SetIsConnected(true)
	err := fileServiceOperator.UpdateFile(ctx, "123", &mpi.File{FileMeta: fileMeta})

	require.NoError(t, err)
	assert.Equal(t, 0, fakeFileServiceClient.UpdateFileCallCount())
	assert.Equal(t, 14, int(fakeClientStreamingClient.sendCount.Load()))

	helpers.RemoveFileWithErrorCheck(t, testFile.Name())
}
