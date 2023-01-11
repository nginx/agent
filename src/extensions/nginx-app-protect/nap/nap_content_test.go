/**
 * Copyright (c) F5, Inc.
 *
 * This source code is licensed under the Apache License, Version 2.0 license found in the
 * LICENSE file in the root directory of this source tree.
 */

package nap

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/nginx/agent/sdk/v2"

	"github.com/stretchr/testify/assert"
)

const (
	nginxID  = "1"
	systemID = "2"
)

var config0 = `daemon            off;
	worker_processes  2;
	user              www-data;
	
	events {
		use           epoll;
		worker_connections  128;
	}
	
	error_log         /tmp/testdata/logs/error.log info;
				
	http {
		log_format upstream_time '$remote_addr - $remote_user [$time_local] '
		'"$request" $status $body_bytes_sent '
		'"$http_referer" "$http_user_agent" '
		'rt=$request_time uct="$upstream_connect_time" uht="$upstream_header_time" urt="$upstream_response_time"';
	
		server_tokens off;
		charset       utf-8;
		
		access_log    /tmp/testdata/logs/access1.log  $upstream_time;
	
		server {
			server_name   localhost;
			listen        127.0.0.1:80;
		
			error_page    500 502 503 504  /50x.html;
			# ssl_certificate /usr/local/nginx/conf/cert.pem;
	
			location      / {
				root      /tmp/testdata/root;
			}

			location /privateapi {
				limit_except GET {
					auth_basic "NGINX Plus API";
					auth_basic_user_file /path/to/passwd/file;
				}
				api write=on;
				allow 127.0.0.1;
				deny  all;
			}	
		}
	
		access_log    /tmp/testdata/logs/access2.log  combined;
	
	}`

var config1 = `daemon            off;
	worker_processes  2;
	user              www-data;
	
	events {
		use           epoll;
		worker_connections  128;
	}
	
	error_log         /tmp/testdata/logs/error.log info;
				
	http {
		app_protect_enable on;
		app_protect_security_log_enable on;

		log_format upstream_time '$remote_addr - $remote_user [$time_local] '
		'"$request" $status $body_bytes_sent '
		'"$http_referer" "$http_user_agent" '
		'rt=$request_time uct="$upstream_connect_time" uht="$upstream_header_time" urt="$upstream_response_time"';
	
		server_tokens off;
		charset       utf-8;
		
		access_log    /tmp/testdata/logs/access1.log  $upstream_time;
		app_protect_policy_file /tmp/testdata/root/my-nap-policy1.json;
		app_protect_security_log "/tmp/testdata/root/log-all.json" /var/log/ssecurity.log;
	
		server {
			server_name   localhost;
			listen        127.0.0.1:80;
			app_protect_policy_file /tmp/testdata/root/my-nap-policy2.json;
			app_protect_security_log "/tmp/testdata/root/log-blocked.json" /var/log/ssecurity.log;
		
			error_page    500 502 503 504  /50x.html;
			# ssl_certificate /usr/local/nginx/conf/cert.pem;
	
			location / {
				root      /tmp/testdata/root;
				app_protect_policy_file /tmp/testdata/root/my-nap-policy3.json;
				app_protect_security_log "/tmp/testdata/root/log-default.json" /var/log/security.log;
			}

			location /home {
				app_protect_policy_file /tmp/testdata/root/my-nap-policy4.json;
				app_protect_security_log "/tmp/testdata/root/log-illegal.json" /var/log/security.log;
			}

			location /privateapi {
				app_protect_policy_file /tmp/testdata/root/my-nap-policy4.json;
				app_protect_security_log "/tmp/testdata/root/log-illegal.json" /var/log/security.log;
				limit_except GET {
					auth_basic "NGINX Plus API";
					auth_basic_user_file /path/to/passwd/file;
				}
				api write=on;
				allow 127.0.0.1;
				deny  all;
			}	
		}
	
		access_log    /tmp/testdata/logs/access2.log  combined;
	
	}`

func TestNAPContent(t *testing.T) {
	testCases := []struct {
		testName    string
		file        string
		config      string
		expPolicies []string
		expProfiles []string
	}{
		{
			testName:    "NoNAPContent",
			file:        "/tmp/testdata/nginx/nginx.conf",
			config:      config0,
			expPolicies: []string{},
			expProfiles: []string{},
		},
		{
			testName:    "ConfigWithNAPContent",
			file:        "/tmp/testdata/nginx/nginx2.conf",
			config:      config1,
			expPolicies: []string{"my-nap-policy2.json", "my-nap-policy1.json", "my-nap-policy3.json", "my-nap-policy4.json"},
			expProfiles: []string{"log-all.json", "log-blocked.json", "log-default.json", "log-illegal.json"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.testName, func(t *testing.T) {
			defer tearDownDirectories()

			err := setUpFile(tc.file, []byte(tc.config))
			assert.NoError(t, err)

			allowedDirs := map[string]struct{}{}

			cfg, err := sdk.GetNginxConfig(tc.file, nginxID, systemID, allowedDirs)
			assert.NoError(t, err)

			policies, profiles := getContent(cfg)
			assert.ElementsMatch(t, tc.expPolicies, policies)
			assert.ElementsMatch(t, tc.expProfiles, profiles)
		})
	}
}

func setUpFile(file string, content []byte) error {
	err := os.MkdirAll(filepath.Dir(file), 0755)
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(file, content, 0644)
	if err != nil {
		return err
	}

	return nil
}

func tearDownDirectories() {
	os.RemoveAll("/tmp/testdata")
}
