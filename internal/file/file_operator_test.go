// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package file

import (
	"context"
	"encoding/json"
	"os"
	"path"
	"path/filepath"
	"sync"
	"testing"

	"github.com/nginx/agent/v3/pkg/files"
	"github.com/nginx/agent/v3/test/protos"

	"github.com/nginx/agent/v3/internal/model"
	"github.com/nginx/agent/v3/test/helpers"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFileOperator_Write(t *testing.T) {
	ctx := context.Background()

	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "nginx.conf")
	fileContent, err := os.ReadFile("../../test/config/nginx/nginx.conf")
	require.NoError(t, err)
	defer helpers.RemoveFileWithErrorCheck(t, filePath)
	fileOp := NewFileOperator(&sync.RWMutex{})

	fileMeta := protos.FileMeta(filePath, files.GenerateHash(fileContent))

	writeErr := fileOp.Write(ctx, fileContent, fileMeta.GetName(), fileMeta.GetPermissions())
	require.NoError(t, writeErr)
	assert.FileExists(t, filePath)

	data, readErr := os.ReadFile(filePath)
	require.NoError(t, readErr)
	assert.Equal(t, fileContent, data)
}

func TestFileOperator_WriteManifestFile(t *testing.T) {
	tempDir := t.TempDir()
	manifestPath := path.Join(tempDir, "manifest.json")

	manifestFiles := map[string]*model.ManifestFile{
		"/etc/nginx/nginx.conf": {
			ManifestFileMeta: &model.ManifestFileMeta{
				Name:       "/etc/nginx/nginx.conf",
				Size:       1024,
				Hash:       "6d232d32d44",
				Referenced: true,
				Unmanaged:  false,
			},
		},
		"/etc/nginx/conf.d/default.conf": {
			ManifestFileMeta: &model.ManifestFileMeta{
				Name:       "/etc/nginx/conf.d/default.conf",
				Size:       32342,
				Hash:       "1eh32hd3792hd329",
				Referenced: true,
				Unmanaged:  false,
			},
		},
	}

	fileOperator := NewFileOperator(&sync.RWMutex{})
	err := fileOperator.WriteManifestFile(t.Context(), manifestFiles, tempDir, manifestPath)
	require.NoError(t, err)

	assert.FileExists(t, manifestPath)
	assert.NoFileExists(t, manifestPath+".tmp")

	// Verify the contents can be read back
	data, err := os.ReadFile(manifestPath)
	require.NoError(t, err)

	var readBack map[string]*model.ManifestFile
	err = json.Unmarshal(data, &readBack)
	require.NoError(t, err)
	assert.Equal(t, manifestFiles, readBack)
}

func TestFileOperator_WriteManifestFile_directoryCreationError(t *testing.T) {
	manifestPath := "/unknown/manifest.json"
	manifestDir := "/unknown"

	fileOperator := NewFileOperator(&sync.RWMutex{})
	err := fileOperator.WriteManifestFile(t.Context(), make(map[string]*model.ManifestFile), manifestDir, manifestPath)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unable to create directory")
}

func TestFileOperator_WriteManifestFile_tempFileCreationError(t *testing.T) {
	tempDir := t.TempDir()

	// Create a file where we want to write the manifest
	manifestPath := path.Join(tempDir, "manifest.json")
	err := os.WriteFile(manifestPath, []byte("existing"), 0o400) // readonly
	require.NoError(t, err)

	// Create readonly directory to prevent temp file creation
	err = os.Chmod(tempDir, 0o444)
	require.NoError(t, err)
	defer func() {
		revertPermissionsError := os.Chmod(tempDir, 0o755)
		require.NoError(t, revertPermissionsError)
	}()

	fileOperator := NewFileOperator(&sync.RWMutex{})
	err = fileOperator.WriteManifestFile(t.Context(), make(map[string]*model.ManifestFile), tempDir, manifestPath)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to open file")
}

func TestFileOperator_WriteManifestFile_fileMissing(t *testing.T) {
	tempDir := t.TempDir()
	manifestPath := "/unknown/manifest.json"

	fileOperator := NewFileOperator(&sync.RWMutex{})
	err := fileOperator.WriteManifestFile(t.Context(), make(map[string]*model.ManifestFile), tempDir, manifestPath)
	require.Error(t, err)

	assert.NoFileExists(t, manifestPath+".tmp")
}

func TestFileOperator_MoveFile_fileExists(t *testing.T) {
	tempDir := t.TempDir()
	tempFile := path.Join(tempDir, "/etc/nginx/nginx.conf")
	newFile := path.Join(tempDir, "/etc/nginx/new_test.conf")

	err := os.MkdirAll(path.Dir(tempFile), 0o755)
	require.NoError(t, err)

	_, err = os.Create(tempFile)
	require.NoError(t, err)

	fileOperator := NewFileOperator(&sync.RWMutex{})
	err = fileOperator.MoveFile(t.Context(), tempFile, newFile)
	require.NoError(t, err)

	assert.FileExists(t, newFile)
}

func TestFileOperator_MoveFile_sourceFileDoesNotExist(t *testing.T) {
	tempDir := t.TempDir()
	tempFile := path.Join(tempDir, "/etc/nginx/nginx.conf")
	newFile := path.Join(tempDir, "/etc/nginx/new_test.conf")

	fileOperator := NewFileOperator(&sync.RWMutex{})
	err := fileOperator.MoveFile(t.Context(), tempFile, newFile)
	require.Error(t, err)

	assert.NoFileExists(t, tempFile)
	assert.NoFileExists(t, newFile)
}

func TestFileOperator_MoveFile_destFileDoesNotExist(t *testing.T) {
	tempDir := t.TempDir()
	tempFile := path.Join(tempDir, "/etc/nginx/nginx.conf")
	newFile := "/unknown/nginx/new_test.conf"

	err := os.MkdirAll(path.Dir(tempFile), 0o755)
	require.NoError(t, err)

	_, err = os.Create(tempFile)
	require.NoError(t, err)

	fileOperator := NewFileOperator(&sync.RWMutex{})
	err = fileOperator.MoveFile(t.Context(), tempFile, newFile)
	require.Error(t, err)

	assert.FileExists(t, tempFile)
	assert.NoFileExists(t, newFile)
}
