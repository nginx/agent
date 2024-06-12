// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package file

import (
	"bytes"
	"context"
	"fmt"
	mpi "github.com/nginx/agent/v3/api/grpc/mpi/v1"
	"github.com/nginx/agent/v3/files"
	"io"
	"log/slog"
	"os"
	"path"
)

type FileOperator struct{}

var _ fileOperator = (*FileOperator)(nil)

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

// ReadFileContents TODO: Need to either handle files not having any changes here or have it so by this point the list of files
// only contains files needing to be updated, added or deleted ??
func (fo *FileOperator) ReadFileContents(files []*mpi.File) (filesContents map[string][]byte, err error) {
	filesContents = make(map[string][]byte)
	for _, file := range files {
		filePath := file.GetFileMeta().GetName()
		if _, err = os.Stat(filePath); os.IsNotExist(err) {
			// File is new and doesn't exist so no previous content to save
			continue
		}
		f, openErr := os.Open(filePath)
		if openErr != nil {
			return nil, err
		}

		content := bytes.NewBuffer([]byte{})
		_, copyErr := io.Copy(content, f)
		if copyErr != nil {
			return nil, copyErr
		}

		filesContents[filePath] = content.Bytes()
	}
	return filesContents, nil
}
