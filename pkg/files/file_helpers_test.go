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

func TestGetFileMeta(t *testing.T) {
	file, err := os.CreateTemp(t.TempDir(), "get_file_meta.txt")
	defer helpers.RemoveFileWithErrorCheck(t, file.Name())
	require.NoError(t, err)

	expected := &v1.FileMeta{
		Name:        file.Name(),
		Hash:        "4ae71336-e44b-39bf-b9d2-752e234818a5",
		Permissions: "-rw-------",
		Size:        0,
	}

	fileMeta, err := FileMeta(file.Name())
	require.NoError(t, err)

	assert.Equal(t, expected.GetName(), fileMeta.GetName())
	assert.Equal(t, expected.GetHash(), fileMeta.GetHash())
	assert.Equal(t, expected.GetPermissions(), fileMeta.GetPermissions())
	assert.Equal(t, expected.GetSize(), fileMeta.GetSize())
	assert.NotNil(t, fileMeta.GetModifiedTime())
}

func TestGetPermissions(t *testing.T) {
	file, err := os.CreateTemp(t.TempDir(), "get_permissions_test.txt")
	defer helpers.RemoveFileWithErrorCheck(t, file.Name())
	require.NoError(t, err)

	info, err := os.Stat(file.Name())
	require.NoError(t, err)

	permissions := Permissions(info.Mode())

	assert.Equal(t, "0600", permissions)
}

func Test_GenerateConfigVersion(t *testing.T) {
	expectedConfigVersion := "b0fe3bfa-5140-3fb2-aee6-7f458f6dae4e"

	file1 := &v1.File{
		FileMeta: &v1.FileMeta{
			Name: "file1",
			Hash: "3151431543",
		},
	}
	file2 := &v1.File{
		FileMeta: &v1.FileMeta{
			Name: "file2",
			Hash: "4234235325",
		},
	}

	files := []*v1.File{
		file1,
		file2,
	}

	configVersion := GenerateConfigVersion(files)
	assert.Equal(t, expectedConfigVersion, configVersion)

	// Reorder files to make sure version is still the same
	files = []*v1.File{
		file2,
		file1,
	}

	configVersion = GenerateConfigVersion(files)
	assert.Equal(t, expectedConfigVersion, configVersion)
}

func Test_GenerateHash(t *testing.T) {
	testFile := helpers.CreateFileWithErrorCheck(t, os.TempDir(), "testFile")
	defer helpers.RemoveFileWithErrorCheck(t, testFile.Name())
	err := os.WriteFile(testFile.Name(), []byte("test data"), 0o600)
	require.NoError(t, err)

	content, err := os.ReadFile(testFile.Name())

	hash := GenerateHash(content)
	require.NoError(t, err)

	assert.Equal(t, "ff8dcd5d-a12f-3895-a6b9-2ac8c98bfd08", hash)
}
