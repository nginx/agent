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
	"strings"

	"github.com/nginx/agent/v3/api/grpc/mpi/v1"
	filesHelper "github.com/nginx/agent/v3/pkg/files"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const defaultFilePermissions = 0o644

type FileService struct {
	v1.UnimplementedFileServiceServer
	overviews          map[string][]*v1.File // Key is the config version UID
	versionDirectories map[string]string     // Key is the version directory name
	configDirectory    string
}

func NewFileService(configDirectory string) (*FileService, error) {
	overviews := make(map[string][]*v1.File)
	versionDirectories := make(map[string]string)

	mapOfVersionedFiles, err := getMapOfVersionedFiles(configDirectory)
	if err != nil {
		return nil, err
	}

	for versionDirectory, versionedFiles := range mapOfVersionedFiles {
		configVersion := filesHelper.GenerateConfigVersion(versionedFiles)
		slog.Info(
			"Found versioned files",
			"version_directory_name", versionDirectory,
			"number_of_files", len(versionedFiles),
			"config_version", configVersion,
		)
		overviews[configVersion] = versionedFiles
		versionDirectories[configVersion] = versionDirectory
		slog.Info("versioned Files", "", versionedFiles)
	}

	return &FileService{
		configDirectory:    configDirectory,
		overviews:          overviews,
		versionDirectories: versionDirectories,
	}, nil
}

func (mgs *FileService) GetOverview(
	_ context.Context,
	request *v1.GetOverviewRequest,
) (*v1.GetOverviewResponse, error) {
	configVersion := request.GetConfigVersion()
	version := configVersion.GetVersion()
	files := mgs.overviews[version]

	slog.Info("Getting overview", "config_version", configVersion)

	if files == nil {
		slog.Error("Config version not found", "config_version", configVersion)
		return nil, status.Errorf(codes.NotFound, "Config version not found")
	}

	return &v1.GetOverviewResponse{
		Overview: &v1.FileOverview{
			ConfigVersion: configVersion,
			Files:         files,
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

	mgs.overviews[overview.GetConfigVersion().GetVersion()] = overview.GetFiles()

	return &v1.UpdateOverviewResponse{}, nil
}

func (mgs *FileService) GetFile(
	_ context.Context,
	request *v1.GetFileRequest,
) (*v1.GetFileResponse, error) {
	fileName := request.GetFileMeta().GetName()
	fileHash := request.GetFileMeta().GetHash()

	slog.Info("Getting file", "name", fileName, "hash", fileHash)

	fileConfigVersions := mgs.getConfigVersions(fileName, fileHash)

	if len(fileConfigVersions) == 0 {
		slog.Error("File not found", "file_name", fileName)
		return nil, status.Errorf(codes.NotFound, "File not found")
	}

	fullFilePath := filepath.Join(mgs.versionDirectories[fileConfigVersions[0]], fileName)

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

	fileConfigVersions := mgs.getConfigVersions(fileName, fileHash)

	for _, fileConfigVersion := range fileConfigVersions {
		fullFilePath := filepath.Join(mgs.configDirectory, mgs.versionDirectories[fileConfigVersion], fileName)

		err := performFileAction(fileAction, fileContents, fullFilePath, filePermissions)
		if err != nil {
			return nil, err
		}
	}

	return &v1.UpdateFileResponse{
		FileMeta: fileMeta,
	}, nil
}

func (mgs *FileService) getConfigVersions(fileName, fileHash string) []string {
	var fileConfigVersions []string

	for configVersion, overview := range mgs.overviews {
		for _, file := range overview {
			if fileName == file.GetFileMeta().GetName() && fileHash == file.GetFileMeta().GetHash() {
				fileConfigVersions = append(fileConfigVersions, configVersion)
				break
			}
		}
	}

	return fileConfigVersions
}

// nolint: gomnd
func getMapOfVersionedFiles(configDirectory string) (map[string][]*v1.File, error) {
	files := make(map[string][]*v1.File)

	slog.Info("Getting map of versioned files", "config_directory", configDirectory)

	err := filepath.Walk(configDirectory, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if isValidFile(info, path) {
			slog.Debug("Found file", "path", path)

			splitPath := strings.SplitN(strings.Split(path, configDirectory)[1], string(filepath.Separator), 3)
			if len(splitPath) == 2 {
				return nil
			}
			version := splitPath[1]
			filePath := string(filepath.Separator) + splitPath[2]

			versionDirectory := filepath.Join(configDirectory, version)

			file, fileErr := createFile(path, filePath)
			if fileErr != nil {
				return fileErr
			}

			slog.Debug("File found:", "path", file.GetFileMeta().GetName(),
				"hash", file.GetFileMeta().GetHash())

			files[versionDirectory] = append(files[versionDirectory], file)
		}

		return nil
	})

	return files, err
}

func isValidFile(info os.FileInfo, path string) bool {
	return !info.IsDir() && !strings.HasSuffix(path, ".DS_Store")
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

func createFile(fullPath, filePath string) (*v1.File, error) {
	content, err := os.ReadFile(fullPath)
	if err != nil {
		return nil, err
	}
	fileHash := filesHelper.GenerateHash(content)

	fileInfo, err := os.Stat(fullPath)
	if err != nil {
		return nil, err
	}

	return &v1.File{
		FileMeta: &v1.FileMeta{
			Name:         filePath,
			Hash:         fileHash,
			ModifiedTime: timestamppb.New(fileInfo.ModTime()),
			Permissions:  filesHelper.Permissions(fileInfo.Mode()),
			Size:         fileInfo.Size(),
		},
	}, nil
}
