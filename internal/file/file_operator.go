// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package file

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path"
	"sync"

	"github.com/nginx/agent/v3/internal/model"

	"google.golang.org/grpc"

	"github.com/nginx/agent/v3/pkg/files"

	mpi "github.com/nginx/agent/v3/api/grpc/mpi/v1"
)

type FileOperator struct {
	manifestLock *sync.RWMutex
}

var _ fileOperator = (*FileOperator)(nil)

// FileOperator only purpose is to write files,

func NewFileOperator(manifestLock *sync.RWMutex) *FileOperator {
	return &FileOperator{
		manifestLock: manifestLock,
	}
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
	slog.DebugContext(ctx, "Content written to file", "file_path", file.GetName())

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
			)
		}
	}()
	if createError != nil {
		return createError
	}

	slog.DebugContext(ctx, "Writing chunked file", "file", file.GetFileMeta().GetName())
	for range header.GetChunks() {
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

func (fo *FileOperator) WriteManifestFile(updatedFiles map[string]*model.ManifestFile, manifestDir,
	manifestPath string,
) (writeError error) {
	manifestJSON, err := json.MarshalIndent(updatedFiles, "", "  ")
	if err != nil {
		return fmt.Errorf("unable to marshal manifest file json: %w", err)
	}

	fo.manifestLock.Lock()
	defer fo.manifestLock.Unlock()
	// 0755 allows read/execute for all, write for owner
	if err = os.MkdirAll(manifestDir, dirPerm); err != nil {
		return fmt.Errorf("unable to create directory %s: %w", manifestDir, err)
	}

	// 0600 ensures only root can read/write
	newFile, err := os.OpenFile(manifestPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, filePerm)
	if err != nil {
		return fmt.Errorf("failed to read manifest file: %w", err)
	}
	defer func() {
		if closeErr := newFile.Close(); closeErr != nil {
			writeError = closeErr
		}
	}()

	_, err = newFile.Write(manifestJSON)
	if err != nil {
		return fmt.Errorf("failed to write manifest file: %w", err)
	}

	return writeError
}
