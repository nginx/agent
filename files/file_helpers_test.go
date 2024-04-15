// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package files

import (
	"os"
	"testing"

	"github.com/nginx/agent/v3/api/grpc/mpi/v1"
	"github.com/nginx/agent/v3/test/helpers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetPermissions(t *testing.T) {
	file, err := os.CreateTemp(t.TempDir(), "get_permissions_test.txt")
	defer helpers.RemoveFileWithErrorCheck(t, file.Name())
	require.NoError(t, err)

	info, err := os.Stat(file.Name())
	require.NoError(t, err)

	permissions := GetPermissions(info.Mode())

	assert.Equal(t, "0600", permissions)
}

func Test_GenerateConfigVersion(t *testing.T) {
	files := []*v1.File{
		{
			FileMeta: &v1.FileMeta{
				Name: "file1",
				Hash: "3151431543",
			},
		},
		{
			FileMeta: &v1.FileMeta{
				Name: "file2",
				Hash: "4234235325",
			},
		},
	}

	configVersion := GenerateConfigVersion(files)

	assert.Equal(t, "a7d6580c-8ac9-376e-acde-b2cbed21d291", configVersion)
}

func Test_GenerateFileHash(t *testing.T) {
	testFile := helpers.CreateFileWithErrorCheck(t, os.TempDir(), "testFile")
	defer helpers.RemoveFileWithErrorCheck(t, testFile.Name())
	err := os.WriteFile(testFile.Name(), []byte("test data"), 0o600)
	require.NoError(t, err)

	hash, err := GenerateFileHash(testFile.Name())
	require.NoError(t, err)

	assert.Equal(t, "\x91o\x00'\xa5u\aL\xe7*3\x17w\xc3G\x8de\x13\xf7\x86\xa5\x91\xbd\x89-\xa1\xa5w\xbf#5\xf9", hash)
}
