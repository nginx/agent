// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package test

import (
	"encoding/json"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"testing"

	"github.com/nginx/agent/v3/api/grpc/instances"

	"github.com/stretchr/testify/assert"
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
	if err != nil {
		t.Logf("error marshaling cache, error: %v", err)
		t.Fail()
	}

	err = os.MkdirAll(path.Dir(cachePath), os.ModePerm)
	if err != nil {
		t.Logf("error creating cache directory, error: %v", err)
		t.Fail()
	}

	for _, file := range cacheData {
		CreateFileWithErrorCheck(t, filepath.Dir(file.GetPath()), filepath.Base(file.GetPath()))
	}
	err = os.WriteFile(cachePath, cache, os.ModePerm)
	if err != nil {
		t.Logf("error writing to file, error: %v", err)
		t.Fail()
	}
}

func RemoveFileWithErrorCheck(t *testing.T, fileName string) {
	t.Helper()
	err := os.Remove(fileName)
	if err != nil {
		assert.Fail(t, fmt.Sprintf("failed on os.Remove of file %s", fileName))
	}
}
