// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package file

import (
	"context"
	"errors"
	"log/slog"
	"os"

	"github.com/cenkalti/backoff/v4"
	"github.com/google/uuid"
	mpi "github.com/nginx/agent/v3/api/grpc/mpi/v1"
	"github.com/nginx/agent/v3/files"
	"github.com/nginx/agent/v3/internal/config"
	"github.com/nginx/agent/v3/internal/grpc"
	"github.com/nginx/agent/v3/internal/logger"
	"google.golang.org/protobuf/types/known/timestamppb"

	backoffHelpers "github.com/nginx/agent/v3/internal/backoff"
)

type (
	fileOperator interface {
		Write(ctx context.Context, fileContent []byte, file *mpi.FileMeta) error
		ReadFileContents(files []*mpi.File) (filesContents map[string][]byte, err error)
	}
)

type FileManagerService struct {
	fileServiceClient mpi.FileServiceClient
	agentConfig       *config.Config
	fileOperator      fileOperator
	fileOverviewCache map[string]*mpi.FileOverview // key is instance ID
	fileContentsCache map[string][]byte            // key is file path
	nginxConfigFiles  map[string][]*mpi.File       // key is instance ID, this is the list of files from the nginxConfigContext
}

func NewFileManagerService(fileServiceClient mpi.FileServiceClient, agentConfig *config.Config) *FileManagerService {
	return &FileManagerService{
		fileServiceClient: fileServiceClient,
		agentConfig:       agentConfig,
		fileOperator:      NewFileOperator(),
		fileOverviewCache: make(map[string]*mpi.FileOverview),
		fileContentsCache: make(map[string][]byte),
		nginxConfigFiles:  make(map[string][]*mpi.File),
	}
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

func (fms *FileManagerService) ConfigApply(ctx context.Context, configApplyRequest *mpi.ConfigApplyRequest) error {
	fileContents := make(map[string]*mpi.FileContents) // key is file path

	fileOverview, err := fms.fileOverview(ctx, configApplyRequest)
	if err != nil {
		return err
	}
	fms.fileOverviewCache[configApplyRequest.GetConfigVersion().GetInstanceId()] = fileOverview

	contents, readErr := fms.fileOperator.ReadFileContents(fileOverview.GetFiles())
	fms.fileContentsCache = contents
	if readErr != nil {
		return readErr
	}

	// TODO:
	// Need to check if file is being updated, added or deleted
	// Updated - compare hash
	// Added - does file exist
	// Deleted is file in nginxConfigFiles and not in FileOverview

	for _, file := range fileOverview.GetFiles() {
		fileResponse, getFileErr := fms.fileServiceClient.GetFile(ctx, &mpi.GetFileRequest{
			MessageMeta: &mpi.MessageMeta{
				MessageId:     uuid.NewString(),
				CorrelationId: logger.GetCorrelationID(ctx),
				Timestamp:     timestamppb.Now(),
			},
			FileMeta: file.GetFileMeta(),
		})

		if getFileErr != nil {
			return getFileErr
		}

		fileContents[file.GetFileMeta().GetName()] = fileResponse.GetContents()

		// TODO:
		// write file
		// compare file content and hash
		// publish update successful
		// keep content and cache

	}

	return nil
}

func (fms *FileManagerService) fileOverview(ctx context.Context,
	configApplyRequest *mpi.ConfigApplyRequest,
) (*mpi.FileOverview, error) {
	var fileOverview *mpi.FileOverview
	if configApplyRequest.GetOverview() == nil {
		overview, err := fms.fileServiceClient.GetOverview(ctx, &mpi.GetOverviewRequest{
			MessageMeta: &mpi.MessageMeta{
				MessageId:     uuid.NewString(),
				CorrelationId: logger.GetCorrelationID(ctx),
				Timestamp:     timestamppb.Now(),
			},
			ConfigVersion: configApplyRequest.GetConfigVersion(),
		})
		if err != nil {
			return nil, err
		}
		fileOverview = overview.GetOverview()
	} else {
		fileOverview = configApplyRequest.GetOverview()
	}

	return fileOverview, nil
}

// TODO: Naming
func (fms *FileManagerService) updateConfigFiles(instanceID string, files []*mpi.File) {
	fms.nginxConfigFiles[instanceID] = files
}
