/**
 * Copyright (c) F5, Inc.
 *
 * This source code is licensed under the Apache License, Version 2.0 license found in the
 * LICENSE file in the root directory of this source tree.
 */

package config

import (
	"fmt"
	"os"
	"testing"

	"github.com/nginx/agent/v3/api/grpc/instances"
	"github.com/nginx/agent/v3/internal/model"
	"github.com/stretchr/testify/assert"
)

func TestNginx_ParseConfig(t *testing.T) {
	file, err := os.CreateTemp("./", "nginx-parse-config.conf")
	defer os.Remove(file.Name())
	assert.NoError(t, err)

	errorLog, err := os.CreateTemp("./", "error.log")
	defer os.Remove(errorLog.Name())
	assert.NoError(t, err)

	accessLog, err := os.CreateTemp("./", "access.log")
	defer os.Remove(accessLog.Name())
	assert.NoError(t, err)

	combinedAccessLog, err := os.CreateTemp("./", "combined_access.log")
	defer os.Remove(combinedAccessLog.Name())
	assert.NoError(t, err)

	ltsvAccessLog, err := os.CreateTemp("./", "ltsv_access.log")
	defer os.Remove(ltsvAccessLog.Name())
	assert.NoError(t, err)

	data := []byte(fmt.Sprintf(`
		user  nginx;
		worker_processes  auto;
		
		error_log  %s notice;
		pid        /var/run/nginx.pid;
		
		
		events {
			worker_connections  1024;
		}

		http {
			log_format upstream_time '$remote_addr - $remote_user [$time_local]';
		
			server {
				access_log %s upstream_time;
				access_log %s combined;
			}
		}

		http {
			log_format ltsv "time:$time_local"
					"\thost:$remote_addr"                                                                                   
					"\tmethod:$request_method"
					"\turi:$request_uri"
					"\tprotocol:$server_protocol"
					"\tstatus:$status"
					"\tsize:$body_bytes_sent"
					"\treferer:$http_referer"
					"\tua:$http_user_agent"
					"\treqtime:$request_time"
					"\tapptime:$upstream_response_time";
		
			server {
				access_log %s ltsv;
			}
		}
	`, errorLog.Name(), accessLog.Name(), combinedAccessLog.Name(), ltsvAccessLog.Name()))
	err = os.WriteFile(file.Name(), data, 0o644)
	assert.NoError(t, err)

	expectedConfigContext := &model.NginxConfigContext{
		AccessLogs: []*model.AccessLog{
			{
				Name:        accessLog.Name(),
				Format:      "$remote_addr - $remote_user [$time_local]",
				Readable:    true,
				Permissions: "0600",
			},
			{
				Name:        combinedAccessLog.Name(),
				Format:      "$remote_addr - $remote_user [$time_local] \"$request\" $status $body_bytes_sent \"$http_referer\" \"$http_user_agent\"",
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

	nginxConfig := NewNginx()
	result, err := nginxConfig.ParseConfig(&instances.Instance{
		Type: instances.Type_NGINX,
		Meta: &instances.Meta{
			Meta: &instances.Meta_NginxMeta{
				NginxMeta: &instances.NginxMeta{
					ConfigPath: file.Name(),
				},
			},
		},
	})

	assert.NoError(t, err)
	assert.Equal(t, expectedConfigContext, result)
}
