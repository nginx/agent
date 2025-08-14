// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package model

import (
	"context"
	"log/slog"
	"path"
)

type ProcessInfo struct {
	ConfigureArgs   map[string]interface{}
	Version         string
	Prefix          string
	ConfPath        string
	ExePath         string
	LoadableModules []string
	DynamicModules  []string
	ProcessID       int32
}

func NginxConfPath(ctx context.Context, nginxInfo *ProcessInfo) string {
	var confPath string

	if nginxInfo.ConfigureArgs["conf-path"] != nil {
		var ok bool
		confPath, ok = nginxInfo.ConfigureArgs["conf-path"].(string)
		if !ok {
			slog.DebugContext(ctx, "failed to cast nginxInfo conf-path to string")
		}
	} else {
		confPath = path.Join(nginxInfo.Prefix, "/conf/nginx.conf")
	}

	return confPath
}

func NginxPrefix(ctx context.Context, nginxInfo *ProcessInfo) string {
	var prefix string

	if nginxInfo.ConfigureArgs["prefix"] != nil {
		var ok bool
		prefix, ok = nginxInfo.ConfigureArgs["prefix"].(string)
		if !ok {
			slog.DebugContext(ctx, "Failed to cast nginxInfo prefix to string")
		}
	} else {
		prefix = "/usr/local/nginx"
	}

	return prefix
}
