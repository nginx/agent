// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package file

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"maps"
	"math"
	"os"
	"slices"
	"sync"
	"sync/atomic"

	"google.golang.org/grpc"

	"github.com/nginx/agent/v3/internal/model"

	"github.com/cenkalti/backoff/v4"

	mpi "github.com/nginx/agent/v3/api/grpc/mpi/v1"
	"github.com/nginx/agent/v3/internal/config"
	internalgrpc "github.com/nginx/agent/v3/internal/grpc"
	"github.com/nginx/agent/v3/internal/logger"
	"github.com/nginx/agent/v3/pkg/files"
	"github.com/nginx/agent/v3/pkg/id"
	"google.golang.org/protobuf/types/known/timestamppb"

	backoffHelpers "github.com/nginx/agent/v3/internal/backoff"
)

//go:generate go run github.com/maxbrunsfeld/counterfeiter/v6@v6.8.1 -generate
//counterfeiter:generate . fileOperator

//go:generate go run github.com/maxbrunsfeld/counterfeiter/v6@v6.8.1 -generate
//counterfeiter:generate . fileManagerServiceInterface

const (
	maxAttempts = 5
	dirPerm     = 0o755
	filePerm    = 0o600
)

type (
	fileOperator interface {
		Write(ctx context.Context, fileContent []byte, file *mpi.FileMeta) error
		CreateFileDirectories(ctx context.Context, fileMeta *mpi.FileMeta, filePermission os.FileMode) error
		WriteChunkedFile(
			ctx context.Context,
			file *mpi.File,
			header *mpi.FileDataChunkHeader,
			stream grpc.ServerStreamingClient[mpi.FileDataChunk],
		) error
		ReadChunk(
			ctx context.Context,
			chunkSize uint32,
			reader *bufio.Reader,
			chunkID uint32,
		) (mpi.FileDataChunk_Content, error)
	}

	fileManagerServiceInterface interface {
		UpdateOverview(ctx context.Context, instanceID string, filesToUpdate []*mpi.File, iteration int) error
		ConfigApply(ctx context.Context, configApplyRequest *mpi.ConfigApplyRequest) (writeStatus model.WriteStatus,
			err error)
		Rollback(ctx context.Context, instanceID string) error
		UpdateFile(ctx context.Context, instanceID string, fileToUpdate *mpi.File) error
		ClearCache()
		UpdateCurrentFilesOnDisk(ctx context.Context, updateFiles map[string]*mpi.File, referenced bool) error
		DetermineFileActions(currentFiles map[string]*mpi.File, modifiedFiles map[string]*model.FileCache) (
			map[string]*model.FileCache, map[string][]byte, error)
		IsConnected() bool
		SetIsConnected(isConnected bool)
	}
)

type FileManagerService struct {
	fileServiceClient mpi.FileServiceClient
	agentConfig       *config.Config
	isConnected       *atomic.Bool
	fileOperator      fileOperator
	// map of files and the actions performed on them during config apply
	fileActions map[string]*model.FileCache // key is file path
	// map of the contents of files which have been updated or deleted during config apply, used during rollback
	rollbackFileContents map[string][]byte // key is file path
	// map of the files currently on disk, used to determine the file action during config apply
	currentFilesOnDisk    map[string]*mpi.File // key is file path
	previousManifestFiles map[string]*model.ManifestFile
	manifestFilePath      string
	filesMutex            sync.RWMutex
}

func NewFileManagerService(fileServiceClient mpi.FileServiceClient, agentConfig *config.Config) *FileManagerService {
	isConnected := &atomic.Bool{}
	isConnected.Store(false)

	return &FileManagerService{
		fileServiceClient:     fileServiceClient,
		agentConfig:           agentConfig,
		fileOperator:          NewFileOperator(),
		fileActions:           make(map[string]*model.FileCache),
		rollbackFileContents:  make(map[string][]byte),
		currentFilesOnDisk:    make(map[string]*mpi.File),
		previousManifestFiles: make(map[string]*model.ManifestFile),
		manifestFilePath:      agentConfig.ManifestDir + "/manifest.json",
		isConnected:           isConnected,
	}
}

func (fms *FileManagerService) UpdateOverview(
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

	newCtx, correlationID := fms.setupIdentifiers(ctx, iteration)

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

	backOffCtx, backoffCancel := context.WithTimeout(newCtx, fms.agentConfig.Client.Backoff.MaxElapsedTime)
	defer backoffCancel()

	sendUpdateOverview := func() (*mpi.UpdateOverviewResponse, error) {
		if fms.fileServiceClient == nil {
			return nil, errors.New("file service client is not initialized")
		}

		if !fms.isConnected.Load() {
			return nil, errors.New("CreateConnection rpc has not being called yet")
		}

		slog.DebugContext(newCtx, "Sending update overview request",
			"instance_id", request.GetOverview().GetConfigVersion().GetInstanceId(),
			"request", request, "parent_correlation_id", correlationID,
		)

		response, updateError := fms.fileServiceClient.UpdateOverview(newCtx, request)

		validatedError := internalgrpc.ValidateGrpcError(updateError)

		if validatedError != nil {
			slog.ErrorContext(newCtx, "Failed to send update overview", "error", validatedError)

			return nil, validatedError
		}

		return response, nil
	}

	backoffSettings := fms.agentConfig.Client.Backoff
	response, err := backoff.RetryWithData(
		sendUpdateOverview,
		backoffHelpers.Context(backOffCtx, backoffSettings),
	)
	if err != nil {
		return err
	}

	slog.DebugContext(newCtx, "UpdateOverview response", "response", response)

	if response.GetOverview() == nil {
		slog.Debug("UpdateOverview response is empty")
		return nil
	}
	delta := files.ConvertToMapOfFiles(response.GetOverview().GetFiles())

	if len(delta) != 0 {
		return fms.updateFiles(ctx, delta, instanceID, iteration)
	}

	return err
}

func (fms *FileManagerService) setupIdentifiers(ctx context.Context, iteration int) (context.Context, string) {
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

func (fms *FileManagerService) updateFiles(
	ctx context.Context,
	delta map[string]*mpi.File,
	instanceID string,
	iteration int,
) error {
	diffFiles := slices.Collect(maps.Values(delta))

	for _, file := range diffFiles {
		updateErr := fms.UpdateFile(ctx, instanceID, file)
		if updateErr != nil {
			return updateErr
		}
	}

	iteration++
	slog.Info("Updating file overview after file updates", "attempt_number", iteration)

	return fms.UpdateOverview(ctx, instanceID, diffFiles, iteration)
}

func (fms *FileManagerService) UpdateFile(
	ctx context.Context,
	instanceID string,
	fileToUpdate *mpi.File,
) error {
	slog.InfoContext(ctx, "Updating file", "file_name", fileToUpdate.GetFileMeta().GetName(), "instance_id", instanceID)

	slog.DebugContext(ctx, "Checking file size",
		"file_size", fileToUpdate.GetFileMeta().GetSize(),
		"max_file_size", int64(fms.agentConfig.Client.Grpc.MaxFileSize),
	)

	if fileToUpdate.GetFileMeta().GetSize() <= int64(fms.agentConfig.Client.Grpc.MaxFileSize) {
		return fms.sendUpdateFileRequest(ctx, fileToUpdate)
	}

	return fms.sendUpdateFileStream(ctx, fileToUpdate, fms.agentConfig.Client.Grpc.FileChunkSize)
}

func (fms *FileManagerService) sendUpdateFileRequest(
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

	backOffCtx, backoffCancel := context.WithTimeout(ctx, fms.agentConfig.Client.Backoff.MaxElapsedTime)
	defer backoffCancel()

	sendUpdateFile := func() (*mpi.UpdateFileResponse, error) {
		slog.DebugContext(ctx, "Sending update file request", "request_file", request.GetFile(),
			"request_message_meta", request.GetMessageMeta())
		if fms.fileServiceClient == nil {
			return nil, errors.New("file service client is not initialized")
		}

		if !fms.isConnected.Load() {
			return nil, errors.New("CreateConnection rpc has not being called yet")
		}

		response, updateError := fms.fileServiceClient.UpdateFile(ctx, request)

		validatedError := internalgrpc.ValidateGrpcError(updateError)

		if validatedError != nil {
			slog.ErrorContext(ctx, "Failed to send update file", "error", validatedError)

			return nil, validatedError
		}

		return response, nil
	}

	response, err := backoff.RetryWithData(
		sendUpdateFile,
		backoffHelpers.Context(backOffCtx, fms.agentConfig.Client.Backoff),
	)
	if err != nil {
		return err
	}

	slog.DebugContext(ctx, "UpdateFile response", "response", response)

	return nil
}

func (fms *FileManagerService) sendUpdateFileStream(
	ctx context.Context,
	fileToUpdate *mpi.File,
	chunkSize uint32,
) error {
	if chunkSize == 0 {
		return fmt.Errorf("file chunk size must be greater than zero")
	}

	updateFileStreamClient, err := fms.fileServiceClient.UpdateFileStream(ctx)
	if err != nil {
		return err
	}

	err = fms.sendUpdateFileStreamHeader(ctx, fileToUpdate, chunkSize, updateFileStreamClient)
	if err != nil {
		return err
	}

	return fms.sendFileUpdateStreamChunks(ctx, fileToUpdate, chunkSize, updateFileStreamClient)
}

func (fms *FileManagerService) sendUpdateFileStreamHeader(
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

	backOffCtx, backoffCancel := context.WithTimeout(ctx, fms.agentConfig.Client.Backoff.MaxElapsedTime)
	defer backoffCancel()

	sendUpdateFileHeader := func() error {
		slog.DebugContext(ctx, "Sending update file stream header", "header", header)
		if fms.fileServiceClient == nil {
			return errors.New("file service client is not initialized")
		}

		if !fms.isConnected.Load() {
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

	return backoff.Retry(sendUpdateFileHeader, backoffHelpers.Context(backOffCtx, fms.agentConfig.Client.Backoff))
}

func (fms *FileManagerService) sendFileUpdateStreamChunks(
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
		chunk, readChunkError := fms.fileOperator.ReadChunk(ctx, chunkSize, reader, chunkID)
		if readChunkError != nil {
			return readChunkError
		}
		if chunk.Content == nil {
			break
		}

		sendError := fms.sendFileUpdateStreamChunk(ctx, chunk, updateFileStreamClient)
		if sendError != nil {
			return sendError
		}

		chunkID++
	}

	return nil
}

func (fms *FileManagerService) sendFileUpdateStreamChunk(
	ctx context.Context,
	chunk mpi.FileDataChunk_Content,
	updateFileStreamClient grpc.ClientStreamingClient[mpi.FileDataChunk, mpi.UpdateFileResponse],
) error {
	messageMeta := &mpi.MessageMeta{
		MessageId:     id.GenerateMessageID(),
		CorrelationId: logger.CorrelationID(ctx),
		Timestamp:     timestamppb.Now(),
	}

	backOffCtx, backoffCancel := context.WithTimeout(ctx, fms.agentConfig.Client.Backoff.MaxElapsedTime)
	defer backoffCancel()

	sendUpdateFileChunk := func() error {
		slog.DebugContext(ctx, "Sending update file stream chunk", "chunk_id", chunk.Content.GetChunkId())
		if fms.fileServiceClient == nil {
			return errors.New("file service client is not initialized")
		}

		if !fms.isConnected.Load() {
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

	return backoff.Retry(sendUpdateFileChunk, backoffHelpers.Context(backOffCtx, fms.agentConfig.Client.Backoff))
}

func (fms *FileManagerService) IsConnected() bool {
	return fms.isConnected.Load()
}

func (fms *FileManagerService) SetIsConnected(isConnected bool) {
	fms.isConnected.Store(isConnected)
}

func (fms *FileManagerService) ConfigApply(ctx context.Context,
	configApplyRequest *mpi.ConfigApplyRequest,
) (status model.WriteStatus, err error) {
	fileOverview := configApplyRequest.GetOverview()

	if fileOverview == nil {
		return model.Error, fmt.Errorf("fileOverview is nil")
	}

	allowedErr := fms.checkAllowedDirectory(fileOverview.GetFiles())
	if allowedErr != nil {
		return model.Error, allowedErr
	}

	diffFiles, fileContent, compareErr := fms.DetermineFileActions(fms.currentFilesOnDisk,
		ConvertToMapOfFileCache(fileOverview.GetFiles()))

	if compareErr != nil {
		return model.Error, compareErr
	}

	if len(diffFiles) == 0 {
		return model.NoChange, nil
	}

	fms.rollbackFileContents = fileContent
	fms.fileActions = diffFiles

	fileErr := fms.executeFileActions(ctx)
	if fileErr != nil {
		return model.RollbackRequired, fileErr
	}
	fileOverviewFiles := files.ConvertToMapOfFiles(fileOverview.GetFiles())
	// Update map of current files on disk
	manifestFileErr := fms.UpdateCurrentFilesOnDisk(ctx, fileOverviewFiles, false)
	if manifestFileErr != nil {
		return model.RollbackRequired, manifestFileErr
	}

	return model.OK, nil
}

func (fms *FileManagerService) ClearCache() {
	clear(fms.rollbackFileContents)
	clear(fms.fileActions)
	clear(fms.previousManifestFiles)
}

// nolint:revive,cyclop
func (fms *FileManagerService) Rollback(ctx context.Context, instanceID string) error {
	slog.InfoContext(ctx, "Rolling back config for instance", "instanceid", instanceID)

	fms.filesMutex.Lock()
	defer fms.filesMutex.Unlock()
	for _, fileAction := range fms.fileActions {
		switch fileAction.Action {
		case model.Add:
			if err := os.Remove(fileAction.File.GetFileMeta().GetName()); err != nil && !os.IsNotExist(err) {
				return fmt.Errorf("error deleting file: %s error: %w", fileAction.File.GetFileMeta().GetName(), err)
			}

			// currentFilesOnDisk needs to be updated after rollback action is performed
			delete(fms.currentFilesOnDisk, fileAction.File.GetFileMeta().GetName())

			continue
		case model.Delete, model.Update:
			content := fms.rollbackFileContents[fileAction.File.GetFileMeta().GetName()]
			err := fms.fileOperator.Write(ctx, content, fileAction.File.GetFileMeta())
			if err != nil {
				return err
			}

			// currentFilesOnDisk needs to be updated after rollback action is performed
			fileAction.File.GetFileMeta().Hash = files.GenerateHash(content)
			fms.currentFilesOnDisk[fileAction.File.GetFileMeta().GetName()] = fileAction.File
		case model.Unchanged:
			fallthrough
		default:
			slog.DebugContext(ctx, "File Action not implemented")
		}
	}

	manifestFileErr := fms.writeManifestFile(fms.previousManifestFiles)
	if manifestFileErr != nil {
		return manifestFileErr
	}

	return nil
}

func (fms *FileManagerService) executeFileActions(ctx context.Context) error {
	for _, fileAction := range fms.fileActions {
		switch fileAction.Action {
		case model.Delete:
			slog.Debug("File action, deleting file", "file", fileAction.File.GetFileMeta().GetName())
			if err := os.Remove(fileAction.File.GetFileMeta().GetName()); err != nil && !os.IsNotExist(err) {
				return fmt.Errorf("error deleting file: %s error: %w",
					fileAction.File.GetFileMeta().GetName(), err)
			}

			continue
		case model.Add, model.Update:
			slog.Debug("File action, add or update file", "file", fileAction.File.GetFileMeta().GetName())
			updateErr := fms.fileUpdate(ctx, fileAction.File)
			if updateErr != nil {
				return updateErr
			}
		case model.Unchanged:
			slog.DebugContext(ctx, "File unchanged")
		}
	}

	return nil
}

func (fms *FileManagerService) fileUpdate(ctx context.Context, file *mpi.File) error {
	if file.GetFileMeta().GetSize() <= int64(fms.agentConfig.Client.Grpc.MaxFileSize) {
		return fms.file(ctx, file)
	}

	return fms.chunkedFile(ctx, file)
}

func (fms *FileManagerService) file(ctx context.Context, file *mpi.File) error {
	slog.DebugContext(ctx, "Getting file", "file", file.GetFileMeta().GetName())

	backOffCtx, backoffCancel := context.WithTimeout(ctx, fms.agentConfig.Client.Backoff.MaxElapsedTime)
	defer backoffCancel()

	getFile := func() (*mpi.GetFileResponse, error) {
		return fms.fileServiceClient.GetFile(ctx, &mpi.GetFileRequest{
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
		backoffHelpers.Context(backOffCtx, fms.agentConfig.Client.Backoff),
	)

	if getFileErr != nil {
		return fmt.Errorf("error getting file data for %s: %w", file.GetFileMeta(), getFileErr)
	}

	if writeErr := fms.fileOperator.Write(ctx, getFileResp.GetContents().GetContents(),
		file.GetFileMeta()); writeErr != nil {
		return writeErr
	}

	return fms.validateFileHash(file.GetFileMeta().GetName())
}

func (fms *FileManagerService) chunkedFile(ctx context.Context, file *mpi.File) error {
	slog.DebugContext(ctx, "Getting chunked file", "file", file.GetFileMeta().GetName())

	stream, err := fms.fileServiceClient.GetFileStream(ctx, &mpi.GetFileRequest{
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

	writeChunkedFileError := fms.fileOperator.WriteChunkedFile(ctx, file, header, stream)
	if writeChunkedFileError != nil {
		return writeChunkedFileError
	}

	return nil
}

func (fms *FileManagerService) validateFileHash(filePath string) error {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}
	fileHash := files.GenerateHash(content)

	if fileHash != fms.fileActions[filePath].File.GetFileMeta().GetHash() {
		return fmt.Errorf("error writing file, file hash does not match for file %s", filePath)
	}

	return nil
}

func (fms *FileManagerService) checkAllowedDirectory(checkFiles []*mpi.File) error {
	for _, file := range checkFiles {
		allowed := fms.agentConfig.IsDirectoryAllowed(file.GetFileMeta().GetName())
		if !allowed {
			return fmt.Errorf("file not in allowed directories %s", file.GetFileMeta().GetName())
		}
	}

	return nil
}

// DetermineFileActions compares two sets of files to determine the file action for each file. Returns a map of files
// that have changed and a map of the contents for each updated and deleted file. Key to both maps is file path
// nolint: revive,cyclop,gocognit
func (fms *FileManagerService) DetermineFileActions(
	currentFiles map[string]*mpi.File,
	modifiedFiles map[string]*model.FileCache,
) (
	map[string]*model.FileCache,
	map[string][]byte,
	error,
) {
	fms.filesMutex.Lock()
	defer fms.filesMutex.Unlock()

	fileDiff := make(map[string]*model.FileCache) // Files that have changed, key is file name
	fileContents := make(map[string][]byte)       // contents of the file, key is file name

	_, filesMap, manifestFileErr := fms.manifestFile()

	if manifestFileErr != nil {
		if errors.Is(manifestFileErr, os.ErrNotExist) {
			filesMap = currentFiles
		} else {
			return nil, nil, manifestFileErr
		}
	}
	// if file is in manifestFiles but not in modified files, file has been deleted
	// copy contents, set file action
	for fileName, manifestFile := range filesMap {
		_, exists := modifiedFiles[fileName]

		if !exists {
			// Read file contents before marking it deleted
			fileContent, readErr := os.ReadFile(fileName)
			if readErr != nil {
				return nil, nil, fmt.Errorf("error reading file %s: %w", fileName, readErr)
			}
			fileContents[fileName] = fileContent

			fileDiff[fileName] = &model.FileCache{
				File:   manifestFile,
				Action: model.Delete,
			}
		}
	}

	for _, modifiedFile := range modifiedFiles {
		fileName := modifiedFile.File.GetFileMeta().GetName()
		currentFile, ok := filesMap[modifiedFile.File.GetFileMeta().GetName()]
		// default to unchanged action
		modifiedFile.Action = model.Unchanged

		// if file is unmanaged, action is set to unchanged so file is skipped when performing actions
		if modifiedFile.File.GetUnmanaged() {
			continue
		}
		// if file doesn't exist in the current files, file has been added
		// set file action
		if _, statErr := os.Stat(modifiedFile.File.GetFileMeta().GetName()); errors.Is(statErr, os.ErrNotExist) {
			modifiedFile.Action = model.Add
			fileDiff[modifiedFile.File.GetFileMeta().GetName()] = modifiedFile

			continue
			// if file currently exists and file hash is different, file has been updated
			// copy contents, set file action
		} else if ok && modifiedFile.File.GetFileMeta().GetHash() != currentFile.GetFileMeta().GetHash() {
			fileContent, readErr := os.ReadFile(fileName)
			if readErr != nil {
				return nil, nil, fmt.Errorf("error reading file %s, error: %w", fileName, readErr)
			}
			modifiedFile.Action = model.Update
			fileContents[fileName] = fileContent
			fileDiff[modifiedFile.File.GetFileMeta().GetName()] = modifiedFile
		}
	}

	return fileDiff, fileContents, nil
}

// UpdateCurrentFilesOnDisk updates the FileManagerService currentFilesOnDisk slice which contains the files
// currently on disk
func (fms *FileManagerService) UpdateCurrentFilesOnDisk(
	ctx context.Context,
	currentFiles map[string]*mpi.File,
	referenced bool,
) error {
	fms.filesMutex.Lock()
	defer fms.filesMutex.Unlock()

	clear(fms.currentFilesOnDisk)

	for _, currentFile := range currentFiles {
		fms.currentFilesOnDisk[currentFile.GetFileMeta().GetName()] = currentFile
	}

	err := fms.UpdateManifestFile(currentFiles, referenced)
	if err != nil {
		return fmt.Errorf("failed to update manifest file: %w", err)
	}

	return nil
}

// seems to be a control flag, avoid control coupling
// nolint: revive
func (fms *FileManagerService) UpdateManifestFile(currentFiles map[string]*mpi.File, referenced bool) (err error) {
	currentManifestFiles, _, readError := fms.manifestFile()
	fms.previousManifestFiles = currentManifestFiles
	if readError != nil && !errors.Is(readError, os.ErrNotExist) {
		return fmt.Errorf("unable to read manifest file: %w", readError)
	}

	updatedFiles := make(map[string]*model.ManifestFile)

	manifestFiles := fms.convertToManifestFileMap(currentFiles, referenced)
	// During a config apply every file is set to unreferenced
	// When a new NGINX config context is detected
	// we update the files in the manifest by setting the referenced bool to true
	if currentManifestFiles != nil && referenced {
		for _, currentManifestFile := range currentManifestFiles {
			// if file from manifest file is unreferenced add it to updatedFiles map
			if !currentManifestFile.ManifestFileMeta.Referenced {
				updatedFiles[currentManifestFile.ManifestFileMeta.Name] = currentManifestFile
			}
		}
		for manifestFileName, manifestFile := range manifestFiles {
			updatedFiles[manifestFileName] = manifestFile
		}
	} else {
		updatedFiles = manifestFiles
	}

	return fms.writeManifestFile(updatedFiles)
}

func (fms *FileManagerService) writeManifestFile(updatedFiles map[string]*model.ManifestFile) (writeError error) {
	manifestJSON, err := json.MarshalIndent(updatedFiles, "", "  ")
	if err != nil {
		return fmt.Errorf("unable to marshal manifest file json: %w", err)
	}

	// 0755 allows read/execute for all, write for owner
	if err = os.MkdirAll(fms.agentConfig.ManifestDir, dirPerm); err != nil {
		return fmt.Errorf("unable to create directory %s: %w", fms.agentConfig.ManifestDir, err)
	}

	// 0600 ensures only root can read/write
	newFile, err := os.OpenFile(fms.manifestFilePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, filePerm)
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

func (fms *FileManagerService) manifestFile() (map[string]*model.ManifestFile, map[string]*mpi.File, error) {
	if _, err := os.Stat(fms.manifestFilePath); err != nil {
		return nil, nil, err
	}

	file, err := os.ReadFile(fms.manifestFilePath)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read manifest file: %w", err)
	}

	var manifestFiles map[string]*model.ManifestFile

	err = json.Unmarshal(file, &manifestFiles)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to parse manifest file: %w", err)
	}

	fileMap := fms.convertToFileMap(manifestFiles)

	return manifestFiles, fileMap, nil
}

func (fms *FileManagerService) convertToManifestFileMap(
	currentFiles map[string]*mpi.File,
	referenced bool,
) map[string]*model.ManifestFile {
	manifestFileMap := make(map[string]*model.ManifestFile)

	for name, currentFile := range currentFiles {
		if currentFile == nil || currentFile.GetFileMeta() == nil {
			continue
		}
		manifestFile := fms.convertToManifestFile(currentFile, referenced)
		manifestFileMap[name] = manifestFile
	}

	return manifestFileMap
}

func (fms *FileManagerService) convertToManifestFile(file *mpi.File, referenced bool) *model.ManifestFile {
	return &model.ManifestFile{
		ManifestFileMeta: &model.ManifestFileMeta{
			Name:       file.GetFileMeta().GetName(),
			Size:       file.GetFileMeta().GetSize(),
			Hash:       file.GetFileMeta().GetHash(),
			Referenced: referenced,
		},
	}
}

func (fms *FileManagerService) convertToFileMap(manifestFiles map[string]*model.ManifestFile) map[string]*mpi.File {
	currentFileMap := make(map[string]*mpi.File)
	for name, manifestFile := range manifestFiles {
		currentFile := fms.convertToFile(manifestFile)
		currentFileMap[name] = currentFile
	}

	return currentFileMap
}

func (fms *FileManagerService) convertToFile(manifestFile *model.ManifestFile) *mpi.File {
	return &mpi.File{
		FileMeta: &mpi.FileMeta{
			Name: manifestFile.ManifestFileMeta.Name,
			Hash: manifestFile.ManifestFileMeta.Hash,
			Size: manifestFile.ManifestFileMeta.Size,
		},
	}
}

// ConvertToMapOfFiles converts a list of files to a map of file caches (file and action) with the file name as the key
func ConvertToMapOfFileCache(convertFiles []*mpi.File) map[string]*model.FileCache {
	filesMap := make(map[string]*model.FileCache)
	for _, convertFile := range convertFiles {
		filesMap[convertFile.GetFileMeta().GetName()] = &model.FileCache{
			File: convertFile,
		}
	}

	return filesMap
}
