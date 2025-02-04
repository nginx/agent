// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package file

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"maps"
	"os"
	"slices"
	"sync"
	"sync/atomic"

	"github.com/nginx/agent/v3/internal/datasource/proto"
	"github.com/nginx/agent/v3/internal/model"

	"github.com/cenkalti/backoff/v4"

	mpi "github.com/nginx/agent/v3/api/grpc/mpi/v1"
	"github.com/nginx/agent/v3/internal/config"
	"github.com/nginx/agent/v3/internal/grpc"
	"github.com/nginx/agent/v3/internal/logger"
	"github.com/nginx/agent/v3/pkg/files"
	"google.golang.org/protobuf/types/known/timestamppb"

	backoffHelpers "github.com/nginx/agent/v3/internal/backoff"
)

//go:generate go run github.com/maxbrunsfeld/counterfeiter/v6@v6.8.1 -generate
//counterfeiter:generate . fileOperator

//go:generate go run github.com/maxbrunsfeld/counterfeiter/v6@v6.8.1 -generate
//counterfeiter:generate . fileManagerServiceInterface

const (
	maxAttempts     = 5
	addAction       = "add"
	updateAction    = "update"
	deleteAction    = "delete"
	unchangedAction = "unchanged"
)

type (
	fileOperator interface {
		Write(ctx context.Context, fileContent []byte, file *mpi.FileMeta) error
	}

	fileManagerServiceInterface interface {
		UpdateOverview(ctx context.Context, instanceID string, filesToUpdate []*mpi.File, iteration int) error
		ConfigApply(ctx context.Context, configApplyRequest *mpi.ConfigApplyRequest) (writeStatus model.WriteStatus,
			err error)
		Rollback(ctx context.Context, instanceID string) error
		UpdateFile(ctx context.Context, instanceID string, fileToUpdate *mpi.File) error
		ClearCache()
		UpdateCurrentFilesOnDisk(updateFiles map[string]model.FileCache)
		DetermineFileActions(currentFiles, modifiedFiles map[string]model.FileCache) (map[string]model.FileCache,
			map[string][]byte, error)
		SetIsConnected(isConnected bool)
	}
)

type FileManagerService struct {
	fileServiceClient mpi.FileServiceClient
	agentConfig       *config.Config
	isConnected       *atomic.Bool
	fileOperator      fileOperator
	// map of files and the actions performed on them during config apply
	fileActions map[string]model.FileCache // key is File path
	// map of the contents of files which have been updated or deleted during config apply, used during rollback
	rollbackFileContents map[string][]byte // key is File path
	// map of the files currently on disk, used to determine the File Action during config apply
	currentFilesOnDisk map[string]model.FileCache // key is File path
	filesMutex         sync.RWMutex
}

func NewFileManagerService(fileServiceClient mpi.FileServiceClient, agentConfig *config.Config) *FileManagerService {
	isConnected := &atomic.Bool{}
	isConnected.Store(false)

	return &FileManagerService{
		fileServiceClient:    fileServiceClient,
		agentConfig:          agentConfig,
		fileOperator:         NewFileOperator(),
		fileActions:          make(map[string]model.FileCache),
		rollbackFileContents: make(map[string][]byte),
		currentFilesOnDisk:   make(map[string]model.FileCache),
		isConnected:          isConnected,
	}
}

func (fms *FileManagerService) UpdateOverview(
	ctx context.Context,
	instanceID string,
	filesToUpdate []*mpi.File,
	iteration int,
) error {
	correlationID := logger.GetCorrelationID(ctx)

	// error case for the UpdateOverview attempts
	if iteration > maxAttempts {
		return errors.New("too many UpdateOverview attempts")
	}

	newCtx, correlationID := fms.setupIdentifiers(ctx, iteration)

	request := &mpi.UpdateOverviewRequest{
		MessageMeta: &mpi.MessageMeta{
			MessageId:     proto.GenerateMessageID(),
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
		slog.DebugContext(newCtx, "Sending update overview request", "request", request,
			"parent_correlation_id", correlationID)
		if fms.fileServiceClient == nil {
			return nil, errors.New("file service client is not initialized")
		}

		if !fms.isConnected.Load() {
			return nil, errors.New("CreateConnection rpc has not being called yet")
		}

		response, updateError := fms.fileServiceClient.UpdateOverview(newCtx, request)

		validatedError := grpc.ValidateGrpcError(updateError)

		if validatedError != nil {
			slog.ErrorContext(newCtx, "Failed to send update overview", "error", validatedError)

			return nil, validatedError
		}

		return response, nil
	}

	response, err := backoff.RetryWithData(
		sendUpdateOverview,
		backoffHelpers.Context(backOffCtx, fms.agentConfig.Client.Backoff),
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
	correlationID := logger.GetCorrelationID(ctx)
	var requestCorrelationID slog.Attr

	if iteration == 0 {
		requestCorrelationID = logger.GenerateCorrelationID()
	} else {
		requestCorrelationID = logger.GetCorrelationIDAttr(ctx)
	}

	newCtx := context.WithValue(ctx, logger.CorrelationIDContextKey, requestCorrelationID)
	slog.InfoContext(newCtx, "Updating file overview", "instance_id", logger.GetCorrelationIDAttr(ctx),
		"parent_correlation_id", correlationID)

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
	slog.Debug("Updating file overview", "attempt_number", iteration)

	return fms.UpdateOverview(ctx, instanceID, diffFiles, iteration)
}

func (fms *FileManagerService) UpdateFile(
	ctx context.Context,
	instanceID string,
	fileToUpdate *mpi.File,
) error {
	slog.InfoContext(ctx, "Updating file", "instance_id", instanceID, "file_name", fileToUpdate.GetFileMeta().GetName())
	contents, err := os.ReadFile(fileToUpdate.GetFileMeta().GetName())
	if err != nil {
		return err
	}

	correlationID := logger.GetCorrelationID(ctx)

	request := &mpi.UpdateFileRequest{
		File: fileToUpdate,
		Contents: &mpi.FileContents{
			Contents: contents,
		},
		MessageMeta: &mpi.MessageMeta{
			MessageId:     proto.GenerateMessageID(),
			CorrelationId: correlationID,
			Timestamp:     timestamppb.Now(),
		},
	}

	backOffCtx, backoffCancel := context.WithTimeout(ctx, fms.agentConfig.Client.Backoff.MaxElapsedTime)
	defer backoffCancel()

	sendUpdateFile := func() (*mpi.UpdateFileResponse, error) {
		slog.DebugContext(ctx, "Sending update file request", "request", request)
		if fms.fileServiceClient == nil {
			return nil, errors.New("file service client is not initialized")
		}

		if !fms.isConnected.Load() {
			return nil, errors.New("CreateConnection rpc has not being called yet")
		}

		response, updateError := fms.fileServiceClient.UpdateFile(ctx, request)

		validatedError := grpc.ValidateGrpcError(updateError)

		if validatedError != nil {
			slog.ErrorContext(ctx, "Failed to send update file", "error", validatedError)

			return nil, validatedError
		}

		return response, nil
	}

	response, err := backoff.RetryWithData(sendUpdateFile, backoffHelpers.Context(backOffCtx,
		fms.agentConfig.Client.Backoff))
	if err != nil {
		return err
	}

	slog.DebugContext(ctx, "UpdateFile response", "response", response)

	return err
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
		files.ConvertToMapOfFileCache(fileOverview.GetFiles()))

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

	// Update map of current files on disk
	fms.UpdateCurrentFilesOnDisk(files.ConvertToMapOfFileCache(fileOverview.GetFiles()))

	return model.OK, nil
}

func (fms *FileManagerService) ClearCache() {
	clear(fms.rollbackFileContents)
	clear(fms.fileActions)
}

func (fms *FileManagerService) Rollback(ctx context.Context, instanceID string) error {
	slog.InfoContext(ctx, "Rolling back config for instance", "instanceid", instanceID)
	fms.filesMutex.Lock()
	defer fms.filesMutex.Unlock()
	for _, fileAction := range fms.fileActions {
		switch fileAction.Action {
		case addAction:
			if err := os.Remove(fileAction.File.GetFileMeta().GetName()); err != nil && !os.IsNotExist(err) {
				return fmt.Errorf("error deleting file: %s error: %w", fileAction.File.GetFileMeta().GetName(), err)
			}

			// currentFilesOnDisk needs to be updated after rollback action is performed
			delete(fms.currentFilesOnDisk, fileAction.File.GetFileMeta().GetName())

			continue
		case deleteAction, updateAction:
			content := fms.rollbackFileContents[fileAction.File.GetFileMeta().GetName()]

			err := fms.fileOperator.Write(ctx, content, fileAction.File.GetFileMeta())
			if err != nil {
				return err
			}

			// currentFilesOnDisk needs to be updated after rollback action is performed
			fileAction.File.GetFileMeta().Hash = files.GenerateHash(content)
			fms.currentFilesOnDisk[fileAction.File.GetFileMeta().GetName()] = fileAction
		case "", unchangedAction:
			fallthrough
		default:
			slog.DebugContext(ctx, "File Action not implemented")
		}
	}

	return nil
}

func (fms *FileManagerService) executeFileActions(ctx context.Context) error {
	for _, fileAction := range fms.fileActions {
		switch fileAction.Action {
		case deleteAction:
			if err := os.Remove(fileAction.File.GetFileMeta().GetName()); err != nil && !os.IsNotExist(err) {
				return fmt.Errorf("error deleting file: %s error: %w", fileAction.File.GetFileMeta().GetName(), err)
			}

			continue
		case addAction, updateAction:
			updateErr := fms.fileUpdate(ctx, fileAction.File)
			if updateErr != nil {
				return updateErr
			}
		case "", unchangedAction:
			fallthrough
		default:
			slog.DebugContext(ctx, "File Action not implemented", "action", fileAction.Action)
		}
	}

	return nil
}

func (fms *FileManagerService) fileUpdate(ctx context.Context, file *mpi.File) error {
	backOffCtx, backoffCancel := context.WithTimeout(ctx, fms.agentConfig.Client.Backoff.MaxElapsedTime)
	defer backoffCancel()

	getFile := func() (*mpi.GetFileResponse, error) {
		return fms.fileServiceClient.GetFile(ctx, &mpi.GetFileRequest{
			MessageMeta: &mpi.MessageMeta{
				MessageId:     proto.GenerateMessageID(),
				CorrelationId: logger.GetCorrelationID(ctx),
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

	validateErr := fms.validateFileHash(file.GetFileMeta().GetName())

	return validateErr
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
// nolint: revive
func (fms *FileManagerService) DetermineFileActions(currentFiles, modifiedFiles map[string]model.FileCache) (
	map[string]model.FileCache, map[string][]byte, error,
) {
	fms.filesMutex.Lock()
	defer fms.filesMutex.Unlock()

	fileDiff := make(map[string]model.FileCache) // Files that have changed, key is file name
	fileContents := make(map[string][]byte)      // contents of the file, key is file name

	// if file is in currentFiles but not in modified files, file has been deleted
	// copy contents, set file action
	for _, currentFile := range currentFiles {
		fileName := currentFile.File.GetFileMeta().GetName()
		_, ok := modifiedFiles[fileName]

		if !ok {
			fileContent, readErr := os.ReadFile(fileName)
			if readErr != nil {
				return nil, nil, fmt.Errorf("error reading file %s, error: %w", fileName, readErr)
			}
			fileContents[fileName] = fileContent
			currentFile.Action = deleteAction
			fileDiff[currentFile.File.GetFileMeta().GetName()] = currentFile
		}
	}

	for _, modifiedFile := range modifiedFiles {
		fileName := modifiedFile.File.GetFileMeta().GetName()
		currentFile, ok := currentFiles[modifiedFile.File.GetFileMeta().GetName()]
		// default to unchanged action
		modifiedFile.Action = unchangedAction

		// if file is unmanaged, action is set to unchanged so file is skipped when performing actions
		if modifiedFile.File.GetUnmanaged() {
			continue
		}
		// if file doesn't exist in the current files, file has been added
		// set file action
		if !ok {
			modifiedFile.Action = addAction
			fileDiff[modifiedFile.File.GetFileMeta().GetName()] = modifiedFile
			// if file currently exists and file hash is different, file has been updated
			// copy contents, set file action
		} else if modifiedFile.File.GetFileMeta().GetHash() != currentFile.File.GetFileMeta().GetHash() {
			fileContent, readErr := os.ReadFile(fileName)
			if readErr != nil {
				return nil, nil, fmt.Errorf("error reading file %s, error: %w", fileName, readErr)
			}
			modifiedFile.Action = updateAction
			fileContents[fileName] = fileContent
			fileDiff[modifiedFile.File.GetFileMeta().GetName()] = modifiedFile
		}
	}

	return fileDiff, fileContents, nil
}

// UpdateCurrentFilesOnDisk updates the FileManagerService currentFilesOnDisk slice which contains the files
// currently on disk
func (fms *FileManagerService) UpdateCurrentFilesOnDisk(currentFiles map[string]model.FileCache) {
	fms.filesMutex.Lock()
	defer fms.filesMutex.Unlock()

	clear(fms.currentFilesOnDisk)

	for _, currentFile := range currentFiles {
		fms.currentFilesOnDisk[currentFile.File.GetFileMeta().GetName()] = currentFile
	}
}
