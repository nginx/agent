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

	"github.com/nginx/agent/v3/pkg/host/exec/execfakes"
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

func TestExeFromCmdline(t *testing.T) {
	tests := []struct {
		cmd      string
		expected string
		name     string
	}{
		{
			name: "Test 1: Extract exe from rootless NGINX",
			cmd: "nginx: master process /home/rootless/myNGINX/usr/sbin/nginx -p /home/rootless/myNGINX " +
				"-c /home/rootless/myNGINX/etc/nginx/nginx.conf",
			expected: "/home/rootless/myNGINX/usr/sbin/nginx",
		},
		{
			name:     "Test 2: Extract exe from regular NGINX",
			cmd:      "nginx: master process /usr/sbin/nginx -g daemon off;",
			expected: "/usr/sbin/nginx",
		},
		{
			name:     "Test 3: Return empty for worker process",
			cmd:      "nginx: worker process",
			expected: "",
		},
		{
			name:     "Test 4: Return empty for non-NGINX process",
			cmd:      "some other process",
			expected: "",
		},
		{
			name:     "Test 5: Return empty for empty string",
			cmd:      "",
			expected: "",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := exeFromCmdline(test.cmd)
			assert.Equal(t, test.expected, result)
		})
	}
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
