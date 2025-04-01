// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package file

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path"

	"github.com/nginx/agent/v3/pkg/files"

	mpi "github.com/nginx/agent/v3/api/grpc/mpi/v1"
)

type FileOperator struct{}

var _ fileOperator = (*FileOperator)(nil)

var (
	manifestDirPath  = "/var/lib/nginx-agent"
	manifestFilePath = manifestDirPath + "/manifest.json"
)

// FileOperator only purpose is to write files,

func NewFileOperator() *FileOperator {
	return &FileOperator{}
}

func (fo *FileOperator) Write(ctx context.Context, fileContent []byte, file *mpi.FileMeta) error {
	filePermission := files.FileMode(file.GetPermissions())
	if _, err := os.Stat(file.GetName()); os.IsNotExist(err) {
		slog.DebugContext(ctx, "File does not exist, creating new file", "file_path", file.GetName())
		err = os.MkdirAll(path.Dir(file.GetName()), filePermission)
		if err != nil {
			return fmt.Errorf("error creating directory %s: %w", path.Dir(file.GetName()), err)
		}
	}

	err := os.WriteFile(file.GetName(), fileContent, filePermission)
	if err != nil {
		return fmt.Errorf("error writing to file %s: %w", file.GetName(), err)
	}
	slog.DebugContext(ctx, "Content written to file", "file_path", file.GetName())

	return nil
}

// nolint: musttag
func (fo *FileOperator) UpdateManifestFile(currentFiles map[string]*mpi.File) (err error) {
	slog.Info("Updating NGINX config manifest file", "current_files", currentFiles)

	manifestJSON, err := json.MarshalIndent(currentFiles, "", "  ")
	if err != nil {
		slog.Error("Unable to marshal manifest file json ", "err", err)
		return err
	}

	// 0755 allows read/execute for all, write for owner
	if err = os.MkdirAll(manifestDirPath, dirPerm); err != nil {
		slog.Error("Unable to create directory", "err", err)
		return err
	}

	// 0600 ensures only root can read/write
	newFile, err := os.OpenFile(manifestFilePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, filePerm)
	if err != nil {
		slog.Error("Failed to read manifest file", "error", err)
		return err
	}
	defer newFile.Close()

	_, err = newFile.Write(manifestJSON)
	if err != nil {
		slog.Error("Failed to write manifest file: %v\n", "error", err)
		return err
	}

	return nil
}

func (fo *FileOperator) CreateManifestFile() error {
	if err := os.MkdirAll(manifestDirPath, dirPerm); err != nil {
		slog.Error("Unable to create directory", "err", err)
		return err
	}

	// 0600 ensures only root can read/write
	newFile, err := os.OpenFile(manifestFilePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, filePerm)
	if err != nil {
		slog.Error("Failed to create manifest file", "error", err)
		return err
	}

	return newFile.Close()
}

// nolint: musttag
func (fo *FileOperator) ManifestFile(currentFiles map[string]*mpi.File) (map[string]*mpi.File, error) {
	if _, err := os.Stat(manifestFilePath); err != nil {
		return currentFiles, err // Return current files if manifest directory still doesn't exist
	}

	file, err := os.ReadFile(manifestFilePath)
	if err != nil {
		slog.Error("Failed to read manifest file", "error", err)
		return nil, err
	}

	var manifestFiles map[string]*mpi.File

	err = json.Unmarshal(file, &manifestFiles)
	if err != nil {
		slog.Error("Failed to parse manifest file", "error", err)
		return nil, err
	}

	fileMap := fo.convertToFileMap(manifestFiles)

	return fileMap, nil
}

func (fo *FileOperator) convertToFileMap(manifestFiles map[string]*mpi.File) map[string]*mpi.File {
	currentFileMap := make(map[string]*mpi.File)
	for name, manifestFile := range manifestFiles {
		currentFile := fo.convertToFile(manifestFile)
		currentFileMap[name] = currentFile
	}

	return currentFileMap
}

func (fo *FileOperator) convertToFile(manifestFile *mpi.File) *mpi.File {
	return manifestFile
}
