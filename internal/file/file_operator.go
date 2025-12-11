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
	"path/filepath"
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

func (fo *FileOperator) Write(ctx context.Context, fileContent []byte, fileName, filePermissions string) error {
	filePermission := files.FileMode(filePermissions)
	err := fo.CreateFileDirectories(ctx, fileName)
	if err != nil {
		return err
	}

	writeErr := os.WriteFile(fileName, fileContent, filePermission)
	if writeErr != nil {
		return fmt.Errorf("error writing to file %s: %w", fileName, writeErr)
	}
	slog.DebugContext(ctx, "Content written to file", "file_path", fileName)

	return nil
}

func (fo *FileOperator) CreateFileDirectories(
	ctx context.Context,
	fileName string,
) error {
	if _, err := os.Stat(fileName); os.IsNotExist(err) {
		parentDirectory := path.Dir(fileName)
		slog.DebugContext(
			ctx, "File does not exist, creating parent directory",
			"directory_path", parentDirectory,
		)
		err = os.MkdirAll(parentDirectory, dirPerm)
		if err != nil {
			return fmt.Errorf("error creating directory %s: %w", parentDirectory, err)
		}
	}

	return nil
}

func (fo *FileOperator) WriteChunkedFile(
	ctx context.Context,
	fileName, filePermissions string,
	header *mpi.FileDataChunkHeader,
	stream grpc.ServerStreamingClient[mpi.FileDataChunk],
) error {
	createFileDirectoriesError := fo.CreateFileDirectories(ctx, fileName)
	if createFileDirectoriesError != nil {
		return createFileDirectoriesError
	}

	fileToWrite, createError := os.Create(fileName)
	defer func() {
		closeError := fileToWrite.Close()
		if closeError != nil {
			slog.WarnContext(
				ctx, "Failed to close file",
				"file", fileName,
				"error", closeError,
			)
		}
	}()
	if createError != nil {
		return createError
	}

	filePermission := files.FileMode(filePermissions)
	if err := os.Chmod(fileName, filePermission); err != nil {
		return fmt.Errorf("error setting permissions for %s file: %w", fileName, err)
	}

	slog.DebugContext(ctx, "Writing chunked file", "file", fileName)
	for range header.GetChunks() {
		chunk, recvError := stream.Recv()
		if recvError != nil {
			return recvError
		}

		_, chunkWriteError := fileToWrite.Write(chunk.GetContent().GetData())
		if chunkWriteError != nil {
			return fmt.Errorf("error writing chunk to file %s: %w", fileName, chunkWriteError)
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

func (fo *FileOperator) WriteManifestFile(
	ctx context.Context, updatedFiles map[string]*model.ManifestFile, manifestDir, manifestPath string,
) error {
	slog.DebugContext(ctx, "Writing manifest file", "updated_files", updatedFiles)
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

	// Write to a temporary file first to ensure atomicity
	tempManifestFilePath := manifestPath + ".tmp"
	tempFile, err := os.OpenFile(tempManifestFilePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, filePerm)
	if err != nil {
		return fmt.Errorf("failed to open temporary manifest file: %w", err)
	}

	if _, err = tempFile.Write(manifestJSON); err != nil {
		closeFile(ctx, tempFile)

		return fmt.Errorf("failed to write to temporary manifest file: %w", err)
	}

	closeFile(ctx, tempFile)

	// Verify the contents of the temporary file is JSON
	file, err := os.ReadFile(tempManifestFilePath)
	if err != nil {
		return fmt.Errorf("failed to read temporary manifest file: %w", err)
	}

	var manifestFiles map[string]*model.ManifestFile

	err = json.Unmarshal(file, &manifestFiles)
	if err != nil {
		if len(file) == 0 {
			return fmt.Errorf("temporary manifest file is empty: %w", err)
		}

		return fmt.Errorf("failed to parse temporary manifest file: %w", err)
	}

	// Rename the temporary file to the actual manifest file path
	if renameError := os.Rename(tempManifestFilePath, manifestPath); renameError != nil {
		return fmt.Errorf("failed to rename temporary manifest file: %w", renameError)
	}

	return nil
}

func (fo *FileOperator) MoveFile(ctx context.Context, sourcePath, destPath string) error {
	inputFile, openErr := os.Open(sourcePath)
	if openErr != nil {
		return fmt.Errorf("failed to open source file %s: %w", sourcePath, openErr)
	}
	defer closeFile(ctx, inputFile)

	fileInfo, statErr := inputFile.Stat()
	if statErr != nil {
		return fmt.Errorf("failed to stat source file %s: %w", sourcePath, statErr)
	}

	if dirErr := os.MkdirAll(filepath.Dir(destPath), dirPerm); dirErr != nil {
		return fmt.Errorf("failed to create directories for %s: %w", destPath, dirErr)
	}

	outputFile, createErr := os.OpenFile(destPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, fileInfo.Mode())
	if createErr != nil {
		return fmt.Errorf("failed to create destination file %s: %w", destPath, createErr)
	}
	defer closeFile(ctx, outputFile)

	_, copyErr := io.Copy(outputFile, inputFile)
	if copyErr != nil {
		return fmt.Errorf("failed to copy data from %s to %s: %w", sourcePath, destPath, copyErr)
	}

	if err := os.Chmod(outputFile.Name(), fileInfo.Mode()); err != nil {
		return fmt.Errorf("failed to change file permissions chmod: %w", err)
	}

	return nil
}

func closeFile(ctx context.Context, file *os.File) {
	err := file.Close()
	if err != nil {
		slog.ErrorContext(ctx, "Error closing file", "error", err, "file", file.Name())
	}
}
