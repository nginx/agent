// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package grpc

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"

	"github.com/google/uuid"
	"github.com/nginx/agent/v3/api/grpc/mpi/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const defaultFilePermissions = 0o644

type FileService struct {
	v1.UnimplementedFileServiceServer
	files            []*v1.File // Key is the config version UID
	versionDirectory string     // Key is the version directory name
	configDirectory  string
	requestChan      chan *v1.ManagementPlaneRequest
}

func NewFileService(configDirectory string, requestChan chan *v1.ManagementPlaneRequest) (*FileService, error) {
	files := []*v1.File{}
	versionDirectory := ""

	return &FileService{
		configDirectory:  configDirectory,
		files:            files,
		versionDirectory: versionDirectory,
		requestChan:      requestChan,
	}, nil
}

func (mgs *FileService) GetOverview(
	_ context.Context,
	request *v1.GetOverviewRequest,
) (*v1.GetOverviewResponse, error) {
	configVersion := request.GetConfigVersion()

	slog.Info("Getting overview", "config_version", configVersion)

	if mgs.files == nil {
		slog.Error("Config version not found", "config_version", configVersion)
		return nil, status.Errorf(codes.NotFound, "Config version not found")
	}

	return &v1.GetOverviewResponse{
		Overview: &v1.FileOverview{
			ConfigVersion: configVersion,
			Files:         mgs.files,
		},
	}, nil
}

// nolint: unparam
func (mgs *FileService) UpdateOverview(
	_ context.Context,
	request *v1.UpdateOverviewRequest,
) (*v1.UpdateOverviewResponse, error) {
	overview := request.GetOverview()
	version := overview.GetConfigVersion().GetVersion()

	slog.Info("Updating overview", "version", version)

	mgs.files = overview.GetFiles()
	mgs.versionDirectory = fmt.Sprintf("%s/%s", mgs.configDirectory, overview.GetConfigVersion().GetInstanceId())

	slog.Info("config Dir", "", mgs.versionDirectory)

	configUploadRequest := &v1.ManagementPlaneRequest{
		MessageMeta: &v1.MessageMeta{
			MessageId:     uuid.NewString(),
			CorrelationId: request.GetMessageMeta().GetCorrelationId(),
			Timestamp:     timestamppb.Now(),
		},
		Request: &v1.ManagementPlaneRequest_ConfigUploadRequest{
			ConfigUploadRequest: &v1.ConfigUploadRequest{
				Overview:   request.GetOverview(),
			},
		},
	}
	mgs.requestChan <- configUploadRequest

	return &v1.UpdateOverviewResponse{}, nil
}

func (mgs *FileService) GetFile(
	_ context.Context,
	request *v1.GetFileRequest,
) (*v1.GetFileResponse, error) {
	fileName := request.GetFileMeta().GetName()
	fileHash := request.GetFileMeta().GetHash()

	slog.Info("Getting file", "name", fileName, "hash", fileHash)

	if mgs.versionDirectory == "" {
		slog.Error("File not found", "file_name", fileName)
		return nil, status.Errorf(codes.NotFound, "File not found")
	}

	fullFilePath := filepath.Join(mgs.versionDirectory, fileName)

	bytes, err := os.ReadFile(fullFilePath)
	if err != nil {
		slog.Error("Failed to get file contents", "full_file_path", fullFilePath, "error", err)
		return nil, status.Errorf(codes.Internal, "Failed to get file contents")
	}

	return &v1.GetFileResponse{
		Contents: &v1.FileContents{
			Contents: bytes,
		},
	}, nil
}

func (mgs *FileService) UpdateFile(
	_ context.Context,
	request *v1.UpdateFileRequest,
) (*v1.UpdateFileResponse, error) {
	fileContents := request.GetContents().GetContents()
	fileAction := request.GetFile().GetAction()
	fileMeta := request.GetFile().GetFileMeta()
	fileName := fileMeta.GetName()
	fileHash := fileMeta.GetHash()
	filePermissions := fileMeta.GetPermissions()

	slog.Info("Updating file", "name", fileName, "hash", fileHash)

	fullFilePath := filepath.Join(mgs.configDirectory, mgs.versionDirectory, fileName)

	err := performFileAction(fileAction, fileContents, fullFilePath, filePermissions)
	if err != nil {
		return nil, err
	}

	return &v1.UpdateFileResponse{
		FileMeta: fileMeta,
	}, nil
}

func performFileAction(fileAction v1.File_FileAction, fileContents []byte, fullFilePath, filePermissions string) error {
	switch fileAction {
	case v1.File_FILE_ACTION_ADD, v1.File_FILE_ACTION_UPDATE:
		// Ensure if file doesn't exist that directories are created before creating the file
		if _, err := os.Stat(fullFilePath); os.IsNotExist(err) {
			statErr := os.MkdirAll(filepath.Dir(fullFilePath), os.ModePerm)
			if statErr != nil {
				slog.Info("Failed to create/update file", "full_file_path", fullFilePath, "error", statErr)
				return status.Errorf(codes.Internal, "Failed to create/update file")
			}
		}

		err := os.WriteFile(fullFilePath, fileContents, getFileMode(filePermissions))
		if err != nil {
			slog.Info("Failed to create/update file", "full_file_path", fullFilePath, "error", err)
			return status.Errorf(codes.Internal, "Failed to create/update file")
		}
	case v1.File_FILE_ACTION_DELETE:
		err := os.Remove(fullFilePath)
		if err != nil {
			slog.Info("Failed to delete file", "full_file_path", fullFilePath, "error", err)
			return status.Errorf(codes.Internal, "Failed to delete file")
		}
	case v1.File_FILE_ACTION_UNSPECIFIED:
		slog.Info("Nothing to update, file action is unspecified", "full_file_path", fullFilePath)
	case v1.File_FILE_ACTION_UNCHANGED:
		slog.Info("Nothing to update, file action is unchanged", "full_file_path", fullFilePath)
	default:
		slog.Info("Nothing to update, unknown file action", "full_file_path", fullFilePath)
	}

	return nil
}

func getFileMode(mode string) os.FileMode {
	result, err := strconv.ParseInt(mode, 8, 32)
	if err != nil {
		return os.FileMode(defaultFilePermissions)
	}

	return os.FileMode(result)
}
