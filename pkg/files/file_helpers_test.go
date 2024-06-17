// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package files

import (
	"os"
	"testing"

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
		Hash:        "47DEQpj8HBSa+/TImW+5JCeuQeRkm5NMpJWZG3hSuFU=",
		Permissions: "-rw-------",
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
	expectedConfigVersion := "a7d6580c-8ac9-376e-acde-b2cbed21d291"

	file1 := &mpi.File{
		FileMeta: &mpi.FileMeta{
			Name: "file1",
			Hash: "3151431543",
		},
	}
	file2 := &mpi.File{
		FileMeta: &mpi.FileMeta{
			Name: "file2",
			Hash: "4234235325",
		},
	}

	files := []*mpi.File{
		file1,
		file2,
	}

	configVersion := GenerateConfigVersion(files)
	assert.Equal(t, expectedConfigVersion, configVersion)

	// Reorder files to make sure version is still the same
	files = []*mpi.File{
		file2,
		file1,
	}

	configVersion = GenerateConfigVersion(files)
	assert.Equal(t, expectedConfigVersion, configVersion)
}

func Test_GenerateFileHash(t *testing.T) {
	testFile := helpers.CreateFileWithErrorCheck(t, os.TempDir(), "testFile")
	defer helpers.RemoveFileWithErrorCheck(t, testFile.Name())
	err := os.WriteFile(testFile.Name(), []byte("test data"), 0o600)
	require.NoError(t, err)

	hash, err := GenerateFileHash(testFile.Name())
	require.NoError(t, err)

	assert.Equal(t, "kW8AJ6V1B0znKjMXd8NHjWUT94alkb2JLaGld78jNfk=", hash)
}

func Test_GenerateFileHashWithContent(t *testing.T) {
	testFile := helpers.CreateFileWithErrorCheck(t, os.TempDir(), "testFile")
	defer helpers.RemoveFileWithErrorCheck(t, testFile.Name())
	err := os.WriteFile(testFile.Name(), []byte("test data"), 0o600)
	require.NoError(t, err)

	content, err := os.ReadFile(testFile.Name())
	require.NoError(t, err)

	hash, err := GenerateFileHashWithContent(content)
	require.NoError(t, err)

	assert.Equal(t, "kW8AJ6V1B0znKjMXd8NHjWUT94alkb2JLaGld78jNfk=", hash)
}

func TestReadFile(t *testing.T) {
	testFile := helpers.CreateFileWithErrorCheck(t, os.TempDir(), "testFile")
	defer helpers.RemoveFileWithErrorCheck(t, testFile.Name())
	expectedFileContent := []byte("test data")
	err := os.WriteFile(testFile.Name(), expectedFileContent, 0o600)
	require.NoError(t, err)

	resultContent, _, err := ReadFileGenerateFile(testFile.Name())
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
	expectedUpdateFileContent := []byte("test data")
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
							Hash:         "kW8AJ6V1B0znKjMXd8NHjWUT94alkb2JLaGld78jNfk=",
							ModifiedTime: nil,
							Permissions:  "",
							Size:         0,
						},
						Action: &deleteAction,
					},
					{
						FileMeta: &mpi.FileMeta{
							Name:         updateTestFile.Name(),
							Hash:         "5xV2LmHSjZou6Bx50v/YRM4tlQ2AtR2mnqbJ/mx3e/w=",
							ModifiedTime: nil,
							Permissions:  "",
							Size:         0,
						},
						Action: &updateAction,
					},
					{
						FileMeta: &mpi.FileMeta{
							Name:         tempDir + "newFileName",
							Hash:         "5xV2LmHSjZou6Bx50v/YRM4tlQ2AtR2mnqbJ/mx3e/w=",
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
						Hash:         "kW8AJ6V1B0znKjMXd8NHjWUT94alkb2JLaGld78jNfk=",
						ModifiedTime: nil,
						Permissions:  "",
						Size:         0,
					},
					Action: &deleteAction,
				},
				updateTestFile.Name(): {
					FileMeta: &mpi.FileMeta{
						Name:         updateTestFile.Name(),
						Hash:         "5xV2LmHSjZou6Bx50v/YRM4tlQ2AtR2mnqbJ/mx3e/w=",
						ModifiedTime: nil,
						Permissions:  "",
						Size:         0,
					},
					Action: &updateAction,
				},
				tempDir + "newFileName": {
					FileMeta: &mpi.FileMeta{
						Name:         tempDir + "newFileName",
						Hash:         "5xV2LmHSjZou6Bx50v/YRM4tlQ2AtR2mnqbJ/mx3e/w=",
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
							Hash:         "kW8AJ6V1B0znKjMXd8NHjWUT94alkb2JLaGld78jNfk=",
							ModifiedTime: nil,
							Permissions:  "",
							Size:         0,
						},
						Action: &deleteAction,
					},
					{
						FileMeta: &mpi.FileMeta{
							Name:         updateTestFile.Name(),
							Hash:         "kW8AJ6V1B0znKjMXd8NHjWUT94alkb2JLaGld78jNfk=",
							ModifiedTime: nil,
							Permissions:  "",
							Size:         0,
						},
						Action: &updateAction,
					},
					{
						FileMeta: &mpi.FileMeta{
							Name:         addTestFile.Name(),
							Hash:         "5xV2LmHSjZou6Bx50v/YRM4tlQ2AtR2mnqbJ/mx3e/w=",
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
						Hash:         "5xV2LmHSjZou6Bx50v/YRM4tlQ2AtR2mnqbJ/mx3e/w=",
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
