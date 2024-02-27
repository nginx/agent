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

	"github.com/google/uuid"

	"github.com/nginx/agent/v3/api/grpc/instances"

	"github.com/stretchr/testify/require"
)

const filePermission = 0o777

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

func CreateCacheFiles(t testing.TB, cachePath string, cacheData map[string]*instances.File) {
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

func RemoveFileWithErrorCheck(t testing.TB, fileName string) {
	t.Helper()
	err := os.Remove(fileName)
	require.NoError(t, err)
}

func CreateTestIDs(t testing.TB) (uuid.UUID, uuid.UUID) {
	t.Helper()
	tenantID, err := uuid.Parse("7332d596-d2e6-4d1e-9e75-70f91ef9bd0e")
	require.NoError(t, err)

	instanceID, err := uuid.Parse("aecea348-62c1-4e3d-b848-6d6cdeb1cb9c")
	require.NoError(t, err)

	return tenantID, instanceID
}
