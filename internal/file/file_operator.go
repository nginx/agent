// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package file

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path"

	"google.golang.org/grpc"

	"github.com/nginx/agent/v3/pkg/files"

	mpi "github.com/nginx/agent/v3/api/grpc/mpi/v1"
)

type FileOperator struct{}

var operatorLogOrigin = slog.String("log_origin", "file_operator.go")

var _ fileOperator = (*FileOperator)(nil)

// FileOperator only purpose is to write files,

func NewFileOperator() *FileOperator {
	return &FileOperator{}
}

func (fo *FileOperator) Write(ctx context.Context, fileContent []byte, file *mpi.FileMeta) error {
	filePermission := files.FileMode(file.GetPermissions())
	err := fo.CreateFileDirectories(ctx, file, filePermission)
	if err != nil {
		return err
	}

	writeErr := os.WriteFile(file.GetName(), fileContent, filePermission)
	if writeErr != nil {
		return fmt.Errorf("error writing to file %s: %w", file.GetName(), writeErr)
	}
	slog.DebugContext(ctx, "Content written to file", "file_path", file.GetName(), operatorLogOrigin)

	return nil
}

func (fo *FileOperator) CreateFileDirectories(
	ctx context.Context,
	fileMeta *mpi.FileMeta,
	filePermission os.FileMode,
) error {
	if _, err := os.Stat(fileMeta.GetName()); os.IsNotExist(err) {
		parentDirectory := path.Dir(fileMeta.GetName())
		slog.DebugContext(
			ctx, "File does not exist, creating parent directory",
			"directory_path", parentDirectory,
		)
		err = os.MkdirAll(parentDirectory, filePermission)
		if err != nil {
			return fmt.Errorf("error creating directory %s: %w", parentDirectory, err)
		}
	}

	return nil
}

func (fo *FileOperator) WriteChunkedFile(
	ctx context.Context,
	file *mpi.File,
	header *mpi.FileDataChunkHeader,
	stream grpc.ServerStreamingClient[mpi.FileDataChunk],
) error {
	filePermissions := files.FileMode(file.GetFileMeta().GetPermissions())
	createFileDirectoriesError := fo.CreateFileDirectories(ctx, file.GetFileMeta(), filePermissions)
	if createFileDirectoriesError != nil {
		return createFileDirectoriesError
	}

	fileToWrite, createError := os.Create(file.GetFileMeta().GetName())
	defer func() {
		closeError := fileToWrite.Close()
		if closeError != nil {
			slog.WarnContext(
				ctx, "Failed to close file",
				"file", file.GetFileMeta().GetName(),
				"error", closeError,
				operatorLogOrigin,
			)
		}
	}()
	if createError != nil {
		return createError
	}

	slog.DebugContext(ctx, "Writing chunked file", "file", file.GetFileMeta().GetName(), operatorLogOrigin)
	for i := uint32(0); i < header.GetChunks(); i++ {
		chunk, recvError := stream.Recv()
		if recvError != nil {
			return recvError
		}

		_, chunkWriteError := fileToWrite.Write(chunk.GetContent().GetData())
		if chunkWriteError != nil {
			return fmt.Errorf("error writing chunk to file %s: %w", file.GetFileMeta().GetName(), chunkWriteError)
		}
	}

	return nil
}

func (fo *FileOperator) ReadChunk(
	ctx context.Context,
	chunkSize uint32,
	reader *bufio.Reader,
	chunkID uint32,
) (mpi.FileDataChunk_Content, error) {
	buf := make([]byte, chunkSize)
	n, err := reader.Read(buf)
	buf = buf[:n]
	if err != nil {
		if err != io.EOF {
			return mpi.FileDataChunk_Content{}, fmt.Errorf("failed to read chunk: %w", err)
		}

		slog.DebugContext(ctx, "No more data to read from file")

		return mpi.FileDataChunk_Content{}, nil
	}

	slog.DebugContext(ctx, "Read file chunk", "chunk_id", chunkID, "chunk_size", len(buf))

	chunk := mpi.FileDataChunk_Content{
		Content: &mpi.FileDataChunkContent{
			ChunkId: chunkID,
			Data:    buf,
		},
	}

	return chunk, err
}
