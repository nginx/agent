// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package grpc

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log/slog"
	"math"
	"os"
	"path/filepath"
	"strconv"

	"github.com/nginx/agent/v3/pkg/files"

	"github.com/cenkalti/backoff/v4"
	backoffHelpers "github.com/nginx/agent/v3/internal/backoff"
	"github.com/nginx/agent/v3/internal/config"
	internalgrpc "github.com/nginx/agent/v3/internal/grpc"
	"github.com/nginx/agent/v3/internal/logger"
	"google.golang.org/grpc"

	"github.com/nginx/agent/v3/api/grpc/mpi/v1"
	"github.com/nginx/agent/v3/pkg/id"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const defaultFilePermissions = 0o644

type FileService struct {
	agentConfig *config.Config
	v1.UnimplementedFileServiceServer
	instanceFiles   map[string][]*v1.File // key is instanceID
	requestChan     chan *v1.ManagementPlaneRequest
	configDirectory string
}

func NewFileService(configDirectory string, requestChan chan *v1.ManagementPlaneRequest,
	agentConfig *config.Config,
) *FileService {
	return &FileService{
		configDirectory: configDirectory,
		instanceFiles:   make(map[string][]*v1.File),
		requestChan:     requestChan,
		agentConfig:     agentConfig,
	}
}

func (mgs *FileService) GetOverview(
	ctx context.Context,
	request *v1.GetOverviewRequest,
) (*v1.GetOverviewResponse, error) {
	configVersion := request.GetConfigVersion()

	slog.InfoContext(ctx, "Getting overview", "config_version", configVersion)

	if _, ok := mgs.instanceFiles[request.GetConfigVersion().GetInstanceId()]; !ok {
		slog.ErrorContext(ctx, "Config version not found", "config_version", configVersion)
		return nil, status.Errorf(codes.NotFound, "Config version not found")
	}

	return &v1.GetOverviewResponse{
		Overview: &v1.FileOverview{
			ConfigVersion: configVersion,
			Files:         mgs.instanceFiles[configVersion.GetInstanceId()],
		},
	}, nil
}

// nolint: unparam
func (mgs *FileService) UpdateOverview(
	ctx context.Context,
	request *v1.UpdateOverviewRequest,
) (*v1.UpdateOverviewResponse, error) {
	overview := request.GetOverview()

	marshaledJSON, errMarshaledJSON := protojson.Marshal(request)
	if errMarshaledJSON != nil {
		return nil, fmt.Errorf("failed to marshal struct back to JSON: %w", errMarshaledJSON)
	}
	slog.InfoContext(ctx, "Updating overview JSON", "overview", marshaledJSON)

	mgs.instanceFiles[overview.GetConfigVersion().GetInstanceId()] = overview.GetFiles()

	configUploadRequest := &v1.ManagementPlaneRequest{
		MessageMeta: &v1.MessageMeta{
			MessageId:     id.GenerateMessageID(),
			CorrelationId: request.GetMessageMeta().GetCorrelationId(),
			Timestamp:     timestamppb.Now(),
		},
		Request: &v1.ManagementPlaneRequest_ConfigUploadRequest{
			ConfigUploadRequest: &v1.ConfigUploadRequest{
				Overview: overview,
			},
		},
	}
	mgs.requestChan <- configUploadRequest

	return &v1.UpdateOverviewResponse{}, nil
}

func (mgs *FileService) GetFile(
	ctx context.Context,
	request *v1.GetFileRequest,
) (*v1.GetFileResponse, error) {
	fileName := request.GetFileMeta().GetName()
	fileHash := request.GetFileMeta().GetHash()

	slog.InfoContext(ctx, "Getting file", "name", fileName, "hash", fileHash)

	fullFilePath := mgs.findFile(request.GetFileMeta())

	if fullFilePath == "" {
		slog.ErrorContext(ctx, "File not found", "file_name", fileName)
		return nil, status.Errorf(codes.NotFound, "File not found")
	}

	bytes, err := os.ReadFile(fullFilePath)
	if err != nil {
		slog.ErrorContext(ctx, "Failed to get file contents", "full_file_path", fullFilePath, "error", err)
		return nil, status.Errorf(codes.Internal, "Failed to get file contents")
	}

	return &v1.GetFileResponse{
		Contents: &v1.FileContents{
			Contents: bytes,
		},
	}, nil
}

func (mgs *FileService) GetFileStream(request *v1.GetFileRequest,
	streamingServer grpc.ServerStreamingServer[v1.FileDataChunk],
) error {
	correlationID := request.GetMessageMeta().GetCorrelationId()
	newCtx := context.WithValue(streamingServer.Context(), logger.CorrelationIDContextKey, correlationID)
	fileName := request.GetFileMeta().GetName()
	fileHash := request.GetFileMeta().GetHash()

	slog.Info("Getting file, stream", "name", fileName, "hash", fileHash)

	fullFilePath := mgs.findFile(request.GetFileMeta())

	if fullFilePath == "" {
		slog.Error("File not found", "file_name", fileName)
		return status.Errorf(codes.NotFound, "File not found")
	}

	err := mgs.sendGetFileStreamHeader(newCtx, request.GetFileMeta(), mgs.agentConfig.Client.Grpc.FileChunkSize,
		streamingServer)
	if err != nil {
		return err
	}

	return mgs.sendGetFileStreamChunks(newCtx, fullFilePath, fileName, mgs.agentConfig.Client.Grpc.FileChunkSize,
		streamingServer)
}

func (mgs *FileService) UpdateFile(
	ctx context.Context,
	request *v1.UpdateFileRequest,
) (*v1.UpdateFileResponse, error) {
	fileContents := request.GetContents().GetContents()
	fileMeta := request.GetFile().GetFileMeta()
	fileName := fileMeta.GetName()
	fileHash := fileMeta.GetHash()
	filePermissions := fileMeta.GetPermissions()

	slog.InfoContext(ctx, "Updating file", "name", fileName, "hash", fileHash)

	fullFilePath := mgs.findFile(request.GetFile().GetFileMeta())

	if _, err := os.Stat(fullFilePath); os.IsNotExist(err) {
		statErr := os.MkdirAll(filepath.Dir(fullFilePath), os.ModePerm)
		if statErr != nil {
			slog.InfoContext(ctx, "Failed to create/update file", "full_file_path", fullFilePath, "error", statErr)
			return nil, status.Errorf(codes.Internal, "Failed to create/update file")
		}
	}

	err := os.WriteFile(fullFilePath, fileContents, fileMode(filePermissions))
	if err != nil {
		slog.InfoContext(ctx, "Failed to create/update file", "full_file_path", fullFilePath, "error", err)
		return nil, status.Errorf(codes.Internal, "Failed to create/update file")
	}

	return &v1.UpdateFileResponse{
		FileMeta: fileMeta,
	}, nil
}

func (mgs *FileService) UpdateFileStream(streamingServer grpc.ClientStreamingServer[v1.FileDataChunk,
	v1.UpdateFileResponse],
) error {
	slog.Info("Updating file, stream")

	headerChunk, headerChunkErr := streamingServer.Recv()
	if headerChunkErr != nil {
		return headerChunkErr
	}

	slog.Debug("File header chunk received", "header_chunk", headerChunk)

	header := headerChunk.GetHeader()
	writeChunkedFileError := mgs.WriteChunkFile(header.GetFileMeta(), header, streamingServer)
	if writeChunkedFileError != nil {
		return writeChunkedFileError
	}

	return nil
}

func (mgs *FileService) WriteChunkFile(fileMeta *v1.FileMeta, header *v1.FileDataChunkHeader,
	stream grpc.ClientStreamingServer[v1.FileDataChunk, v1.UpdateFileResponse],
) error {
	fileName := mgs.findFile(fileMeta)
	filePermissions := files.FileMode(fileMeta.GetPermissions())
	fullFilePath := mgs.findFile(fileMeta)

	if err := mgs.createDirectories(fullFilePath, filePermissions); err != nil {
		return err
	}

	fileToWrite, createError := os.Create(fullFilePath)
	defer func() {
		closeError := fileToWrite.Close()
		if closeError != nil {
			slog.Warn("Failed to close file",
				"file", fileMeta.GetName(),
				"error", closeError,
			)
		}
	}()

	if createError != nil {
		return createError
	}
	slog.Debug("Writing chunked file", "file", fileName)
	for range header.GetChunks() {
		chunk, recvError := stream.Recv()
		if recvError != nil {
			return recvError
		}

		_, chunkWriteError := fileToWrite.Write(chunk.GetContent().GetData())
		if chunkWriteError != nil {
			return fmt.Errorf("error writing chunk to file %s: %w", fileMeta.GetName(), chunkWriteError)
		}
	}

	return nil
}

func (mgs *FileService) sendGetFileStreamChunks(ctx context.Context, fullFilePath, filePath string, chunkSize uint32,
	streamingServer grpc.ServerStreamingServer[v1.FileDataChunk],
) error {
	f, err := os.Open(fullFilePath)
	defer func() {
		closeError := f.Close()
		if closeError != nil {
			slog.Warn("Failed to close file",
				"file", filePath,
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
		chunk, readChunkError := readChunk(chunkSize, reader, chunkID)
		if readChunkError != nil {
			return readChunkError
		}
		if chunk.Content == nil {
			break
		}

		sendErr := mgs.sendGetFileStreamChunk(ctx, chunk, streamingServer)
		if sendErr != nil {
			return sendErr
		}

		chunkID++
	}

	return nil
}

func (mgs *FileService) sendGetFileStreamHeader(ctx context.Context,
	fileToUpdate *v1.FileMeta,
	chunkSize uint32, streamingServer grpc.ServerStreamingServer[v1.FileDataChunk],
) error {
	messageMeta := &v1.MessageMeta{
		MessageId:     id.GenerateMessageID(),
		CorrelationId: logger.CorrelationID(ctx),
		Timestamp:     timestamppb.Now(),
	}

	numberOfChunks := uint32(math.Ceil(float64(fileToUpdate.GetSize()) / float64(chunkSize)))

	header := v1.FileDataChunk_Header{
		Header: &v1.FileDataChunkHeader{
			FileMeta:  fileToUpdate,
			Chunks:    numberOfChunks,
			ChunkSize: chunkSize,
		},
	}

	backOffCtx, backoffCancel := context.WithTimeout(ctx, mgs.agentConfig.Client.Backoff.MaxElapsedTime)
	defer backoffCancel()

	sendGetFileHeader := func() error {
		slog.DebugContext(ctx, "Sending update file stream header", "header", header)
		err := streamingServer.Send(
			&v1.FileDataChunk{
				Meta:  messageMeta,
				Chunk: &header,
			},
		)

		validatedError := internalgrpc.ValidateGrpcError(err)

		if validatedError != nil {
			slog.ErrorContext(ctx, "Failed to send get file stream header", "error", validatedError)

			return validatedError
		}

		return nil
	}

	return backoff.Retry(sendGetFileHeader, backoffHelpers.Context(backOffCtx, mgs.agentConfig.Client.Backoff))
}

func (mgs *FileService) sendGetFileStreamChunk(ctx context.Context, chunk v1.FileDataChunk_Content,
	streamingServer grpc.ServerStreamingServer[v1.FileDataChunk],
) error {
	messageMeta := &v1.MessageMeta{
		MessageId:     id.GenerateMessageID(),
		CorrelationId: logger.CorrelationID(ctx),
		Timestamp:     timestamppb.Now(),
	}

	backOffCtx, backoffCancel := context.WithTimeout(ctx,
		mgs.agentConfig.Client.Backoff.MaxElapsedTime)
	defer backoffCancel()

	sendGetFileChunk := func() error {
		slog.DebugContext(ctx, "Sending get file stream chunk", "chunk_id", chunk.Content.GetChunkId())
		err := streamingServer.Send(
			&v1.FileDataChunk{
				Meta:  messageMeta,
				Chunk: &chunk,
			})
		validatedError := internalgrpc.ValidateGrpcError(err)
		if validatedError != nil {
			slog.ErrorContext(ctx, "Failed to send get file stream chunk", "error", validatedError)

			return validatedError
		}

		return nil
	}

	return backoff.Retry(sendGetFileChunk, backoffHelpers.Context(backOffCtx, mgs.agentConfig.Client.Backoff))
}

func readChunk(
	chunkSize uint32,
	reader *bufio.Reader,
	chunkID uint32,
) (v1.FileDataChunk_Content, error) {
	buf := make([]byte, chunkSize)
	n, err := reader.Read(buf)
	buf = buf[:n]
	if err != nil {
		if err != io.EOF {
			return v1.FileDataChunk_Content{}, fmt.Errorf("failed to read chunk: %w", err)
		}

		slog.Debug("No more data to read from file")

		return v1.FileDataChunk_Content{}, nil
	}

	slog.Debug("read file chunk", "chunk_id", chunkID, "chunk_size", len(buf))

	chunk := v1.FileDataChunk_Content{
		Content: &v1.FileDataChunkContent{
			ChunkId: chunkID,
			Data:    buf,
		},
	}

	return chunk, err
}

func (mgs *FileService) createDirectories(fullFilePath string, filePermissions os.FileMode) error {
	if _, err := os.Stat(fullFilePath); os.IsNotExist(err) {
		statErr := os.MkdirAll(filepath.Dir(fullFilePath), filePermissions)
		if statErr != nil {
			return status.Errorf(codes.Internal, "Failed to create/update file")
		}
	}

	return nil
}

func (mgs *FileService) findFile(fileMeta *v1.FileMeta) (fullFilePath string) {
	for instanceID, instanceFiles := range mgs.instanceFiles {
		for _, file := range instanceFiles {
			if file.GetFileMeta().GetName() == fileMeta.GetName() {
				fullFilePath = filepath.Join(mgs.configDirectory, instanceID, fileMeta.GetName())
			}
		}
	}

	return fullFilePath
}

func fileMode(mode string) os.FileMode {
	result, err := strconv.ParseInt(mode, 8, 32)
	if err != nil {
		return os.FileMode(defaultFilePermissions)
	}

	return os.FileMode(result)
}
