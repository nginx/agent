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

	expected := protos.FileMeta(file.Name(), GenerateHash([]byte("")))

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
		expected string
		input    []*mpi.File
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
		expected string
		input    []byte
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
