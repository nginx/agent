// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package resource

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/nginx/agent/v3/internal/datasource/host/exec/execfakes"
	"github.com/nginx/agent/v3/test/helpers"
	"github.com/nginx/agent/v3/test/protos"
	"github.com/nginx/agent/v3/test/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInstanceOperator_ValidateConfigCheckResponse(t *testing.T) {
	tests := []struct {
		expected interface{}
		name     string
		out      string
	}{
		{
			name:     "Test 1: Valid response",
			out:      "nginx [info]",
			expected: nil,
		},
		{
			name:     "Test 2: Error response",
			out:      "nginx [emerg]",
			expected: errors.New("error running nginx -t -c:\nnginx [emerg]"),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			operator := NewInstanceOperator(types.AgentConfig())

			err := operator.validateConfigCheckResponse([]byte(test.out))
			assert.Equal(t, test.expected, err)
		})
	}
}

func TestInstanceOperator_Validate(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		out      *bytes.Buffer
		err      error
		expected error
		name     string
	}{
		{
			name:     "Test 1: Validate successful",
			out:      bytes.NewBufferString(""),
			err:      nil,
			expected: nil,
		},
		{
			name:     "Test 2: Validate failed",
			out:      bytes.NewBufferString("[emerg]"),
			err:      errors.New("error validating"),
			expected: fmt.Errorf("NGINX config test failed %w: [emerg]", errors.New("error validating")),
		},
		{
			name:     "Test 3: Validate Config failed",
			out:      bytes.NewBufferString("nginx [emerg]"),
			err:      nil,
			expected: errors.New("error running nginx -t -c:\nnginx [emerg]"),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			mockExec := &execfakes.FakeExecInterface{}
			mockExec.RunCmdReturns(test.out, test.err)

			instance := protos.NginxOssInstance([]string{})

			operator := NewInstanceOperator(types.AgentConfig())
			operator.executer = mockExec

			err := operator.Validate(ctx, instance)

			assert.Equal(t, test.expected, err)
		})
	}
}

func TestInstanceOperator_Reload(t *testing.T) {
	ctx := context.Background()

	errorLogFile := helpers.CreateFileWithErrorCheck(t, t.TempDir(), "error.log")
	defer helpers.RemoveFileWithErrorCheck(t, errorLogFile.Name())

	tests := []struct {
		err      error
		expected error
		name     string
	}{
		{
			name:     "Test 1: Successful reload",
			err:      nil,
			expected: nil,
		},
		{
			name:     "Test 2: Failed reload",
			err:      errors.New("error reloading"),
			expected: errors.New("error reloading"),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			mockExec := &execfakes.FakeExecInterface{}
			mockExec.KillProcessReturns(test.err)

			instance := protos.NginxOssInstance([]string{})

			operator := NewInstanceOperator(types.AgentConfig())
			operator.executer = mockExec

			err := operator.Reload(ctx, instance)
			assert.Equal(t, test.expected, err)
		})
	}
}

func TestInstanceOperator_ReloadAndMonitor(t *testing.T) {
	ctx := context.Background()

	errorLogFile := helpers.CreateFileWithErrorCheck(t, t.TempDir(), "error.log")
	defer helpers.RemoveFileWithErrorCheck(t, errorLogFile.Name())

	tests := []struct {
		expectedErr     error
		name            string
		errorLogs       string
		errorLogContent string
	}{
		{
			name:            "Test 1: Successful reload",
			errorLogs:       errorLogFile.Name(),
			errorLogContent: "",
			expectedErr:     nil,
		},
		{
			name:            "Test 2: Failed reload - error in logs",
			errorLogs:       errorLogFile.Name(),
			errorLogContent: errorLogLine,
			expectedErr:     errors.Join(fmt.Errorf("%s", errorLogLine)),
		},
		{
			name:            "Test 3: Successful reload - no error log",
			errorLogs:       "",
			errorLogContent: "",
			expectedErr:     nil,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(tt *testing.T) {
			mockExec := &execfakes.FakeExecInterface{}
			mockExec.KillProcessReturns(nil)

			instance := protos.NginxOssInstance([]string{})
			if test.errorLogs != "" {
				instance.GetInstanceRuntime().GetNginxRuntimeInfo().ErrorLogs = []string{test.errorLogs}
			}

			agentConfig := types.AgentConfig()
			agentConfig.DataPlaneConfig.Nginx.ReloadMonitoringPeriod = 10 * time.Second
			operator := NewInstanceOperator(types.AgentConfig())
			operator.executer = mockExec

			var wg sync.WaitGroup
			wg.Add(1)
			go func(expected error) {
				defer wg.Done()
				reloadError := operator.Reload(ctx, instance)
				assert.Equal(tt, expected, reloadError)
			}(test.expectedErr)

			time.Sleep(200 * time.Millisecond)

			if test.errorLogContent != "" {
				_, err := errorLogFile.WriteString(test.errorLogContent)
				require.NoError(tt, err, "Error writing data to error log file")
			}

			wg.Wait()
		})
	}
}
