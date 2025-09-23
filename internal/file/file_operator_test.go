// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package file

import (
	"context"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"
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

func TestFileOperator_WriteManifestFile_fileMissing(t *testing.T) {
	tempDir := t.TempDir()
	manifestPath := "/unknown/manifest.json"

	fileOperator := NewFileOperator(&sync.RWMutex{})
	err := fileOperator.WriteManifestFile(t.Context(), make(map[string]*model.ManifestFile), tempDir, manifestPath)
	assert.Error(t, err)
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

	assert.NoFileExists(t, tempFile)
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

func TestFileOperator_runHelper(t *testing.T) {
	// Common setup for all subtests.
	fo := NewFileOperator(&sync.RWMutex{})
	ctx := context.Background()
	tmpDir := t.TempDir()
	helperPath := filepath.Join(tmpDir, "helper-script")

	createScript := func(content string, permissions os.FileMode) {
		err := os.WriteFile(helperPath, []byte(content), permissions)
		require.NoError(t, err)
	}

	t.Run("Test 1 : Success", func(t *testing.T) {
		script := "#!/bin/sh\nprintf 'test content'"
		createScript(script, 0o755)

		tmpFilePath, err := fo.runHelper(ctx, helperPath, "http://example.com", 100)
		require.NoError(t, err)
		defer os.Remove(tmpFilePath)

		data, readErr := os.ReadFile(tmpFilePath)
		require.NoError(t, readErr)
		assert.Equal(t, "test content", string(data))
	})

	t.Run("Test 2 : ExceedsMaxBytes", func(t *testing.T) {
		largeContent := strings.Repeat("a", 101)
		script := fmt.Sprintf("#!/bin/sh\nprintf '%%s' '%s'", largeContent)
		createScript(script, 0o755)

		_, err := fo.runHelper(ctx, helperPath, "http://example.com", 100)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "exceeds the maximum allowed size")
	})

	t.Run("Test 3 : JSONError", func(t *testing.T) {
		jsonError := `{"error":"HelperExecutionError","message":"URL not found"}`
		script := fmt.Sprintf("#!/bin/sh\nprintf '%%s' '%s' 1>&2\nexit 1", jsonError)
		createScript(script, 0o755)

		_, err := fo.runHelper(ctx, helperPath, "http://example.com", 100)
		require.Error(t, err)
		assert.EqualError(t, err, "helper process failed with error 'HelperExecutionError' and message 'URL not found'")
	})

	os.RemoveAll(tmpDir)
}
