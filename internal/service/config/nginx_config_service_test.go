// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package config

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/nginx/agent/v3/internal/client/clientfakes"

	"github.com/nginx/agent/v3/test/helpers"
	modelHelpers "github.com/nginx/agent/v3/test/model"
	"github.com/nginx/agent/v3/test/protos"
	"github.com/nginx/agent/v3/test/types"
	crossplane "github.com/nginxinc/nginx-go-crossplane"

	"github.com/stretchr/testify/require"

	"github.com/nginx/agent/v3/internal/datasource/host/exec/execfakes"
	testconfig "github.com/nginx/agent/v3/test/config"

	"github.com/nginx/agent/v3/internal/model"
	"github.com/stretchr/testify/assert"
)

const (
	errorLogLine   = "2023/03/14 14:16:23 [emerg] 3871#3871: bind() to 0.0.0.0:8081 failed (98: Address already in use)"
	warningLogLine = "2023/03/14 14:16:23 nginx: [warn] 2048 worker_connections exceed open file resource limit: 1024"
	instanceID     = "7332d596-d2e6-4d1e-9e75-70f91ef9bd0e"
)

func TestNginx_ParseConfig(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()

	file := helpers.CreateFileWithErrorCheck(t, dir, "nginx-parse-config.conf")
	defer helpers.RemoveFileWithErrorCheck(t, file.Name())

	errorLog := helpers.CreateFileWithErrorCheck(t, dir, "error.log")
	defer helpers.RemoveFileWithErrorCheck(t, errorLog.Name())

	accessLog := helpers.CreateFileWithErrorCheck(t, dir, "access.log")
	defer helpers.RemoveFileWithErrorCheck(t, accessLog.Name())

	combinedAccessLog := helpers.CreateFileWithErrorCheck(t, dir, "combined_access.log")
	defer helpers.RemoveFileWithErrorCheck(t, combinedAccessLog.Name())

	ltsvAccessLog := helpers.CreateFileWithErrorCheck(t, dir, "ltsv_access.log")
	defer helpers.RemoveFileWithErrorCheck(t, ltsvAccessLog.Name())

	content := testconfig.GetNginxConfigWithMultipleAccessLogs(
		errorLog.Name(),
		accessLog.Name(),
		combinedAccessLog.Name(),
		ltsvAccessLog.Name(),
	)

	err := os.WriteFile(file.Name(), []byte(content), 0o600)
	require.NoError(t, err)

	expectedConfigContext := modelHelpers.GetConfigContextWithNames(
		accessLog.Name(),
		combinedAccessLog.Name(),
		ltsvAccessLog.Name(),
		errorLog.Name())

	instance := protos.GetNginxOssInstance([]string{})
	instance.InstanceRuntime.ConfigPath = file.Name()

	nginxConfig := NewNginx(ctx, instance, types.GetAgentConfig(), &clientfakes.FakeConfigClient{})
	result, err := nginxConfig.ParseConfig(ctx)

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
			nginxConfig := NewNginx(context.Background(), protos.GetNginxOssInstance([]string{}),
				types.GetAgentConfig(), &clientfakes.FakeConfigClient{})

			err := nginxConfig.validateConfigCheckResponse([]byte(test.out))
			assert.Equal(t, test.expected, err)
		})
	}
}

func TestNginx_Apply(t *testing.T) {
	ctx := context.Background()

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

			instance := protos.GetNginxOssInstance([]string{})

			nginxConfig := NewNginx(
				ctx,
				instance,
				types.GetAgentConfig(), &clientfakes.FakeConfigClient{})

			nginxConfig.executor = mockExec
			nginxConfig.SetConfigContext(&model.NginxConfigContext{
				ErrorLogs: test.errorLogs,
			})

			var wg sync.WaitGroup
			wg.Add(1)
			go func(expected error) {
				defer wg.Done()
				reloadError := nginxConfig.Apply(ctx)
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

			nginxConfig := NewNginx(ctx, instance, types.GetAgentConfig(), &clientfakes.FakeConfigClient{})
			nginxConfig.executor = mockExec

			err := nginxConfig.Validate(ctx)

			assert.Equal(t, test.expected, err)
		})
	}
}

var (
	testConf01 = `server {
    listen       80 default_server;
    server_name  localhost;
    location /api/ {
        api write=on;
        allow 127.0.0.1;
        deny all;
    }
}
`
	testConf02 = `server {
    listen       *:80 default_server;
    location /api/ {
        api write=on;
        allow 127.0.0.1;
        deny all;
    }
}
`
	testConf03 = `server {
    listen       80 default_server;
	server_name _;
    location /api/ {
        api write=on;
        allow 127.0.0.1;
        deny all;
    }
}
`
	testConf04 = `server {
    listen 8888 default_server;
    server_name status.internal.com;
    location /api/ {
        api write=on;
        allow 127.0.0.1;
        deny all;
    }
}
`
	testConf05 = `server {
		listen 127.0.0.1:8080;
		location /privateapi {
			api write=on;
			allow 127.0.0.1;
			deny all;
		}
}`
	testConf06 = `server {
    listen 80 default_server;
	listen [::]:80 default_server;
	server_name _;
    location /api/ {
        api write=on;
        allow 127.0.0.1;
        deny all;
    }
}`
	testConf07 = `server {
    listen 127.0.0.1;
	server_name _;
    location /api/ {
        api write=on;
        allow 127.0.0.1;
        deny all;
    }
}`
	testConf08 = `server {
    listen 127.0.0.1;
	server_name _;
    location = /api/ {
        api write=on;
        allow 127.0.0.1;
        deny all;
    }
}`
	testConf09 = `server {
    listen 80;
	server_name _;
    location = /api/ {
        api write=on;
        allow 127.0.0.1;
        deny all;
    }
}`
	testConf10 = `server {
    listen :80;
	server_name _;
    location = /api/ {
        api write=on;
        allow 127.0.0.1;
        deny all;
    }
}`
	testConf11 = `server {
    listen localhost;
	server_name _;
    location /api/ {
        api write=on;
        allow 127.0.0.1;
        deny all;
    }
}`
	testConf12 = `server {
    listen [::1];
	server_name _;
    location /api/ {
        api write=on;
        allow 127.0.0.1;
        deny all;
    }
}`
	testConf13 = `server {
    listen [::]:8000;
	server_name _;
    location /api/ {
        api write=on;
        allow 127.0.0.1;
        deny all;
    }
}`
	testConf14 = `server {
	server_name   localhost;
	listen        127.0.0.1:80;

	error_page    500 502 503 504  /50x.html;
	# ssl_certificate /usr/local/nginx/conf/cert.pem;

	location      / {
		root      /tmp/testdata/foo;
	}

	location /stub_status {
		stub_status;
	}
}`
	testConf15 = `server {
	server_name   localhost;
	listen        :80;

	error_page    500 502 503 504  /50x.html;
	# ssl_certificate /usr/local/nginx/conf/cert.pem;

	location      / {
		root      /tmp/testdata/foo;
	}

	location /stub_status {
		stub_status;
	}
}`
	testConf16 = `server {
	server_name   localhost;
	listen        80;

	error_page    500 502 503 504  /50x.html;
	# ssl_certificate /usr/local/nginx/conf/cert.pem;

	location      / {
		root      /tmp/testdata/foo;
	}

	location /stub_status {
		stub_status;
	}
}`
	testConf17 = `server {
	server_name   localhost;
	listen        80;

	error_page    500 502 503 504  /50x.html;
	# ssl_certificate /usr/local/nginx/conf/cert.pem;

	location      / {
		root      /tmp/testdata/foo;
	}

	location = /stub_status {
		stub_status;
	}
}`
	testConf18 = `server {
	server_name   localhost;
	listen        80;

	error_page    500 502 503 504  /50x.html;
	# ssl_certificate /usr/local/nginx/conf/cert.pem;

	location      / {
		root      /tmp/testdata/foo;
	}

	location = /stub_status {
		stub_status;
	}

	location /api/ {
        api write=on;
        allow 127.0.0.1;
        deny all;
    }
}`
	testConf19 = `server {
	server_name 127.0.0.1;
	listen 127.0.0.1:49151;
	access_log off;
	location /api {
		api;
	}
}`
)

func TestParseStatusAPIEndpoints(t *testing.T) {
	tmpDir := t.TempDir()
	for _, tt := range []struct {
		oss  []string
		plus []string
		conf string
	}{
		{
			plus: []string{
				"http://127.0.0.1:80/api/",
				"http://localhost:80/api/",
			},
			conf: testConf01,
		},
		{
			plus: []string{
				"http://127.0.0.1:80/api/",
			},
			conf: testConf02,
		},
		{
			plus: []string{
				"http://127.0.0.1:80/api/",
			},
			conf: testConf03,
		},
		{
			plus: []string{
				"http://127.0.0.1:8888/api/",
				"http://status.internal.com:8888/api/",
			},
			conf: testConf04,
		},
		{
			plus: []string{
				"http://127.0.0.1:8080/privateapi",
			},
			conf: testConf05,
		},
		{
			plus: []string{
				"http://127.0.0.1:80/api/",
				"http://[::1]:80/api/",
			},
			conf: testConf06,
		},
		{
			plus: []string{
				"http://127.0.0.1:80/api/",
			},
			conf: testConf07,
		},
		{
			plus: []string{
				"http://127.0.0.1:80/api/",
			},
			conf: testConf08,
		},
		{
			plus: []string{
				"http://127.0.0.1:80/api/",
			},
			conf: testConf09,
		},
		{
			plus: []string{
				"http://127.0.0.1:80/api/",
			},
			conf: testConf10,
		},
		{
			plus: []string{
				"http://localhost:80/api/",
			},
			conf: testConf11,
		},
		{
			plus: []string{
				"http://[::1]:80/api/",
			},
			conf: testConf12,
		},
		{
			plus: []string{
				"http://[::1]:8000/api/",
			},
			conf: testConf13,
		},
		{
			oss: []string{
				"http://localhost:80/stub_status",
				"http://127.0.0.1:80/stub_status",
			},
			conf: testConf14,
		},
		{
			oss: []string{
				"http://localhost:80/stub_status",
				"http://127.0.0.1:80/stub_status",
			},
			conf: testConf15,
		},
		{
			oss: []string{
				"http://localhost:80/stub_status",
				"http://127.0.0.1:80/stub_status",
			},
			conf: testConf16,
		},
		{
			oss: []string{
				"http://localhost:80/stub_status",
				"http://127.0.0.1:80/stub_status",
			},
			conf: testConf17,
		},
		{
			oss: []string{
				"http://localhost:80/stub_status",
				"http://127.0.0.1:80/stub_status",
			},
			plus: []string{
				"http://localhost:80/api/",
				"http://127.0.0.1:80/api/",
			},
			conf: testConf18,
		},
		{
			plus: []string{
				"http://127.0.0.1:49151/api",
				"http://127.0.0.1:49151/api",
			},
			conf: testConf19,
		},
	} {
		ctx := context.Background()
		f, err := os.CreateTemp(tmpDir, "conf")
		require.NoError(t, err)
		parseOptions := &crossplane.ParseOptions{
			SingleFile:         false,
			StopParsingOnError: true,
		}

		err = os.WriteFile(f.Name(), []byte(fmt.Sprintf("http{ %s }", tt.conf)), 0o600)
		require.NoError(t, err)

		payload, err := crossplane.Parse(f.Name(), parseOptions)
		require.NoError(t, err)
		instance := protos.GetNginxOssInstance([]string{})
		nginxConfig := NewNginx(context.Background(), instance, types.GetAgentConfig(), &clientfakes.FakeConfigClient{})

		var oss, plus []string

		assert.Len(t, payload.Config, 1)
		for _, xpConf := range payload.Config {
			assert.Len(t, xpConf.Parsed, 1)
			err = nginxConfig.crossplaneConfigTraverse(ctx, &xpConf,
				func(ctx context.Context, parent, directive *crossplane.Directive) error {
					_oss := nginxConfig.urlsForLocationDirective(parent, directive, stubStatusAPIDirective)
					_plus := nginxConfig.urlsForLocationDirective(parent, directive, plusAPIDirective)
					oss = append(oss, _oss...)
					plus = append(plus, _plus...)

					return nil
				})
			require.NoError(t, err)
		}

		assert.Equal(t, tt.plus, plus)
		assert.Equal(t, tt.oss, oss)
	}
}
