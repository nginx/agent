// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

// Package files implements utility routines for gathering information about files and their contents.
package files

import (
	"bytes"
	"cmp"
	"fmt"
	"io"
	"os"
	"slices"
	"strconv"

	"github.com/google/uuid"
	mpi "github.com/nginx/agent/v3/api/grpc/mpi/v1"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const permissions = 0o644

func GetFileMeta(filePath string) (*mpi.FileMeta, error) {
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return nil, err
	}

	content, err := ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	fileHash := GenerateHash(content)

	return &mpi.FileMeta{
		Name:         filePath,
		Hash:         fileHash,
		ModifiedTime: timestamppb.New(fileInfo.ModTime()),
		Permissions:  GetPermissions(fileInfo.Mode()),
		Size:         fileInfo.Size(),
	}, nil
}

// GetPermissions returns a file's permissions as a string.
func GetPermissions(fileMode os.FileMode) string {
	return fmt.Sprintf("%#o", fileMode.Perm())
}

func FileMode(mode string) os.FileMode {
	result, err := strconv.ParseInt(mode, 8, 32)
	if err != nil {
		return os.FileMode(permissions)
	}

	return os.FileMode(result)
}

// GenerateConfigVersion returns a unique config version for a set of files.
// The config version is calculated by joining the file hashes together and generating a unique ID.
func GenerateConfigVersion(fileSlice []*mpi.File) string {
	var hashes string

	slices.SortFunc(fileSlice, func(a, b *mpi.File) int {
		return cmp.Compare(a.GetFileMeta().GetName(), b.GetFileMeta().GetName())
	})

	for _, file := range fileSlice {
		hashes += file.GetFileMeta().GetHash()
	}

	return GenerateHash([]byte(hashes))
}

func GenerateHash(b []byte) string {
	return uuid.NewMD5(uuid.Nil, b).String()
}

// ReadFile returns the content of a file
func ReadFile(filePath string) ([]byte, error) {
	f, openErr := os.Open(filePath)
	if openErr != nil {
		return nil, openErr
	}

	content := bytes.NewBuffer([]byte{})
	_, copyErr := io.Copy(content, f)
	if copyErr != nil {
		return nil, copyErr
	}

	return content.Bytes(), nil
}

// CompareFileHash compares files from the FileOverview to files on disk and returns a map with the files that have
// changed and a map with the contents of those files. Key to both maps is file path
// nolint: revive,cyclop
func CompareFileHash(fileOverview *mpi.FileOverview) (fileDiff map[string]*mpi.File,
	fileContents map[string][]byte, err error,
) {
	fileDiff = make(map[string]*mpi.File)
	fileContents = make(map[string][]byte)

	for _, file := range fileOverview.GetFiles() {
		fileName := file.GetFileMeta().GetName()
		switch file.GetAction() {
		case mpi.File_FILE_ACTION_DELETE:
			if _, err = os.Stat(fileName); os.IsNotExist(err) {
				// File is already deleted, skip
				continue
			}
			fileContent, readErr := ReadFile(fileName)
			if readErr != nil {
				return nil, nil, fmt.Errorf("error reading file %s, error: %w", fileName, readErr)
			}
			fileContents[fileName] = fileContent
			fileDiff[fileName] = file
		case mpi.File_FILE_ACTION_ADD:
			if _, err = os.Stat(fileName); os.IsNotExist(err) {
				// file is new, nothing to compare
				fileDiff[fileName] = file
				continue
			}
			// file already exists and needs to be updated instead
			updateAction := mpi.File_FILE_ACTION_UPDATE
			file.Action = &updateAction

			fallthrough
		case mpi.File_FILE_ACTION_UPDATE:
			fileContent, readErr := ReadFile(fileName)
			if readErr != nil {
				return nil, nil, fmt.Errorf("error generating hash for file %s, error: %w", fileName, readErr)
			}
			fileHash := GenerateHash(fileContent)

			if fileHash == file.GetFileMeta().GetHash() {
				// file is same as on disk, skip
				continue
			}

			fileContents[fileName] = fileContent
			fileDiff[fileName] = file
		case mpi.File_FILE_ACTION_UNSPECIFIED, mpi.File_FILE_ACTION_UNCHANGED:
			// FileAction is UNSPECIFIED or UNCHANGED skipping. Treat UNSPECIFIED as if it is UNCHANGED.
			fallthrough
		default:
			continue
		}
	}

	return fileDiff, fileContents, nil
}
