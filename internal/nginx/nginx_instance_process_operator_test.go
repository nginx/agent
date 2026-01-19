// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package nginx

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/nginx/agent/v3/pkg/host/exec/execfakes"
	"github.com/nginx/agent/v3/pkg/nginxprocess"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProcessOperator_FindParentProcessID(t *testing.T) {
	ctx := context.Background()

	modulePath := t.TempDir() + "/usr/lib/nginx/modules"

	configArgs := fmt.Sprintf(ossConfigArgs, modulePath)
	nginxVersionCommandOutput := `nginx version: nginx/1.25.3
					built by clang 14.0.0 (clang-1400.0.29.202)
					built with OpenSSL 1.1.1s  1 Nov 2022 (running with OpenSSL 1.1.1t  7 Feb 2023)
					TLS SNI support enabled
					configure arguments: ` + configArgs

	tests := []struct {
		name           string                  // 16 bytes
		instanceID     string                  // 16 bytes
		expectErr      error                   // 16 bytes (interface)
		nginxProcesses []*nginxprocess.Process // 24 bytes (slice header)
		expectedPPID   int32
	}{
		{
			name:         "Test 1: Found parent process",
			instanceID:   "e1374cb1-462d-3b6c-9f3b-f28332b5f10c",
			expectErr:    nil,
			expectedPPID: 1234,
			nginxProcesses: []*nginxprocess.Process{
				{
					PID:     567,
					Created: time.Date(2025, 8, 13, 8, 1, 0, 0, time.Local),
					PPID:    1234,
					Name:    "nginx",
					Cmd:     "nginx: worker process",
					Exe:     exePath,
				},
				{
					PID:     789,
					PPID:    1234,
					Created: time.Date(2025, 8, 13, 8, 1, 0, 0, time.Local),
					Name:    "nginx",
					Cmd:     "nginx: worker process",
					Exe:     exePath,
				},
				{
					PID:     1234,
					Created: time.Date(2025, 8, 13, 8, 1, 0, 0, time.Local),
					PPID:    1,
					Name:    "nginx",
					Cmd:     "nginx: master process /usr/local/opt/nginx/bin/nginx -g daemon off;",
					Exe:     exePath,
				},
			},
		},
		{
			name:         "Test 2: unable to find parent process",
			instanceID:   "e1374cb1-462d-3b6c-9f3b-f28332b5f10c",
			expectErr:    errors.New("unable to find parent process"),
			expectedPPID: 0,
			nginxProcesses: []*nginxprocess.Process{
				{
					PID:     567,
					Created: time.Date(2025, 8, 13, 8, 1, 0, 0, time.Local),
					PPID:    1234,
					Name:    "nginx",
					Cmd:     "nginx: worker process",
					Exe:     exePath,
				},
				{
					PID:     789,
					PPID:    1234,
					Created: time.Date(2025, 8, 13, 8, 1, 0, 0, time.Local),
					Name:    "nginx",
					Cmd:     "nginx: worker process",
					Exe:     exePath,
				},
				{
					PID:     4567,
					Created: time.Date(2025, 8, 13, 8, 1, 0, 0, time.Local),
					PPID:    1,
					Name:    "nginx",
					Cmd:     "nginx: master process /usr/local/opt/nginx/bin/nginx -g daemon off;",
					Exe:     exePath2,
				},
			},
		},
		{
			name:         "Test 3: Found parent process, multiple NGINX processes",
			instanceID:   "e1374cb1-462d-3b6c-9f3b-f28332b5f10c",
			expectErr:    nil,
			expectedPPID: 1234,
			nginxProcesses: []*nginxprocess.Process{
				{
					PID:     567,
					Created: time.Date(2025, 8, 13, 8, 1, 0, 0, time.Local),
					PPID:    1234,
					Name:    "nginx",
					Cmd:     "nginx: worker process",
					Exe:     exePath,
				},
				{
					PID:     789,
					PPID:    1234,
					Created: time.Date(2025, 8, 13, 8, 1, 0, 0, time.Local),
					Name:    "nginx",
					Cmd:     "nginx: worker process",
					Exe:     exePath,
				},
				{
					PID:     1234,
					Created: time.Date(2025, 8, 13, 8, 1, 0, 0, time.Local),
					PPID:    1,
					Name:    "nginx",
					Cmd:     "nginx: master process /usr/local/opt/nginx/bin/nginx -g daemon off;",
					Exe:     exePath,
				},
				{
					PID:     5678,
					Created: time.Date(2025, 8, 13, 5, 1, 0, 0, time.Local),
					PPID:    1,
					Name:    "nginx",
					Cmd:     "nginx: master process /usr/local/opt/nginx/bin/nginx -g daemon off;",
					Exe:     exePath2,
				},
				{
					PID:     567,
					Created: time.Date(2025, 8, 13, 8, 1, 0, 0, time.Local),
					PPID:    1234,
					Name:    "nginx",
					Cmd:     "nginx: worker process",
					Exe:     exePath,
				},
				{
					PID:     789,
					PPID:    1234,
					Created: time.Date(2025, 8, 13, 8, 1, 0, 0, time.Local),
					Name:    "nginx",
					Cmd:     "nginx: worker process",
					Exe:     exePath,
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(tt *testing.T) {
			processOperator := NewNginxInstanceProcessOperator()
			mockExec := &execfakes.FakeExecInterface{}
			mockExec.RunCmdReturnsOnCall(0, bytes.NewBufferString(nginxVersionCommandOutput), nil)
			ppid, err := processOperator.FindParentProcessID(ctx, test.instanceID, test.nginxProcesses, mockExec)

			if test.expectErr != nil {
				require.Error(tt, err)
			} else {
				require.NoError(tt, err)
			}

			assert.Equal(tt, test.expectedPPID, ppid)
		})
	}
}
