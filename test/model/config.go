// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package model

import "github.com/nginx/agent/v3/internal/model"

func GetConfigContext() *model.NginxConfigContext {
	return &model.NginxConfigContext{
		AccessLogs: []*model.AccessLog{{Name: "access.logs"}},
		ErrorLogs:  []*model.ErrorLog{{Name: "error.log"}},
	}
}

func GetConfigContextWithNames(
	accessLogName,
	combinedAccessLogName,
	ltsvAccessLogName,
	errorLogName string,
	instanceID string,
) *model.NginxConfigContext {
	return &model.NginxConfigContext{
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
		InstanceID: instanceID,
	}
}
