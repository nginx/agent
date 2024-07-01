// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package files

import (
	"os"
	"testing"

	"github.com/google/uuid"

	"github.com/nginx/agent/v3/test/protos"

	mpi "github.com/nginx/agent/v3/api/grpc/mpi/v1"
	"github.com/nginx/agent/v3/test/helpers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetFileMeta(t *testing.T) {
	file, err := os.CreateTemp(t.TempDir(), "get_file_meta.txt")
	defer helpers.RemoveFileWithErrorCheck(t, file.Name())
	require.NoError(t, err)

	expected := protos.GetFileMeta(file.Name(), GenerateHash([]byte("")))

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
	tests := []struct {
		name     string
		input    []*mpi.File
		expected string
	}{
		{
			name:     "Test 1: empty file slice",
			input:    []*mpi.File{},
			expected: GenerateHash([]byte{}),
		},
		{
			name: "Test 2: one file",
			input: []*mpi.File{
				{
					FileMeta: &mpi.FileMeta{
						Name: "file1",
						Hash: "hash1",
					},
					Action: nil,
				},
			},
			expected: GenerateHash([]byte("hash1")),
		},
		{
			name: "Test 3: multiple files",
			input: []*mpi.File{
				{
					FileMeta: &mpi.FileMeta{
						Name: "file1",
						Hash: "hash1",
					},
					Action: nil,
				},
				{
					FileMeta: &mpi.FileMeta{
						Name: "file2",
						Hash: "hash2",
					},
					Action: nil,
				},
			},
			expected: func() string {
				hashes := "hash1hash2"
				return GenerateHash([]byte(hashes))
			}(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GenerateConfigVersion(tt.input)
			if result != tt.expected {
				t.Errorf("GenerateConfigVersion(%v) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestGenerateHash(t *testing.T) {
	tests := []struct {
		name     string
		input    []byte
		expected string
	}{
		{
			name:     "Test 1: empty byte slice",
			input:    []byte{},
			expected: uuid.NewMD5(uuid.Nil, []byte("")).String(),
		},
		{
			name:     "Test 2: non-empty byte slice",
			input:    []byte("test"),
			expected: uuid.NewMD5(uuid.Nil, []byte("test")).String(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GenerateHash(tt.input)
			if result != tt.expected {
				t.Errorf("GenerateHash(%v) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestCompareFileHash_Delete(t *testing.T) {
	tempDir := os.TempDir()

	deleteTestFile := helpers.CreateFileWithErrorCheck(t, tempDir, "deleteTestFile")
	defer helpers.RemoveFileWithErrorCheck(t, deleteTestFile.Name())
	expectedFileContent, readErr := os.ReadFile("../../test/config/nginx/nginx.conf")
	require.NoError(t, readErr)
	err := os.WriteFile(deleteTestFile.Name(), expectedFileContent, 0o600)
	require.NoError(t, err)

	updateTestFile := helpers.CreateFileWithErrorCheck(t, tempDir, "updateTestFile")
	defer helpers.RemoveFileWithErrorCheck(t, updateTestFile.Name())
	expectedUpdateFileContent := []byte("test update data")
	updateErr := os.WriteFile(updateTestFile.Name(), expectedUpdateFileContent, 0o600)
	require.NoError(t, updateErr)

	addTestFile := helpers.CreateFileWithErrorCheck(t, tempDir, "addTestFile")
	defer helpers.RemoveFileWithErrorCheck(t, addTestFile.Name())
	addErr := os.WriteFile(addTestFile.Name(), expectedFileContent, 0o600)
	require.NoError(t, addErr)

	// Go doesn't allow address of numeric constant
	deleteAction := mpi.File_FILE_ACTION_DELETE
	updateAction := mpi.File_FILE_ACTION_UPDATE
	addAction := mpi.File_FILE_ACTION_ADD

	// protos.GetFileMeta(deleteTestFile.Name(), GenerateHash([]byte("")))

	tests := []struct {
		name             string
		fileOverview     *mpi.FileOverview
		expectedDiff     map[string]*mpi.File
		expectedContents map[string][]byte
	}{
		{
			name: "Test 1: Delete, Add & Update Files",
			fileOverview: &mpi.FileOverview{
				Files: []*mpi.File{
					{
						FileMeta: protos.GetFileMeta(deleteTestFile.Name(), GenerateHash(expectedFileContent)),
						Action:   &deleteAction,
					},
					{
						FileMeta: protos.GetFileMeta(updateTestFile.Name(), GenerateHash(expectedFileContent)),
						Action:   &updateAction,
					},
					{
						FileMeta: protos.GetFileMeta(tempDir+"random/new/file", GenerateHash(expectedFileContent)),
						Action:   &addAction,
					},
				},
				ConfigVersion: &mpi.ConfigVersion{
					InstanceId: protos.GetNginxOssInstance([]string{}).GetInstanceMeta().GetInstanceId(),
					Version:    "a7d6580c-8ac9-376e-acde-b2cbed21d291",
				},
			},
			expectedContents: map[string][]byte{
				deleteTestFile.Name(): expectedFileContent,
				updateTestFile.Name(): expectedUpdateFileContent,
			},
			expectedDiff: map[string]*mpi.File{
				deleteTestFile.Name(): {
					FileMeta: protos.GetFileMeta(deleteTestFile.Name(), GenerateHash(expectedFileContent)),
					Action:   &deleteAction,
				},
				updateTestFile.Name(): {
					FileMeta: protos.GetFileMeta(updateTestFile.Name(), GenerateHash(expectedFileContent)),
					Action:   &updateAction,
				},
				tempDir + "random/new/file": {
					FileMeta: protos.GetFileMeta(tempDir+"random/new/file", GenerateHash(expectedFileContent)),
					Action:   &addAction,
				},
			},
		},
		{
			name: "Test 2: File Already Deleted, File Already Updated, File Already Added",
			fileOverview: &mpi.FileOverview{
				Files: []*mpi.File{
					{
						FileMeta: protos.GetFileMeta(tempDir+"deletedFile", GenerateHash(expectedFileContent)),
						Action:   &deleteAction,
					},
					{
						FileMeta: protos.GetFileMeta(updateTestFile.Name(), GenerateHash(expectedUpdateFileContent)),
						Action:   &updateAction,
					},
					{
						FileMeta: protos.GetFileMeta(addTestFile.Name(), GenerateHash(expectedUpdateFileContent)),
						Action:   &addAction,
					},
				},
				ConfigVersion: protos.CreateConfigVersion(),
			},
			expectedContents: map[string][]byte{
				addTestFile.Name(): expectedFileContent,
			},
			expectedDiff: map[string]*mpi.File{
				addTestFile.Name(): {
					FileMeta: protos.GetFileMeta(addTestFile.Name(), GenerateHash(expectedUpdateFileContent)),
					Action:   &updateAction,
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(tt *testing.T) {
			diff, contents, compareErr := CompareFileHash(test.fileOverview)

			assert.Equal(tt, test.expectedDiff, diff)
			assert.Equal(tt, test.expectedContents, contents)
			require.NoError(tt, compareErr)
		})
	}
}
