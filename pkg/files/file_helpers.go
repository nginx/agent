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

	"github.com/google/uuid"

	"github.com/nginx/agent/v3/api/grpc/mpi/v1"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func GetFileMeta(filePath string) (*v1.FileMeta, error) {
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return nil, err
	}

	content, _ := ReadFile(filePath)
	hash := GenerateHash(content)

	return &v1.FileMeta{
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
