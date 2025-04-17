// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package file

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path"

	"github.com/nginx/agent/v3/pkg/files"

	mpi "github.com/nginx/agent/v3/api/grpc/mpi/v1"
)

type FileOperator struct{}

var _ fileOperator = (*FileOperator)(nil)

// FileOperator only purpose is to write files,

func NewFileOperator() *FileOperator {
	return &FileOperator{}
}

func (fo *FileOperator) Write(ctx context.Context, fileContent []byte, file *mpi.FileMeta) error {
	filePermission := files.FileMode(file.GetPermissions())
	err := fo.CreateFileDirectories(ctx, file, filePermission)
	if err != nil {
		return err
	}

	writeErr := os.WriteFile(file.GetName(), fileContent, filePermission)
	if writeErr != nil {
		return fmt.Errorf("error writing to file %s: %w", file.GetName(), writeErr)
	}
	slog.DebugContext(ctx, "Content written to file", "file_path", file.GetName())

	return nil
}

func (fo *FileOperator) CreateFileDirectories(ctx context.Context, fileMeta *mpi.FileMeta, filePermission os.FileMode) error {
	if _, err := os.Stat(fileMeta.GetName()); os.IsNotExist(err) {
		slog.DebugContext(ctx, "File does not exist, creating new file", "file_path", fileMeta.GetName())
		err = os.MkdirAll(path.Dir(fileMeta.GetName()), filePermission)
		if err != nil {
			return fmt.Errorf("error creating directory %s: %w", path.Dir(fileMeta.GetName()), err)
		}
	}

	return nil
}
