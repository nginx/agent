// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package file

import (
	"context"
	"os"
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

func TestFileOperator_WriteManifestFile_fileMissing(t *testing.T) {
	tempDir := t.TempDir()
	manifestPath := "/unknown/manifest.json"

	fileOperator := NewFileOperator(&sync.RWMutex{})
	err := fileOperator.WriteManifestFile(t.Context(), make(map[string]*model.ManifestFile), tempDir, manifestPath)
	assert.Error(t, err)
}
