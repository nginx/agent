// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

// Package files implements utility routines for gathering information about files nad their contents.
package files

import (
	"crypto/sha256"
	"fmt"
	"io"
	"os"

	"github.com/nginx/agent/v3/api/grpc/mpi/v1"
	"github.com/nginx/agent/v3/internal/uuid"
)

// GetPermissions returns a file's permissions as a string.
func GetPermissions(fileMode os.FileMode) string {
	return fmt.Sprintf("%#o", fileMode.Perm())
}

// GenerateConfigVersion returns a unique config version for a set of files.
// The config version is calculated by joining the file hashes together and generating a unique ID.
func GenerateConfigVersion(files []*v1.File) string {
	var hashes string

	for _, file := range files {
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

	return string(h.Sum(nil)), nil
}
