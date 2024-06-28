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
	"testing"

	"github.com/nginx/agent/v3/internal/datasource/host/exec/execfakes"
	"github.com/nginx/agent/v3/test/helpers"
	"github.com/nginx/agent/v3/test/protos"
	"github.com/stretchr/testify/assert"
)

func TestInstanceOperator_ValidateConfigCheckResponse(t *testing.T) {
	tests := []struct {
		name     string
		out      string
		expected interface{}
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
			operator := NewInstanceOperator()

			err := operator.validateConfigCheckResponse([]byte(test.out))
			assert.Equal(t, test.expected, err)
		})
	}
}

func TestInstanceOperator_Validate(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name     string
		out      *bytes.Buffer
		error    error
		expected error
	}{
		{
			name:     "Test 1: Validate successful",
			out:      bytes.NewBufferString(""),
			error:    nil,
			expected: nil,
		},
		{
			name:     "Test 2: Validate failed",
			out:      bytes.NewBufferString("[emerg]"),
			error:    errors.New("error validating"),
			expected: fmt.Errorf("NGINX config test failed %w: [emerg]", errors.New("error validating")),
		},
		{
			name:     "Test 3: Validate Config failed",
			out:      bytes.NewBufferString("nginx [emerg]"),
			error:    nil,
			expected: fmt.Errorf("error running nginx -t -c:\nnginx [emerg]"),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			mockExec := &execfakes.FakeExecInterface{}
			mockExec.RunCmdReturns(test.out, test.error)

			instance := protos.GetNginxOssInstance([]string{})

			operator := NewInstanceOperator()
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
		name     string
		error    error
		expected error
	}{
		{
			name:     "Test 1: Successful reload",
			error:    nil,
			expected: nil,
		},
		{
			name:     "Test 2: Failed reload",
			error:    errors.New("error reloading"),
			expected: errors.New("error reloading"),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			mockExec := &execfakes.FakeExecInterface{}
			mockExec.KillProcessReturns(test.error)

			instance := protos.GetNginxOssInstance([]string{})

			operator := NewInstanceOperator()
			operator.executer = mockExec

			err := operator.Reload(ctx, instance)
			assert.Equal(t, test.expected, err)
		})
	}
}
