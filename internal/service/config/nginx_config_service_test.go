// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package config

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"testing"

	helpers "github.com/nginx/agent/v3/test"

	"github.com/nginx/agent/v3/internal/config"

	"github.com/stretchr/testify/require"

	"github.com/nginx/agent/v3/internal/datasource/host/exec/execfakes"
	testconfig "github.com/nginx/agent/v3/test/config"

	"github.com/nginx/agent/v3/api/grpc/instances"
	"github.com/nginx/agent/v3/internal/model"
	"github.com/stretchr/testify/assert"
)

const instanceID = "7332d596-d2e6-4d1e-9e75-70f91ef9bd0e"

func TestNginx_ParseConfig(t *testing.T) {
	file, err := os.CreateTemp("./", "nginx-parse-config.conf")
	defer helpers.RemoveFileWithErrorCheck(t, file.Name())
	require.NoError(t, err)

	errorLog, err := os.CreateTemp("./", "error.log")
	defer helpers.RemoveFileWithErrorCheck(t, errorLog.Name())
	require.NoError(t, err)

	accessLog, err := os.CreateTemp("./", "access.log")
	defer helpers.RemoveFileWithErrorCheck(t, accessLog.Name())
	require.NoError(t, err)

	combinedAccessLog, err := os.CreateTemp("./", "combined_access.log")
	defer helpers.RemoveFileWithErrorCheck(t, combinedAccessLog.Name())
	require.NoError(t, err)

	ltsvAccessLog, err := os.CreateTemp("./", "ltsv_access.log")
	defer helpers.RemoveFileWithErrorCheck(t, ltsvAccessLog.Name())
	require.NoError(t, err)

	content, err := testconfig.GetNginxConfigWithMultipleAccessLogs(
		errorLog.Name(),
		accessLog.Name(),
		combinedAccessLog.Name(),
		ltsvAccessLog.Name(),
	)
	require.NoError(t, err)

	data := []byte(content)

	err = os.WriteFile(file.Name(), data, 0o600)
	require.NoError(t, err)

	expectedConfigContext := &model.NginxConfigContext{
		AccessLogs: []*model.AccessLog{
			{
				Name:        accessLog.Name(),
				Format:      "$remote_addr - $remote_user [$time_local]",
				Readable:    true,
				Permissions: "0600",
			},
			{
				Name: combinedAccessLog.Name(),
				Format: "$remote_addr - $remote_user [$time_local] " +
					"\"$request\" $status $body_bytes_sent \"$http_referer\" \"$http_user_agent\"",
				Readable:    true,
				Permissions: "0600",
			},
			{
				Name:        ltsvAccessLog.Name(),
				Format:      "ltsv",
				Readable:    true,
				Permissions: "0600",
			},
		},
		ErrorLogs: []*model.ErrorLog{
			{
				Name:        errorLog.Name(),
				LogLevel:    "notice",
				Readable:    true,
				Permissions: "0600",
			},
		},
	}

	instance := &instances.Instance{
		Type:       instances.Type_NGINX,
		InstanceId: instanceID,
		Meta: &instances.Meta{
			Meta: &instances.Meta_NginxMeta{
				NginxMeta: &instances.NginxMeta{
					ConfigPath: file.Name(),
				},
			},
		},
	}

	nginxConfig := NewNginx(instance, &config.Config{})
	result, err := nginxConfig.ParseConfig()

	require.NoError(t, err)
	assert.Equal(t, expectedConfigContext, result)
}

func TestValidateConfigCheckResponse(t *testing.T) {
	tests := []struct {
		name     string
		out      string
		expected interface{}
	}{
		{
			name:     "valid response",
			out:      "nginx [info]",
			expected: nil,
		},
		{
			name:     "err response",
			out:      "nginx [emerg]",
			expected: errors.New("error running nginx -t -c:\nnginx [emerg]"),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := validateConfigCheckResponse([]byte(test.out))
			assert.Equal(t, test.expected, err)
		})
	}
}

func TestNginx_Apply(t *testing.T) {
	tests := []struct {
		name     string
		error    error
		expected error
	}{
		{
			name:     "successful reload",
			error:    nil,
			expected: nil,
		},
		{
			name:     "failed reload",
			error:    errors.New("error reloading"),
			expected: fmt.Errorf("failed to reload NGINX %w: ", errors.New("error reloading")),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			mockExec := &execfakes.FakeExecInterface{}
			mockExec.KillProcessReturns(test.error)

			instance := &instances.Instance{
				Type:       instances.Type_NGINX,
				InstanceId: instanceID,
				Meta: &instances.Meta{
					Meta: &instances.Meta_NginxMeta{
						NginxMeta: &instances.NginxMeta{
							ExePath:   "nginx",
							ProcessId: 1,
						},
					},
				},
			}
			nginxConfig := NewNginx(instance, &config.Config{})
			nginxConfig.executor = mockExec

			err := nginxConfig.Apply()

			if test.error != nil {
				assert.Equal(t, fmt.Errorf("failed to reload NGINX, %w", test.error), err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestNginx_Validate(t *testing.T) {
	tests := []struct {
		name     string
		out      *bytes.Buffer
		error    error
		expected error
	}{
		{
			name:     "validate successful",
			out:      bytes.NewBufferString(""),
			error:    nil,
			expected: nil,
		},
		{
			name:     "validate failed",
			out:      bytes.NewBufferString("[emerg]"),
			error:    errors.New("error validating"),
			expected: fmt.Errorf("NGINX config test failed %w: [emerg]", errors.New("error validating")),
		},
		{
			name:     "validate Config failed",
			out:      bytes.NewBufferString("nginx [emerg]"),
			error:    nil,
			expected: fmt.Errorf("error running nginx -t -c:\nnginx [emerg]"),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			mockExec := &execfakes.FakeExecInterface{}
			mockExec.RunCmdReturns(test.out, test.error)
			instance := &instances.Instance{
				Type:       instances.Type_NGINX,
				InstanceId: instanceID,
				Meta: &instances.Meta{
					Meta: &instances.Meta_NginxMeta{
						NginxMeta: &instances.NginxMeta{
							ExePath: "nginx",
						},
					},
				},
			}
			nginxConfig := NewNginx(instance, &config.Config{})
			nginxConfig.executor = mockExec

			err := nginxConfig.Validate()

			assert.Equal(t, test.expected, err)
		})
	}
}
