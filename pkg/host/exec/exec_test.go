// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package exec

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRunCmd(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		cmd       string
		args      []string
		wantError bool
	}{
		{
			name:      "Test 1: valid command returns output",
			cmd:       "/bin/ls",
			wantError: false,
		},
		{
			name:      "Test 2: non-existent command returns error",
			cmd:       "/bin/this-does-not-exist-XYZ",
			wantError: true,
		},
		{
			name:      "Test 3: command with non-zero exit returns error and output",
			cmd:       "/bin/ls",
			args:      []string{"/no-such-path-XYZ"},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ex := Exec{}
			output, err := ex.RunCmd(context.Background(), tt.cmd, tt.args...)
			// output buffer is always returned even on error
			require.NotNil(t, output)
			if tt.wantError {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.NotEmpty(t, output.String())
			}
		})
	}
}

func TestFindExecutable(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		exec      string
		wantError bool
	}{
		{
			name:      "Test 1: known executable is found",
			exec:      "ls",
			wantError: false,
		},
		{
			name:      "Test 2: non-existent executable returns error",
			exec:      "this-does-not-exist-XYZ",
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ex := Exec{}
			p, err := ex.FindExecutable(tt.exec)
			if tt.wantError {
				assert.Error(t, err)
				assert.Empty(t, p)
			} else {
				require.NoError(t, err)
				assert.NotEmpty(t, p)
			}
		})
	}
}
