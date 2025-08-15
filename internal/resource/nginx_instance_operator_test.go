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
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/nginx/agent/v3/pkg/host/exec/execfakes"

	"github.com/nginx/agent/v3/test/helpers"
	"github.com/nginx/agent/v3/test/protos"
	"github.com/nginx/agent/v3/test/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	exePath       = "/usr/local/Cellar/nginx/1.25.3/bin/nginx"
	exePath2      = "/usr/local/Cellar/nginx/1.26.3/bin/nginx"
	ossConfigArgs = "--prefix=/usr/local/Cellar/nginx/1.25.3 --sbin-path=/usr/local/Cellar/nginx/1.25.3/bin/nginx " +
		"--modules-path=%s --with-cc-opt='-I/usr/local/opt/pcre2/include -I/usr/local/opt/openssl@1.1/include' " +
		"--with-ld-opt='-L/usr/local/opt/pcre2/lib -L/usr/local/opt/openssl@1.1/lib' " +
		"--conf-path=/usr/local/etc/nginx/nginx.conf --pid-path=/usr/local/var/run/nginx.pid " +
		"--lock-path=/usr/local/var/run/nginx.lock " +
		"--http-client-body-temp-path=/usr/local/var/run/nginx/client_body_temp " +
		"--http-proxy-temp-path=/usr/local/var/run/nginx/proxy_temp " +
		"--http-fastcgi-temp-path=/usr/local/var/run/nginx/fastcgi_temp " +
		"--http-uwsgi-temp-path=/usr/local/var/run/nginx/uwsgi_temp " +
		"--http-scgi-temp-path=/usr/local/var/run/nginx/scgi_temp " +
		"--http-log-path=/usr/local/var/log/nginx/access.log " +
		"--error-log-path=/usr/local/var/log/nginx/error.log --with-compat --with-debug " +
		"--with-http_addition_module --with-http_auth_request_module --with-http_dav_module " +
		"--with-http_degradation_module --with-http_flv_module --with-http_gunzip_module " +
		"--with-http_gzip_static_module --with-http_mp4_module --with-http_random_index_module " +
		"--with-http_realip_module --with-http_secure_link_module --with-http_slice_module " +
		"--with-http_ssl_module --with-http_stub_status_module --with-http_sub_module " +
		"--with-http_v2_module --with-ipv6 --with-mail --with-mail_ssl_module --with-pcre " +
		"--with-pcre-jit --with-stream --with-stream_realip_module --with-stream_ssl_module " +
		"--with-stream_ssl_preread_module"
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
		{
			name:     "Test 3: Warn response",
			out:      "nginx [warn]",
			expected: errors.New("error running nginx -t -c:\nnginx [warn]"),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			operator := NewInstanceOperator(types.AgentConfig())
			operator.treatWarningsAsErrors = true
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

			agentConfig := types.AgentConfig()
			agentConfig.DataPlaneConfig.Nginx.ReloadBackoff = &config.BackOff{
				InitialInterval:     config.DefNginxReloadBackoffInitialInterval,
				MaxInterval:         config.DefNginxReloadBackoffMaxInterval,
				MaxElapsedTime:      config.DefNginxReloadBackoffMaxElapsedTime,
				RandomizationFactor: config.DefNginxReloadBackoffRandomizationFactor,
				Multiplier:          config.DefNginxReloadBackoffMultiplier,
			}
			operator := NewInstanceOperator(agentConfig)
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
			agentConfig.DataPlaneConfig.Nginx.ReloadBackoff = &config.BackOff{
				InitialInterval:     config.DefNginxReloadBackoffInitialInterval,
				MaxInterval:         config.DefNginxReloadBackoffMaxInterval,
				MaxElapsedTime:      config.DefNginxReloadBackoffMaxElapsedTime,
				RandomizationFactor: config.DefNginxReloadBackoffRandomizationFactor,
				Multiplier:          config.DefNginxReloadBackoffMultiplier,
			}
			operator := NewInstanceOperator(agentConfig)
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

func TestInstanceOperator_checkWorkers(t *testing.T) {
	ctx := context.Background()

	modulePath := t.TempDir() + "/usr/lib/nginx/modules"

	configArgs := fmt.Sprintf(ossConfigArgs, modulePath)
	nginxVersionCommandOutput := `nginx version: nginx/1.25.3
					built by clang 14.0.0 (clang-1400.0.29.202)
					built with OpenSSL 1.1.1s  1 Nov 2022 (running with OpenSSL 1.1.1t  7 Feb 2023)
					TLS SNI support enabled
					configure arguments: ` + configArgs

	tests := []struct {
		expectedLog   string
		name          string
		instanceID    string
		reloadTime    time.Time
		workers       []*nginxprocess.Process
		masterProcess []*nginxprocess.Process
	}{
		{
			name:        "Test 1: Successful reload",
			expectedLog: "All NGINX workers have been reloaded",
			reloadTime:  time.Date(2025, 8, 13, 8, 0, 0, 0, time.Local),
			instanceID:  "e1374cb1-462d-3b6c-9f3b-f28332b5f10c",
			workers: []*nginxprocess.Process{
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
			masterProcess: []*nginxprocess.Process{
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
			name: "Test 2: Unsuccessful reload",
			expectedLog: "\"Failed to check if NGINX worker processes have successfully reloaded, timed out " +
				"waiting\" error=\"waiting for NGINX worker processes\"",
			reloadTime: time.Date(2025, 8, 13, 8, 0, 0, 0, time.Local),
			instanceID: "e1374cb1-462d-3b6c-9f3b-f28332b5f10c",
			masterProcess: []*nginxprocess.Process{
				{
					PID:     1234,
					Created: time.Date(2025, 8, 13, 8, 1, 0, 0, time.Local),
					PPID:    1,
					Name:    "nginx",
					Cmd:     "nginx: master process /usr/local/opt/nginx/bin/nginx -g daemon off;",
					Exe:     exePath,
				},
			},
			workers: []*nginxprocess.Process{
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
					Created: time.Date(2025, 8, 13, 7, 1, 0, 0, time.Local),
					Name:    "nginx",
					Cmd:     "nginx: worker process",
					Exe:     exePath,
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(tt *testing.T) {
			mockExec := &execfakes.FakeExecInterface{}
			mockExec.RunCmdReturnsOnCall(0, bytes.NewBufferString(nginxVersionCommandOutput), nil)
			mockExec.RunCmdReturnsOnCall(1, bytes.NewBufferString(nginxVersionCommandOutput), nil)
			mockExec.RunCmdReturnsOnCall(2, bytes.NewBufferString(nginxVersionCommandOutput), nil)
			mockExec.RunCmdReturnsOnCall(3, bytes.NewBufferString(nginxVersionCommandOutput), nil)

			mockProcessOp := &resourcefakes.FakeProcessOperator{}
			allProcesses := slices.Concat(test.workers, test.masterProcess)
			mockProcessOp.FindNginxProcessesReturnsOnCall(0, allProcesses, nil)
			mockProcessOp.NginxWorkerProcessesReturnsOnCall(0, test.workers)
			mockProcessOp.FindParentProcessIDReturnsOnCall(0, test.masterProcess[0].PID, nil)

			logBuf := &bytes.Buffer{}
			stub.StubLoggerWith(logBuf)

			agentConfig := types.AgentConfig()
			agentConfig.DataPlaneConfig.Nginx.ReloadMonitoringPeriod = 10 * time.Second
			agentConfig.DataPlaneConfig.Nginx.ReloadBackoff = &config.BackOff{
				InitialInterval:     config.DefNginxReloadBackoffInitialInterval,
				MaxInterval:         config.DefNginxReloadBackoffMaxInterval,
				MaxElapsedTime:      config.DefNginxReloadBackoffMaxElapsedTime,
				RandomizationFactor: config.DefNginxReloadBackoffRandomizationFactor,
				Multiplier:          config.DefNginxReloadBackoffMultiplier,
			}
			operator := NewInstanceOperator(agentConfig)
			operator.executer = mockExec
			operator.nginxProcessOperator = mockProcessOp

			operator.checkWorkers(ctx, test.instanceID, test.reloadTime, allProcesses)

			helpers.ValidateLog(t, test.expectedLog, logBuf)

			t.Logf("Logs: %s", logBuf.String())
			logBuf.Reset()
		})
	}
}
