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
	"time"

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
	}, filePath, 0)

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
	}, filePath, 0)

	require.Error(t, err)
	assert.Equal(t, "too many UpdateOverview attempts", err.Error())
}

func TestFileServiceOperator_UpdateOverview_NoConnection(t *testing.T) {
	ctx, cancel := context.WithTimeout(t.Context(), 500*time.Millisecond)
	defer cancel()

	filePath := filepath.Join(t.TempDir(), "nginx.conf")
	fileMeta := protos.FileMeta(filePath, "")

	fakeFileServiceClient := &v1fakes.FakeFileServiceClient{}

	agentConfig := types.AgentConfig()
	agentConfig.Client.Backoff.MaxElapsedTime = 200 * time.Millisecond

	fileServiceOperator := NewFileServiceOperator(types.AgentConfig(), fakeFileServiceClient, &sync.RWMutex{})
	fileServiceOperator.SetIsConnected(false)

	err := fileServiceOperator.UpdateOverview(ctx, "123", []*mpi.File{
		{
			FileMeta: fileMeta,
		},
	}, filePath, 0)

	assert.ErrorIs(t, err, context.DeadlineExceeded)
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

//nolint:revive // complexity is 21.
func TestFileServiceOperator_RenameExternalFile(t *testing.T) {
	type testCase struct {
		name               string
		wantErrMsg         string
		setupFailurePath   string
		destinationPath    string
		expectedDstContent []byte
		srcContent         []byte
		wantErr            bool
	}

	tests := []testCase{
		{
			name:               "Test 1: success",
			srcContent:         []byte("hello world"),
			setupFailurePath:   "",
			destinationPath:    "subdir/dest.txt",
			wantErr:            false,
			expectedDstContent: []byte("hello world"),
		},
		{
			name:               "Test 2: mkdirall_fail",
			srcContent:         []byte("content"),
			setupFailurePath:   "not_a_dir",
			destinationPath:    "not_a_dir/dest.txt",
			wantErr:            true,
			wantErrMsg:         "failed to create directories for",
			expectedDstContent: nil,
		},
		{
			name:               "Test 3: rename_fail (src does not exist)",
			srcContent:         nil,
			setupFailurePath:   "",
			destinationPath:    "subdir/dest.txt",
			wantErr:            true,
			wantErrMsg:         "failed to move file",
			expectedDstContent: nil,
		},
		{
			name:               "Test 4: No destination specified (empty dst path)",
			srcContent:         []byte("source content"),
			setupFailurePath:   "",
			destinationPath:    "",
			wantErr:            true,
			wantErrMsg:         "failed to move file:",
			expectedDstContent: nil,
		},
		{
			name:               "Test 5: Restricted directory (simulated permission fail)",
			srcContent:         []byte("source content"),
			setupFailurePath:   "",
			destinationPath:    "restricted_dir/dest.txt",
			wantErr:            true,
			wantErrMsg:         "permission denied",
			expectedDstContent: nil,
		},
		{
			name:               "Test 6: Two files to the same destination",
			srcContent:         []byte("source content 1"),
			setupFailurePath:   "",
			destinationPath:    "collision_dir/file.txt",
			wantErr:            false,
			expectedDstContent: []byte("source content 2"),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Helper()
			ctx := context.Background()
			fso := NewFileServiceOperator(types.AgentConfig(), nil, &sync.RWMutex{})
			tmp := t.TempDir()

			src := filepath.Join(tmp, "src.txt")
			dst := filepath.Join(tmp, tc.destinationPath)

			if tc.setupFailurePath != "" {
				parentFile := filepath.Join(tmp, tc.setupFailurePath)
				require.NoError(t, os.WriteFile(parentFile, []byte("block"), 0o600))
			}

			if tc.srcContent != nil {
				require.NoError(t, os.WriteFile(src, tc.srcContent, 0o600))
			}

			if tc.name == "Test 6: Two files to the same destination" {
				src1 := src
				dstCollision := dst
				require.NoError(t, fso.RenameExternalFile(ctx, src1, dstCollision), "initial rename must succeed")

				src2 := filepath.Join(tmp, "src2.txt")
				content2 := []byte("source content 2")
				require.NoError(t, os.WriteFile(src2, content2, 0o600))

				src = src2
				dst = dstCollision
			}

			if tc.name == "Test 5: Restricted directory (simulated permission fail)" {
				parentDir := filepath.Dir(dst)
				require.NoError(t, os.MkdirAll(parentDir, 0o500))
			}

			err := fso.RenameExternalFile(ctx, src, dst)

			if tc.wantErr {
				require.Error(t, err)
				if tc.wantErrMsg != "" {
					require.Contains(t, err.Error(), tc.wantErrMsg)
				}

				return
			}

			require.NoError(t, err)

			dstContent, readErr := os.ReadFile(dst)
			require.NoError(t, readErr)
			require.Equal(t, tc.expectedDstContent, dstContent)

			_, statErr := os.Stat(src)
			require.True(t, os.IsNotExist(statErr), "Source file should not exist after successful rename")
		})
	}
}
