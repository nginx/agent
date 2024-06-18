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

	expected := &mpi.FileMeta{
		Name:        file.Name(),
		Hash:        "4ae71336-e44b-39bf-b9d2-752e234818a5",
		Permissions: "0777",
		Size:        0,
	}

	fileMeta, err := GetFileMeta(file.Name())
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

	permissions := GetPermissions(info.Mode())

	assert.Equal(t, "0600", permissions)
}

func Test_GenerateConfigVersion(t *testing.T) {
	tests := []struct {
		name     string
		input    []*mpi.File
		expected string
	}{
		{
			name:     "Test with empty file slice",
			input:    []*mpi.File{},
			expected: GenerateHash([]byte{}),
		},
		{
			name: "Test with one file",
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
			name: "Test with multiple files",
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
			name:     "Test with empty byte slice",
			input:    []byte{},
			expected: uuid.NewMD5(uuid.Nil, []byte("")).String(),
		},
		{
			name:     "Test with non-empty byte slice",
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

func TestReadFile(t *testing.T) {
	testFile := helpers.CreateFileWithErrorCheck(t, os.TempDir(), "testFile")
	defer helpers.RemoveFileWithErrorCheck(t, testFile.Name())
	expectedFileContent := []byte("test data")
	err := os.WriteFile(testFile.Name(), expectedFileContent, 0o600)
	require.NoError(t, err)

	resultContent, _, err := GenerateHashWithReadFile(testFile.Name())
	require.NoError(t, err)

	assert.Equal(t, expectedFileContent, resultContent)
}

func TestCompareFileHash_Delete(t *testing.T) {
	tempDir := os.TempDir()

	deleteTestFile := helpers.CreateFileWithErrorCheck(t, tempDir, "deleteTestFile")
	defer helpers.RemoveFileWithErrorCheck(t, deleteTestFile.Name())
	expectedFileContent := []byte("test data")
	err := os.WriteFile(deleteTestFile.Name(), expectedFileContent, 0o600)
	require.NoError(t, err)

	updateTestFile := helpers.CreateFileWithErrorCheck(t, tempDir, "updateTestFile")
	defer helpers.RemoveFileWithErrorCheck(t, updateTestFile.Name())
	expectedUpdateFileContent := []byte("test update data")
	updateErr := os.WriteFile(updateTestFile.Name(), expectedUpdateFileContent, 0o600)
	require.NoError(t, updateErr)

	addTestFile := helpers.CreateFileWithErrorCheck(t, tempDir, "addTestFile")
	defer helpers.RemoveFileWithErrorCheck(t, addTestFile.Name())
	expectedAddFileContent := []byte("test data")
	addErr := os.WriteFile(addTestFile.Name(), expectedAddFileContent, 0o600)
	require.NoError(t, addErr)

	// Go doesn't allow address of numeric constant
	deleteAction := mpi.File_FILE_ACTION_DELETE
	updateAction := mpi.File_FILE_ACTION_UPDATE
	addAction := mpi.File_FILE_ACTION_ADD

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
						FileMeta: &mpi.FileMeta{
							Name:         deleteTestFile.Name(),
							Hash:         "f0ebf313-853b-3582-b74a-eff115f6e4d3",
							ModifiedTime: nil,
							Permissions:  "",
							Size:         0,
						},
						Action: &deleteAction,
					},
					{
						FileMeta: &mpi.FileMeta{
							Name:         updateTestFile.Name(),
							Hash:         "ff8dcd5d-a12f-3895-a6b9-2ac8c98bfd08",
							ModifiedTime: nil,
							Permissions:  "",
							Size:         0,
						},
						Action: &updateAction,
					},
					{
						FileMeta: &mpi.FileMeta{
							Name:         tempDir + "newFileName",
							Hash:         "ff8dcd5d-a12f-3895-a6b9-2ac8c98bfd08",
							ModifiedTime: nil,
							Permissions:  "",
							Size:         0,
						},
						Action: &addAction,
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
					FileMeta: &mpi.FileMeta{
						Name:         deleteTestFile.Name(),
						Hash:         "f0ebf313-853b-3582-b74a-eff115f6e4d3",
						ModifiedTime: nil,
						Permissions:  "",
						Size:         0,
					},
					Action: &deleteAction,
				},
				updateTestFile.Name(): {
					FileMeta: &mpi.FileMeta{
						Name:         updateTestFile.Name(),
						Hash:         "ff8dcd5d-a12f-3895-a6b9-2ac8c98bfd08",
						ModifiedTime: nil,
						Permissions:  "",
						Size:         0,
					},
					Action: &updateAction,
				},
				tempDir + "newFileName": {
					FileMeta: &mpi.FileMeta{
						Name:         tempDir + "newFileName",
						Hash:         "ff8dcd5d-a12f-3895-a6b9-2ac8c98bfd08",
						ModifiedTime: nil,
						Permissions:  "",
						Size:         0,
					},
					Action: &addAction,
				},
			},
		},
		{
			name: "Test 2: File Already Deleted, File Already Updated, File Already Added",
			fileOverview: &mpi.FileOverview{
				Files: []*mpi.File{
					{
						FileMeta: &mpi.FileMeta{
							Name:         tempDir + "deletedFile",
							Hash:         "f0ebf313-853b-3582-b74a-eff115f6e4d3",
							ModifiedTime: nil,
							Permissions:  "",
							Size:         0,
						},
						Action: &deleteAction,
					},
					{
						FileMeta: &mpi.FileMeta{
							Name:         updateTestFile.Name(),
							Hash:         "3ea160ae-b15e-3ce6-ac61-5b27f926c8b0",
							ModifiedTime: nil,
							Permissions:  "",
							Size:         0,
						},
						Action: &updateAction,
					},
					{
						FileMeta: &mpi.FileMeta{
							Name:         addTestFile.Name(),
							Hash:         "3ea160ae-b15e-3ce6-ac61-5b27f926c8b0",
							ModifiedTime: nil,
							Permissions:  "",
							Size:         0,
						},
						Action: &addAction,
					},
				},
				ConfigVersion: &mpi.ConfigVersion{
					InstanceId: protos.GetNginxOssInstance([]string{}).GetInstanceMeta().GetInstanceId(),
					Version:    "a7d6580c-8ac9-376e-acde-b2cbed21d291",
				},
			},
			expectedContents: map[string][]byte{
				addTestFile.Name(): expectedAddFileContent,
			},
			expectedDiff: map[string]*mpi.File{
				addTestFile.Name(): {
					FileMeta: &mpi.FileMeta{
						Name:         addTestFile.Name(),
						Hash:         "3ea160ae-b15e-3ce6-ac61-5b27f926c8b0",
						ModifiedTime: nil,
						Permissions:  "",
						Size:         0,
					},
					Action: &updateAction,
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
