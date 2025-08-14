// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package nginx

import (
	"bytes"
	"context"
	"errors"
	"testing"

	"github.com/nginx/agent/v3/internal/datasource/host/exec/execfakes"
	"github.com/stretchr/testify/assert"
)

func TestGetConfigPathFromCommand(t *testing.T) {
	result := ConfPathFromCommand("nginx: master process nginx -c /tmp/nginx.conf")
	assert.Equal(t, "/tmp/nginx.conf", result)

	result = ConfPathFromCommand("nginx: master process nginx -c")
	assert.Empty(t, result)

	result = ConfPathFromCommand("")
	assert.Empty(t, result)
}

func TestNginxProcessParser_GetExe(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		commandError  error
		name          string
		expected      string
		commandOutput []byte
	}{
		{
			name:          "Test 1: Default exe if error executing command -v nginx",
			commandOutput: []byte{},
			commandError:  errors.New("command error"),
			expected:      "/usr/bin/nginx",
		},
		{
			name:          "Test 2: Sanitize Exe Deleted Path",
			commandOutput: []byte("/usr/sbin/nginx (deleted)"),
			commandError:  nil,
			expected:      "/usr/sbin/nginx",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(tt *testing.T) {
			mockExec := &execfakes.FakeExecInterface{}
			mockExec.RunCmdReturns(bytes.NewBuffer(test.commandOutput), test.commandError)
			mockExec.FindExecutableReturns("/usr/bin/nginx", nil)

			result := Exe(ctx, mockExec)

			assert.Equal(tt, test.expected, result)
		})
	}
}
