// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package test

import (
	"encoding/json"
	"os"
	"path"
	"path/filepath"
	"testing"

	"github.com/nginx/agent/v3/api/grpc/instances"

	"github.com/stretchr/testify/require"
)

func CreateDirWithErrorCheck(t *testing.T, dirName string) {
	t.Helper()
	err := os.Mkdir(dirName, os.ModePerm)
	require.NoError(t, err)
}

func CreateFileWithErrorCheck(t *testing.T, dir, fileName string) *os.File {
	t.Helper()

	testConf, err := os.CreateTemp(dir, fileName)
	require.NoError(t, err)

	return testConf
}

func CreateCacheFiles(t *testing.T, cachePath string, cacheData map[string]*instances.File) {
	t.Helper()
	cache, err := json.MarshalIndent(cacheData, "", "  ")
	require.NoError(t, err)

	err = os.MkdirAll(path.Dir(cachePath), os.ModePerm)
	require.NoError(t, err)

	for _, file := range cacheData {
		CreateFileWithErrorCheck(t, filepath.Dir(file.GetPath()), filepath.Base(file.GetPath()))
	}

	err = os.WriteFile(cachePath, cache, os.ModePerm)
	require.NoError(t, err)
}

func RemoveFileWithErrorCheck(t *testing.T, fileName string) {
	t.Helper()
	err := os.Remove(fileName)
	require.NoError(t, err)
}
