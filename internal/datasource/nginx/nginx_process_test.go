// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package nginx

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/nginx/agent/v3/internal/datasource/host/exec/execfakes"
	"github.com/stretchr/testify/assert"
)

func TestGetExe(t *testing.T) {
	tests := []struct {
		name          string
		commandOutput []byte
		commandError  error
		expected      string
	}{
		{
			name:          "Default exe if error executing command -v nginx",
			commandOutput: []byte{},
			commandError:  fmt.Errorf("command error"),
			expected:      "/usr/bin/nginx",
		},
		{
			name:          "Sanitize Exe Deleted Path",
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

			n := New(mockExec)
			result := n.GetExe()

			assert.Equal(tt, test.expected, result)
		})
	}
}
