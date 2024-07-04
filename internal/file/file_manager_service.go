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
	"os"
	"sync/atomic"

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

	RollbackRequiredError struct {
		Err error
	}

	NoChangeError struct{}

	fileManagerServiceInterface interface {
		UpdateOverview(ctx context.Context, instanceID string, filesToUpdate []*mpi.File) error
		ConfigApply(ctx context.Context, configApplyRequest *mpi.ConfigApplyRequest) (err error)
		Rollback(ctx context.Context, instanceID string) error
		UpdateFile(ctx context.Context, instanceID string, fileToUpdate *mpi.File) error
		ClearCache()
		SetIsConnected(isConnected bool)
	}
)

type FileManagerService struct {
	fileServiceClient mpi.FileServiceClient
	agentConfig       *config.Config
	isConnected       *atomic.Bool
	fileOperator      fileOperator
	filesCache        map[string]*mpi.File // key is file path
	fileContentsCache map[string][]byte    // key is file path
}

func NewFileManagerService(fileServiceClient mpi.FileServiceClient, agentConfig *config.Config) *FileManagerService {
	isConnected := &atomic.Bool{}
	isConnected.Store(false)

	return &FileManagerService{
		fileServiceClient: fileServiceClient,
		agentConfig:       agentConfig,
		fileOperator:      NewFileOperator(),
		filesCache:        make(map[string]*mpi.File),
		fileContentsCache: make(map[string][]byte),
		isConnected:       isConnected,
	}
}

func (r *RollbackRequiredError) Error() string {
	return fmt.Sprintf("rollback required: %v", r.Err)
}

func (r *NoChangeError) Error() string {
	return "no change required"
}

func (fms *FileManagerService) UpdateOverview(
	ctx context.Context,
	instanceID string,
	filesToUpdate []*mpi.File,
) error {
	slog.InfoContext(ctx, "Updating file overview", "instance_id", instanceID)
	correlationID := logger.GetCorrelationID(ctx)

	request := &mpi.UpdateOverviewRequest{
		MessageMeta: &mpi.MessageMeta{
			MessageId:     uuid.NewString(),
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

	backOffCtx, backoffCancel := context.WithTimeout(ctx, fms.agentConfig.Common.MaxElapsedTime)
	defer backoffCancel()

	sendUpdateOverview := func() (*mpi.UpdateOverviewResponse, error) {
		slog.DebugContext(ctx, "Sending update overview request", "request", request)
		if fms.fileServiceClient == nil {
			return nil, errors.New("file service client is not initialized")
		}

		if !fms.isConnected.Load() {
			return nil, errors.New("CreateConnection rpc has not being called yet")
		}

		response, updateError := fms.fileServiceClient.UpdateOverview(ctx, request)

		validatedError := grpc.ValidateGrpcError(updateError)

		if validatedError != nil {
			slog.ErrorContext(ctx, "Failed to send update overview", "error", validatedError)

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

	slog.DebugContext(ctx, "UpdateOverview response", "response", response)

	return err
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

	request := &mpi.UpdateFileRequest{
		File: fileToUpdate,
		Contents: &mpi.FileContents{
			Contents: contents,
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
) (err error) {
	fileOverview := configApplyRequest.GetOverview()

	if fileOverview == nil {
		return fmt.Errorf("fileOverview is nil")
	}

	allowedErr := fms.checkAllowedDirectory(fileOverview.GetFiles())
	if allowedErr != nil {
		return allowedErr
	}

	diffFiles, fileContent, compareErr := files.CompareFileHash(fileOverview)
	if compareErr != nil {
		return compareErr
	}

	if len(diffFiles) == 0 {
		return &NoChangeError{}
	}

	fms.fileContentsCache = fileContent
	fms.filesCache = diffFiles

	fileErr := fms.executeFileActions(ctx)
	if fileErr != nil {
		return &RollbackRequiredError{Err: fileErr}
	}

	return nil
}

func (fms *FileManagerService) ClearCache() {
	clear(fms.fileContentsCache)
	clear(fms.filesCache)
}

func (fms *FileManagerService) Rollback(ctx context.Context, instanceID string) error {
	slog.InfoContext(ctx, "Rolling back config for instance", "instanceid", instanceID)
	for _, file := range fms.filesCache {
		switch file.GetAction() {
		case mpi.File_FILE_ACTION_ADD:
			if err := os.Remove(file.GetFileMeta().GetName()); err != nil && !os.IsNotExist(err) {
				return fmt.Errorf("error deleting file: %s error: %w", file.GetFileMeta().GetName(), err)
			}

			continue
		case mpi.File_FILE_ACTION_DELETE, mpi.File_FILE_ACTION_UPDATE:
			content := fms.fileContentsCache[file.GetFileMeta().GetName()]

			err := fms.fileOperator.Write(ctx, content, file.GetFileMeta())
			if err != nil {
				return err
			}

		case mpi.File_FILE_ACTION_UNSPECIFIED, mpi.File_FILE_ACTION_UNCHANGED:
			fallthrough
		default:
			slog.DebugContext(ctx, "File Action not implemented")
		}
	}

	return nil
}

func (fms *FileManagerService) executeFileActions(ctx context.Context) error {
	for _, file := range fms.filesCache {
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

	if fileHash != fms.filesCache[filePath].GetFileMeta().GetHash() {
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
