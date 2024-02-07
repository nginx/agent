// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package instance

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/nginx/agent/v3/api/grpc/instances"
	"github.com/nginx/agent/v3/internal/datasource/host/exec/execfakes"
	"github.com/nginx/agent/v3/internal/model"
	"github.com/stretchr/testify/assert"
)

const (
	exePath       = "/usr/local/Cellar/nginx/1.23.3/bin/nginx"
	ossConfigArgs = "--prefix=/usr/local/Cellar/nginx/1.23.3 --sbin-path=/usr/local/Cellar/nginx/1.23.3/bin/nginx " +
		"--with-cc-opt='-I/usr/local/opt/pcre2/include -I/usr/local/opt/openssl@1.1/include' " +
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
	plusConfigArgs = "--prefix=/etc/nginx --sbin-path=/usr/sbin/nginx --modules-path=/usr/lib/nginx/modules " +
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
		"--build=nginx-plus-r30-p1 --with-http_auth_jwt_module --with-http_f4f_module " +
		"--with-http_hls_module --with-http_proxy_protocol_vendor_module " +
		"--with-http_session_log_module --with-stream_mqtt_filter_module " +
		"--with-stream_mqtt_preread_module --with-stream_proxy_protocol_vendor_module " +
		"--with-cc-opt='-g -O2 " +
		"-fdebug-prefix-map=" +
		"/data/builder/debuild/nginx-plus-1.25.1/debian/debuild-base/nginx-plus-1.25.1=. " +
		"-fstack-protector-strong -Wformat -Werror=format-security -Wp,-D_FORTIFY_SOURCE=2 -fPIC' " +
		"--with-ld-opt='-Wl,-Bsymbolic-functions -Wl,-z,relro -Wl,-z,now -Wl,--as-needed -pie'"
)

func TestGetInstances(t *testing.T) {
	processes := []*model.Process{
		{
			Pid:  123,
			Ppid: 456,
			Name: "nginx",
			Cmd:  "nginx: worker process",
			Exe:  exePath,
		},
		{
			Pid:  789,
			Ppid: 123,
			Name: "nginx",
			Cmd:  "nginx: master process /usr/local/opt/nginx/bin/nginx -g daemon off;",
			Exe:  exePath,
		},
		{
			Pid:  543,
			Ppid: 454,
			Name: "grep",
			Cmd: "grep --color=auto --exclude-dir=.bzr --exclude-dir=CVS --exclude-dir=.git " +
				"--exclude-dir=.hg --exclude-dir=.svn --exclude-dir=.idea --exclude-dir=.tox nginx",
			Exe: exePath,
		},
	}

	tests := []struct {
		name                      string
		nginxVersionCommandOutput string
		expected                  []*instances.Instance
	}{
		{
			name: "NGINX open source",
			nginxVersionCommandOutput: fmt.Sprintf(`nginx version: nginx/1.23.3
					built by clang 14.0.0 (clang-1400.0.29.202)
					built with OpenSSL 1.1.1s  1 Nov 2022 (running with OpenSSL 1.1.1t  7 Feb 2023)
					TLS SNI support enabled
					configure arguments: %s`, ossConfigArgs),
			expected: []*instances.Instance{
				{
					InstanceId: "5975af28-e028-3f87-a848-a39e50cdf0e9",
					Type:       instances.Type_NGINX,
					Version:    "1.23.3",
					Meta: &instances.Meta{
						Meta: &instances.Meta_NginxMeta{
							NginxMeta: &instances.NginxMeta{
								ConfigPath: "/usr/local/etc/nginx/nginx.conf",
								ExePath:    exePath,
							},
						},
					},
				},
			},
		}, {
			name: "NGINX plus",
			nginxVersionCommandOutput: fmt.Sprintf(`
				nginx version: nginx/1.25.1 (nginx-plus-r30-p1)
				built by gcc 9.4.0 (Ubuntu 9.4.0-1ubuntu1~20.04.2)
				built with OpenSSL 1.1.1f  31 Mar 2020
				TLS SNI support enabled
				configure arguments: %s`, plusConfigArgs),
			expected: []*instances.Instance{
				{
					InstanceId: "2dbc5cb9-464f-30da-b703-5e63c62bf31e",
					Type:       instances.Type_NGINX_PLUS,
					Version:    "nginx-plus-r30-p1",
					Meta: &instances.Meta{
						Meta: &instances.Meta_NginxMeta{
							NginxMeta: &instances.NginxMeta{
								ConfigPath: "/etc/nginx/nginx.conf",
								ExePath:    exePath,
							},
						},
					},
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(tt *testing.T) {
			mockExec := &execfakes.FakeExecInterface{}
			mockExec.RunCmdReturns(bytes.NewBufferString(test.nginxVersionCommandOutput), nil)

			n := NewNginx(NginxParameters{executer: mockExec})
			result := n.GetInstances(processes)

			assert.Equal(tt, test.expected, result)
		})
	}
}

func TestGetInfo(t *testing.T) {
	tests := []struct {
		name                      string
		nginxVersionCommandOutput string
		process                   *model.Process
		expected                  *Info
	}{
		{
			name: "NGINX open source",
			nginxVersionCommandOutput: fmt.Sprintf(`
				nginx version: nginx/1.23.3
				built by clang 14.0.0 (clang-1400.0.29.202)
				built with OpenSSL 1.1.1s  1 Nov 2022 (running with OpenSSL 1.1.1t  7 Feb 2023)
				TLS SNI support enabled
				configure arguments: %s`, ossConfigArgs),
			process: &model.Process{
				Exe: exePath,
			},
			expected: &Info{
				Version:     "1.23.3",
				PlusVersion: "",
				Prefix:      "/usr/local/Cellar/nginx/1.23.3",
				ConfPath:    "/usr/local/etc/nginx/nginx.conf",
				ExePath:     exePath,
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
					"pid-path":                   "/usr/local/var/run/nginx.pid",
					"prefix":                     "/usr/local/Cellar/nginx/1.23.3",
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
			},
		},
		{
			name: "NGINX plus",
			nginxVersionCommandOutput: fmt.Sprintf(`
				nginx version: nginx/1.25.1 (nginx-plus-r30-p1)
				built by gcc 9.4.0 (Ubuntu 9.4.0-1ubuntu1~20.04.2)
				built with OpenSSL 1.1.1f  31 Mar 2020
				TLS SNI support enabled
				configure arguments: %s`, plusConfigArgs),
			process: &model.Process{
				Exe: exePath,
			},
			expected: &Info{
				Version:     "1.25.1",
				PlusVersion: "nginx-plus-r30-p1",
				Prefix:      "/etc/nginx",
				ConfPath:    "/etc/nginx/nginx.conf",
				ExePath:     exePath,
				ConfigureArgs: map[string]interface{}{
					"build":                                  "nginx-plus-r30-p1",
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
					"modules-path":                           "/usr/lib/nginx/modules",
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
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(tt *testing.T) {
			mockExec := &execfakes.FakeExecInterface{}
			mockExec.RunCmdReturns(bytes.NewBufferString(test.nginxVersionCommandOutput), nil)

			n := NewNginx(NginxParameters{executer: mockExec})
			result, err := n.getInfo(test.process)

			assert.Equal(tt, test.expected, result)
			require.NoError(tt, err)
		})
	}
}
