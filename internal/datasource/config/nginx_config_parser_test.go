// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package config

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"sort"
	"testing"

	"google.golang.org/protobuf/proto"

	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/nginx/agent/v3/internal/model"
	"github.com/nginx/agent/v3/pkg/files"
	"github.com/nginx/agent/v3/test/stub"

	mpi "github.com/nginx/agent/v3/api/grpc/mpi/v1"
	testconfig "github.com/nginx/agent/v3/test/config"
	"github.com/nginx/agent/v3/test/helpers"
	modelHelpers "github.com/nginx/agent/v3/test/model"
	"github.com/nginx/agent/v3/test/protos"
	"github.com/nginx/agent/v3/test/types"
	crossplane "github.com/nginxinc/nginx-go-crossplane"
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

	testConf20 = `server {
    listen       unix:/var/run/nginx/nginx-status.sock;
    location /stub_status {
        stub_status;
    }
}
`
	testConf21 = `server {
    listen unix:/var/run/nginx/nginx-plus-api.sock;
    access_log off;

    location /api {
        api write=on;
    }
  }
`

	testConf22 = ` server {
	listen unix:/var/lib/nginx/nginx-plus-api.sock;
	access_log off;

	# $config_version_mismatch is defined in /etc/nginx/config-version.conf
	location /configVersionCheck {
		if ($config_version_mismatch) {
			return 503;
		}
		return 200;
	}

	location /api {
		api write=on;
	}
}

server {
	listen unix:/var/lib/nginx/nginx-418-server.sock;
	access_log off;

	return 418;
}
`
)

// nolint: maintidx
func TestNginxConfigParser_Parse(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	notAllowedDir := t.TempDir()

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

	notAllowedFile := helpers.CreateFileWithErrorCheck(t, notAllowedDir, "file_outside_allowed.conf")
	defer helpers.RemoveFileWithErrorCheck(t, notAllowedFile.Name())

	allowedFile := helpers.CreateFileWithErrorCheck(t, dir, "file_allowed.conf")
	defer helpers.RemoveFileWithErrorCheck(t, allowedFile.Name())
	fileMetaAllowedFiles, err := files.FileMeta(allowedFile.Name())
	require.NoError(t, err)
	allowedFileWithMetas := mpi.File{FileMeta: fileMetaAllowedFiles}

	_, cert := helpers.GenerateSelfSignedCert(t)
	certContents := helpers.Cert{Name: "nginx.cert", Type: "CERTIFICATE", Contents: cert}
	certFile := helpers.WriteCertFiles(t, dir, certContents)
	require.NotNil(t, certFile)
	fileMetaCertFiles, err := files.FileMetaWithCertificate(certFile)
	require.NoError(t, err)
	certFileWithMetas := mpi.File{FileMeta: fileMetaCertFiles}

	_, diffCert := helpers.GenerateSelfSignedCert(t)
	diffCertContents := helpers.Cert{Name: "nginx1.cert", Type: "CERTIFICATE", Contents: diffCert}
	diffCertFile := helpers.WriteCertFiles(t, dir, diffCertContents)
	require.NotNil(t, diffCertFile)
	diffFileMetaCertFiles, err := files.FileMetaWithCertificate(diffCertFile)
	require.NoError(t, err)
	diffCertFileWithMetas := mpi.File{FileMeta: diffFileMetaCertFiles}

	tests := []struct {
		instance              *mpi.Instance
		name                  string
		content               string
		expectedConfigContext *model.NginxConfigContext
		expectedLog           string
		allowedDirectories    []string
	}{
		{
			name:     "Test 1: Valid response",
			instance: protos.NginxOssInstance([]string{}),
			content: testconfig.NginxConfigWithMultipleAccessLogs(
				errorLog.Name(),
				accessLog.Name(),
				combinedAccessLog.Name(),
				ltsvAccessLog.Name(),
			),
			expectedConfigContext: modelHelpers.ConfigContextWithNames(
				accessLog.Name(),
				combinedAccessLog.Name(),
				ltsvAccessLog.Name(),
				errorLog.Name(),
				protos.NginxOssInstance([]string{}).GetInstanceMeta().GetInstanceId(),
				[]string{"127.0.0.1:1515"},
			),
			expectedLog:        "",
			allowedDirectories: []string{dir},
		},
		{
			name:     "Test 2: Error response",
			instance: protos.NginxPlusInstance([]string{}),
			content: testconfig.NginxConfigWithMultipleAccessLogs(
				errorLog.Name(),
				accessLog.Name(),
				combinedAccessLog.Name(),
				ltsvAccessLog.Name(),
			),
			expectedConfigContext: modelHelpers.ConfigContextWithNames(
				accessLog.Name(),
				combinedAccessLog.Name(),
				ltsvAccessLog.Name(),
				errorLog.Name(),
				protos.NginxPlusInstance([]string{}).GetInstanceMeta().GetInstanceId(),
				[]string{"127.0.0.1:1515"},
			),
			expectedLog:        "",
			allowedDirectories: []string{dir},
		},
		{
			name:     "Test 3: File outside allowed directories",
			instance: protos.NginxPlusInstance([]string{}),
			content: testconfig.NginxConfigWithNotAllowedDir(errorLog.Name(), allowedFile.Name(),
				notAllowedFile.Name(), accessLog.Name()),
			expectedConfigContext: modelHelpers.ConfigContextWithFiles(
				accessLog.Name(),
				errorLog.Name(),
				[]*mpi.File{&allowedFileWithMetas},
				protos.NginxPlusInstance([]string{}).GetInstanceMeta().GetInstanceId(),
				[]string{},
			),
			expectedLog:        "",
			allowedDirectories: []string{dir},
		},
		{
			name:     "Test 4: SSL Certificate file path containing variables",
			instance: protos.NginxPlusInstance([]string{}),
			content:  testconfig.NginxConfWithSSLCertsWithVariables(),
			expectedConfigContext: &model.NginxConfigContext{
				StubStatus:       &model.APIDetails{},
				PlusAPI:          &model.APIDetails{},
				InstanceID:       protos.NginxPlusInstance([]string{}).GetInstanceMeta().GetInstanceId(),
				Files:            []*mpi.File{},
				AccessLogs:       []*model.AccessLog{},
				ErrorLogs:        []*model.ErrorLog{},
				NAPSysLogServers: []string{},
			},
			allowedDirectories: []string{dir},
		},
		{
			name:     "Test 5: Error Log outputting to stderr",
			instance: protos.NginxPlusInstance([]string{}),
			content: testconfig.NginxConfigWithMultipleAccessLogs(
				"stderr",
				accessLog.Name(),
				combinedAccessLog.Name(),
				ltsvAccessLog.Name(),
			),
			expectedConfigContext: modelHelpers.ConfigContextWithoutErrorLog(
				accessLog.Name(),
				combinedAccessLog.Name(),
				ltsvAccessLog.Name(),
				protos.NginxPlusInstance([]string{}).GetInstanceMeta().GetInstanceId(),
				[]string{"127.0.0.1:1515"},
			),
			expectedLog: "Currently error log outputs to stderr. Log monitoring is disabled while applying a " +
				"config; log errors to file to enable error monitoring",
			allowedDirectories: []string{dir},
		},
		{
			name:     "Test 6: Error Log outputting to stdout",
			instance: protos.NginxPlusInstance([]string{}),
			content: testconfig.NginxConfigWithMultipleAccessLogs(
				"stdout",
				accessLog.Name(),
				combinedAccessLog.Name(),
				ltsvAccessLog.Name(),
			),
			expectedConfigContext: modelHelpers.ConfigContextWithoutErrorLog(
				accessLog.Name(),
				combinedAccessLog.Name(),
				ltsvAccessLog.Name(),
				protos.NginxPlusInstance([]string{}).GetInstanceMeta().GetInstanceId(),
				[]string{"127.0.0.1:1515"},
			),
			expectedLog: "Currently error log outputs to stdout. Log monitoring is disabled while applying a " +
				"config; log errors to file to enable error monitoring",
			allowedDirectories: []string{dir},
		},
		{
			name:     "Test 7: Check Parser for SSL Certs",
			instance: protos.NginxPlusInstance([]string{}),
			content: testconfig.NginxConfigWithSSLCerts(
				errorLog.Name(),
				accessLog.Name(),
				certFile,
			),
			expectedConfigContext: modelHelpers.ConfigContextWithFiles(
				accessLog.Name(),
				errorLog.Name(),
				[]*mpi.File{&certFileWithMetas},
				protos.NginxPlusInstance([]string{}).GetInstanceMeta().GetInstanceId(),
				[]string{},
			),
			allowedDirectories: []string{dir},
		},
		{
			name:     "Test 8: Check for multiple different SSL Certs",
			instance: protos.NginxPlusInstance([]string{}),
			content: testconfig.NginxConfigWithMultipleSSLCerts(
				errorLog.Name(),
				accessLog.Name(),
				certFile,
				diffCertFile,
			),
			expectedConfigContext: modelHelpers.ConfigContextWithFiles(
				accessLog.Name(),
				errorLog.Name(),
				[]*mpi.File{&diffCertFileWithMetas, &certFileWithMetas},
				protos.NginxPlusInstance([]string{}).GetInstanceMeta().GetInstanceId(),
				[]string{},
			),
			allowedDirectories: []string{dir},
		},
		{
			name:     "Test 9: Check for multiple same SSL Certs",
			instance: protos.NginxPlusInstance([]string{}),
			content: testconfig.NginxConfigWithMultipleSSLCerts(
				errorLog.Name(),
				accessLog.Name(),
				certFile,
				certFile,
			),
			expectedConfigContext: modelHelpers.ConfigContextWithFiles(
				accessLog.Name(),
				errorLog.Name(),
				[]*mpi.File{&certFileWithMetas},
				protos.NginxPlusInstance([]string{}).GetInstanceMeta().GetInstanceId(),
				[]string{},
			),
			allowedDirectories: []string{dir},
		},
		{
			name:     "Test 10: Available NAP syslog server",
			instance: protos.NginxPlusInstance([]string{}),
			content: testconfig.NginxConfigWithMultipleSysLogs(errorLog.Name(), accessLog.Name(),
				"192.168.12.34:1517", "my.domain.com:1517", "127.0.0.1:1515"),
			expectedConfigContext: modelHelpers.ConfigContextWithSysLog(
				accessLog.Name(),
				errorLog.Name(),
				protos.NginxPlusInstance([]string{}).GetInstanceMeta().GetInstanceId(),
				[]string{"127.0.0.1:1515"},
			),
			allowedDirectories: []string{dir},
			expectedLog:        "Found NAP syslog server",
		},
		{
			name:     "Test 11: Unavailable NAP syslog server",
			instance: protos.NginxPlusInstance([]string{}),
			content: testconfig.NginxConfigWithMultipleSysLogs(errorLog.Name(), accessLog.Name(),
				"192.168.12.34:1517", "my.domain.com:1517", "not.allowed:1515"),
			expectedConfigContext: modelHelpers.ConfigContextWithSysLog(
				accessLog.Name(),
				errorLog.Name(),
				protos.NginxPlusInstance([]string{}).GetInstanceMeta().GetInstanceId(),
				[]string{},
			),
			allowedDirectories: []string{dir},
			expectedLog: "Could not find available local NGINX App Protect syslog server. " +
				"Security violations will not be collected.",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			writeErr := os.WriteFile(file.Name(), []byte(test.content), 0o600)
			require.NoError(t, writeErr)

			fileMeta, fileMetaErr := files.FileMeta(file.Name())
			require.NoError(t, fileMetaErr)

			test.expectedConfigContext.Files = append(test.expectedConfigContext.Files, &mpi.File{
				FileMeta: fileMeta,
			})

			test.instance.InstanceRuntime.ConfigPath = file.Name()

			agentConfig := types.AgentConfig()
			agentConfig.AllowedDirectories = test.allowedDirectories

			nginxConfig := NewNginxConfigParser(agentConfig)

			logBuf := &bytes.Buffer{}
			stub.StubLoggerWith(logBuf)

			result, parseError := nginxConfig.Parse(ctx, test.instance)
			require.NoError(t, parseError)

			t.Logf("Log: %s", logBuf.String())
			helpers.ValidateLog(t, test.expectedLog, logBuf)
			logBuf.Reset()

			sort.Slice(test.expectedConfigContext.Files, func(i, j int) bool {
				return test.expectedConfigContext.Files[i].GetFileMeta().GetName() >
					test.expectedConfigContext.Files[j].GetFileMeta().GetName()
			})

			sort.Slice(result.Files, func(i, j int) bool {
				return result.Files[i].GetFileMeta().GetName() >
					result.Files[j].GetFileMeta().GetName()
			})

			assert.Truef(t,
				protoListEqual(test.expectedConfigContext.Files, result.Files),
				"Expect %s Got %s", test.expectedConfigContext.Files, result.Files)
			assert.Equal(t, test.expectedConfigContext.NAPSysLogServers, result.NAPSysLogServers)
			assert.Equal(t, test.expectedConfigContext.PlusAPI, result.PlusAPI)
			assert.ElementsMatch(t, test.expectedConfigContext.AccessLogs, result.AccessLogs)
			assert.ElementsMatch(t, test.expectedConfigContext.ErrorLogs, result.ErrorLogs)
			assert.Equal(t, test.expectedConfigContext.StubStatus, result.StubStatus)
			assert.Equal(t, test.expectedConfigContext.InstanceID, result.InstanceID)
			assert.Len(t, result.Files, len(test.expectedConfigContext.Files))
		})
	}
}

func TestNginxConfigParser_sslCert(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()

	_, cert := helpers.GenerateSelfSignedCert(t)

	certContents := helpers.Cert{Name: "nginx.cert", Type: "CERTIFICATE", Contents: cert}

	certFile := helpers.WriteCertFiles(t, dir, certContents)
	require.NotNil(t, certFile)

	// Not in allowed directory
	nginxConfig := NewNginxConfigParser(types.AgentConfig())
	nginxConfig.agentConfig.AllowedDirectories = []string{}
	sslCert := nginxConfig.sslCert(ctx, certFile, dir)
	assert.Nil(t, sslCert)

	// In allowed directory
	nginxConfig.agentConfig.AllowedDirectories = []string{dir}
	sslCert = nginxConfig.sslCert(ctx, certFile, dir)
	assert.Equal(t, certFile, sslCert.GetFileMeta().GetName())
}

func TestNginxConfigParser_SyslogServerParse(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()

	file := helpers.CreateFileWithErrorCheck(t, dir, "nginx-parse-config.conf")
	defer helpers.RemoveFileWithErrorCheck(t, file.Name())

	errorLog := helpers.CreateFileWithErrorCheck(t, dir, "error.log")
	defer helpers.RemoveFileWithErrorCheck(t, errorLog.Name())

	accessLog := helpers.CreateFileWithErrorCheck(t, dir, "access.log")
	defer helpers.RemoveFileWithErrorCheck(t, accessLog.Name())

	instance := protos.NginxOssInstance([]string{})
	instance.InstanceRuntime.ConfigPath = file.Name()

	tests := []struct {
		content               string
		expectedLog           string
		name                  string
		expectedSyslogServers []string
		portInUse             bool
	}{
		{
			name:                  "Test 1: Valid port",
			expectedSyslogServers: []string{"127.0.0.1:1515"},
			content: testconfig.NginxConfigWithMultipleSysLogs(errorLog.Name(), accessLog.Name(),
				"192.168.12.34:1517", "my.domain.com:1517", "127.0.0.1:1515"),
			expectedLog: "Found NAP syslog server",
		},
		{
			name:                  "Test 2: No valid server",
			expectedSyslogServers: []string{},
			content: testconfig.NginxConfigWithMultipleSysLogs(errorLog.Name(), accessLog.Name(),
				"random.domain:1515", "192.168.12.34:1517", "my.domain.com:1517"),
			expectedLog: "Could not find available local NGINX App Protect syslog server. " +
				"Security violations will not be collected.",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			logBuf := &bytes.Buffer{}
			stub.StubLoggerWith(logBuf)

			agentConfig := types.AgentConfig()
			agentConfig.AllowedDirectories = []string{dir}
			nginxConfig := NewNginxConfigParser(agentConfig)

			writeErr := os.WriteFile(file.Name(), []byte(test.content), 0o600)
			require.NoError(t, writeErr)

			result, parseError := nginxConfig.Parse(ctx, instance)
			require.NoError(t, parseError)

			t.Logf("Log: %s", logBuf.String())
			helpers.ValidateLog(t, test.expectedLog, logBuf)
			logBuf.Reset()

			assert.Equal(t, test.expectedSyslogServers, result.NAPSysLogServers)
		})
	}
}

func TestNginxConfigParser_findValidSysLogServers(t *testing.T) {
	servers := []string{
		"syslog:server=192.168.12.34:1517", "syslog:server=my.domain.com:1517", "syslog:server=127.0.0.1:1515",
		"syslog:server=localhost:1516", "syslog:server=127.255.255.255:1517",
	}
	expected := []string{"", "", "127.0.0.1:1515", "localhost:1516", "127.255.255.255:1517"}
	ncp := NewNginxConfigParser(types.AgentConfig())

	for i, server := range servers {
		result := ncp.findLocalSysLogServers(server)

		assert.Equal(t, expected[i], result)
	}
}

func TestNginxConfigParser_checkLog(t *testing.T) {
	logBuf := &bytes.Buffer{}
	stub.StubLoggerWith(logBuf)
	tests := []struct {
		name               string
		expectedLog        string
		accessLog          *model.AccessLog
		currentAccessLogs  []*model.AccessLog
		expectedAccessLogs []*model.AccessLog
	}{
		{
			name: "Test 1: valid access log",
			accessLog: &model.AccessLog{
				Name: "/var/log/nginx/access2.log",
				Format: "$remote_addr - $remote_user [$time_local] \"$request\" \"$http_user_agent\" " +
					"\"$http_x_forwarded_for\"$status $body_bytes_sent \"$http_referer\"",
				Permissions: "",
				Readable:    true,
			},
			currentAccessLogs: []*model.AccessLog{
				{
					Name: "/var/log/nginx/access.log",
					Format: "$remote_addr - $remote_user [$time_local] \"$request\" \"$http_user_agent\" " +
						"\"$http_x_forwarded_for\"$status $body_bytes_sent \"$http_referer\"",
					Permissions: "",
					Readable:    true,
				},
			},
			expectedAccessLogs: []*model.AccessLog{
				{
					Name: "/var/log/nginx/access.log",
					Format: "$remote_addr - $remote_user [$time_local] \"$request\" \"$http_user_agent\" " +
						"\"$http_x_forwarded_for\"$status $body_bytes_sent \"$http_referer\"",
					Permissions: "",
					Readable:    true,
				},
				{
					Name: "/var/log/nginx/access2.log",
					Format: "$remote_addr - $remote_user [$time_local] \"$request\" \"$http_user_agent\" " +
						"\"$http_x_forwarded_for\"$status $body_bytes_sent \"$http_referer\"",
					Permissions: "",
					Readable:    true,
				},
			},
			expectedLog: "Found valid access log",
		},
		{
			name: "Test 2: Duplicate access log, with same format",
			accessLog: &model.AccessLog{
				Name: "/var/log/nginx/access.log",
				Format: "$remote_addr - $remote_user [$time_local] \"$request\" \"$http_user_agent\" " +
					"\"$http_x_forwarded_for\"$status $body_bytes_sent \"$http_referer\"",
				Permissions: "",
				Readable:    true,
			},
			currentAccessLogs: []*model.AccessLog{
				{
					Name: "/var/log/nginx/access.log",
					Format: "$remote_addr - $remote_user [$time_local] \"$request\" \"$http_user_agent\" " +
						"\"$http_x_forwarded_for\"$status $body_bytes_sent \"$http_referer\"",
					Permissions: "",
					Readable:    true,
				},
			},
			expectedAccessLogs: []*model.AccessLog{
				{
					Name: "/var/log/nginx/access.log",
					Format: "$remote_addr - $remote_user [$time_local] \"$request\" \"$http_user_agent\" " +
						"\"$http_x_forwarded_for\"$status $body_bytes_sent \"$http_referer\"",
					Permissions: "",
					Readable:    true,
				},
			},
			expectedLog: "Found duplicate access log, skipping",
		},

		{
			name: "Test 3: invalid access log, duplicate access log with different format",
			accessLog: &model.AccessLog{
				Name: "/var/log/nginx/access.log",
				Format: "$remote_addr - $remote_user [$time_local] \"$request\" \"$http_user_agent\" " +
					"\"$http_x_forwarded_for\"$status $body_bytes_sent",
				Permissions: "",
				Readable:    true,
			},
			currentAccessLogs: []*model.AccessLog{
				{
					Name: "/var/log/nginx/access.log",
					Format: "$remote_addr - $remote_user [$time_local] \"$request\" \"$http_user_agent\" " +
						"\"$http_x_forwarded_for\"$status $body_bytes_sent \"$http_referer\"",
					Permissions: "",
					Readable:    true,
				},
			},
			expectedAccessLogs: []*model.AccessLog{},
			expectedLog: "Found multiple log_format directives for the same access log. " +
				"Multiple log formats are not supported in the same access log, metrics from this access log " +
				"will not be collected",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ncp := NewNginxConfigParser(types.AgentConfig())
			logs := ncp.addAccessLog(test.accessLog, test.currentAccessLogs)
			assert.Equal(t, test.expectedAccessLogs, logs)

			helpers.ValidateLog(t, test.expectedLog, logBuf)

			logBuf.Reset()
		})
	}
}

// nolint: maintidx
func TestNginxConfigParser_urlsForLocationDirective(t *testing.T) {
	tmpDir := t.TempDir()
	for _, tt := range []struct {
		name string
		conf string
		oss  []*model.APIDetails
		plus []*model.APIDetails
	}{
		{
			plus: []*model.APIDetails{
				{
					URL:      "http://127.0.0.1:80/api/",
					Listen:   "127.0.0.1:80",
					Location: "/api/",
				},
				{
					URL:      "http://localhost:80/api/",
					Listen:   "localhost:80",
					Location: "/api/",
				},
			},
			name: "Test 1: listen localhost 80, allow 127.0.0.1 - Plus",
			conf: testConf01,
		},
		{
			plus: []*model.APIDetails{
				{
					URL:      "http://127.0.0.1:80/api/",
					Listen:   "127.0.0.1:80",
					Location: "/api/",
				},
			},
			name: "Test 2: listen *:80 - Plus",
			conf: testConf02,
		},
		{
			plus: []*model.APIDetails{
				{
					URL:      "http://127.0.0.1:80/api/",
					Listen:   "127.0.0.1:80",
					Location: "/api/",
				},
			},
			name: "Test 3: server_name _ - Plus",
			conf: testConf03,
		},
		{
			plus: []*model.APIDetails{
				{
					URL:      "http://127.0.0.1:8888/api/",
					Listen:   "127.0.0.1:8888",
					Location: "/api/",
				},
				{
					URL:      "http://status.internal.com:8888/api/",
					Listen:   "status.internal.com:8888",
					Location: "/api/",
				},
			},
			name: "Test 4:  server_name status.internal.com - Plus",
			conf: testConf04,
		},
		{
			plus: []*model.APIDetails{
				{
					URL:      "http://127.0.0.1:8080/privateapi",
					Listen:   "127.0.0.1:8080",
					Location: "/privateapi",
				},
			},
			name: "Test 5:  location /privateapi - Plus",
			conf: testConf05,
		},
		{
			plus: []*model.APIDetails{
				{
					URL:      "http://127.0.0.1:80/api/",
					Listen:   "127.0.0.1:80",
					Location: "/api/",
				},
				{
					URL:      "http://[::1]:80/api/",
					Listen:   "[::1]:80",
					Location: "/api/",
				},
			},
			name: "Test 6:  listen [::]:80 default_server - Plus",
			conf: testConf06,
		},
		{
			plus: []*model.APIDetails{
				{
					URL:      "http://127.0.0.1:80/api/",
					Listen:   "127.0.0.1:80",
					Location: "/api/",
				},
			},
			name: "Test 7:  listen 127.0.0.1, server_name _ - Plus",
			conf: testConf07,
		},
		{
			plus: []*model.APIDetails{
				{
					URL:      "http://127.0.0.1:80/api/",
					Listen:   "127.0.0.1:80",
					Location: "/api/",
				},
			},
			name: "Test 8: location = /api/, listen 127.0.0.1 - Plus",
			conf: testConf08,
		},
		{
			plus: []*model.APIDetails{
				{
					URL:      "http://127.0.0.1:80/api/",
					Listen:   "127.0.0.1:80",
					Location: "/api/",
				},
			},
			name: "Test 9:  location = /api/ , listen 80 - Plus",
			conf: testConf09,
		},
		{
			plus: []*model.APIDetails{
				{
					URL:      "http://127.0.0.1:80/api/",
					Listen:   "127.0.0.1:80",
					Location: "/api/",
				},
			},
			name: "Test 10: listen :80 - Plus",
			conf: testConf10,
		},
		{
			plus: []*model.APIDetails{
				{
					URL:      "http://localhost:80/api/",
					Listen:   "localhost:80",
					Location: "/api/",
				},
			},
			name: "Test 11: listen localhost - Plus",
			conf: testConf11,
		},
		{
			plus: []*model.APIDetails{
				{
					URL:      "http://[::1]:80/api/",
					Listen:   "[::1]:80",
					Location: "/api/",
				},
			},
			name: "Test 12: listen [::1] - Plus",
			conf: testConf12,
		},
		{
			plus: []*model.APIDetails{
				{
					URL:      "http://[::1]:8000/api/",
					Listen:   "[::1]:8000",
					Location: "/api/",
				},
			},
			name: "Test 13: listen [::]:8000 - Plus",
			conf: testConf13,
		},
		{
			oss: []*model.APIDetails{
				{
					URL:      "http://localhost:80/stub_status",
					Listen:   "localhost:80",
					Location: "/stub_status",
				},
				{
					URL:      "http://127.0.0.1:80/stub_status",
					Listen:   "127.0.0.1:80",
					Location: "/stub_status",
				},
			},
			name: "Test 14: listen 127.0.0.1:80, server_name localhost - OSS",
			conf: testConf14,
		},
		{
			oss: []*model.APIDetails{
				{
					URL:      "http://localhost:80/stub_status",
					Listen:   "localhost:80",
					Location: "/stub_status",
				},
				{
					URL:      "http://127.0.0.1:80/stub_status",
					Listen:   "127.0.0.1:80",
					Location: "/stub_status",
				},
			},
			name: "Test 15: listen :80, server_name localhost - OSS",
			conf: testConf15,
		},
		{
			oss: []*model.APIDetails{
				{
					URL:      "http://localhost:80/stub_status",
					Listen:   "localhost:80",
					Location: "/stub_status",
				},
				{
					URL:      "http://127.0.0.1:80/stub_status",
					Listen:   "127.0.0.1:80",
					Location: "/stub_status",
				},
			},
			name: "Test 16: listen 80, server_name localhost - OSS",
			conf: testConf16,
		},
		{
			oss: []*model.APIDetails{
				{
					URL:      "http://localhost:80/stub_status",
					Listen:   "localhost:80",
					Location: "/stub_status",
				},
				{
					URL:      "http://127.0.0.1:80/stub_status",
					Listen:   "127.0.0.1:80",
					Location: "/stub_status",
				},
			},
			name: "Test 17: location = /stub_status - OSS",
			conf: testConf17,
		},
		{
			oss: []*model.APIDetails{
				{
					URL:      "http://localhost:80/stub_status",
					Listen:   "localhost:80",
					Location: "/stub_status",
				},
				{
					URL:      "http://127.0.0.1:80/stub_status",
					Listen:   "127.0.0.1:80",
					Location: "/stub_status",
				},
			},
			plus: []*model.APIDetails{
				{
					URL:      "http://localhost:80/api/",
					Listen:   "localhost:80",
					Location: "/api/",
				},
				{
					URL:      "http://127.0.0.1:80/api/",
					Listen:   "127.0.0.1:80",
					Location: "/api/",
				},
			},
			name: "Test 18: listen 80 - OSS & Plus",
			conf: testConf18,
		},
		{
			plus: []*model.APIDetails{
				{
					URL:      "http://127.0.0.1:49151/api",
					Listen:   "127.0.0.1:49151",
					Location: "/api",
				},
				{
					URL:      "http://127.0.0.1:49151/api",
					Listen:   "127.0.0.1:49151",
					Location: "/api",
				},
			},
			name: "Test 19: listen 127.0.0.1:49151 - Plus",
			conf: testConf19,
		},
		{
			oss: []*model.APIDetails{
				{
					URL:      "http://config-status/stub_status",
					Listen:   "unix:/var/run/nginx/nginx-status.sock",
					Location: "/stub_status",
				},
			},
			name: "Test 20: unix:/var/run/nginx/nginx-status.sock - OSS Unix Socket",
			conf: testConf20,
		},
		{
			plus: []*model.APIDetails{
				{
					URL:      "http://nginx-plus-api/api",
					Listen:   "unix:/var/run/nginx/nginx-plus-api.sock",
					Location: "/api",
				},
			},
			name: "Test 21: listen unix:/var/run/nginx/nginx-plus-api.sock - Plus Unix Socket",
			conf: testConf21,
		},
		{
			plus: []*model.APIDetails{
				{
					URL:      "http://nginx-plus-api/api",
					Listen:   "unix:/var/lib/nginx/nginx-plus-api.sock",
					Location: "/api",
				},
			},
			name: "Test 22: Multiple Plus Unix Sockets",
			conf: testConf22,
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
		ncp := NewNginxConfigParser(types.AgentConfig())

		var oss, plus []*model.APIDetails

		assert.Len(t, payload.Config, 1)
		for _, xpConf := range payload.Config {
			assert.Len(t, xpConf.Parsed, 1)
			err = ncp.crossplaneConfigTraverse(ctx, &xpConf,
				func(ctx context.Context, parent, directive *crossplane.Directive) error {
					_oss := ncp.urlsForLocationDirectiveAPIDetails(ctx, parent, directive,
						stubStatusAPIDirective)
					_plus := ncp.urlsForLocationDirectiveAPIDetails(ctx, parent, directive, plusAPIDirective)
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
func TestNginxConfigParser_pingAPIEndpoint_PlusAPI(t *testing.T) {
	handler := http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		if req.URL.String() == "/good_api" {
			data := []byte("[1,2,3,4,5,6,7,8]")
			_, err := rw.Write(data)
			// go-require: do not use require in http handlers (testifylint), using assert instead
			assert.NoError(t, err)
		} else if req.URL.String() == "/invalid_body_api" {
			data := []byte("Invalid")
			_, err := rw.Write(data)
			// go-require: do not use require in http handlers (testifylint), using assert instead
			assert.NoError(t, err)
		} else {
			rw.WriteHeader(http.StatusInternalServerError)
			data := []byte("")
			_, err := rw.Write(data)
			// go-require: do not use require in http handlers (testifylint), using assert instead
			assert.NoError(t, err)
		}
	})

	fakeServer := httptest.NewServer(handler)
	defer fakeServer.Close()

	nginxConfigParser := NewNginxConfigParser(types.AgentConfig())

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
			result := nginxConfigParser.pingAPIEndpoint(ctx, &model.APIDetails{
				URL:    fmt.Sprintf("%s%s", fakeServer.URL, test.endpoint),
				Listen: "",
			}, "api")
			assert.Equal(t, test.expected, result)
		})
	}
}

// linter doesn't like the duplicate handler and server function
// nolint: dupl
func TestNginxConfigParser_pingAPIEndpoint_StubStatus(t *testing.T) {
	handler := http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		if req.URL.String() == "/good_api" {
			data := []byte(`
Active connections: 2
server accepts handled requests
	18 18 3266
Reading: 0 Writing: 1 Waiting: 1
			`)
			_, err := rw.Write(data)

			// go-require: do not use require in http handlers (testifylint), using assert instead
			assert.NoError(t, err)
		} else if req.URL.String() == "/invalid_body_api" {
			data := []byte("Invalid")
			_, err := rw.Write(data)

			// go-require: do not use require in http handlers (testifylint), using assert instead
			assert.NoError(t, err)
		} else {
			rw.WriteHeader(http.StatusInternalServerError)
			data := []byte("")
			_, err := rw.Write(data)

			// go-require: do not use require in http handlers (testifylint), using assert instead
			assert.NoError(t, err)
		}
	})

	fakeServer := httptest.NewServer(handler)
	defer fakeServer.Close()

	nginxConfigParser := NewNginxConfigParser(types.AgentConfig())

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
			statusAPI := &model.APIDetails{
				URL:    fmt.Sprintf("%s%s", fakeServer.URL, test.endpoint),
				Listen: "",
			}
			result := nginxConfigParser.pingAPIEndpoint(ctx, statusAPI, "stub_status")
			assert.Equal(t, test.expected, result)
		})
	}
}

func TestNginxConfigParser_ignoreLog(t *testing.T) {
	tests := []struct {
		name        string
		logPath     string
		expectedLog string
		excludeLogs []string
		expected    bool
	}{
		{
			name:        "Test 1: allowed log path",
			logPath:     "/tmp/var/log/nginx/access.log",
			excludeLogs: []string{},
			expected:    false,
			expectedLog: "",
		},
		{
			name:        "Test 2: syslog",
			logPath:     "syslog:server=unix:/var/log/nginx.sock,nohostname;",
			excludeLogs: []string{},
			expected:    true,
			expectedLog: "",
		},
		{
			name:        "Test 3: log off",
			logPath:     "off",
			excludeLogs: []string{},
			expected:    true,
			expectedLog: "",
		},
		{
			name:        "Test 4: log /dev/stderr",
			logPath:     "/dev/stderr",
			excludeLogs: []string{},
			expected:    true,
			expectedLog: "",
		},
		{
			name:        "Test 5: log /dev/stdout",
			logPath:     "/dev/stdout",
			excludeLogs: []string{},
			expected:    true,
			expectedLog: "",
		},
		{
			name:        "Test 6: log /dev/null",
			logPath:     "/dev/null",
			excludeLogs: []string{},
			expected:    true,
			expectedLog: "",
		},
		{
			name:        "Test 7: exclude logs set, log path should be excluded - regex",
			logPath:     "/tmp/var/log/nginx/alert.log",
			excludeLogs: []string{"\\.log$"},
			expected:    true,
			expectedLog: "",
		},
		{
			name:        "Test 8: exclude logs set, log path should be excluded - full path",
			logPath:     "/tmp/var/log/nginx/alert.log",
			excludeLogs: []string{"/tmp/var/log/nginx/alert.log"},
			expected:    true,
			expectedLog: "",
		},
		{
			name:        "Test 9: exclude logs set, log path is allowed",
			logPath:     "/tmp/var/log/nginx/access.log",
			excludeLogs: []string{"/tmp/var/log/nginx/alert.log", "\\.swp$"},
			expected:    false,
			expectedLog: "",
		},
		{
			name:        "Test 10: log path outside allowed dir",
			logPath:     "/var/log/nginx/access.log",
			excludeLogs: []string{"/tmp/var/log/nginx/alert.log", "\\.swp$"},
			expected:    false,
			expectedLog: "Log being read is outside of allowed directories",
		},
		{
			name:        "Test 10: log stderr",
			logPath:     "stderr",
			excludeLogs: []string{},
			expected:    true,
			expectedLog: "",
		},
		{
			name:        "Test 11: log stdout",
			logPath:     "stdout",
			excludeLogs: []string{},
			expected:    true,
			expectedLog: "",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			logBuf := &bytes.Buffer{}
			stub.StubLoggerWith(logBuf)

			agentConfig := types.AgentConfig()
			agentConfig.DataPlaneConfig.Nginx.ExcludeLogs = test.excludeLogs

			ncp := NewNginxConfigParser(agentConfig)
			assert.Equal(t, test.expected, ncp.ignoreLog(test.logPath))

			helpers.ValidateLog(t, test.expectedLog, logBuf)

			logBuf.Reset()
		})
	}
}

func TestNginxConfigParser_checkDuplicate(t *testing.T) {
	fileContent := []byte("location /test {\n    return 200 \"Test location\\n\";\n}")
	fileContentNew := []byte("some test data")
	fileHash := files.GenerateHash(fileContent)
	fileHashNew := files.GenerateHash(fileContentNew)

	tests := []struct {
		file     *mpi.File
		name     string
		expected bool
	}{
		{
			name: "Test 1: File already in files",
			file: &mpi.File{
				FileMeta: &mpi.FileMeta{
					Name:         "/etc/nginx/certs/nginx-repo.crt",
					Hash:         fileHashNew,
					ModifiedTime: timestamppb.Now(),
					Permissions:  "0640",
					Size:         0,
				},
			},
			expected: true,
		},
		{
			name: "Test 2: File not in files",
			file: &mpi.File{
				FileMeta: &mpi.FileMeta{
					Name:         "/etc/nginx/certs/nginx-repo-new.crt",
					Hash:         fileHashNew,
					ModifiedTime: timestamppb.Now(),
					Permissions:  "0640",
					Size:         0,
				},
			},
			expected: false,
		},
	}

	nginxConfigContextFiles := model.NginxConfigContext{
		Files: []*mpi.File{
			{
				FileMeta: &mpi.FileMeta{
					Name:         "/etc/nginx/certs/nginx-repo.crt",
					Hash:         fileHash,
					ModifiedTime: timestamppb.Now(),
					Permissions:  "0640",
					Size:         0,
				},
			},
			{
				FileMeta: &mpi.FileMeta{
					Name:         "/etc/nginx/keys/nginx-repo.key",
					Hash:         fileHash,
					ModifiedTime: timestamppb.Now(),
					Permissions:  "0640",
					Size:         0,
				},
			},
			{
				FileMeta: &mpi.FileMeta{
					Name:         "/etc/nginx/keys/inline_key.pem",
					Hash:         fileHash,
					ModifiedTime: timestamppb.Now(),
					Permissions:  "0640",
					Size:         0,
				},
			},
			{
				FileMeta: &mpi.FileMeta{
					Name:         "/etc/nginx/certs/inline_cert.pem",
					Hash:         fileHash,
					ModifiedTime: timestamppb.Now(),
					Permissions:  "0640",
					Size:         0,
				},
			},
		},
	}

	for _, test := range tests {
		ncp := NewNginxConfigParser(types.AgentConfig())
		t.Run(test.name, func(t *testing.T) {
			assert.Equal(t, test.expected, ncp.isDuplicateFile(nginxConfigContextFiles.Files, test.file))
		})
	}
}

func TestNginxConfigParser_parseIncludeDirective(t *testing.T) {
	parser := NewNginxConfigParser(types.AgentConfig())

	tests := []struct {
		name     string
		confFile string
		expected string
		args     []string
	}{
		{
			name:     "Test 1: relative path",
			args:     []string{"test.conf"},
			confFile: "/etc/nginx/nginx.conf",
			expected: "/etc/nginx/test.conf",
		},
		{
			name:     "Test 2: absolute path",
			args:     []string{"/usr/local/nginx/conf/vhost.conf"},
			confFile: "/etc/nginx/nginx.conf",
			expected: "/usr/local/nginx/conf/vhost.conf",
		},
		{
			name:     "Test 3: wildcard",
			args:     []string{"/etc/nginx/conf.d/*.conf"},
			confFile: "/etc/nginx/nginx.conf",
			expected: "/etc/nginx/conf.d/*.conf",
		},
		{
			name:     "Test 4: relative path with subdirectory",
			args:     []string{"conf.d/default.conf"},
			confFile: "/etc/nginx/nginx.conf",
			expected: "/etc/nginx/conf.d/default.conf",
		},
		{
			name:     "Test 5: parent directory reference",
			args:     []string{"../sites-enabled/*.conf"},
			confFile: "/etc/nginx/conf.d/nginx.conf",
			expected: "/etc/nginx/sites-enabled/*.conf",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			include := parser.parseIncludeDirective(
				&crossplane.Directive{
					Args: tc.args,
				},
				&crossplane.Config{
					File: tc.confFile,
				},
			)
			assert.Equal(t, tc.expected, include)
		})
	}
}

func protoListEqual(protoListA, protoListB []*mpi.File) bool {
	for i := range protoListA {
		res := proto.Equal(protoListA[i], protoListB[i])
		if !res {
			return false
		}
	}

	return true
}
