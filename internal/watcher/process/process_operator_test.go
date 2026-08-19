// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package process

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewProcessOperator(t *testing.T) {
	po := NewProcessOperator()
	assert.NotNil(t, po)
}

func TestProcessOperator_Processes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		ctx       func() context.Context
		name      string
		wantError bool
	}{
		{
			name:      "Test 1: active context returns no error",
			ctx:       context.Background,
			wantError: false,
		},
		{
			name: "Test 2: canceled context returns error",
			ctx: func() context.Context {
				ctx, cancel := context.WithCancel(context.Background())
				cancel()

				return ctx
			},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			po := NewProcessOperator()
			_, err := po.Processes(tt.ctx())

			if tt.wantError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestProcessOperator_Process(t *testing.T) {
	t.Parallel()

	validPID := int32(os.Getpid())

	tests := []struct {
		name      string
		pid       int32
		wantError bool
		wantPID   int32
	}{
		{
			name:      "Test 1: current process PID returns populated struct",
			pid:       validPID,
			wantError: false,
			wantPID:   validPID,
		},
		{
			name:      "Test 2: invalid PID returns error and nil result",
			pid:       -1,
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			po := NewProcessOperator()
			proc, err := po.Process(context.Background(), tt.pid)

			if tt.wantError {
				require.Error(t, err)
				assert.Nil(t, proc)
			} else {
				require.NoError(t, err)
				require.NotNil(t, proc)
				assert.Equal(t, tt.wantPID, proc.PID)
				assert.NotEmpty(t, proc.Name)
			}
		})
	}
}
