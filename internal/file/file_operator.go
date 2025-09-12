// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package file

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"path"
	"sync"

	"github.com/nginx/agent/v3/internal/config"

	"github.com/nginx/agent/v3/internal/model"

	"google.golang.org/grpc"

	"github.com/nginx/agent/v3/pkg/files"

	mpi "github.com/nginx/agent/v3/api/grpc/mpi/v1"
)

// Helper struct to unmarshal the JSON error from stderr.
type helperError struct {
	Error     string `json:"error"`
	Message   string `json:"message"`
	Retryable bool   `json:"retryable"`
	Status    int    `json:"status"`
}
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

// Executes the external helper script to download a file using URL.
func (fo *FileOperator) runHelper(
	ctx context.Context,
	helperPath string,
	url string,
	maxBytes int64,
	tlsConfig *config.TLSConfig,
) (string, error) {
	args := []string{}

	// Add TLS arguments if configured in nginx-config.
	if tlsConfig != nil {
		if tlsConfig.SkipVerify {
			args = append(args, "--skip-verify")
		}
		if tlsConfig.Ca != "" {
			args = append(args, "--ca", tlsConfig.Ca)
		}
		if tlsConfig.ServerName != "" {
			args = append(args, "--server-name", tlsConfig.ServerName)
		}
	}

	// Adding the arguments in the command while calling helper script.
	args = append(args, "--url", url)

	cmd := exec.CommandContext(ctx, helperPath, args...)

	// Create a temporary file to store the downloaded content.
	tmpFile, err := os.CreateTemp("", "external-file-*.tmp")
	if err != nil {
		return "", fmt.Errorf("failed to create temp file: %w", err)
	}
	// Use a variable to track if the file needs to be removed. Set it to true by default for error cases.
	removeTmpFile := true
	defer func() {
		if removeTmpFile {
			os.Remove(tmpFile.Name())
		}
	}()
	defer tmpFile.Close() // Closing the file handle.

	// Set up stdout to stream the output from the helper.
	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		return "", fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	// Set up stderr to capture error messages from the helper script if any.
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	// Execute the command using the helper Script and the URL.
	if err := cmd.Start(); err != nil {
		return "", fmt.Errorf("failed to start helper process: %w", err)
	}

	// Read from stdout in chunks and check the size on the fly.
	var totalBytesRead int64
	reader := bufio.NewReader(stdoutPipe)
	chunk := make([]byte, 4096) // Read in 4KB chunks.

	for {
		n, readErr := reader.Read(chunk)
		if n > 0 {
			totalBytesRead += int64(n)
			// Check if the total downloaded size exceeds the maximum allowed.
			if totalBytesRead > maxBytes {
				// Kill the helper process to stop further downloads if the file exceeds the limit.
				if killErr := cmd.Process.Kill(); killErr != nil {
					slog.WarnContext(ctx, "Failed to kill helper process after exceeding maxBytes", "error", killErr)
				}
				// Clean up the partially downloaded temp file.
				os.Remove(tmpFile.Name())

				return "", fmt.Errorf("downloaded file size (%d bytes) exceeds the maximum allowed size of %d bytes",
					totalBytesRead, maxBytes)
			}
			// Write the chunk to the temporary file.
			if _, writeErr := tmpFile.Write(chunk[:n]); writeErr != nil {
				if killErr := cmd.Process.Kill(); killErr != nil {
					slog.WarnContext(ctx, "Failed to kill helper process after write error", "error", killErr)
				}
				os.Remove(tmpFile.Name())

				return "", fmt.Errorf("error writing to temp file: %w", writeErr)
			}
		}
		// Handle read errors.
		if readErr != nil {
			if readErr == io.EOF {
				break // End of file, download complete.
			}
			// If there's another read error, kill the process and clean up.
			if killErr := cmd.Process.Kill(); killErr != nil {
				slog.WarnContext(ctx, "Failed to kill helper process after read error", "error", killErr)
			}
			os.Remove(tmpFile.Name())

			return "", fmt.Errorf("error reading from helper stdout: %w", readErr)
		}
	}

	// Wait for the helper process to finish.
	if err := cmd.Wait(); err != nil {
		// Clean up if the helper process exited with an error.
		os.Remove(tmpFile.Name())
		// Parse stderr for structured error messages.
		var errObj helperError
		if unmarshalErr := json.Unmarshal(stderr.Bytes(), &errObj); unmarshalErr != nil {
			// If JSON parsing fails, return the raw stderr.
			return "", fmt.Errorf("helper process failed with exit code %d, stderr: %s", cmd.ProcessState.ExitCode(),
				stderr.String())
		}
		// Return a formatted error from the parsed JSON object.
		return "", fmt.Errorf("helper process failed with error '%s' and message '%s'", errObj.Error, errObj.Message)
	}

	// If successful, set removeTmpFile to false so the deferred cleanup doesn't remove it,
	removeTmpFile = false
	// If successful, return the path to the temporary file.
	return tmpFile.Name(), nil
}
