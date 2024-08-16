// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package file

import (
	"context"
	"fmt"
	"log/slog"
	"path/filepath"
	"testing"

	mpi "github.com/nginx/agent/v3/api/grpc/mpi/v1"
	"github.com/nginx/agent/v3/internal/grpc"
	"github.com/nginx/agent/v3/pkg/files"
	"github.com/nginx/agent/v3/test/types"
	"github.com/stretchr/testify/require"
)

var configFilePaths = []string{
	"../../test/config/nginx/nginx.conf",
	// "../../test/config/nginx/nginx-with-1k-lines.conf",
	//"../../test/config/nginx/nginx-with-2k-lines.conf",
	"../../test/config/nginx/nginx-with-3k-lines.conf",
	"../../test/config/nginx/nginx-with-10k-lines.conf",
	"../../test/config/nginx/nginx-10MB.conf",
	//"../../test/config/nginx/nginx-100MB.conf",
	// "../../test/config/nginx/nginx-200MB.conf",
}

var chunksizes = []int{
	0,           // no chunking
	16 * 1024,   // the only thing tested so far
	1024 * 1024, // a larger chunk size to be a bit closer to (1/4) the default gRPC message size
}

// runAllFilePaths will run the given test function foreach configFilePaths.
// This checks the file exists and builds the FileMeta
func runAllFilePaths(b *testing.B, tester func(b *testing.B, fileMeta *mpi.FileMeta)) {
	b.Helper()
	for _, configFilePath := range configFilePaths {
		configFilePath := configFilePath
		absFilePath, err := filepath.Abs(configFilePath)
		require.NoError(b, err)

		fileMeta, err := files.GetFileMeta(absFilePath)
		require.NoError(b, err)
		name := filepath.Base(configFilePath)
		b.Run(name, func(b *testing.B) {
			tester(b, fileMeta)
		})
	}
}

// runAllChunkSizes runs the given test function foreach chunksize
func runAllChunkSizes(b *testing.B, tester func(b *testing.B, chunksize int)) {
	b.Helper()
	for _, chunksize := range chunksizes {
		chunksize := chunksize
		chunking := "chunk:off "
		if chunksize > 0 {
			chunking = fmt.Sprintf("chunk:%v ", chunksize)
		}
		b.Run(chunking, func(b *testing.B) {
			tester(b, chunksize)
		})
	}
}

func BenchmarkFileManagerService_Sending_1_File(b *testing.B) {
	slog.SetLogLoggerLevel(slog.LevelError)

	ctx := context.Background()
	agentConfig := types.AgentConfig()

	grpcConnection, err := grpc.NewGrpcConnection(ctx, agentConfig)
	require.NoError(b, err)

	fileManagerService := NewFileManagerService(grpcConnection.FileServiceClient(), agentConfig)
	fileManagerService.isConnected.Store(true)

	runAllFilePaths(b, func(b *testing.B, fileMeta *mpi.FileMeta) {
		b.Run("Unary_UpdateFile", func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				err := fileManagerService.UpdateFile(
					ctx, "123", &mpi.File{FileMeta: fileMeta},
				)
				require.NoError(b, err)
			}
		})
		b.Run("Stream_UploadFile", func(b *testing.B) {
			runAllChunkSizes(b, func(b *testing.B, chunksize int) {
				for i := 0; i < b.N; i++ {
					err := fileManagerService.UploadFile(
						ctx, "123", &mpi.File{FileMeta: fileMeta}, chunksize,
					)
					require.NoError(b, err)
				}
			})
		})
	})
}

func BenchmarkFileManagerService_SendingMultipleFiles_1000_files(b *testing.B) {
	slog.SetLogLoggerLevel(slog.LevelError)

	ctx := context.Background()
	agentConfig := types.AgentConfig()

	grpcConnection, err := grpc.NewGrpcConnection(ctx, agentConfig)
	require.NoError(b, err)

	fileManagerService := NewFileManagerService(grpcConnection.FileServiceClient(), agentConfig)
	fileManagerService.isConnected.Store(true)

	runAllFilePaths(b, func(b *testing.B, fileMeta *mpi.FileMeta) {
		fileList := []*mpi.File{}
		for range 1000 {
			fileList = append(fileList, &mpi.File{
				FileMeta: fileMeta,
			})
		}
		b.Run("Unary", func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				err := fileManagerService.UpdateMultipleFiles(ctx, "123", fileList)
				require.NoError(b, err)
			}
		})

		b.Run("Stream-sequential", func(b *testing.B) {
			runAllChunkSizes(b, func(b *testing.B, chunksize int) {
				for i := 0; i < b.N; i++ {
					err := fileManagerService.UploadMultipleFiles(ctx, "123", fileList, chunksize)
					require.NoError(b, err)
				}
			})
		})
	})
}

// func BenchmarkFileManagerService_GetFile_RPC(b *testing.B) {
// 	slog.SetLogLoggerLevel(slog.LevelError)

// 	ctx := context.Background()
// 	agentConfig := types.AgentConfig()

// 	grpcConnection, err := grpc.NewGrpcConnection(ctx, agentConfig)
// 	require.NoError(b, err)

// 	fileManagerService := NewFileManagerService(grpcConnection.FileServiceClient(), agentConfig)
// 	fileManagerService.isConnected.Store(true)

// 	runAllFilePaths(b, func(b *testing.B, fileMeta *mpi.FileMeta) {
// 		for i := 0; i < b.N; i++ {
// 			err := fileManagerService.GetFile(
// 				ctx,
// 				"123",
// 				&mpi.File{
// 					FileMeta: fileMeta,
// 				},
// 			)
// 			require.NoError(b, err)
// 		}
// 	})
// }

// func BenchmarkFileManagerService_DownloadFile_Stream(b *testing.B) {
// 	slog.SetLogLoggerLevel(slog.LevelError)

// 	ctx := context.Background()
// 	agentConfig := types.AgentConfig()
// 	agentConfig.Common.MaxElapsedTime = time.Hour

// 	grpcConnection, err := grpc.NewGrpcConnection(ctx, agentConfig)
// 	require.NoError(b, err)

// 	fileManagerService := NewFileManagerService(grpcConnection.FileServiceClient(), agentConfig)
// 	fileManagerService.isConnected.Store(true)

// 	runAllFilePaths(b, func(b *testing.B, fileMeta *mpi.FileMeta) {
// 		for i := 0; i < b.N; i++ {
// 			err := fileManagerService.DownloadFile(
// 				ctx,
// 				"123",
// 				&mpi.File{
// 					FileMeta: fileMeta,
// 				},
// 			)
// 			require.NoError(b, err)
// 		}
// 	})
// }
