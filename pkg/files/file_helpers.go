// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

// Package files implements utility routines for gathering information about files and their contents.
package files

import (
	"cmp"
	"fmt"
	"os"
	"slices"
	"strconv"

	"github.com/google/uuid"

	mpi "github.com/nginx/agent/v3/api/grpc/mpi/v1"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const permissions = 0o644

// FileMeta returns a proto FileMeta struct from a given file path.
func FileMeta(filePath string) (*mpi.FileMeta, error) {
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return nil, err
	}

	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	fileHash := GenerateHash(content)

	return &mpi.FileMeta{
		Name:         filePath,
		Hash:         fileHash,
		ModifiedTime: timestamppb.New(fileInfo.ModTime()),
		Permissions:  Permissions(fileInfo.Mode()),
		Size:         fileInfo.Size(),
	}, nil
}

// Permissions returns a file's permissions as a string.
func Permissions(fileMode os.FileMode) string {
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

// GenerateHash returns the hash value of a file's contents.
func GenerateHash(b []byte) string {
	return uuid.NewMD5(uuid.Nil, b).String()
}

// ConvertToMapOfFiles converts a list of files to a map of files with the file name as the key
func ConvertToMapOfFiles(files []*mpi.File) map[string]*mpi.File {
	filesMap := make(map[string]*mpi.File)
	for _, file := range files {
		filesMap[file.GetFileMeta().GetName()] = file
	}

	return filesMap
}
