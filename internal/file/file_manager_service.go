// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package file

import (
	"bytes"
	"context"
	"io"
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
		Write()
	}
)

type FileManagerService struct {
	fileServiceClient mpi.FileServiceClient
	agentConfig       *config.Config
	fileOperator      fileOperator
	// TODO: Naming of these
	fileOverviewCache map[string]*mpi.FileOverview // key is instance ID
	fileContentsCache map[string][]byte            // key is file path
}

func NewFileManagerService(fileServiceClient mpi.FileServiceClient, agentConfig *config.Config) *FileManagerService {
	return &FileManagerService{
		fileServiceClient: fileServiceClient,
		agentConfig:       agentConfig,
		fileOperator:      NewFileOperator(),
		fileOverviewCache: make(map[string]*mpi.FileOverview),
		fileContentsCache: make(map[string][]byte),
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
	for _, file := range fileOverview.GetFiles() {
		action := file.GetAction()
		switch action {
		case mpi.File_FILE_ACTION_UNCHANGED, mpi.File_FILE_ACTION_UNSPECIFIED:
			break
		case mpi.File_FILE_ACTION_UPDATE, mpi.File_FILE_ACTION_ADD:
			fileResponse, getFileErr := fms.fileServiceClient.GetFile(ctx, &mpi.GetFileRequest{
				MessageMeta: &mpi.MessageMeta{
					MessageId:     uuid.NewString(),
					CorrelationId: logger.GetCorrelationID(ctx),
					Timestamp:     timestamppb.Now(),
				},
				FileMeta: file.GetFileMeta(),
			})
			// TODO: should this be more specific ? mention file ??
			if getFileErr != nil {
				return getFileErr
			}

			// not sure if this is needed ?
			fileContents[file.GetFileMeta().GetName()] = fileResponse.GetContents()
			fallthrough
		case mpi.File_FILE_ACTION_DELETE:
			readErr := fms.readFileContent(file.GetFileMeta().GetName())
			if readErr != nil {
				return readErr
			}

			// write file

			// compare file content and hash

			// publish update successful

			// keep content and cache
		}

	}

	return nil
}

func (fms *FileManagerService) readFileContent(filePath string) error {
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		// File is new and doesn't exist so no previous content to save
		return nil
	}
	file, err := os.Open(filePath)
	// TODO: should this be more specific ? mention file ??
	if err != nil {
		return err
	}

	content := bytes.NewBuffer([]byte{})
	// TODO: should this be more specific ? mention file ??
	_, copyErr := io.Copy(content, file)
	if copyErr != nil {
		return copyErr
	}

	fms.fileContentsCache[filePath] = content.Bytes()

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
		// TODO: should this be more specific ? mention file overview ??
		if err != nil {
			return nil, err
		}
		fileOverview = overview.GetOverview()
	} else {
		fileOverview = configApplyRequest.GetOverview()
	}

	return fileOverview, nil
}

// call file operator write
//func write() {
//}
//
//// compare hash from file overview with contents gotten from getFileResponse
//func verify() {
//}
//
//// clear fileOverviews and file contents for specific instance
//func clearFiles() {
//}

// Receive config apply request with file overview
// save file overview in fileOverviews map
// get files from management plane
// add contents of files to fileContents map

// write each file
// compare files content and file overview hash
// publish file update successful
// keep file contents and overview until resource plugin says apply was successful
// if it fails and rollback is needed
// compare the cache to the file overview to see if a file was added as the file will be in file overview
// but won't be in the cache as its new and won't have previous content. this means the file should be deleted
// files that were deleted should be created using the cache
// files that were updated should be written to using the cache
