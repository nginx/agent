// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

// Package files implements utility routines for gathering information about files and their contents.
package files

import (
	"bytes"
	"cmp"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"io"
	"os"
	"slices"
	"strconv"

	mpi "github.com/nginx/agent/v3/api/grpc/mpi/v1"
	"github.com/nginx/agent/v3/pkg/uuid"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const permissions = 0o644

func GetFileMeta(filePath string) (*mpi.FileMeta, error) {
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return nil, err
	}

	hash, err := GenerateFileHash(filePath)
	if err != nil {
		return nil, err
	}

	return &mpi.FileMeta{
		Name:         filePath,
		Hash:         hash,
		ModifiedTime: timestamppb.New(fileInfo.ModTime()),
		Permissions:  fileInfo.Mode().Perm().String(),
		Size:         fileInfo.Size(),
	}, nil
}

// GetPermissions returns a file's permissions as a string.
func GetPermissions(fileMode os.FileMode) string {
	return fmt.Sprintf("%#o", fileMode.Perm())
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

	return uuid.Generate("%s", hashes)
}

// GenerateFileHash returns the hash value of a file's contents.
func GenerateFileHash(filePath string) (string, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer f.Close()

	h := sha256.New()
	if _, copyErr := io.Copy(h, f); copyErr != nil {
		return "", copyErr
	}

	return base64.StdEncoding.EncodeToString(h.Sum(nil)), nil
}

func FileMode(mode string) os.FileMode {
	result, err := strconv.ParseInt(mode, 8, 32)
	if err != nil {
		return os.FileMode(permissions)
	}

	return os.FileMode(result)
}

func GenerateFileHashWithContent(content []byte) (string, error) {
	h := sha256.New()
	_, err := h.Write(content)
	if err != nil {
		return "", err
	}

	return base64.StdEncoding.EncodeToString(h.Sum(nil)), nil
}

func ReadFileGenerateFile(filePath string) ([]byte, string, error) {
	f, openErr := os.Open(filePath)
	if openErr != nil {
		return nil, "", openErr
	}

	content := bytes.NewBuffer([]byte{})
	_, copyErr := io.Copy(content, f)
	if copyErr != nil {
		return nil, "", copyErr
	}

	hash, err := GenerateFileHashWithContent(content.Bytes())

	return content.Bytes(), hash, err
}

// TODO: reduce complexity
// nolint: revive, cyclop
// CompareFileHash compares files from the FileOverview to files on disk and returns a map with the files that have
// changed and a map with the contents of those files. Key to both maps is file path
func CompareFileHash(fileOverview *mpi.FileOverview) (diffFiles map[string]*mpi.File,
	fileContents map[string][]byte, err error,
) {
	diffFiles = make(map[string]*mpi.File)
	fileContents = make(map[string][]byte)
	for _, file := range fileOverview.GetFiles() {
		fileName := file.GetFileMeta().GetName()
		switch file.GetAction() {
		case mpi.File_FILE_ACTION_DELETE:
			if _, err = os.Stat(fileName); os.IsNotExist(err) {
				// File is already deleted skip
				continue
			}
			fileContent, _, readErr := ReadFileGenerateFile(fileName)
			if readErr != nil {
				return nil, nil, fmt.Errorf("error reading file %s, error: %w", fileName, readErr)
			}
			fileContents[fileName] = fileContent
			diffFiles[fileName] = file

			continue
		case mpi.File_FILE_ACTION_ADD:
			if _, err = os.Stat(fileName); os.IsNotExist(err) {
				// file is new nothing to compare
				diffFiles[fileName] = file
				continue
			}
			updateAction := mpi.File_FILE_ACTION_UPDATE
			file.Action = &updateAction

			fallthrough
		case mpi.File_FILE_ACTION_UPDATE:
			fileContent, hash, hashErr := ReadFileGenerateFile(fileName)
			if hashErr != nil {
				return nil, nil, fmt.Errorf("error generating hash for file %s, error: %w", fileName, err)
			}

			if hash == file.GetFileMeta().GetHash() {
				// file is same as on disk skip
				continue
			}
			fileContents[fileName] = fileContent
			diffFiles[fileName] = file
		case mpi.File_FILE_ACTION_UNSPECIFIED, mpi.File_FILE_ACTION_UNCHANGED:
			// FileAction is UNSPECIFIED or UNCHANGED skipping, treat UNSPECIFIED as if it is UNCHANGED
			fallthrough
		default:
			continue
		}
	}

	return diffFiles, fileContents, nil
}
