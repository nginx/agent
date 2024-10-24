// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package helpers

import (
	"os"
	"testing"

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
