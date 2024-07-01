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

	"github.com/google/uuid"

	"github.com/nginx/agent/v3/api/grpc/mpi/v1"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// FileMeta returns a proto FileMeta struct from a given file path.
func FileMeta(filePath string) (*v1.FileMeta, error) {
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return nil, err
	}

	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	hash := GenerateHash(content)

	return &v1.FileMeta{
		Name:         filePath,
		Hash:         hash,
		ModifiedTime: timestamppb.New(fileInfo.ModTime()),
		Permissions:  fileInfo.Mode().Perm().String(),
		Size:         fileInfo.Size(),
	}, nil
}

// Permissions returns a file's permissions as a string.
func Permissions(fileMode os.FileMode) string {
	return fmt.Sprintf("%#o", fileMode.Perm())
}

// GenerateConfigVersion returns a unique config version for a set of files.
// The config version is calculated by joining the file hashes together and generating a unique ID.
func GenerateConfigVersion(fileSlice []*v1.File) string {
	var hashes string

	slices.SortFunc(fileSlice, func(a, b *v1.File) int {
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
