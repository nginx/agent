// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package file

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"maps"
	"math"
	"os"
	"slices"
	"sync"
	"sync/atomic"

	"github.com/cenkalti/backoff/v4"
	mpi "github.com/nginx/agent/v3/api/grpc/mpi/v1"
	backoffHelpers "github.com/nginx/agent/v3/internal/backoff"
	"github.com/nginx/agent/v3/internal/config"
	internalgrpc "github.com/nginx/agent/v3/internal/grpc"
	"github.com/nginx/agent/v3/internal/logger"
	"github.com/nginx/agent/v3/internal/model"
	"github.com/nginx/agent/v3/pkg/files"
	"github.com/nginx/agent/v3/pkg/id"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// File service operator handles requests to the grpc file service

type FileServiceOperator struct {
	fileServiceClient mpi.FileServiceClient
	agentConfig       *config.Config
	fileOperator      fileOperator
	isConnected       *atomic.Bool
}

var _ fileServiceOperatorInterface = (*FileServiceOperator)(nil)

func NewFileServiceOperator(agentConfig *config.Config, fileServiceClient mpi.FileServiceClient,
	manifestLock *sync.RWMutex,
) *FileServiceOperator {
	isConnected := &atomic.Bool{}
	isConnected.Store(false)

	return &FileServiceOperator{
		fileServiceClient: fileServiceClient,
		agentConfig:       agentConfig,
		fileOperator:      NewFileOperator(manifestLock),
		isConnected:       isConnected,
	}
}

func (fso *FileServiceOperator) SetIsConnected(isConnected bool) {
	fso.isConnected.Store(isConnected)
}

func (fso *FileServiceOperator) IsConnected() bool {
	return fso.isConnected.Load()
}

func (fso *FileServiceOperator) File(ctx context.Context, file *mpi.File,
	fileActions map[string]*model.FileCache,
) error {
	slog.DebugContext(ctx, "Getting file", "file", file.GetFileMeta().GetName())

	backOffCtx, backoffCancel := context.WithTimeout(ctx, fso.agentConfig.Client.Backoff.MaxElapsedTime)
	defer backoffCancel()

	getFile := func() (*mpi.GetFileResponse, error) {
		return fso.fileServiceClient.GetFile(ctx, &mpi.GetFileRequest{
			MessageMeta: &mpi.MessageMeta{
				MessageId:     id.GenerateMessageID(),
				CorrelationId: logger.CorrelationID(ctx),
				Timestamp:     timestamppb.Now(),
			},
			FileMeta: file.GetFileMeta(),
		})
	}

	getFileResp, getFileErr := backoff.RetryWithData(
		getFile,
		backoffHelpers.Context(backOffCtx, fso.agentConfig.Client.Backoff),
	)

	if getFileErr != nil {
		return fmt.Errorf("error getting file data for %s: %w", file.GetFileMeta(), getFileErr)
	}

	if writeErr := fso.fileOperator.Write(ctx, getFileResp.GetContents().GetContents(),
		file.GetFileMeta()); writeErr != nil {
		return writeErr
	}

	return fso.validateFileHash(file.GetFileMeta().GetName(), fileActions)
}

func (fso *FileServiceOperator) UpdateOverview(
	ctx context.Context,
	instanceID string,
	filesToUpdate []*mpi.File,
	iteration int,
) error {
	correlationID := logger.CorrelationID(ctx)

	// error case for the UpdateOverview attempts
	if iteration > maxAttempts {
		return errors.New("too many UpdateOverview attempts")
	}

	newCtx, correlationID := fso.setupIdentifiers(ctx, iteration)

	request := &mpi.UpdateOverviewRequest{
		MessageMeta: &mpi.MessageMeta{
			MessageId:     id.GenerateMessageID(),
			CorrelationId: correlationID,
			Timestamp:     timestamppb.Now(),
		},
		Overview: &mpi.FileOverview{
			Files: filesToUpdate,
			ConfigVersion: &mpi.ConfigVersion{
				InstanceId: instanceID,
				Version:    files.GenerateConfigVersion(filesToUpdate),
			},
		},
	}

	backOffCtx, backoffCancel := context.WithTimeout(newCtx, fso.agentConfig.Client.Backoff.MaxElapsedTime)
	defer backoffCancel()

	sendUpdateOverview := func() (*mpi.UpdateOverviewResponse, error) {
		if fso.fileServiceClient == nil {
			return nil, errors.New("file service client is not initialized")
		}

		if !fso.isConnected.Load() {
			return nil, errors.New("CreateConnection rpc has not being called yet")
		}

		slog.DebugContext(newCtx, "Sending update overview request",
			"instance_id", request.GetOverview().GetConfigVersion().GetInstanceId(),
			"request", request, "parent_correlation_id", correlationID,
		)

		response, updateError := fso.fileServiceClient.UpdateOverview(newCtx, request)

		validatedError := internalgrpc.ValidateGrpcError(updateError)

		if validatedError != nil {
			slog.ErrorContext(newCtx, "Failed to send update overview", "error", validatedError)

			return nil, validatedError
		}

		return response, nil
	}

	backoffSettings := fso.agentConfig.Client.Backoff
	response, err := backoff.RetryWithData(
		sendUpdateOverview,
		backoffHelpers.Context(backOffCtx, backoffSettings),
	)
	if err != nil {
		return err
	}

	slog.DebugContext(newCtx, "UpdateOverview response", "response", response)

	if response.GetOverview() == nil {
		slog.DebugContext(ctx, "UpdateOverview response is empty")
		return nil
	}
	delta := files.ConvertToMapOfFiles(response.GetOverview().GetFiles())

	if len(delta) != 0 {
		return fso.updateFiles(ctx, delta, instanceID, iteration)
	}

	return err
}

func (fso *FileServiceOperator) ChunkedFile(ctx context.Context, file *mpi.File) error {
	slog.DebugContext(ctx, "Getting chunked file", "file", file.GetFileMeta().GetName())

	stream, err := fso.fileServiceClient.GetFileStream(ctx, &mpi.GetFileRequest{
		MessageMeta: &mpi.MessageMeta{
			MessageId:     id.GenerateMessageID(),
			CorrelationId: logger.CorrelationID(ctx),
			Timestamp:     timestamppb.Now(),
		},
		FileMeta: file.GetFileMeta(),
	})
	if err != nil {
		return fmt.Errorf("error getting file stream for %s: %w", file.GetFileMeta().GetName(), err)
	}

	// Get header chunk first
	headerChunk, recvHeaderChunkError := stream.Recv()
	if recvHeaderChunkError != nil {
		return recvHeaderChunkError
	}

	slog.DebugContext(ctx, "File header chunk received", "header_chunk", headerChunk)

	header := headerChunk.GetHeader()

	writeChunkedFileError := fso.fileOperator.WriteChunkedFile(ctx, file, header, stream)
	if writeChunkedFileError != nil {
		return writeChunkedFileError
	}

	return nil
}

func (fso *FileServiceOperator) UpdateFile(
	ctx context.Context,
	instanceID string,
	fileToUpdate *mpi.File,
) error {
	slog.InfoContext(ctx, "Updating file", "file_name", fileToUpdate.GetFileMeta().GetName(), "instance_id", instanceID)

	slog.DebugContext(ctx, "Checking file size",
		"file_size", fileToUpdate.GetFileMeta().GetSize(),
		"max_file_size", int64(fso.agentConfig.Client.Grpc.MaxFileSize),
	)

	if fileToUpdate.GetFileMeta().GetSize() <= int64(fso.agentConfig.Client.Grpc.MaxFileSize) {
		return fso.sendUpdateFileRequest(ctx, fileToUpdate)
	}

	return fso.sendUpdateFileStream(ctx, fileToUpdate, fso.agentConfig.Client.Grpc.FileChunkSize)
}

func (fso *FileServiceOperator) validateFileHash(filePath string, fileActions map[string]*model.FileCache) error {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}
	fileHash := files.GenerateHash(content)

	if fileHash != fileActions[filePath].File.GetFileMeta().GetHash() {
		return fmt.Errorf("error writing file, file hash does not match for file %s", filePath)
	}

	return nil
}

func (fso *FileServiceOperator) updateFiles(
	ctx context.Context,
	delta map[string]*mpi.File,
	instanceID string,
	iteration int,
) error {
	diffFiles := slices.Collect(maps.Values(delta))

	for _, file := range diffFiles {
		updateErr := fso.UpdateFile(ctx, instanceID, file)
		if updateErr != nil {
			return updateErr
		}
	}

	iteration++
	slog.InfoContext(ctx, "Updating file overview after file updates", "attempt_number", iteration)

	return fso.UpdateOverview(ctx, instanceID, diffFiles, iteration)
}

func (fso *FileServiceOperator) sendUpdateFileRequest(
	ctx context.Context,
	fileToUpdate *mpi.File,
) error {
	messageMeta := &mpi.MessageMeta{
		MessageId:     id.GenerateMessageID(),
		CorrelationId: logger.CorrelationID(ctx),
		Timestamp:     timestamppb.Now(),
	}

	contents, err := os.ReadFile(fileToUpdate.GetFileMeta().GetName())
	if err != nil {
		return err
	}

	request := &mpi.UpdateFileRequest{
		File: fileToUpdate,
		Contents: &mpi.FileContents{
			Contents: contents,
		},
		MessageMeta: messageMeta,
	}

	backOffCtx, backoffCancel := context.WithTimeout(ctx, fso.agentConfig.Client.Backoff.MaxElapsedTime)
	defer backoffCancel()

	sendUpdateFile := func() (*mpi.UpdateFileResponse, error) {
		slog.DebugContext(ctx, "Sending update file request", "request_file", request.GetFile(),
			"request_message_meta", request.GetMessageMeta())
		if fso.fileServiceClient == nil {
			return nil, errors.New("file service client is not initialized")
		}

		if !fso.isConnected.Load() {
			return nil, errors.New("CreateConnection rpc has not being called yet")
		}

		response, updateError := fso.fileServiceClient.UpdateFile(ctx, request)

		validatedError := internalgrpc.ValidateGrpcError(updateError)

		if validatedError != nil {
			slog.ErrorContext(ctx, "Failed to send update file", "error", validatedError)

			return nil, validatedError
		}

		return response, nil
	}

	response, err := backoff.RetryWithData(
		sendUpdateFile,
		backoffHelpers.Context(backOffCtx, fso.agentConfig.Client.Backoff),
	)
	if err != nil {
		return err
	}

	slog.DebugContext(ctx, "UpdateFile response", "response", response)

	return nil
}

func (fso *FileServiceOperator) sendUpdateFileStream(
	ctx context.Context,
	fileToUpdate *mpi.File,
	chunkSize uint32,
) error {
	if chunkSize == 0 {
		return errors.New("file chunk size must be greater than zero")
	}

	updateFileStreamClient, err := fso.fileServiceClient.UpdateFileStream(ctx)
	if err != nil {
		return err
	}

	err = fso.sendUpdateFileStreamHeader(ctx, fileToUpdate, chunkSize, updateFileStreamClient)
	if err != nil {
		return err
	}

	return fso.sendFileUpdateStreamChunks(ctx, fileToUpdate, chunkSize, updateFileStreamClient)
}

func (fso *FileServiceOperator) sendUpdateFileStreamHeader(
	ctx context.Context,
	fileToUpdate *mpi.File,
	chunkSize uint32,
	updateFileStreamClient grpc.ClientStreamingClient[mpi.FileDataChunk, mpi.UpdateFileResponse],
) error {
	messageMeta := &mpi.MessageMeta{
		MessageId:     id.GenerateMessageID(),
		CorrelationId: logger.CorrelationID(ctx),
		Timestamp:     timestamppb.Now(),
	}

	numberOfChunks := uint32(math.Ceil(float64(fileToUpdate.GetFileMeta().GetSize()) / float64(chunkSize)))

	header := mpi.FileDataChunk_Header{
		Header: &mpi.FileDataChunkHeader{
			FileMeta:  fileToUpdate.GetFileMeta(),
			Chunks:    numberOfChunks,
			ChunkSize: chunkSize,
		},
	}

	backOffCtx, backoffCancel := context.WithTimeout(ctx, fso.agentConfig.Client.Backoff.MaxElapsedTime)
	defer backoffCancel()

	sendUpdateFileHeader := func() error {
		slog.DebugContext(ctx, "Sending update file stream header", "header", header)
		if fso.fileServiceClient == nil {
			return errors.New("file service client is not initialized")
		}

		if !fso.isConnected.Load() {
			return errors.New("CreateConnection rpc has not being called yet")
		}

		err := updateFileStreamClient.Send(
			&mpi.FileDataChunk{
				Meta:  messageMeta,
				Chunk: &header,
			},
		)

		validatedError := internalgrpc.ValidateGrpcError(err)

		if validatedError != nil {
			slog.ErrorContext(ctx, "Failed to send update file stream header", "error", validatedError)

			return validatedError
		}

		return nil
	}

	return backoff.Retry(sendUpdateFileHeader, backoffHelpers.Context(backOffCtx, fso.agentConfig.Client.Backoff))
}

func (fso *FileServiceOperator) sendFileUpdateStreamChunks(
	ctx context.Context,
	fileToUpdate *mpi.File,
	chunkSize uint32,
	updateFileStreamClient grpc.ClientStreamingClient[mpi.FileDataChunk, mpi.UpdateFileResponse],
) error {
	f, err := os.Open(fileToUpdate.GetFileMeta().GetName())
	defer func() {
		closeError := f.Close()
		if closeError != nil {
			slog.WarnContext(
				ctx, "Failed to close file",
				"file", fileToUpdate.GetFileMeta().GetName(),
				"error", closeError,
			)
		}
	}()
	if err != nil {
		return err
	}

	var chunkID uint32

	reader := bufio.NewReader(f)
	for {
		chunk, readChunkError := fso.fileOperator.ReadChunk(ctx, chunkSize, reader, chunkID)
		if readChunkError != nil {
			return readChunkError
		}
		if chunk.Content == nil {
			break
		}

		sendError := fso.sendFileUpdateStreamChunk(ctx, chunk, updateFileStreamClient)
		if sendError != nil {
			return sendError
		}

		chunkID++
	}

	return nil
}

func (fso *FileServiceOperator) sendFileUpdateStreamChunk(
	ctx context.Context,
	chunk mpi.FileDataChunk_Content,
	updateFileStreamClient grpc.ClientStreamingClient[mpi.FileDataChunk, mpi.UpdateFileResponse],
) error {
	messageMeta := &mpi.MessageMeta{
		MessageId:     id.GenerateMessageID(),
		CorrelationId: logger.CorrelationID(ctx),
		Timestamp:     timestamppb.Now(),
	}

	backOffCtx, backoffCancel := context.WithTimeout(ctx, fso.agentConfig.Client.Backoff.MaxElapsedTime)
	defer backoffCancel()

	sendUpdateFileChunk := func() error {
		slog.DebugContext(ctx, "Sending update file stream chunk", "chunk_id", chunk.Content.GetChunkId())
		if fso.fileServiceClient == nil {
			return errors.New("file service client is not initialized")
		}

		if !fso.isConnected.Load() {
			return errors.New("CreateConnection rpc has not being called yet")
		}

		err := updateFileStreamClient.Send(
			&mpi.FileDataChunk{
				Meta:  messageMeta,
				Chunk: &chunk,
			},
		)

		validatedError := internalgrpc.ValidateGrpcError(err)

		if validatedError != nil {
			slog.ErrorContext(ctx, "Failed to send update file stream chunk", "error", validatedError)

			return validatedError
		}

		return nil
	}

	return backoff.Retry(sendUpdateFileChunk, backoffHelpers.Context(backOffCtx, fso.agentConfig.Client.Backoff))
}

func (fso *FileServiceOperator) setupIdentifiers(ctx context.Context, iteration int) (context.Context, string) {
	correlationID := logger.CorrelationID(ctx)
	var requestCorrelationID slog.Attr

	if iteration == 0 {
		requestCorrelationID = logger.GenerateCorrelationID()
	} else {
		requestCorrelationID = logger.CorrelationIDAttr(ctx)
	}

	newCtx := context.WithValue(ctx, logger.CorrelationIDContextKey, requestCorrelationID)

	return newCtx, correlationID
}
