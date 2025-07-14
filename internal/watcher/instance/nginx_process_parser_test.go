// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package instance

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"google.golang.org/protobuf/proto"

	mpi "github.com/nginx/agent/v3/api/grpc/mpi/v1"
	"github.com/nginx/agent/v3/internal/datasource/host/exec/execfakes"
	"github.com/nginx/agent/v3/pkg/nginxprocess"
	"github.com/nginx/agent/v3/test/helpers"
	"github.com/nginx/agent/v3/test/protos"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

func TestNginxProcessParser_Parse(t *testing.T) {
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
	processes := []*nginxprocess.Process{
		{
			PID:  789,
			PPID: 1234,
			Name: "nginx",
			Cmd:  "nginx: worker process",
			Exe:  exePath,
		},
		{
			PID:  567,
			PPID: 1234,
			Name: "nginx",
			Cmd:  "nginx: worker process",
			Exe:  exePath,
		},
		{
			PID:  1234,
			PPID: 1,
			Name: "nginx",
			Cmd:  "nginx: master process /usr/local/opt/nginx/bin/nginx -g daemon off;",
			Exe:  exePath,
		},
	}

	tests := []struct {
		expected                  map[string]*mpi.Instance
		name                      string
		nginxVersionCommandOutput string
	}{
		{
			name: "Test 1: NGINX open source",
			nginxVersionCommandOutput: `nginx version: nginx/1.25.3
					built by clang 14.0.0 (clang-1400.0.29.202)
					built with OpenSSL 1.1.1s  1 Nov 2022 (running with OpenSSL 1.1.1t  7 Feb 2023)
					TLS SNI support enabled
					configure arguments: ` + ossArgs,
			expected: map[string]*mpi.Instance{
				protos.NginxOssInstance([]string{}).GetInstanceMeta().GetInstanceId(): protos.NginxOssInstance(
					[]string{expectedModules}),
			},
		},
		{
			name: "Test 2: NGINX plus",
			nginxVersionCommandOutput: `
				nginx version: nginx/1.25.3 (nginx-plus-r31-p1)
				built by gcc 9.4.0 (Ubuntu 9.4.0-1ubuntu1~20.04.2)
				built with OpenSSL 1.1.1f  31 Mar 2020
				TLS SNI support enabled
				configure arguments: ` + plusArgs,
			expected: map[string]*mpi.Instance{
				protos.NginxPlusInstance([]string{}).GetInstanceMeta().GetInstanceId(): protos.NginxPlusInstance(
					[]string{expectedModules}),
			},
		},
		{
			name: "Test 3: No Modules",
			nginxVersionCommandOutput: `nginx version: nginx/1.25.3
					built by clang 14.0.0 (clang-1400.0.29.202)
					built with OpenSSL 1.1.1s  1 Nov 2022 (running with OpenSSL 1.1.1t  7 Feb 2023)
					TLS SNI support enabled
					configure arguments: ` + noModuleArgs,
			expected: map[string]*mpi.Instance{
				protos.NginxOssInstance([]string{}).GetInstanceMeta().GetInstanceId(): protos.
					NginxOssInstance(nil),
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(tt *testing.T) {
			mockExec := &execfakes.FakeExecInterface{}
			mockExec.RunCmdReturnsOnCall(0, bytes.NewBufferString(test.nginxVersionCommandOutput), nil)

			n := NewNginxProcessParser()
			n.executer = mockExec
			result := n.Parse(ctx, processes)

			for id, instance := range result {
				resultRun := instance.GetInstanceRuntime()
				expectedRun := test.expected[id].GetInstanceRuntime()
				expectedRun.InstanceChildren = protos.SortInstanceChildren(expectedRun.GetInstanceChildren())
				resultRun.InstanceChildren = protos.SortInstanceChildren(resultRun.GetInstanceChildren())

				if resultRun.GetNginxRuntimeInfo() != nil {
					sort.Strings(resultRun.GetNginxRuntimeInfo().GetDynamicModules())
					assert.True(tt, proto.Equal(test.expected[id], instance))
				} else {
					sort.Strings(resultRun.GetNginxPlusRuntimeInfo().GetDynamicModules())
					assert.True(tt, proto.Equal(test.expected[id], instance))
				}
			}

			assert.Len(tt, result, len(test.expected))
		})
	}
}

func TestNginxProcessParser_Parse_Processes(t *testing.T) {
	ctx := context.Background()
	modulePath := t.TempDir() + "/usr/lib/nginx/modules"

	configArgs := fmt.Sprintf(ossConfigArgs, modulePath)

	nginxVersionCommandOutput := `nginx version: nginx/1.25.3
					built by clang 14.0.0 (clang-1400.0.29.202)
					built with OpenSSL 1.1.1s  1 Nov 2022 (running with OpenSSL 1.1.1t  7 Feb 2023)
					TLS SNI support enabled
					configure arguments: ` + configArgs

	process1 := protos.NginxOssInstance(nil)
	instancesTest1 := map[string]*mpi.Instance{
		process1.GetInstanceMeta().GetInstanceId(): process1,
	}

	noChildrenInstance := protos.NginxOssInstance(nil)
	noChildrenInstance.GetInstanceRuntime().InstanceChildren = nil
	instancesTest2 := map[string]*mpi.Instance{
		noChildrenInstance.GetInstanceMeta().GetInstanceId(): noChildrenInstance,
	}

	noParentInstanceList := protos.InstancesNoParentProcess(nil)
	instancesTest3 := map[string]*mpi.Instance{
		noParentInstanceList[0].GetInstanceMeta().GetInstanceId(): noParentInstanceList[0],
		noParentInstanceList[1].GetInstanceMeta().GetInstanceId(): noParentInstanceList[1],
	}

	instancesList := protos.MultipleInstances(nil)
	instancesTest4 := map[string]*mpi.Instance{
		instancesList[0].GetInstanceMeta().GetInstanceId(): instancesList[0],
		instancesList[1].GetInstanceMeta().GetInstanceId(): instancesList[1],
	}

	tests := []struct {
		expected  map[string]*mpi.Instance
		name      string
		processes []*nginxprocess.Process
	}{
		{
			name: "Test 1: 1 master process, 2 workers",
			processes: []*nginxprocess.Process{
				{
					PID:  567,
					PPID: 1234,
					Name: "nginx",
					Cmd:  "nginx: worker process",
					Exe:  exePath,
				},
				{
					PID:  789,
					PPID: 1234,
					Name: "nginx",
					Cmd:  "nginx: worker process",
					Exe:  exePath,
				},
				{
					PID:  1234,
					PPID: 1,
					Name: "nginx",
					Cmd:  "nginx: master process /usr/local/opt/nginx/bin/nginx -g daemon off;",
					Exe:  exePath,
				},
			},
			expected: instancesTest1,
		},
		{
			name: "Test 2: 1 master process, no workers",
			processes: []*nginxprocess.Process{
				{
					PID:  1234,
					PPID: 1,
					Name: "nginx",
					Cmd:  "nginx: master process /usr/local/opt/nginx/bin/nginx -g daemon off;",
					Exe:  exePath,
				},
			},
			expected: instancesTest2,
		},
		{
			name: "Test 3: no master process, 2 workers for each killed master",
			processes: []*nginxprocess.Process{
				{
					PID:  789,
					PPID: 1234,
					Name: "nginx",
					Cmd:  "nginx: worker process",
					Exe:  exePath,
				},
				{
					PID:  567,
					PPID: 1234,
					Name: "nginx",
					Cmd:  "nginx: worker process",
					Exe:  exePath,
				},
				{
					PID:  987,
					PPID: 4321,
					Name: "nginx",
					Cmd:  "nginx: worker process",
					Exe:  "/opt/homebrew/etc/nginx/1.25.3/bin/nginx",
				},
				{
					PID:  321,
					PPID: 4321,
					Name: "nginx",
					Cmd:  "nginx: worker process",
					Exe:  "/opt/homebrew/etc/nginx/1.25.3/bin/nginx",
				},
			},
			expected: instancesTest3,
		},
		{
			name: "Test 4: 2 master process each with 2 workers",
			processes: []*nginxprocess.Process{
				{
					PID:  789,
					PPID: 1234,
					Name: "nginx",
					Cmd:  "nginx: worker process",
					Exe:  exePath,
				},
				{
					PID:  567,
					PPID: 1234,
					Name: "nginx",
					Cmd:  "nginx: worker process",
					Exe:  exePath,
				},
				{
					PID:  1234,
					PPID: 1,
					Name: "nginx",
					Cmd:  "nginx: master process /usr/local/opt/nginx/bin/nginx -g daemon off;",
					Exe:  exePath,
				},
				{
					PID:  987,
					PPID: 5678,
					Name: "nginx",
					Cmd:  "nginx: worker process",
					Exe:  "/opt/homebrew/etc/nginx/1.25.3/bin/nginx",
				},
				{
					PID:  321,
					PPID: 5678,
					Name: "nginx",
					Cmd:  "nginx: worker process",
					Exe:  "/opt/homebrew/etc/nginx/1.25.3/bin/nginx",
				},
				{
					PID:  5678,
					PPID: 1,
					Name: "nginx",
					Cmd:  "nginx: master process /usr/local/opt/nginx/bin/nginx -g daemon off;",
					Exe:  "/opt/homebrew/etc/nginx/1.25.3/bin/nginx",
				},
			},
			expected: instancesTest4,
		},
		{
			name: "Test 5: 1 cache process",
			processes: []*nginxprocess.Process{
				{
					PID:  1234,
					PPID: 1,
					Name: "nginx",
					Cmd:  "nginx: cache manager process",
					Exe:  exePath,
				},
			},
			expected: make(map[string]*mpi.Instance),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(tt *testing.T) {
			mockExec := &execfakes.FakeExecInterface{}
			mockExec.RunCmdReturnsOnCall(0, bytes.NewBufferString(nginxVersionCommandOutput), nil)
			mockExec.RunCmdReturnsOnCall(1, bytes.NewBufferString(nginxVersionCommandOutput), nil)
			mockExec.RunCmdReturnsOnCall(2, bytes.NewBufferString(nginxVersionCommandOutput), nil)
			mockExec.RunCmdReturnsOnCall(3, bytes.NewBufferString(nginxVersionCommandOutput), nil)

			n := NewNginxProcessParser()
			n.executer = mockExec
			result := n.Parse(ctx, test.processes)

			for id, instance := range result {
				resultRun := instance.GetInstanceRuntime()
				expectedRun := test.expected[id].GetInstanceRuntime()

				sort.Strings(resultRun.GetNginxRuntimeInfo().GetDynamicModules())

				expectedRun.InstanceChildren = protos.SortInstanceChildren(expectedRun.GetInstanceChildren())
				resultRun.InstanceChildren = protos.SortInstanceChildren(resultRun.GetInstanceChildren())

				assert.True(tt, proto.Equal(test.expected[id], instance))
			}

			assert.Len(tt, result, len(test.expected))
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
		process                   *nginxprocess.Process
		expected                  *Info
		name                      string
		nginxVersionCommandOutput string
	}{
		{
			name: "Test 1: NGINX open source",
			nginxVersionCommandOutput: `
				nginx version: nginx/1.25.3
				built by clang 14.0.3 (clang-1403.0.22.14.1)
				built with OpenSSL 3.1.3 19 Sep 2023 (running with OpenSSL 3.2.0 23 Nov 2023)
				TLS SNI support enabled
				configure arguments: ` + ossArgs,
			process: &nginxprocess.Process{
				PID: 1123,
				Exe: exePath,
			},
			expected: &Info{
				ProcessID: 1123,
				Version:   "1.25.3",
				Prefix:    "/usr/local/Cellar/nginx/1.25.3",
				ConfPath:  "/usr/local/etc/nginx/nginx.conf",
				ExePath:   exePath,
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
				DynamicModules: protos.NginxOssInstance([]string{}).GetInstanceRuntime().GetNginxRuntimeInfo().
					GetDynamicModules(),
			},
		},
		{
			name: "Test 2: NGINX plus",
			nginxVersionCommandOutput: `
				nginx version: nginx/1.25.3 (nginx-plus-r31-p1)
				built by gcc 9.4.0 (Ubuntu 9.4.0-1ubuntu1~20.04.2)
				built with OpenSSL 1.1.1f  31 Mar 2020
				TLS SNI support enabled
				configure arguments: ` + plusArgs,
			process: &nginxprocess.Process{
				PID: 3141,
				Exe: exePath,
			},
			expected: &Info{
				ProcessID: 3141,
				Version:   "1.25.3 (nginx-plus-r31-p1)",
				Prefix:    "/etc/nginx",
				ConfPath:  "/etc/nginx/nginx.conf",
				ExePath:   exePath,
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
				DynamicModules: protos.NginxPlusInstance([]string{}).GetInstanceRuntime().GetNginxPlusRuntimeInfo().
					GetDynamicModules(),
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(tt *testing.T) {
			mockExec := &execfakes.FakeExecInterface{}
			mockExec.RunCmdReturns(bytes.NewBufferString(test.nginxVersionCommandOutput), nil)

			nginxProcessParser := NewNginxProcessParser()
			nginxProcessParser.executer = mockExec
			result, err := nginxProcessParser.info(ctx, test.process)
			sort.Strings(result.DynamicModules)

			assert.Equal(tt, test.expected, result)
			require.NoError(tt, err)
		})
	}
}

func TestNginxProcessParser_GetExe(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		commandError  error
		name          string
		expected      string
		commandOutput []byte
	}{
		{
			name:          "Test 1: Default exe if error executing command -v nginx",
			commandOutput: []byte{},
			commandError:  errors.New("command error"),
			expected:      "/usr/bin/nginx",
		},
		{
			name:          "Test 2: Sanitize Exe Deleted Path",
			commandOutput: []byte("/usr/sbin/nginx (deleted)"),
			commandError:  nil,
			expected:      "/usr/sbin/nginx",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(tt *testing.T) {
			mockExec := &execfakes.FakeExecInterface{}
			mockExec.RunCmdReturns(bytes.NewBuffer(test.commandOutput), test.commandError)
			mockExec.FindExecutableReturns("/usr/bin/nginx", nil)

			nginxProcessParser := NewNginxProcessParser()
			nginxProcessParser.executer = mockExec
			result := nginxProcessParser.exe(ctx)

			assert.Equal(tt, test.expected, result)
		})
	}
}

func TestGetConfigPathFromCommand(t *testing.T) {
	result := confPathFromCommand("nginx: master process nginx -c /tmp/nginx.conf")
	assert.Equal(t, "/tmp/nginx.conf", result)

	result = confPathFromCommand("nginx: master process nginx -c")
	assert.Empty(t, result)

	result = confPathFromCommand("")
	assert.Empty(t, result)
}
