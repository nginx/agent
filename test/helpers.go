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
	"time"

	"github.com/nginx/agent/v3/api/grpc/instances"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func CreateDirWithErrorCheck(t *testing.T, dirName string) {
	t.Helper()
	err := os.Mkdir(dirName, 0o700)
	require.NoError(t, err)
}

func CreateFileWithErrorCheck(t *testing.T, dir, fileName string) *os.File {
	t.Helper()

	testConf, err := os.CreateTemp(dir, fileName)
	require.NoError(t, err)

	return testConf
}

func CreateCacheFiles(t *testing.T, cachePath string, cacheData map[string]*instances.File) {
	cache, err := json.MarshalIndent(cacheData, "", "  ")
	if err != nil {
		t.Logf("error marshaling cache, error: %v", err)
		t.Fail()
	}

	err = os.MkdirAll(path.Dir(cachePath), 0o750)
	if err != nil {
		t.Logf("error creating cache directory, error: %v", err)
		t.Fail()
	}

	for _, file := range cacheData {
		CreateFileWithErrorCheck(t, filepath.Dir(file.Path), filepath.Base(file.Path))
		t.Logf("creating writing to file, error: %v", file)
	}
	err = os.WriteFile(cachePath, cache, 0o600)
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

func CreateProtoTime(t *testing.T, timeString string) *timestamppb.Timestamp {
	t.Helper()
	newTime, err := time.Parse(time.RFC3339, timeString)
	require.NoError(t, err)

	protoTime := timestamppb.New(newTime)
	if err != nil {
		assert.Fail(t, fmt.Sprintf("failed on creating timestamp %s", protoTime))
	}

	return protoTime
}
