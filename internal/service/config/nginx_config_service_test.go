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

	"github.com/nginx/agent/v3/test/helpers"

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
	file := helpers.CreateFileWithErrorCheck(t, t.TempDir(), "nginx-parse-config.conf")
	defer helpers.RemoveFileWithErrorCheck(t, file.Name())

	errorLog := helpers.CreateFileWithErrorCheck(t, t.TempDir(), "error.log")
	defer helpers.RemoveFileWithErrorCheck(t, errorLog.Name())

	accessLog := helpers.CreateFileWithErrorCheck(t, t.TempDir(), "access.log")
	defer helpers.RemoveFileWithErrorCheck(t, accessLog.Name())

	combinedAccessLog := helpers.CreateFileWithErrorCheck(t, t.TempDir(), "combined_access.log")
	defer helpers.RemoveFileWithErrorCheck(t, combinedAccessLog.Name())

	ltsvAccessLog := helpers.CreateFileWithErrorCheck(t, t.TempDir(), "ltsv_access.log")
	defer helpers.RemoveFileWithErrorCheck(t, ltsvAccessLog.Name())

	content := testconfig.GetNginxConfigWithMultipleAccessLogs(
		errorLog.Name(),
		accessLog.Name(),
		combinedAccessLog.Name(),
		ltsvAccessLog.Name(),
	)

	data := []byte(content)

	err := os.WriteFile(file.Name(), data, 0o600)
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

	nginxConfig := NewNginx(instance, &config.Config{
		Client: &config.Client{
			Timeout: 5 * time.Second,
		},
	})
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
			err := validateConfigCheckResponse([]byte(test.out))
			assert.Equal(t, test.expected, err)
		})
	}
}

func TestNginx_Apply(t *testing.T) {
	errorLogFile := helpers.CreateFileWithErrorCheck(t, t.TempDir(), "error.log")
	defer helpers.RemoveFileWithErrorCheck(t, errorLogFile.Name())

	tests := []struct {
		name             string
		out              *bytes.Buffer
		errorLogs        []*model.ErrorLog
		errorLogContents string
		error            error
		expected         error
	}{
		{
			name: "Test 1: Successful reload",
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
			name: "Test 2: Successful reload - unknown error log location",
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
			name:     "Test 3: Successful reload - no error logs",
			out:      bytes.NewBufferString(""),
			error:    nil,
			expected: nil,
		},
		{
			name:     "Test 4: Failed reload",
			error:    errors.New("error reloading"),
			expected: fmt.Errorf("failed to reload NGINX, %w", errors.New("error reloading")),
		},
		{
			name: "Test 5: Failed reload due to error in error logs",
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
			name: "Test 6: Failed reload due to warning in error logs",
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
			nginxConfig := NewNginx(
				instance,
				&config.Config{
					DataPlaneConfig: &config.DataPlaneConfig{
						Nginx: &config.NginxDataPlaneConfig{
							TreatWarningsAsError:   true,
							ReloadMonitoringPeriod: 400 * time.Millisecond,
						},
					},
					Client: &config.Client{
						Timeout: 5 * time.Second,
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
				_, err := errorLogFile.WriteString(test.errorLogContents)
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
			name:     "Test 1: Validate Config failed",
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
			nginxConfig := NewNginx(instance, &config.Config{
				Client: &config.Client{
					Timeout: 5 * time.Second,
				},
			})
			nginxConfig.executor = mockExec

			err := nginxConfig.Validate()

			assert.Equal(t, test.expected, err)
		})
	}
}
