// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package file

import (
	"context"
	"fmt"
	"os"
	"testing"

	mpi "github.com/nginx/agent/v3/api/grpc/mpi/v1"
	"github.com/nginx/agent/v3/test/helpers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFileOperator_Write(t *testing.T) {
	ctx := context.Background()

	tempDir := t.TempDir()
	filePath := fmt.Sprintf("%s/nginx.conf", tempDir)
	fileContent := []byte("location /test {\n    return 200 \"Test location\\n\";\n}")
	fileOp := NewFileOperator()
	fileMeta := &mpi.FileMeta{
		Name:         filePath,
		Hash:         "kW8AJ6V1B0znKjMXd8NHjWUT94alkb2JLaGld78jNfk=",
		ModifiedTime: nil,
		Permissions:  "0644",
		Size:         0,
	}

	err := fileOp.Write(ctx, fileContent, fileMeta)
	require.NoError(t, err)
	assert.FileExists(t, filePath)

	data, err := os.ReadFile(filePath)
	require.NoError(t, err)
	assert.Equal(t, fileContent, data)

	helpers.RemoveFileWithErrorCheck(t, filePath)
	assert.NoFileExists(t, filePath)
}
