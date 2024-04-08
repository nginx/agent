// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package helpers

import (
	"encoding/json"
	"os"
	"path"
	"path/filepath"
	"testing"

	"github.com/nginx/agent/v3/api/grpc/mpi/v1"

	"github.com/stretchr/testify/require"
)

const (
	filePermission = 0o700
)

func CreateDirWithErrorCheck(t testing.TB, dirName string) {
	t.Helper()

	err := os.MkdirAll(dirName, filePermission)

	require.NoError(t, err)
}

func CreateFileWithErrorCheck(t testing.TB, dir, fileName string) *os.File {
	t.Helper()

	testConf, err := os.CreateTemp(dir, fileName)
	require.NoError(t, err)

	return testConf
}

func RemoveFileWithErrorCheck(t testing.TB, fileName string) {
	t.Helper()

	err := os.Remove(fileName)

	require.NoError(t, err)
}

func CreateCacheFiles(t testing.TB, cachePath string, cacheData map[string]*v1.FileMeta) {
	t.Helper()
	cache, err := json.MarshalIndent(cacheData, "", "  ")
	require.NoError(t, err)

	err = os.MkdirAll(path.Dir(cachePath), os.ModePerm)
	require.NoError(t, err)

	for _, file := range cacheData {
		CreateFileWithErrorCheck(t, filepath.Dir(file.GetName()), filepath.Base(file.GetName()))
	}

	err = os.WriteFile(cachePath, cache, os.ModePerm)
	require.NoError(t, err)
}
