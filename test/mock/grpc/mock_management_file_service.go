// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package grpc

import (
	"context"
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
	instanceFiles   map[string][]*v1.File
	requestChan     chan *v1.ManagementPlaneRequest
	configDirectory string
}

func NewFileService(configDirectory string, requestChan chan *v1.ManagementPlaneRequest) *FileService {
	return &FileService{
		configDirectory: configDirectory,
		instanceFiles:   make(map[string][]*v1.File),
		requestChan:     requestChan,
	}
}

func (mgs *FileService) GetOverview(
	_ context.Context,
	request *v1.GetOverviewRequest,
) (*v1.GetOverviewResponse, error) {
	configVersion := request.GetConfigVersion()

	slog.Info("Getting overview", "config_version", configVersion)

	if _, ok := mgs.instanceFiles[request.GetConfigVersion().GetInstanceId()]; !ok {
		slog.Error("Config version not found", "config_version", configVersion)
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
	_ context.Context,
	request *v1.UpdateOverviewRequest,
) (*v1.UpdateOverviewResponse, error) {
	overview := request.GetOverview()
	version := overview.GetConfigVersion().GetVersion()

	slog.Info("Updating overview", "version", version)

	mgs.instanceFiles[overview.GetConfigVersion().GetInstanceId()] = overview.GetFiles()

	configUploadRequest := &v1.ManagementPlaneRequest{
		MessageMeta: &v1.MessageMeta{
			MessageId:     uuid.NewString(),
			CorrelationId: request.GetMessageMeta().GetCorrelationId(),
			Timestamp:     timestamppb.Now(),
		},
		Request: &v1.ManagementPlaneRequest_ConfigUploadRequest{
			ConfigUploadRequest: &v1.ConfigUploadRequest{
				Overview: request.GetOverview(),
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

	fullFilePath := mgs.findFile(request.GetFileMeta())

	if fullFilePath == "" {
		slog.Error("File not found", "file_name", fileName)
		return nil, status.Errorf(codes.NotFound, "File not found")
	}

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
	fileMeta := request.GetFile().GetFileMeta()
	fileName := fileMeta.GetName()
	fileHash := fileMeta.GetHash()
	filePermissions := fileMeta.GetPermissions()

	slog.Info("Updating file", "name", fileName, "hash", fileHash)

	fullFilePath := mgs.findFile(request.GetFile().GetFileMeta())

	if _, err := os.Stat(fullFilePath); os.IsNotExist(err) {
		statErr := os.MkdirAll(filepath.Dir(fullFilePath), os.ModePerm)
		if statErr != nil {
			slog.Info("Failed to create/update file", "full_file_path", fullFilePath, "error", statErr)
			return nil, status.Errorf(codes.Internal, "Failed to create/update file")
		}
	}

	err := os.WriteFile(fullFilePath, fileContents, getFileMode(filePermissions))
	if err != nil {
		slog.Info("Failed to create/update file", "full_file_path", fullFilePath, "error", err)
		return nil, status.Errorf(codes.Internal, "Failed to create/update file")
	}

	return &v1.UpdateFileResponse{
		FileMeta: fileMeta,
	}, nil
}

func (mgs *FileService) findFile(fileMeta *v1.FileMeta) (fullFilePath string) {
	for instanceID, files := range mgs.instanceFiles {
		for _, file := range files {
			if file.GetFileMeta().GetName() == fileMeta.GetName() {
				fullFilePath = filepath.Join(mgs.configDirectory, instanceID, fileMeta.GetName())
			}
		}
	}

	return fullFilePath
}

func getFileMode(mode string) os.FileMode {
	result, err := strconv.ParseInt(mode, 8, 32)
	if err != nil {
		return os.FileMode(defaultFilePermissions)
	}

	return os.FileMode(result)
}
