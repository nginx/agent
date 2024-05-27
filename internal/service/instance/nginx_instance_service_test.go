// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package instance

import (
	"bytes"
	"context"
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/nginx/agent/v3/internal/datasource/host"

	"github.com/nginx/agent/v3/test/helpers"

	"github.com/stretchr/testify/require"

	v1 "github.com/nginx/agent/v3/api/grpc/mpi/v1"
	"github.com/nginx/agent/v3/internal/datasource/host/exec/execfakes"
	"github.com/nginx/agent/v3/internal/model"
	"github.com/nginx/agent/v3/test/protos"
	"github.com/stretchr/testify/assert"
)

const (
	exePath       = "/usr/local/Cellar/nginx/1.25.3/bin/nginx"
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
	plusConfigArgs = "--prefix=/etc/nginx --sbin-path=/usr/sbin/nginx --modules-path=%s " +
		"--conf-path=/etc/nginx/nginx.conf --error-log-path=/var/log/nginx/error.log " +
		"--http-log-path=/var/log/nginx/access.log --pid-path=/var/run/nginx.pid " +
		"--lock-path=/var/run/nginx.lock --http-client-body-temp-path=/var/cache/nginx/client_temp " +
		"--http-proxy-temp-path=/var/cache/nginx/proxy_temp " +
		"--http-fastcgi-temp-path=/var/cache/nginx/fastcgi_temp " +
		"--http-uwsgi-temp-path=/var/cache/nginx/uwsgi_temp " +
		"--http-scgi-temp-path=/var/cache/nginx/scgi_temp --user=nginx --group=nginx --with-compat " +
		"--with-file-aio --with-threads --with-http_addition_module --with-http_auth_request_module " +
		"--with-http_dav_module --with-http_flv_module --with-http_gunzip_module " +
		"--with-http_gzip_static_module --with-http_mp4_module --with-http_random_index_module " +
		"--with-http_realip_module --with-http_secure_link_module --with-http_slice_module " +
		"--with-http_ssl_module --with-http_stub_status_module --with-http_sub_module " +
		"--with-http_v2_module --with-http_v3_module --with-mail --with-mail_ssl_module --with-stream " +
		"--with-stream_realip_module --with-stream_ssl_module --with-stream_ssl_preread_module " +
		"--build=nginx-plus-r31-p1 --with-http_auth_jwt_module --with-http_f4f_module " +
		"--with-http_hls_module --with-http_proxy_protocol_vendor_module " +
		"--with-http_session_log_module --with-stream_mqtt_filter_module " +
		"--with-stream_mqtt_preread_module --with-stream_proxy_protocol_vendor_module " +
		"--with-cc-opt='-g -O2 " +
		"-fdebug-prefix-map=" +
		"/data/builder/debuild/nginx-plus-1.25.3/debian/debuild-base/nginx-plus-1.25.3=. " +
		"-fstack-protector-strong -Wformat -Werror=format-security -Wp,-D_FORTIFY_SOURCE=2 -fPIC' " +
		"--with-ld-opt='-Wl,-Bsymbolic-functions -Wl,-z,relro -Wl,-z,now -Wl,--as-needed -pie'"
)

func TestGetInstances(t *testing.T) {
	ctx := context.Background()
	tempDir := t.TempDir()
	modulePath := tempDir + "/usr/lib/nginx/modules"
	noModulesPath := t.TempDir() + "/usr/lib/nginx/modules"

	helpers.CreateDirWithErrorCheck(t, modulePath)
	defer helpers.RemoveFileWithErrorCheck(t, modulePath)

	helpers.CreateDirWithErrorCheck(t, noModulesPath)
	defer helpers.RemoveFileWithErrorCheck(t, noModulesPath)

	testModule := helpers.CreateFileWithErrorCheck(t, modulePath, "test.so")
	defer helpers.RemoveFileWithErrorCheck(t, testModule.Name())

	plusArgs := fmt.Sprintf(plusConfigArgs, modulePath)
	ossArgs := fmt.Sprintf(ossConfigArgs, modulePath)
	noModuleArgs := fmt.Sprintf(ossConfigArgs, noModulesPath)

	expectedModules := strings.ReplaceAll(filepath.Base(testModule.Name()), ".so", "")
	processes := host.NginxProcesses{
		789: {
			PID:  789,
			PPID: 1234,
			Name: "nginx",
			Cmd:  "nginx: worker process",
			Exe:  exePath,
		},
		1234: {
			PID:  1234,
			PPID: 1,
			Name: "nginx",
			Cmd:  "nginx: master process /usr/local/opt/nginx/bin/nginx -g daemon off;",
			Exe:  exePath,
		},
	}

	process1 := protos.GetNginxOssInstance([]string{expectedModules})
	process1.GetInstanceRuntime().InstanceChildren = nil
	process2 := protos.GetNginxPlusInstance([]string{expectedModules})
	process2.GetInstanceRuntime().InstanceChildren = nil
	process3 := protos.GetNginxOssInstance(nil)
	process3.GetInstanceRuntime().InstanceChildren = nil

	tests := []struct {
		name                      string
		nginxVersionCommandOutput string
		expected                  []*v1.Instance
	}{
		{
			name: "Test 1: NGINX open source",
			nginxVersionCommandOutput: fmt.Sprintf(`nginx version: nginx/1.25.3
					built by clang 14.0.0 (clang-1400.0.29.202)
					built with OpenSSL 1.1.1s  1 Nov 2022 (running with OpenSSL 1.1.1t  7 Feb 2023)
					TLS SNI support enabled
					configure arguments: %s`, ossArgs),
			expected: []*v1.Instance{
				process1,
			},
		},
		{
			name: "Test 2: NGINX plus",
			nginxVersionCommandOutput: fmt.Sprintf(`
				nginx version: nginx/1.25.3 (nginx-plus-r31-p1)
				built by gcc 9.4.0 (Ubuntu 9.4.0-1ubuntu1~20.04.2)
				built with OpenSSL 1.1.1f  31 Mar 2020
				TLS SNI support enabled
				configure arguments: %s`, plusArgs),
			expected: []*v1.Instance{
				process2,
			},
		},
		{
			name: "Test 3: No Modules",
			nginxVersionCommandOutput: fmt.Sprintf(`nginx version: nginx/1.25.3
					built by clang 14.0.0 (clang-1400.0.29.202)
					built with OpenSSL 1.1.1s  1 Nov 2022 (running with OpenSSL 1.1.1t  7 Feb 2023)
					TLS SNI support enabled
					configure arguments: %s`, noModuleArgs),
			expected: []*v1.Instance{
				process3,
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(tt *testing.T) {
			mockExec := &execfakes.FakeExecInterface{}
			mockExec.RunCmdReturns(bytes.NewBufferString(test.nginxVersionCommandOutput), nil)

			n := NewNginx(NginxParameters{executer: mockExec})
			result := n.GetInstances(ctx, processes)

			for _, instance := range result {
				if instance.GetInstanceRuntime().GetNginxRuntimeInfo() != nil {
					sort.Strings(instance.GetInstanceRuntime().GetNginxRuntimeInfo().GetDynamicModules())
				} else {
					sort.Strings(instance.GetInstanceRuntime().GetNginxPlusRuntimeInfo().GetDynamicModules())
				}
			}

			assert.Equal(tt, test.expected, result)
		})
	}
}

func TestGetInfo(t *testing.T) {
	ctx := context.Background()
	tempDir := t.TempDir()
	modulePath := tempDir + "/usr/lib/nginx/modules"

	helpers.CreateDirWithErrorCheck(t, modulePath)
	defer helpers.RemoveFileWithErrorCheck(t, modulePath)

	testModule := helpers.CreateFileWithErrorCheck(t, modulePath, "test.so")
	defer helpers.RemoveFileWithErrorCheck(t, testModule.Name())

	plusArgs := fmt.Sprintf(plusConfigArgs, modulePath)
	ossArgs := fmt.Sprintf(ossConfigArgs, modulePath)

	expectedModules := strings.ReplaceAll(filepath.Base(testModule.Name()), ".so", "")

	tests := []struct {
		name                      string
		nginxVersionCommandOutput string
		process                   *model.Process
		expected                  *Info
	}{
		{
			name: "Test 1: NGINX open source",
			nginxVersionCommandOutput: fmt.Sprintf(`
				nginx version: nginx/1.25.3
				built by clang 14.0.3 (clang-1403.0.22.14.1)
				built with OpenSSL 3.1.3 19 Sep 2023 (running with OpenSSL 3.2.0 23 Nov 2023)
				TLS SNI support enabled
				configure arguments: %s`, ossArgs),
			process: &model.Process{
				Exe: exePath,
			},
			expected: &Info{
				Version:  "1.25.3",
				Prefix:   "/usr/local/Cellar/nginx/1.25.3",
				ConfPath: "/usr/local/etc/nginx/nginx.conf",
				ExePath:  exePath,
				ConfigureArgs: map[string]interface{}{
					"conf-path":                  "/usr/local/etc/nginx/nginx.conf",
					"error-log-path":             "/usr/local/var/log/nginx/error.log",
					"http-client-body-temp-path": "/usr/local/var/run/nginx/client_body_temp",
					"http-fastcgi-temp-path":     "/usr/local/var/run/nginx/fastcgi_temp",
					"http-log-path":              "/usr/local/var/log/nginx/access.log",
					"http-proxy-temp-path":       "/usr/local/var/run/nginx/proxy_temp",
					"http-scgi-temp-path":        "/usr/local/var/run/nginx/scgi_temp",
					"http-uwsgi-temp-path":       "/usr/local/var/run/nginx/uwsgi_temp",
					"lock-path":                  "/usr/local/var/run/nginx.lock",
					"modules-path":               modulePath,
					"pid-path":                   "/usr/local/var/run/nginx.pid",
					"prefix":                     "/usr/local/Cellar/nginx/1.25.3",
					"sbin-path":                  exePath,
					"with-cc-opt": "'-I/usr/local/opt/pcre2/include " +
						"-I/usr/local/opt/openssl@1.1/include'",
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
					"with-ld-opt":                    "'-L/usr/local/opt/pcre2/lib -L/usr/local/opt/openssl@1.1/lib'",
					"with-mail":                      true,
					"with-mail_ssl_module":           true,
					"with-pcre":                      true,
					"with-pcre-jit":                  true,
					"with-stream":                    true,
					"with-stream_realip_module":      true,
					"with-stream_ssl_module":         true,
					"with-stream_ssl_preread_module": true,
				},
				LoadableModules: []string{expectedModules},
				DynamicModules: protos.GetNginxOssInstance([]string{}).GetInstanceRuntime().GetNginxRuntimeInfo().
					GetDynamicModules(),
			},
		},
		{
			name: "Test 2: NGINX plus",
			nginxVersionCommandOutput: fmt.Sprintf(`
				nginx version: nginx/1.25.3 (nginx-plus-r31-p1)
				built by gcc 9.4.0 (Ubuntu 9.4.0-1ubuntu1~20.04.2)
				built with OpenSSL 1.1.1f  31 Mar 2020
				TLS SNI support enabled
				configure arguments: %s`, plusArgs),
			process: &model.Process{
				Exe: exePath,
			},
			expected: &Info{
				Version:  "1.25.3 (nginx-plus-r31-p1)",
				Prefix:   "/etc/nginx",
				ConfPath: "/etc/nginx/nginx.conf",
				ExePath:  exePath,
				ConfigureArgs: map[string]interface{}{
					"build":                                  "nginx-plus-r31-p1",
					"conf-path":                              "/etc/nginx/nginx.conf",
					"error-log-path":                         "/var/log/nginx/error.log",
					"group":                                  "nginx",
					"http-client-body-temp-path":             "/var/cache/nginx/client_temp",
					"http-fastcgi-temp-path":                 "/var/cache/nginx/fastcgi_temp",
					"http-log-path":                          "/var/log/nginx/access.log",
					"http-proxy-temp-path":                   "/var/cache/nginx/proxy_temp",
					"http-scgi-temp-path":                    "/var/cache/nginx/scgi_temp",
					"http-uwsgi-temp-path":                   "/var/cache/nginx/uwsgi_temp",
					"lock-path":                              "/var/run/nginx.lock",
					"modules-path":                           modulePath,
					"pid-path":                               "/var/run/nginx.pid",
					"prefix":                                 "/etc/nginx",
					"sbin-path":                              "/usr/sbin/nginx",
					"user":                                   "nginx",
					"with-compat":                            true,
					"with-file-aio":                          true,
					"with-http_addition_module":              true,
					"with-http_auth_jwt_module":              true,
					"with-http_auth_request_module":          true,
					"with-http_dav_module":                   true,
					"with-http_f4f_module":                   true,
					"with-http_flv_module":                   true,
					"with-http_gunzip_module":                true,
					"with-http_gzip_static_module":           true,
					"with-http_hls_module":                   true,
					"with-http_mp4_module":                   true,
					"with-http_proxy_protocol_vendor_module": true,
					"with-http_random_index_module":          true,
					"with-http_realip_module":                true,
					"with-http_secure_link_module":           true,
					"with-http_session_log_module":           true,
					"with-http_slice_module":                 true,
					"with-http_ssl_module":                   true,
					"with-http_stub_status_module":           true,
					"with-http_sub_module":                   true,
					"with-http_v2_module":                    true,
					"with-http_v3_module":                    true,
					"with-ld-opt": "'-Wl,-Bsymbolic-functions -Wl,-z,relro " +
						"-Wl,-z,now -Wl,--as-needed -pie'",
					"with-mail":                                true,
					"with-mail_ssl_module":                     true,
					"with-stream":                              true,
					"with-stream_mqtt_filter_module":           true,
					"with-stream_mqtt_preread_module":          true,
					"with-stream_proxy_protocol_vendor_module": true,
					"with-stream_realip_module":                true,
					"with-stream_ssl_module":                   true,
					"with-stream_ssl_preread_module":           true,
					"with-threads":                             true,
				},
				LoadableModules: []string{expectedModules},
				DynamicModules: protos.GetNginxPlusInstance([]string{}).GetInstanceRuntime().GetNginxPlusRuntimeInfo().
					GetDynamicModules(),
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(tt *testing.T) {
			mockExec := &execfakes.FakeExecInterface{}
			mockExec.RunCmdReturns(bytes.NewBufferString(test.nginxVersionCommandOutput), nil)

			n := NewNginx(NginxParameters{executer: mockExec})
			result, err := n.getInfo(ctx, test.process)
			sort.Strings(result.DynamicModules)

			assert.Equal(tt, test.expected, result)
			require.NoError(tt, err)
		})
	}
}
