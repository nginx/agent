/**
 * Copyright (c) F5, Inc.
 *
 * This source code is licensed under the Apache License, Version 2.0 license found in the
 * LICENSE file in the root directory of this source tree.
 */

package core

import (
	"bytes"
	"encoding/base64"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/nginx/agent/sdk/v2/proto"
	"github.com/nginx/agent/sdk/v2/zip"
	"github.com/nginx/agent/v2/src/core/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const CONF_TEMPLATE = `
user www-data;
worker_processes auto;
pid /run/nginx.pid;

events {
worker_connections 768;
}

http {
sendfile on;
tcp_nopush on;
tcp_nodelay on;
keepalive_timeout 65;
types_hash_max_size 2048;

access_log /var/log/nginx/access.log;
error_log /var/log/nginx/error.log;

server {
	listen 80 default_server;
	listen [::]:80 default_server;
	server_name  localhost;

	location / {
		root %s/aux/;
	}
}

gzip on;
}
					
`

func TestGetNginxInfoFromBuffer(t *testing.T) {
	tests := []struct {
		name              string
		input             string
		expectedNginxInfo *nginxInfo
	}{
		{
			name: "normal nginx install",
			input: `nginx version: nginx/1.19.10
			built by clang 12.0.0 (clang-1200.0.32.29)
			built with OpenSSL 1.1.1k  25 Mar 2021
			TLS SNI support enabled
			configure arguments: --prefix=/usr/local/Cellar/nginx/1.19.10 --modules-path=/tmp/modules --sbin-path=/usr/local/Cellar/nginx/1.19.10/bin/nginx --with-cc-opt='-I/usr/local/opt/pcre/include -I/usr/local/opt/openssl@1.1/include' --with-ld-opt='-L/usr/local/opt/pcre/lib -L/usr/local/opt/openssl@1.1/lib' --conf-path=/usr/local/etc/nginx/nginx.conf --pid-path=/usr/local/var/run/nginx.pid --lock-path=/usr/local/var/run/nginx.lock --http-client-body-temp-path=/usr/local/var/run/nginx/client_body_temp --http-proxy-temp-path=/usr/local/var/run/nginx/proxy_temp --http-fastcgi-temp-path=/usr/local/var/run/nginx/fastcgi_temp --http-uwsgi-temp-path=/usr/local/var/run/nginx/uwsgi_temp --http-scgi-temp-path=/usr/local/var/run/nginx/scgi_temp --http-log-path=/usr/local/var/log/nginx/access.log --error-log-path=/usr/local/var/log/nginx/error.log --with-compat --with-debug --with-http_addition_module --with-http_auth_request_module --with-http_dav_module --with-http_degradation_module --with-http_flv_module --with-http_gunzip_module --with-http_gzip_static_module --with-http_mp4_module --with-http_random_index_module --with-http_realip_module --with-http_secure_link_module --with-http_slice_module --with-http_ssl_module --with-http_stub_status_module --with-http_sub_module --with-http_v2_module --with-ipv6 --with-mail --with-mail_ssl_module --with-pcre --with-pcre-jit --with-stream --with-stream_realip_module --with-stream_ssl_module --with-stream_ssl_preread_module`, expectedNginxInfo: &nginxInfo{
				prefix:    "/usr/local/Cellar/nginx/1.19.10",
				confPath:  "/usr/local/etc/nginx/nginx.conf",
				logPath:   "/usr/local/var/log/nginx/access.log",
				errorPath: "/usr/local/var/log/nginx/error.log",
				version:   "1.19.10",
				plusver:   "",
				source:    "built by clang 12.0.0 (clang-1200.0.32.29)",
				ssl: []string{
					"OpenSSL",
					"1.1.1k",
					"25 Mar 2021",
				},
				cfgf: map[string]interface{}{
					"conf-path":                      "/usr/local/etc/nginx/nginx.conf",
					"error-log-path":                 "/usr/local/var/log/nginx/error.log",
					"modules-path":                   "/tmp/modules",
					"http-client-body-temp-path":     "/usr/local/var/run/nginx/client_body_temp",
					"http-fastcgi-temp-path":         "/usr/local/var/run/nginx/fastcgi_temp",
					"http-log-path":                  "/usr/local/var/log/nginx/access.log",
					"http-proxy-temp-path":           "/usr/local/var/run/nginx/proxy_temp",
					"http-scgi-temp-path":            "/usr/local/var/run/nginx/scgi_temp",
					"http-uwsgi-temp-path":           "/usr/local/var/run/nginx/uwsgi_temp",
					"lock-path":                      "/usr/local/var/run/nginx.lock",
					"pid-path":                       "/usr/local/var/run/nginx.pid",
					"prefix":                         "/usr/local/Cellar/nginx/1.19.10",
					"sbin-path":                      "/usr/local/Cellar/nginx/1.19.10/bin/nginx",
					"with-cc-opt":                    "'-I/usr/local/opt/pcre/include -I/usr/local/opt/openssl@1.1/include'",
					"with-compat":                    true,
					"with-debug":                     true,
					"with-http_addition_module":      true,
					"with-http_auth_request_module":  true,
					"with-http_dav_module":           true,
					"with-http_degradation_module":   true,
					"with-http_flv_module":           true,
					"with-http_gunzip_module":        true,
					"with-http_gzip_static_module":   true,
					"with-http_mp4_module":           true,
					"with-http_random_index_module":  true,
					"with-http_realip_module":        true,
					"with-http_secure_link_module":   true,
					"with-http_slice_module":         true,
					"with-http_ssl_module":           true,
					"with-http_stub_status_module":   true,
					"with-http_sub_module":           true,
					"with-http_v2_module":            true,
					"with-ipv6":                      true,
					"with-ld-opt":                    "'-L/usr/local/opt/pcre/lib -L/usr/local/opt/openssl@1.1/lib'",
					"with-mail":                      true,
					"with-mail_ssl_module":           true,
					"with-pcre":                      true,
					"with-pcre-jit":                  true,
					"with-stream":                    true,
					"with-stream_realip_module":      true,
					"with-stream_ssl_module":         true,
					"with-stream_ssl_preread_module": true,
				},
				configureArgs: []string{
					"",
					"prefix=/usr/local/Cellar/nginx/1.19.10",
					"modules-path=/tmp/modules",
					"sbin-path=/usr/local/Cellar/nginx/1.19.10/bin/nginx",
					"with-cc-opt='-I/usr/local/opt/pcre/include -I/usr/local/opt/openssl@1.1/include'",
					"with-ld-opt='-L/usr/local/opt/pcre/lib -L/usr/local/opt/openssl@1.1/lib'",
					"conf-path=/usr/local/etc/nginx/nginx.conf",
					"pid-path=/usr/local/var/run/nginx.pid",
					"lock-path=/usr/local/var/run/nginx.lock",
					"http-client-body-temp-path=/usr/local/var/run/nginx/client_body_temp",
					"http-proxy-temp-path=/usr/local/var/run/nginx/proxy_temp",
					"http-fastcgi-temp-path=/usr/local/var/run/nginx/fastcgi_temp",
					"http-uwsgi-temp-path=/usr/local/var/run/nginx/uwsgi_temp",
					"http-scgi-temp-path=/usr/local/var/run/nginx/scgi_temp",
					"http-log-path=/usr/local/var/log/nginx/access.log",
					"error-log-path=/usr/local/var/log/nginx/error.log",
					"with-compat",
					"with-debug",
					"with-http_addition_module",
					"with-http_auth_request_module",
					"with-http_dav_module",
					"with-http_degradation_module",
					"with-http_flv_module",
					"with-http_gunzip_module",
					"with-http_gzip_static_module",
					"with-http_mp4_module",
					"with-http_random_index_module",
					"with-http_realip_module",
					"with-http_secure_link_module",
					"with-http_slice_module",
					"with-http_ssl_module",
					"with-http_stub_status_module",
					"with-http_sub_module",
					"with-http_v2_module",
					"with-ipv6",
					"with-mail",
					"with-mail_ssl_module",
					"with-pcre",
					"with-pcre-jit",
					"with-stream",
					"with-stream_realip_module",
					"with-stream_ssl_module",
					"with-stream_ssl_preread_module",
				},
				loadableModules: nil,
				modulesPath:     "/tmp/modules",
			},
		},
		{
			name: "custom nginx install",
			input: `nginx version: nginx/1.19.10
			TLS SNI support enabled
			configure arguments: --prefix=/usr/local/Cellar/nginx/1.19.10 --sbin-path=/usr/local/Cellar/nginx/1.19.10/bin/nginx --with-cc-opt='-I/usr/local/opt/pcre/include -I/usr/local/opt/openssl@1.1/include'`,
			expectedNginxInfo: &nginxInfo{
				prefix:   "/usr/local/Cellar/nginx/1.19.10",
				confPath: "/usr/local/Cellar/nginx/1.19.10/conf/nginx.conf",
				version:  "1.19.10",
				plusver:  "",
				source:   "",
				cfgf: map[string]interface{}{
					"prefix":      "/usr/local/Cellar/nginx/1.19.10",
					"sbin-path":   "/usr/local/Cellar/nginx/1.19.10/bin/nginx",
					"with-cc-opt": "'-I/usr/local/opt/pcre/include -I/usr/local/opt/openssl@1.1/include'",
				},
				configureArgs: []string{
					"",
					"prefix=/usr/local/Cellar/nginx/1.19.10",
					"sbin-path=/usr/local/Cellar/nginx/1.19.10/bin/nginx",
					"with-cc-opt='-I/usr/local/opt/pcre/include -I/usr/local/opt/openssl@1.1/include'",
				},
				loadableModules: nil,
				modulesPath:     "",
			},
		},
		{
			name: "custom nginx install no config args",
			input: `nginx version: nginx/1.19.10
			TLS SNI support enabled
			configure arguments: `,
			expectedNginxInfo: &nginxInfo{
				prefix:          "/usr/local/nginx",
				confPath:        "/usr/local/nginx/conf/nginx.conf",
				version:         "1.19.10",
				plusver:         "",
				source:          "",
				cfgf:            map[string]interface{}{},
				configureArgs:   []string{""},
				loadableModules: nil,
				modulesPath:     "",
			},
		},
	}

	err := os.Mkdir("/tmp/modules", 0700)
	assert.NoError(t, err)

	tempDir := t.TempDir()
	mockNginx, err := os.CreateTemp(tempDir, "mock_nginx_executable")
	assert.NoError(t, err)

	defer func() {
		_ = mockNginx.Close()
		_ = os.RemoveAll(mockNginx.Name())
		_ = os.RemoveAll("/tmp/modules")
	}()

	for _, test := range tests {
		t.Run(test.name, func(tt *testing.T) {
			binary := NginxBinaryType{
				env: &EnvironmentType{},
			}

			var buffer bytes.Buffer
			buffer.WriteString(test.input)
			nginxInfo := binary.getNginxInfoFromBuffer(filepath.Join(tempDir, mockNginx.Name()), &buffer)

			assert.Equal(t, test.expectedNginxInfo.cfgf, nginxInfo.cfgf)
			assert.Equal(t, test.expectedNginxInfo.confPath, nginxInfo.confPath)
			assert.Equal(t, test.expectedNginxInfo.configureArgs, nginxInfo.configureArgs)
			assert.Equal(t, test.expectedNginxInfo.errorPath, nginxInfo.errorPath)
			assert.Equal(t, test.expectedNginxInfo.loadableModules, nginxInfo.loadableModules)
			assert.Equal(t, test.expectedNginxInfo.logPath, nginxInfo.logPath)
			assert.Equal(t, test.expectedNginxInfo.modulesPath, nginxInfo.modulesPath)
			assert.Equal(t, test.expectedNginxInfo.plusver, nginxInfo.plusver)
			assert.Equal(t, test.expectedNginxInfo.prefix, nginxInfo.prefix)
			assert.Equal(t, test.expectedNginxInfo.source, nginxInfo.source)
			assert.Equal(t, test.expectedNginxInfo.ssl, nginxInfo.ssl)
			assert.Equal(t, test.expectedNginxInfo.version, nginxInfo.version)
			assert.NotNil(t, nginxInfo.mtime)
		})
	}
}

func TestParseConfigureArguemtns(t *testing.T) {
	input := `configure arguments: --prefix=/usr/local/Cellar/nginx/1.19.10 --sbin-path=/usr/local/Cellar/nginx/1.19.10/bin/nginx --with-cc-opt='-I/usr/local/opt/pcre/include -I/usr/local/opt/openssl@1.1/include' --with-ld-opt='-L/usr/local/opt/pcre/lib -L/usr/local/opt/openssl@1.1/lib' --conf-path=/usr/local/etc/nginx/nginx.conf --pid-path=/usr/local/var/run/nginx.pid --lock-path=/usr/local/var/run/nginx.lock --http-client-body-temp-path=/usr/local/var/run/nginx/client_body_temp --http-proxy-temp-path=/usr/local/var/run/nginx/proxy_temp --http-fastcgi-temp-path=/usr/local/var/run/nginx/fastcgi_temp --http-uwsgi-temp-path=/usr/local/var/run/nginx/uwsgi_temp --http-scgi-temp-path=/usr/local/var/run/nginx/scgi_temp --http-log-path=/usr/local/var/log/nginx/access.log --error-log-path=/usr/local/var/log/nginx/error.log --with-compat --with-debug --with-http_addition_module --with-http_auth_request_module --with-http_dav_module --with-http_degradation_module --with-http_flv_module --with-http_gunzip_module --with-http_gzip_static_module --with-http_mp4_module --with-http_random_index_module --with-http_realip_module --with-http_secure_link_module --with-http_slice_module --with-http_ssl_module --with-http_stub_status_module --with-http_sub_module --with-http_v2_module --with-ipv6 --with-mail --with-mail_ssl_module --with-pcre --with-pcre-jit --with-stream --with-stream_realip_module --with-stream_ssl_module --with-stream_ssl_preread_module`

	expected := map[string]interface{}{
		"conf-path":                      "/usr/local/etc/nginx/nginx.conf",
		"error-log-path":                 "/usr/local/var/log/nginx/error.log",
		"http-client-body-temp-path":     "/usr/local/var/run/nginx/client_body_temp",
		"http-fastcgi-temp-path":         "/usr/local/var/run/nginx/fastcgi_temp",
		"http-log-path":                  "/usr/local/var/log/nginx/access.log",
		"http-proxy-temp-path":           "/usr/local/var/run/nginx/proxy_temp",
		"http-scgi-temp-path":            "/usr/local/var/run/nginx/scgi_temp",
		"http-uwsgi-temp-path":           "/usr/local/var/run/nginx/uwsgi_temp",
		"lock-path":                      "/usr/local/var/run/nginx.lock",
		"pid-path":                       "/usr/local/var/run/nginx.pid",
		"prefix":                         "/usr/local/Cellar/nginx/1.19.10",
		"sbin-path":                      "/usr/local/Cellar/nginx/1.19.10/bin/nginx",
		"with-cc-opt":                    "'-I/usr/local/opt/pcre/include -I/usr/local/opt/openssl@1.1/include'",
		"with-compat":                    true,
		"with-debug":                     true,
		"with-http_addition_module":      true,
		"with-http_auth_request_module":  true,
		"with-http_dav_module":           true,
		"with-http_degradation_module":   true,
		"with-http_flv_module":           true,
		"with-http_gunzip_module":        true,
		"with-http_gzip_static_module":   true,
		"with-http_mp4_module":           true,
		"with-http_random_index_module":  true,
		"with-http_realip_module":        true,
		"with-http_secure_link_module":   true,
		"with-http_slice_module":         true,
		"with-http_ssl_module":           true,
		"with-http_stub_status_module":   true,
		"with-http_sub_module":           true,
		"with-http_v2_module":            true,
		"with-ipv6":                      true,
		"with-ld-opt":                    "'-L/usr/local/opt/pcre/lib -L/usr/local/opt/openssl@1.1/lib'",
		"with-mail":                      true,
		"with-mail_ssl_module":           true,
		"with-pcre":                      true,
		"with-pcre-jit":                  true,
		"with-stream":                    true,
		"with-stream_realip_module":      true,
		"with-stream_ssl_module":         true,
		"with-stream_ssl_preread_module": true,
	}

	result, args := parseConfigureArguments(input)

	assert.Equal(t, expected, result)
	assert.NotNil(t, args)
}

func TestParseNginxVersion(t *testing.T) {
	tests := []struct {
		input       string
		plusVersion string
		version     string
	}{
		{
			input:       "nginx version: nginx/1.19.10",
			plusVersion: "",
			version:     "1.19.10",
		},
	}

	for _, test := range tests {
		versionResult, plusResult := parseNginxVersion(test.input)

		assert.Equal(t, test.plusVersion, plusResult)
		assert.Equal(t, test.version, versionResult)
	}
}

func TestGetConfPath(t *testing.T) {
	result := getConfPathFromCommand("nginx: master process nginx -c /tmp/nginx.conf")
	assert.Equal(t, "/tmp/nginx.conf", result)

	result = getConfPathFromCommand("nginx: master process nginx -c")
	assert.Equal(t, "", result)

	result = getConfPathFromCommand("-c")
	assert.Equal(t, "", result)

	result = getConfPathFromCommand("")
	assert.Equal(t, "", result)
}

func TestBuildSslRun(t *testing.T) {
	input := []string{"hello"}
	result := buildSsl(input, "")
	expected := &proto.NginxSslMetaData{
		SslType: proto.NginxSslMetaData_RUN,
		Details: input,
	}
	assert.Equal(t, expected, result)
}

func TestBuildSslBuilt(t *testing.T) {
	input := []string{"bye"}
	result := buildSsl(input, "built by")
	expected := &proto.NginxSslMetaData{
		SslType: proto.NginxSslMetaData_BUILT,
		Details: input,
	}
	assert.Equal(t, expected, result)
}

func TestWriteBackup(t *testing.T) {
	zippedFile := &proto.ZippedFile{
		RootDirectory: "/tmp",
	}

	tests := []struct {
		name           string
		config         config.Config
		nginxConfig    *proto.NginxConfig
		confFiles      []*proto.File
		auxFiles       []*proto.File
		expectedResult int
	}{
		{
			name:        "enabled test",
			config:      config.Config{Nginx: config.Nginx{Debug: true, TreatWarningsAsErrors: true}},
			nginxConfig: &proto.NginxConfig{Zconfig: zippedFile, Zaux: zippedFile},
			confFiles: []*proto.File{
				{
					Name: "/tmp/file1.html",
				},
			},
			auxFiles: []*proto.File{
				{
					Name: "/tmp/auxfile1.html",
				},
			},
			expectedResult: 2,
		},
		{
			name:           "not enabled test",
			config:         config.Config{Nginx: config.Nginx{Debug: false, TreatWarningsAsErrors: false}},
			nginxConfig:    &proto.NginxConfig{},
			confFiles:      []*proto.File{},
			auxFiles:       []*proto.File{},
			expectedResult: 0,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(tt *testing.T) {
			fakeEnv := FakeEnvironment{}

			binary := NginxBinaryType{config: &test.config, env: &fakeEnv}
			binary.writeBackup(test.nginxConfig, test.confFiles, test.auxFiles)

			assert.Equal(t, test.expectedResult, fakeEnv.WriteFilesCallCount())
		})
	}
}

func TestWriteConfig(t *testing.T) {
	tmpDir := t.TempDir()
	expectedExisting := map[string]struct{}{}
	expectedNotExisting := map[string]struct{}{
		filepath.Join(tmpDir, "/aux/test1.html"): {},
	}

	allowedDirs := make(map[string]struct{})
	allowedDirs[tmpDir] = struct{}{}
	fakeConfig := config.Config{
		AllowedDirectoriesMap: allowedDirs,
	}

	env := EnvironmentType{}
	n := NewNginxBinary(&env, &fakeConfig)

	n.nginxDetailsMap = map[string]*proto.NginxDetails{
		"151d8728e792f42e129337573a21bb30ab3065d59102f075efc2ded28e713ff8": {
			NginxId:     "151d8728e792f42e129337573a21bb30ab3065d59102f075efc2ded28e713ff8",
			ConfPath:    filepath.Join(tmpDir, "/nginx.conf"),
			ProcessId:   "777",
			ProcessPath: "/usr/sbin/nginx",
		},
	}

	if err := os.WriteFile(filepath.Join(tmpDir, "nginx.conf"),
		[]byte(fmt.Sprintf(CONF_TEMPLATE, tmpDir)), 0755); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}

	if err := os.Mkdir(tmpDir+"/aux/", 0755); err != nil {
		t.Fatalf("failed to create aux directory: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "/aux/test2.html"), []byte("<html><html>"), 0755); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}

	nginxConfig, err := buildConfig(tmpDir)
	if err != nil {
		t.Fatal("failed to create test config")
	}
	configApply, err := n.WriteConfig(nginxConfig)

	// Verify configApply
	assert.Equal(t, expectedExisting, configApply.GetExisting())
	assert.Equal(t, expectedNotExisting, configApply.GetNotExists())
	assert.Nil(t, err)

	err = configApply.Complete()
	assert.Nil(t, err)

	// Verify aux file test1.html was created
	_, err = os.Stat(tmpDir + "/aux/test1.html")
	assert.Nil(t, err)
	// Verify aux file test2.html was deleted
	_, err = os.Stat(tmpDir + "/aux/test2.html")
	assert.NotNil(t, err)

	// Verify that rollback on failure works as expected
	err = configApply.Rollback(errors.New("config validation failed"))
	assert.Nil(t, err)

	// Verify aux file test1.html was removed
	_, err = os.Stat(tmpDir + "/aux/test1.html")
	assert.NotNil(t, err)
	// Verify aux file test2.html was restored again
	_, err = os.Stat(tmpDir + "/aux/test2.html")
	assert.Nil(t, err)
}

func TestWriteConfigWithFileAction(t *testing.T) {
	tmpDir := t.TempDir()
	expectedExisting := map[string]struct{}{}
	expectedNotExisting := map[string]struct{}{
		filepath.Join(tmpDir, "/aux/test1.html"): {},
	}

	allowedDirs := make(map[string]struct{})
	allowedDirs[tmpDir] = struct{}{}
	fakeConfig := config.Config{
		AllowedDirectoriesMap: allowedDirs,
	}

	env := EnvironmentType{}
	n := NewNginxBinary(&env, &fakeConfig)

	n.nginxDetailsMap = map[string]*proto.NginxDetails{
		"151d8728e792f42e129337573a21bb30ab3065d59102f075efc2ded28e713ff8": {
			NginxId:     "151d8728e792f42e129337573a21bb30ab3065d59102f075efc2ded28e713ff8",
			ConfPath:    filepath.Join(tmpDir, "/nginx.conf"),
			ProcessId:   "777",
			ProcessPath: "/usr/sbin/nginx",
		},
	}

	if err := os.WriteFile(filepath.Join(tmpDir, "nginx.conf"),
		[]byte(fmt.Sprintf(CONF_TEMPLATE, tmpDir)), 0755); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}

	if err := os.Mkdir(filepath.Join(tmpDir, "aux"), 0755); err != nil {
		t.Fatalf("failed to create aux directory: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "/aux/test2.html"), []byte("<html><html>"), 0755); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}

	if err := os.WriteFile(filepath.Join(tmpDir, "/aux/test3.html"), []byte("<html><html>"), 0755); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}

	nginxConfig, err := buildConfig(tmpDir)
	if err != nil {
		t.Fatal("failed to create test config")
	}
	var auxDir *proto.Directory
	for _, dir := range nginxConfig.DirectoryMap.Directories {
		for _, f := range dir.Files {
			f.Action = proto.File_unchanged
		}
		if filepath.Clean(dir.Name) == filepath.Join(tmpDir, "aux") {
			auxDir = dir
		}
	}
	if auxDir == nil {
		t.Fatalf("no aux dir found")
	}
	auxDir.Files = append(auxDir.Files, &proto.File{
		Name:   "test2.html",
		Action: proto.File_delete,
	})
	auxDir.Files = append(auxDir.Files, &proto.File{
		Name:   "test3.html",
		Action: proto.File_delete,
	})
	auxDir.Files = append(auxDir.Files, &proto.File{
		Name:   "test4.html",
		Action: proto.File_delete,
	})
	configApply, err := n.WriteConfig(nginxConfig)

	// Verify configApply
	assert.NoError(t, err)
	assert.Equal(t, expectedExisting, configApply.GetExisting())
	assert.Equal(t, expectedNotExisting, configApply.GetNotExists())

	err = configApply.Complete()
	assert.Nil(t, err)
	// Verify aux file test1.html was created
	_, err = os.Stat(tmpDir + "/aux/test1.html")
	assert.NoError(t, err)
	// Verify aux file test2.html was deleted
	_, err = os.Stat(tmpDir + "/aux/test2.html")
	assert.Error(t, err)
	_, err = os.Stat(tmpDir + "/aux/test3.html")
	assert.Error(t, err)

	// Verify that rollback on failure works as expected
	assert.NoError(t, configApply.Rollback(errors.New("config validation failed")))

	// Verify aux file test1.html was removed
	_, err = os.Stat(tmpDir + "/aux/test1.html")
	assert.NotNil(t, err)
	// Verify aux file test2.html was restored again
	_, err = os.Stat(tmpDir + "/aux/test2.html")
	assert.NoError(t, err)

	_, err = os.Stat(tmpDir + "/aux/test3.html")
	assert.NoError(t, err)
}

func TestWriteConfigWithFileActionDeleteWithPermError(t *testing.T) {
	tmpDir := t.TempDir()

	allowedDirs := make(map[string]struct{})
	allowedDirs[tmpDir] = struct{}{}
	fakeConfig := config.Config{
		AllowedDirectoriesMap: allowedDirs,
	}

	env := EnvironmentType{}
	n := NewNginxBinary(&env, &fakeConfig)

	n.nginxDetailsMap = map[string]*proto.NginxDetails{
		"151d8728e792f42e129337573a21bb30ab3065d59102f075efc2ded28e713ff8": {
			NginxId:     "151d8728e792f42e129337573a21bb30ab3065d59102f075efc2ded28e713ff8",
			ConfPath:    filepath.Join(tmpDir, "/nginx.conf"),
			ProcessId:   "777",
			ProcessPath: "/usr/sbin/nginx",
		},
	}

	if err := os.WriteFile(filepath.Join(tmpDir, "nginx.conf"),
		[]byte(fmt.Sprintf(CONF_TEMPLATE, tmpDir)), 0755); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}

	if err := os.Mkdir(filepath.Join(tmpDir, "aux"), 0755); err != nil {
		t.Fatalf("failed to create aux directory: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "/aux/test2.html"), []byte("<html><html>"), 0755); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}

	nginxConfig, err := buildConfig(tmpDir)
	if err != nil {
		t.Fatal("failed to create test config")
	}
	var auxDir *proto.Directory
	auxTmpDir := filepath.Join(tmpDir, "aux")
	for _, dir := range nginxConfig.DirectoryMap.Directories {
		for _, f := range dir.Files {
			f.Action = proto.File_unchanged
		}
		// set aux dir directory map
		if filepath.Clean(dir.Name) == auxTmpDir {
			auxDir = dir
		}
	}
	if auxDir == nil {
		t.Fatalf("no aux dir found")
	}
	auxDir.Files = append(auxDir.Files, &proto.File{
		Name:   "test2.html",
		Action: proto.File_delete,
	})

	modDir := &proto.Directory{
		Name:  tmpDir,
		Files: make([]*proto.File, 0),
	}
	modDir.Files = append(modDir.Files, &proto.File{
		Name:   "test3.html",
		Action: proto.File_delete,
	})
	nginxConfig.DirectoryMap.Directories = append(nginxConfig.DirectoryMap.Directories, modDir)

	permFile := filepath.Join(tmpDir, "test3.html")
	if err = os.WriteFile(permFile, []byte("<html><html>"), 0755); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}
	require.NoError(t, os.Chmod(permFile, 0000))

	ca, err := n.WriteConfig(nginxConfig)
	// Verify configApply
	assert.Error(t, err)
	assert.ErrorIs(t, err, os.ErrPermission)
	assert.NotNil(t, ca)
}

func TestGetDirectoryMapDiff(t *testing.T) {
	tests := []struct {
		name                 string
		currentDirectoryMap  []*proto.Directory
		incomingDirectoryMap []*proto.Directory
		expectedResult       []string
	}{
		{
			name:                 "2 Empty Directory Maps",
			currentDirectoryMap:  []*proto.Directory{},
			incomingDirectoryMap: []*proto.Directory{},
			expectedResult:       []string{},
		},
		{
			name:                "Empty Current Directory Map",
			currentDirectoryMap: []*proto.Directory{},
			incomingDirectoryMap: []*proto.Directory{
				{
					Name: "/dir1",
					Files: []*proto.File{
						{
							Name: "file1.html",
						},
					},
				},
			},
			expectedResult: []string{},
		},
		{
			name: "Empty Incoming Directory Map",
			currentDirectoryMap: []*proto.Directory{
				{
					Name: "/dir1",
					Files: []*proto.File{
						{
							Name: "file1.html",
						},
					},
				},
			},
			incomingDirectoryMap: []*proto.Directory{},
			expectedResult:       []string{"/dir1/file1.html"},
		},
		{
			name: "Same Directory Maps",
			currentDirectoryMap: []*proto.Directory{
				{
					Name: "/dir1",
					Files: []*proto.File{
						{
							Name: "file1.html",
						},
					},
				},
			},
			incomingDirectoryMap: []*proto.Directory{
				{
					Name: "/dir1",
					Files: []*proto.File{
						{
							Name: "file1.html",
						},
					},
				},
			},
			expectedResult: []string{},
		},
		{
			name: "Multiple directories and files with differences",
			currentDirectoryMap: []*proto.Directory{
				{
					Name: "/dir1",
					Files: []*proto.File{
						{
							Name: "file1.html",
						},
						{
							Name: "file2.html",
						},
						{
							Name: "file3.html",
						},
					},
				},
				{
					Name: "/dir2",
					Files: []*proto.File{
						{
							Name: "file1.html",
						},
						{
							Name: "file2.html",
						},
					},
				},
				{
					Name: "/dir3",
					Files: []*proto.File{
						{
							Name: "file1.html",
						},
						{
							Name: "file2.html",
						},
					},
				},
			},
			incomingDirectoryMap: []*proto.Directory{
				{
					Name: "/dir1",
					Files: []*proto.File{
						{
							Name: "file1.html",
						},
						{
							Name: "file3.html",
						},
					},
				},
				{
					Name: "/dir2",
					Files: []*proto.File{
						{
							Name: "file2.html",
						},
					},
				},
			},
			expectedResult: []string{
				"/dir1/file2.html",
				"/dir2/file1.html",
				"/dir3/file1.html",
				"/dir3/file2.html",
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(tt *testing.T) {
			actualResult := getDirectoryMapDiff(test.currentDirectoryMap, test.incomingDirectoryMap)
			assert.Equal(tt, test.expectedResult, actualResult)
		})
	}
}

func TestDeepCopyWithNewPath(t *testing.T) {
	tests := []struct {
		name    string
		files   []*proto.File
		oldPath string
		newPath string
	}{
		{
			name: "happy path",
			files: []*proto.File{
				{
					Name: "/tmp/file1.html",
				},
				{
					Name: "/tmp/file2.html",
				},
				{
					Name: "/tmp/file3.html",
				},
			},
			oldPath: "/tmp/",
			newPath: "/changed/",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(tt *testing.T) {
			actualResult := deepCopyWithNewPath(test.files, test.oldPath, test.newPath)

			for _, file := range actualResult {
				assert.True(t, strings.HasPrefix(file.Name, test.newPath))
			}
		})
	}
}

func buildConfig(rootDirectory string) (*proto.NginxConfig, error) {
	nginxConfig := &proto.NginxConfig{}
	defaultFileMode := fs.FileMode(0644)

	// Add config file
	configWriter, err := zip.NewWriter("testconfig")
	if err != nil {
		return nginxConfig, err
	}

	confString := fmt.Sprintf(CONF_TEMPLATE, rootDirectory)
	confBytes := []byte(confString)
	b := bytes.NewReader(confBytes)
	err = configWriter.Add("nginx.conf", defaultFileMode, b)
	if err != nil {
		return nginxConfig, err
	}

	nginxConfig.Action = proto.NginxConfigAction_APPLY
	nginxConfig.ConfigData = &proto.ConfigDescriptor{
		SystemId: "59633a13-f50b-3c46-89e5-d9bbb9080dcf",
		NginxId:  "151d8728e792f42e129337573a21bb30ab3065d59102f075efc2ded28e713ff8",
	}
	nginxConfig.Zconfig, _ = configWriter.Proto()

	// Add aux files
	auxWriter, err := zip.NewWriter("testaux")
	if err != nil {
		return nginxConfig, err
	}
	buf, err := base64.StdEncoding.DecodeString("")
	if err != nil {
		return nginxConfig, err
	}
	b = bytes.NewReader(buf)
	err = auxWriter.Add(rootDirectory+"/aux/test1.html", defaultFileMode, b)
	if err != nil {
		return nginxConfig, err
	}

	nginxConfig.Zaux, _ = auxWriter.Proto()

	// Add Directory Map
	nginxConfig.DirectoryMap = &proto.DirectoryMap{
		Directories: []*proto.Directory{
			{
				Name: rootDirectory,
				Files: []*proto.File{
					{
						Name: "nginx.conf",
					},
				},
			},
			{
				Name: rootDirectory + "/aux/",
				Files: []*proto.File{
					{
						Name: "test1.html",
					},
				},
			},
		},
	}

	return nginxConfig, nil
}

// TestNginxBinaryType_sanitizeProcessPath validate correct parsing of the nginx path when nginx binary has been updated.
func TestNginxBinaryType_sanitizeProcessPath(t *testing.T) {
	type testDef struct {
		desc      string
		path      string
		expect    string
		defaulted bool
	}

	// no test case for process lookup, that would require running nginx or proc somewhere
	for _, def := range []testDef{
		{desc: "deleted path", path: "/usr/sbin/nginx (deleted)", expect: "/usr/sbin/nginx"},
		{desc: "no change path", path: "/usr/sbin/nginx", expect: "/usr/sbin/nginx"},
	} {
		t.Run(def.desc, func(tt *testing.T) {
			p := Process{
				Path: def.path,
			}
			binary := NginxBinaryType{
				env: &EnvironmentType{},
			}
			assert.Equal(tt, def.defaulted, binary.sanitizeProcessPath(&p))
			assert.Equal(tt, def.expect, p.Path)
		})
	}
}

func TestNginxBinaryType_validateConfigCheckResponse(t *testing.T) {
	type testDef struct {
		name                  string
		response              string
		expected              interface{}
		treatWarningsAsErrors bool
	}

	// no test case for process lookup, that would require running nginx or proc somewhere
	for _, test := range []testDef{
		{name: "validation fails, emerg respected", response: "nginx [emerg]", treatWarningsAsErrors: false, expected: errors.New("error running nginx -t -c :\nnginx [emerg]")},
		{name: "validation fails, emerg respected, config irrelevant", response: "nginx [emerg]", treatWarningsAsErrors: true, expected: errors.New("error running nginx -t -c :\nnginx [emerg]")},
		{name: "validation fails, alert respected", response: "nginx [alert]", treatWarningsAsErrors: false, expected: errors.New("error running nginx -t -c :\nnginx [alert]")},
		{name: "validation fails, alert respected, config irrelevant", response: "nginx [alert]", treatWarningsAsErrors: true, expected: errors.New("error running nginx -t -c :\nnginx [alert]")},
		{name: "validation passes, warn ignored", response: "nginx [warn]", treatWarningsAsErrors: false, expected: nil},
		{name: "validation fails, warn respected", response: "nginx [warn]", treatWarningsAsErrors: true, expected: errors.New("error running nginx -t -c :\nnginx [warn]")},
		{name: "validation passes, info irrelevant", response: "nginx [info]", treatWarningsAsErrors: false, expected: nil},
		{name: "validation passes, info irrelevant, config irrelevant", response: "nginx [info]", treatWarningsAsErrors: true, expected: nil},
		{name: "validation fails unknown directive", response: "nginx: [emerg] unknown directive \"location/\" in /etc/nginx/sites-enabled/someapp:5", treatWarningsAsErrors: false, expected: errors.New("error running nginx -t -c :\nnginx: [emerg] unknown directive \"location/\" in /etc/nginx/sites-enabled/someapp:5")},
		{name: "validation fails conflicting server name", response: "nginx: [warn] conflicting server name \"example.com\" on 0.0.0.0:80, ignored", treatWarningsAsErrors: true, expected: errors.New("error running nginx -t -c :\nnginx: [warn] conflicting server name \"example.com\" on 0.0.0.0:80, ignored")},
		{name: "validation fails limit_req", response: "nginx: [emerg] 96300#96300: limit_req \"default\" uses the \"$binary_remote_addr\" key while previously it used the \"$http_apiKey\" key", treatWarningsAsErrors: true, expected: errors.New("error running nginx -t -c :\nnginx: [emerg] 96300#96300: limit_req \"default\" uses the \"$binary_remote_addr\" key while previously it used the \"$http_apiKey\" key")},
		{name: "validation fails host not found in upstream", response: "nginx: [emerg] 5191#5191: host not found in upstream \"example.com:80\" in /etc/nginx/nginx.conf:111", treatWarningsAsErrors: false, expected: errors.New("error running nginx -t -c :\nnginx: [emerg] 5191#5191: host not found in upstream \"example.com:80\" in /etc/nginx/nginx.conf:111")},
		{name: "validation fails worker_connections", response: "nginx: [warn] 2048 worker_connections exceed open file resource limit: 1024", treatWarningsAsErrors: true, expected: errors.New("error running nginx -t -c :\nnginx: [warn] 2048 worker_connections exceed open file resource limit: 1024")},
		{name: "validation passes worker_connections", response: "nginx: [warn] 2048 worker_connections exceed open file resource limit: 1024", treatWarningsAsErrors: false, expected: nil},
	} {
		t.Run(test.name, func(tt *testing.T) {
			binary := NginxBinaryType{
				env:    &EnvironmentType{},
				config: &config.Config{Nginx: config.Nginx{Debug: true, TreatWarningsAsErrors: test.treatWarningsAsErrors}},
			}
			buffer := bytes.NewBuffer([]byte(test.response))
			err := binary.validateConfigCheckResponse(buffer, "")
			assert.Equal(tt, test.expected, err)
		})
	}
}
