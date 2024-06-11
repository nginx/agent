// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package instance

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/google/go-cmp/cmp"
	mpi "github.com/nginx/agent/v3/api/grpc/mpi/v1"
	"github.com/nginx/agent/v3/files"
	testconfig "github.com/nginx/agent/v3/test/config"
	"github.com/nginx/agent/v3/test/helpers"
	modelHelpers "github.com/nginx/agent/v3/test/model"
	"github.com/nginx/agent/v3/test/protos"
	"github.com/nginx/agent/v3/test/types"
	crossplane "github.com/nginxinc/nginx-go-crossplane"
	"google.golang.org/protobuf/testing/protocmp"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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

func TestNginxConfigParser_Parse(t *testing.T) {
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

	fileMeta, err := files.GetFileMeta(file.Name())
	require.NoError(t, err)

	tests := []struct {
		name     string
		instance *mpi.Instance
	}{
		{
			name:     "Test 1: Valid response",
			instance: protos.GetNginxOssInstance([]string{}),
		},
		{
			name:     "Test 2: Error response",
			instance: protos.GetNginxPlusInstance([]string{}),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			expectedConfigContext := modelHelpers.GetConfigContextWithNames(
				accessLog.Name(),
				combinedAccessLog.Name(),
				ltsvAccessLog.Name(),
				errorLog.Name(),
				test.instance.GetInstanceMeta().GetInstanceId(),
			)
			expectedConfigContext.Files = append(expectedConfigContext.Files, &mpi.File{
				FileMeta: fileMeta,
			})

			test.instance.InstanceRuntime.ConfigPath = file.Name()

			nginxConfig := NewNginxConfigParser(types.GetAgentConfig())
			result, parseError := nginxConfig.Parse(ctx, test.instance)
			require.NoError(t, parseError)

			if diff := cmp.Diff(expectedConfigContext, result, protocmp.Transform()); diff != "" {
				t.Errorf("\n%v", diff)
			}
		})
	}
}

func TestNginxConfigParser_rootFiles(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()

	file1 := helpers.CreateFileWithErrorCheck(t, dir, "nginx-1.conf")
	defer helpers.RemoveFileWithErrorCheck(t, file1.Name())
	file2 := helpers.CreateFileWithErrorCheck(t, dir, "nginx-2.conf")
	defer helpers.RemoveFileWithErrorCheck(t, file2.Name())

	// Not in allowed directory
	nginxConfig := NewNginxConfigParser(types.GetAgentConfig())
	nginxConfig.agentConfig.AllowedDirectories = []string{}
	rootfiles := nginxConfig.rootFiles(ctx, dir)
	assert.Empty(t, rootfiles)

	// In allowed directory
	nginxConfig.agentConfig.AllowedDirectories = []string{dir}
	rootfiles = nginxConfig.rootFiles(ctx, dir)
	assert.Len(t, rootfiles, 2)
	assert.Equal(t, file1.Name(), rootfiles[0].GetFileMeta().GetName())
	assert.Equal(t, file2.Name(), rootfiles[1].GetFileMeta().GetName())
}

func TestNginxConfigParser_sslCert(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()

	file1 := helpers.CreateFileWithErrorCheck(t, dir, "nginx-1.conf")
	defer helpers.RemoveFileWithErrorCheck(t, file1.Name())

	// Not in allowed directory
	nginxConfig := NewNginxConfigParser(types.GetAgentConfig())
	nginxConfig.agentConfig.AllowedDirectories = []string{}
	sslCert := nginxConfig.sslCert(ctx, file1.Name(), dir)
	assert.Nil(t, sslCert)

	// In allowed directory
	nginxConfig.agentConfig.AllowedDirectories = []string{dir}
	sslCert = nginxConfig.sslCert(ctx, file1.Name(), dir)
	assert.Equal(t, file1.Name(), sslCert.GetFileMeta().GetName())
}

func TestNginxConfigParser_urlsForLocationDirective(t *testing.T) {
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
		nginxConfigParser := NewNginxConfigParser(types.GetAgentConfig())

		var oss, plus []string

		assert.Len(t, payload.Config, 1)
		for _, xpConf := range payload.Config {
			assert.Len(t, xpConf.Parsed, 1)
			err = nginxConfigParser.crossplaneConfigTraverse(ctx, &xpConf,
				func(ctx context.Context, parent, directive *crossplane.Directive) error {
					_oss := nginxConfigParser.urlsForLocationDirective(parent, directive, stubStatusAPIDirective)
					_plus := nginxConfigParser.urlsForLocationDirective(parent, directive, plusAPIDirective)
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

// linter doesn't like the duplicate handler and server function
// nolint: dupl
func TestNginxConfigParser_pingPlusAPIEndpoint(t *testing.T) {
	handler := http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		if req.URL.String() == "/good_api" {
			data := []byte("[1,2,3,4,5,6,7,8]")
			_, err := rw.Write(data)
			require.NoError(t, err)
		} else if req.URL.String() == "/invalid_body_api" {
			data := []byte("Invalid")
			_, err := rw.Write(data)
			require.NoError(t, err)
		} else {
			rw.WriteHeader(http.StatusInternalServerError)
			data := []byte("")
			_, err := rw.Write(data)
			require.NoError(t, err)
		}
	})

	fakeServer := httptest.NewServer(handler)
	defer fakeServer.Close()

	nginxConfigParser := NewNginxConfigParser(types.GetAgentConfig())

	tests := []struct {
		name     string
		endpoint string
		expected bool
	}{
		{
			name:     "Test 1: valid API",
			endpoint: "/good_api",
			expected: true,
		},
		{
			name:     "Test 2: invalid response status code",
			endpoint: "/bad_api",
			expected: false,
		},
		{
			name:     "Test 3: invalid response body",
			endpoint: "/invalid_body_api",
			expected: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx := context.Background()
			result := nginxConfigParser.pingPlusAPIEndpoint(ctx, fmt.Sprintf("%s%s", fakeServer.URL, test.endpoint))
			assert.Equal(t, test.expected, result)
		})
	}
}

// linter doesn't like the duplicate handler and server function
// nolint: dupl
func TestNginxConfigParser_pingStubStatusAPIEndpoint(t *testing.T) {
	handler := http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		if req.URL.String() == "/good_api" {
			data := []byte(`
Active connections: 2
server accepts handled requests
	18 18 3266
Reading: 0 Writing: 1 Waiting: 1
			`)
			_, err := rw.Write(data)
			require.NoError(t, err)
		} else if req.URL.String() == "/invalid_body_api" {
			data := []byte("Invalid")
			_, err := rw.Write(data)
			require.NoError(t, err)
		} else {
			rw.WriteHeader(http.StatusInternalServerError)
			data := []byte("")
			_, err := rw.Write(data)
			require.NoError(t, err)
		}
	})

	fakeServer := httptest.NewServer(handler)
	defer fakeServer.Close()

	nginxConfigParser := NewNginxConfigParser(types.GetAgentConfig())

	tests := []struct {
		name     string
		endpoint string
		expected bool
	}{
		{
			name:     "Test 1: valid API",
			endpoint: "/good_api",
			expected: true,
		},
		{
			name:     "Test 2: invalid response status code",
			endpoint: "/bad_api",
			expected: false,
		},
		{
			name:     "Test 3: invalid response body",
			endpoint: "/invalid_body_api",
			expected: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx := context.Background()
			statusAPI := fmt.Sprintf("%s%s", fakeServer.URL, test.endpoint)
			result := nginxConfigParser.pingStubStatusAPIEndpoint(ctx, statusAPI)
			assert.Equal(t, test.expected, result)
		})
	}
}
