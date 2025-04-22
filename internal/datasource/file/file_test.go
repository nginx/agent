// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package file

import (
	"os"
	"testing"

	"github.com/nginx/agent/v3/test/helpers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_RetrieveTokenFromFile(t *testing.T) {
	dir := t.TempDir()
	tokenFile := helpers.CreateFileWithErrorCheck(t, dir, "test-tkn")
	defer helpers.RemoveFileWithErrorCheck(t, tokenFile.Name())
	tests := []struct {
		name           string
		path           string
		expected       string
		expectedErrMsg string
		createToken    bool
	}{
		{
			name:           "Test 1: File exists",
			createToken:    true,
			path:           tokenFile.Name(),
			expected:       "test-tkn",
			expectedErrMsg: "",
		},
		{
			name:           "Test 2: File does not exist",
			createToken:    false,
			path:           "test-tkn",
			expected:       "",
			expectedErrMsg: "unable to read from file: open test-tkn: no such file or directory",
		},
		{
			name:           "Test 3: Empty path",
			createToken:    false,
			path:           "",
			expected:       "",
			expectedErrMsg: "failed to read file since file path is empty",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.createToken {
				writeErr := os.WriteFile(tokenFile.Name(), []byte("  test-tkn\n"), 0o600)
				require.NoError(t, writeErr)
			}

			token, err := ReadFromFile(tt.path)
			if err != nil {
				assert.Equal(t, tt.expectedErrMsg, err.Error())
			}
			assert.Equal(t, tt.expected, token)
		})
	}
}
