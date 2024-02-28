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
	"sync"
	"testing"
	"time"

	helpers "github.com/nginx/agent/v3/test"

	"github.com/nginx/agent/v3/internal/config"

	"github.com/stretchr/testify/require"

	"github.com/nginx/agent/v3/internal/datasource/host/exec/execfakes"
	testconfig "github.com/nginx/agent/v3/test/config"

	"github.com/nginx/agent/v3/api/grpc/instances"
	"github.com/nginx/agent/v3/internal/model"
	"github.com/stretchr/testify/assert"
)

const (
	errorLogLine   = "2023/03/14 14:16:23 [emerg] 3871#3871: bind() to 0.0.0.0:8081 failed (98: Address already in use)"
	warningLogLine = "2023/03/14 14:16:23 nginx: [warn] 2048 worker_connections exceed open file resource limit: 1024"
	instanceID     = "7332d596-d2e6-4d1e-9e75-70f91ef9bd0e"
)

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
	errorLogFile, err := os.CreateTemp(".", "error.log")
	require.NoError(t, err)
	defer os.Remove(errorLogFile.Name())

	tests := []struct {
		name             string
		out              *bytes.Buffer
		errorLogs        []*model.ErrorLog
		errorLogContents string
		error            error
		expected         error
	}{
		{
			name: "successful reload",
			out:  bytes.NewBufferString(""),
			errorLogs: []*model.ErrorLog{
				{
					Name: errorLogFile.Name(),
				},
			},
			errorLogContents: "",
			error:            nil,
			expected:         nil,
		},
		{
			name: "successful reload - unknown error log location",
			out:  bytes.NewBufferString(""),
			errorLogs: []*model.ErrorLog{
				{
					Name: "/unknown/error.log",
				},
			},
			errorLogContents: "",
			error:            nil,
			expected:         nil,
		},
		{
			name:     "successful reload - no error logs",
			out:      bytes.NewBufferString(""),
			error:    nil,
			expected: nil,
		},
		{
			name:     "failed reload",
			out:      bytes.NewBufferString(""),
			error:    errors.New("error reloading"),
			expected: fmt.Errorf("failed to reload NGINX %w: ", errors.New("error reloading")),
		},
		{
			name: "failed reload due to error in error logs",
			out:  bytes.NewBufferString(""),
			errorLogs: []*model.ErrorLog{
				{
					Name: errorLogFile.Name(),
				},
			},
			errorLogContents: errorLogLine,
			error:            nil,
			expected:         errors.Join(fmt.Errorf(errorLogLine)),
		},
		{
			name: "failed reload due to warning in error logs",
			out:  bytes.NewBufferString(""),
			errorLogs: []*model.ErrorLog{
				{
					Name: errorLogFile.Name(),
				},
			},
			errorLogContents: warningLogLine,
			error:            nil,
			expected:         errors.Join(fmt.Errorf(warningLogLine)),
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
			nginxConfig := NewNginx(
				instance,
				&config.Config{
					DataPlaneConfig: config.DataPlaneConfig{
						Nginx: config.NginxDataPlaneConfig{
							TreatWarningsAsError:   true,
							ReloadMonitoringPeriod: 400 * time.Millisecond,
						},
					},
				},
			)
			nginxConfig.executor = mockExec
			nginxConfig.SetConfigContext(&model.NginxConfigContext{
				ErrorLogs: test.errorLogs,
			})

			var wg sync.WaitGroup
			wg.Add(1)
			go func(expected error) {
				defer wg.Done()
				reloadError := nginxConfig.Apply()
				assert.Equal(t, expected, reloadError)
			}(test.expected)

			time.Sleep(200 * time.Millisecond)

			if test.errorLogContents != "" {
				_, err = errorLogFile.WriteString(test.errorLogContents)
				require.NoError(t, err, "Error writing data to error log file")
			}

			wg.Wait()
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
