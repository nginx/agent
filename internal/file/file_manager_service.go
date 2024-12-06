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

	"github.com/nginx/agent/v3/internal/model"

	"github.com/cenkalti/backoff/v4"
	"github.com/google/uuid"
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
		UpdateCurrentFilesOnDisk(updateFiles map[string]*mpi.File)
		DetermineFileActions(currentFiles, modifiedFiles map[string]*mpi.File) (map[string]*mpi.File,
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
	fileActions map[string]*mpi.File // key is file path
	// map of the contents of files which have been updated or deleted during config apply, used during rollback
	rollbackFileContents map[string][]byte // key is file path
	// map of the files currently on disk, used to determine the file action during config apply
	currentFilesOnDisk map[string]*mpi.File // key is file path
	filesMutex         sync.RWMutex
}

func NewFileManagerService(fileServiceClient mpi.FileServiceClient, agentConfig *config.Config) *FileManagerService {
	isConnected := &atomic.Bool{}
	isConnected.Store(false)

	return &FileManagerService{
		fileServiceClient:    fileServiceClient,
		agentConfig:          agentConfig,
		fileOperator:         NewFileOperator(),
		fileActions:          make(map[string]*mpi.File),
		rollbackFileContents: make(map[string][]byte),
		currentFilesOnDisk:   make(map[string]*mpi.File),
		isConnected:          isConnected,
	}
}

func (fms *FileManagerService) UpdateOverview(
	ctx context.Context,
	instanceID string,
	filesToUpdate []*mpi.File,
	iteration int,
) error {
	const maxAttempts = 5
	correlationID := logger.GetCorrelationID(ctx)
	var requestCorrelationID slog.Attr

	// error case for the UpdateOverview attempts
	if iteration > maxAttempts {
		return errors.New("too many UpdateOverview attempts")
	}

	newCtx, correlationID := fms.setupIdentifiers(ctx, iteration)

	request := &mpi.UpdateOverviewRequest{
		MessageMeta: &mpi.MessageMeta{
			MessageId:     uuid.NewString(),
			CorrelationId: requestCorrelationID.Value.String(),
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

	backOffCtx, backoffCancel := context.WithTimeout(newCtx, fms.agentConfig.Common.MaxElapsedTime)
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
		backoffHelpers.Context(backOffCtx, fms.agentConfig.Common),
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
	slog.Debug("iteration value", "iteration", iteration)

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
			MessageId:     uuid.NewString(),
			CorrelationId: correlationID,
			Timestamp:     timestamppb.Now(),
		},
	}

	backOffCtx, backoffCancel := context.WithTimeout(ctx, fms.agentConfig.Common.MaxElapsedTime)
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

	response, err := backoff.RetryWithData(sendUpdateFile, backoffHelpers.Context(backOffCtx, fms.agentConfig.Common))
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
		files.ConvertToMapOfFiles(fileOverview.GetFiles()))

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
	fms.UpdateCurrentFilesOnDisk(files.ConvertToMapOfFiles(fileOverview.GetFiles()))

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
	for _, file := range fms.fileActions {
		switch file.GetAction() {
		case mpi.File_FILE_ACTION_ADD:
			if err := os.Remove(file.GetFileMeta().GetName()); err != nil && !os.IsNotExist(err) {
				return fmt.Errorf("error deleting file: %s error: %w", file.GetFileMeta().GetName(), err)
			}

			// currentFilesOnDisk needs to be updated after rollback action is performed
			delete(fms.currentFilesOnDisk, file.GetFileMeta().GetName())

			continue
		case mpi.File_FILE_ACTION_DELETE, mpi.File_FILE_ACTION_UPDATE:
			content := fms.rollbackFileContents[file.GetFileMeta().GetName()]

			err := fms.fileOperator.Write(ctx, content, file.GetFileMeta())
			if err != nil {
				return err
			}

			// currentFilesOnDisk needs to be updated after rollback action is performed
			file.GetFileMeta().Hash = files.GenerateHash(content)
			fms.currentFilesOnDisk[file.GetFileMeta().GetName()] = file
		case mpi.File_FILE_ACTION_UNSPECIFIED, mpi.File_FILE_ACTION_UNCHANGED:
			fallthrough
		default:
			slog.DebugContext(ctx, "File Action not implemented")
		}
	}

	return nil
}

func (fms *FileManagerService) executeFileActions(ctx context.Context) error {
	for _, file := range fms.fileActions {
		switch file.GetAction() {
		case mpi.File_FILE_ACTION_DELETE:
			if err := os.Remove(file.GetFileMeta().GetName()); err != nil && !os.IsNotExist(err) {
				return fmt.Errorf("error deleting file: %s error: %w", file.GetFileMeta().GetName(), err)
			}

			continue
		case mpi.File_FILE_ACTION_ADD, mpi.File_FILE_ACTION_UPDATE:
			updateErr := fms.fileUpdate(ctx, file)
			if updateErr != nil {
				return updateErr
			}
		case mpi.File_FILE_ACTION_UNSPECIFIED, mpi.File_FILE_ACTION_UNCHANGED:
			fallthrough
		default:
			slog.DebugContext(ctx, "File Action not implemented", "action", file.GetAction())
		}
	}

	return nil
}

func (fms *FileManagerService) fileUpdate(ctx context.Context, file *mpi.File) error {
	backOffCtx, backoffCancel := context.WithTimeout(ctx, fms.agentConfig.Common.MaxElapsedTime)
	defer backoffCancel()

	getFile := func() (*mpi.GetFileResponse, error) {
		return fms.fileServiceClient.GetFile(ctx, &mpi.GetFileRequest{
			MessageMeta: &mpi.MessageMeta{
				MessageId:     uuid.NewString(),
				CorrelationId: logger.GetCorrelationID(ctx),
				Timestamp:     timestamppb.Now(),
			},
			FileMeta: file.GetFileMeta(),
		})
	}

	getFileResp, getFileErr := backoff.RetryWithData(
		getFile,
		backoffHelpers.Context(backOffCtx, fms.agentConfig.Common),
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

	if fileHash != fms.fileActions[filePath].GetFileMeta().GetHash() {
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
func (fms *FileManagerService) DetermineFileActions(currentFiles, modifiedFiles map[string]*mpi.File) (
	map[string]*mpi.File, map[string][]byte, error,
) {
	fms.filesMutex.Lock()
	defer fms.filesMutex.Unlock()
	// Go doesn't allow address of numeric constant
	addAction := mpi.File_FILE_ACTION_ADD
	updateAction := mpi.File_FILE_ACTION_UPDATE
	deleteAction := mpi.File_FILE_ACTION_DELETE
	unchangedAction := mpi.File_FILE_ACTION_UNCHANGED

	fileDiff := make(map[string]*mpi.File)  // Files that have changed, key is file name
	fileContents := make(map[string][]byte) // contents of the file, key is file name

	// if file is in currentFiles but not in modified files, file has been deleted
	// copy contents, set file action
	for _, currentFile := range currentFiles {
		fileName := currentFile.GetFileMeta().GetName()
		_, ok := modifiedFiles[fileName]

		if !ok {
			fileContent, readErr := os.ReadFile(fileName)
			if readErr != nil {
				return nil, nil, fmt.Errorf("error reading file %s, error: %w", fileName, readErr)
			}
			fileContents[fileName] = fileContent
			currentFile.Action = &deleteAction
			fileDiff[currentFile.GetFileMeta().GetName()] = currentFile
		}
	}

	for _, file := range modifiedFiles {
		fileName := file.GetFileMeta().GetName()
		currentFile, ok := currentFiles[file.GetFileMeta().GetName()]
		// default to unchanged action
		file.Action = &unchangedAction
		// if file doesn't exist in the current files, file has been added
		// set file action
		if !ok {
			file.Action = &addAction
			fileDiff[file.GetFileMeta().GetName()] = file
			// if file currently exists and file hash is different, file has been updated
			// copy contents, set file action
		} else if file.GetFileMeta().GetHash() != currentFile.GetFileMeta().GetHash() {
			fileContent, readErr := os.ReadFile(fileName)
			if readErr != nil {
				return nil, nil, fmt.Errorf("error reading file %s, error: %w", fileName, readErr)
			}
			file.Action = &updateAction
			fileContents[fileName] = fileContent
			fileDiff[file.GetFileMeta().GetName()] = file
		}
	}

	return fileDiff, fileContents, nil
}

// UpdateCurrentFilesOnDisk updates the FileManagerService currentFilesOnDisk slice which contains the files
// currently on disk
func (fms *FileManagerService) UpdateCurrentFilesOnDisk(currentFiles map[string]*mpi.File) {
	fms.filesMutex.Lock()
	defer fms.filesMutex.Unlock()

	clear(fms.currentFilesOnDisk)

	for _, file := range currentFiles {
		fms.currentFilesOnDisk[file.GetFileMeta().GetName()] = file
	}
}
