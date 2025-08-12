// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package model

import (
	mpi "github.com/nginx/agent/v3/api/grpc/mpi/v1"
	"github.com/nginx/agent/v3/internal/model"
)

func ConfigContext() *model.NginxConfigContext {
	return &model.NginxConfigContext{
		StubStatus: &model.APIDetails{
			URL:      "",
			Listen:   "",
			Location: "",
		},
		AccessLogs: []*model.AccessLog{{Name: "access.logs"}},
		ErrorLogs:  []*model.ErrorLog{{Name: "error.log"}},
	}
}

// nolint: revive
func ConfigContextWithNames(
	accessLogName,
	combinedAccessLogName,
	ltsvAccessLogName,
	errorLogName string,
	instanceID string,
	syslogServers []string,
) *model.NginxConfigContext {
	return &model.NginxConfigContext{
		StubStatus: &model.APIDetails{
			URL:      "",
			Listen:   "",
			Location: "",
		},
		PlusAPI: &model.APIDetails{
			URL:      "",
			Listen:   "",
			Location: "",
		},
		AccessLogs: []*model.AccessLog{
			{
				Name:        accessLogName,
				Format:      "$remote_addr - $remote_user [$time_local]",
				Readable:    true,
				Permissions: "0600",
			},
			{
				Name: combinedAccessLogName,
				Format: "$remote_addr - $remote_user [$time_local] " +
					"\"$request\" $status $body_bytes_sent \"$http_referer\" \"$http_user_agent\"",
				Readable:    true,
				Permissions: "0600",
			},
			{
				Name:        ltsvAccessLogName,
				Format:      "ltsv",
				Readable:    true,
				Permissions: "0600",
			},
		},
		ErrorLogs: []*model.ErrorLog{
			{
				Name:        errorLogName,
				LogLevel:    "notice",
				Readable:    true,
				Permissions: "0600",
			},
		},
		InstanceID:       instanceID,
		NAPSysLogServers: syslogServers,
	}
}

func ConfigContextWithoutErrorLog(
	accessLogName,
	combinedAccessLogName,
	ltsvAccessLogName,
	instanceID string,
	syslogServers []string,
) *model.NginxConfigContext {
	return &model.NginxConfigContext{
		StubStatus: &model.APIDetails{
			URL:      "",
			Listen:   "",
			Location: "",
		},
		PlusAPI: &model.APIDetails{
			URL:      "",
			Listen:   "",
			Location: "",
		},
		AccessLogs: []*model.AccessLog{
			{
				Name:        accessLogName,
				Format:      "$remote_addr - $remote_user [$time_local]",
				Readable:    true,
				Permissions: "0600",
			},
			{
				Name: combinedAccessLogName,
				Format: "$remote_addr - $remote_user [$time_local] " +
					"\"$request\" $status $body_bytes_sent \"$http_referer\" \"$http_user_agent\"",
				Readable:    true,
				Permissions: "0600",
			},
			{
				Name:        ltsvAccessLogName,
				Format:      "ltsv",
				Readable:    true,
				Permissions: "0600",
			},
		},
		InstanceID:       instanceID,
		NAPSysLogServers: syslogServers,
	}
}

func ConfigContextWithFiles(
	accessLogName,
	errorLogName string,
	files []*mpi.File,
	instanceID string,
	syslogServers []string,
) *model.NginxConfigContext {
	return &model.NginxConfigContext{
		StubStatus: &model.APIDetails{
			URL:      "",
			Listen:   "",
			Location: "",
		},
		PlusAPI: &model.APIDetails{
			URL:      "",
			Listen:   "",
			Location: "",
		},
		Files: files,
		AccessLogs: []*model.AccessLog{
			{
				Name: accessLogName,
				Format: "$remote_addr - $remote_user [$time_local] \"$request\" $status $body_bytes_sent " +
					"\"$http_referer\" \"$http_user_agent\" \"$http_x_forwarded_for\" \"$bytes_sent\" " +
					"\"$request_length\" \"$request_time\" \"$gzip_ratio\" $server_protocol ",
				Readable:    true,
				Permissions: "0600",
			},
		},
		ErrorLogs: []*model.ErrorLog{
			{
				Name:        errorLogName,
				Readable:    true,
				Permissions: "0600",
			},
		},
		InstanceID:       instanceID,
		NAPSysLogServers: syslogServers,
	}
}

func ConfigContextWithSysLog(
	accessLogName,
	errorLogName string,
	instanceID string,
	syslogServers []string,
) *model.NginxConfigContext {
	return &model.NginxConfigContext{
		StubStatus: &model.APIDetails{
			URL:      "",
			Listen:   "",
			Location: "",
		},
		PlusAPI: &model.APIDetails{
			URL:      "",
			Listen:   "",
			Location: "",
		},
		AccessLogs: []*model.AccessLog{
			{
				Name:        accessLogName,
				Format:      "$remote_addr - $remote_user [$time_local]",
				Readable:    true,
				Permissions: "0600",
			},
		},
		ErrorLogs: []*model.ErrorLog{
			{
				Name:        errorLogName,
				Readable:    true,
				LogLevel:    "notice",
				Permissions: "0600",
			},
		},
		InstanceID:       instanceID,
		NAPSysLogServers: syslogServers,
	}
}
