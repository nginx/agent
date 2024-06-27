// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package file

import (
	"context"
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
	"../../test/config/nginx/nginx-with-1k-lines.conf",
	"../../test/config/nginx/nginx-with-2k-lines.conf",
	"../../test/config/nginx/nginx-with-3k-lines.conf",
	"../../test/config/nginx/nginx-with-10k-lines.conf",
}

func BenchmarkFileManagerService_UpdateFile_RPC(b *testing.B) {
	slog.SetLogLoggerLevel(slog.LevelError)
	
	ctx := context.Background()
	agentConfig := types.AgentConfig()

	grpcConnection, err := grpc.NewGrpcConnection(ctx, agentConfig)
	require.NoError(b, err)
	
	fileManagerService := NewFileManagerService(grpcConnection.FileServiceClient(), agentConfig)
	fileManagerService.isConnected.Store(true)

	for _, configFilePath := range configFilePaths {
		func(configFilePath string) {
			absFilePath, err := filepath.Abs(configFilePath)
			require.NoError(b, err)

			fileMeta, err := files.GetFileMeta(absFilePath)
			require.NoError(b, err)
			 
			b.Run(configFilePath, func(bb *testing.B) {
				for i := 0; i < bb.N; i++ {
					err := fileManagerService.UpdateFile(
						ctx,
						"123",
						&mpi.File{
							FileMeta: fileMeta,
						},
					)
					require.NoError(bb, err)
				}
			})
		}(configFilePath)
	}
}

func BenchmarkFileManagerService_UploadFile_Stream(b *testing.B) {
	slog.SetLogLoggerLevel(slog.LevelError)
	
	ctx := context.Background()
	agentConfig := types.AgentConfig()

	grpcConnection, err := grpc.NewGrpcConnection(ctx, agentConfig)
	require.NoError(b, err)
	
	fileManagerService := NewFileManagerService(grpcConnection.FileServiceClient(), agentConfig)
	fileManagerService.isConnected.Store(true)

	for _, configFilePath := range configFilePaths {
		func(configFilePath string) {
			absFilePath, err := filepath.Abs(configFilePath)
			require.NoError(b, err)

			fileMeta, err := files.GetFileMeta(absFilePath)
			require.NoError(b, err)
			 
			b.Run(configFilePath, func(bb *testing.B) {
				for i := 0; i < bb.N; i++ {
					err := fileManagerService.UploadFile(
						ctx,
						"123",
						&mpi.File{
							FileMeta: fileMeta,
						},
					)
					require.NoError(bb, err)
				}
			})
		}(configFilePath)
	}
}

func BenchmarkFileManagerService_GetFile_RPC(b *testing.B) {
	slog.SetLogLoggerLevel(slog.LevelError)
	
	ctx := context.Background()
	agentConfig := types.AgentConfig()

	grpcConnection, err := grpc.NewGrpcConnection(ctx, agentConfig)
	require.NoError(b, err)
	
	fileManagerService := NewFileManagerService(grpcConnection.FileServiceClient(), agentConfig)
	fileManagerService.isConnected.Store(true)

	for _, configFilePath := range configFilePaths {
		func(configFilePath string) {
			absFilePath, err := filepath.Abs(configFilePath)
			require.NoError(b, err)

			fileMeta, err := files.GetFileMeta(absFilePath)
			fileMeta.Name = filepath.Join("/", filepath.Base(configFilePath))
			require.NoError(b, err)
			 
			b.Run(configFilePath, func(bb *testing.B) {
				for i := 0; i < bb.N; i++ {
					err := fileManagerService.GetFile(
						ctx,
						"123",
						&mpi.File{
							FileMeta: fileMeta,
						},
					)
					require.NoError(bb, err)
				}
			})
		}(configFilePath)
	}
}

func BenchmarkFileManagerService_DownloadFile_Stream(b *testing.B) {
	slog.SetLogLoggerLevel(slog.LevelError)
	
	ctx := context.Background()
	agentConfig := types.AgentConfig()

	grpcConnection, err := grpc.NewGrpcConnection(ctx, agentConfig)
	require.NoError(b, err)
	
	fileManagerService := NewFileManagerService(grpcConnection.FileServiceClient(), agentConfig)
	fileManagerService.isConnected.Store(true)

	for _, configFilePath := range configFilePaths {
		func(configFilePath string) {
			absFilePath, err := filepath.Abs(configFilePath)
			require.NoError(b, err)

			fileMeta, err := files.GetFileMeta(absFilePath)
			fileMeta.Name = filepath.Join("/", filepath.Base(configFilePath))
			require.NoError(b, err)
			 
			b.Run(configFilePath, func(bb *testing.B) {
				for i := 0; i < bb.N; i++ {
					err := fileManagerService.DownloadFile(
						ctx,
						"123",
						&mpi.File{
							FileMeta: fileMeta,
						},
					)
					require.NoError(bb, err)
				}
			})
		}(configFilePath)
	}
}
