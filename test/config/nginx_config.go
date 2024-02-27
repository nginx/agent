// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package config

import (
	"embed"
	"fmt"
)

//go:embed nginx/nginx.conf
var embedNginxConf embed.FS

//go:embed nginx/nginx-with-multiple-access-logs.conf
var embedNginxConfWithMultipleAccessLogs embed.FS

func GetNginxConfig() (string, error) {
	content, err := embedNginxConf.ReadFile("nginx/nginx.conf")
	if err != nil {
		return "", err
	}

	return string(content), nil
}

func GetNginxConfigWithMultipleAccessLogs(
	errorLogName,
	accessLogName,
	combinedAccessLogName,
	ltsvAccessLogName string,
) (string, error) {
	content, err := embedNginxConfWithMultipleAccessLogs.ReadFile("nginx/nginx-with-multiple-access-logs.conf")
	if err != nil {
		return "", err
	}

	return fmt.Sprintf(string(content), errorLogName, accessLogName, combinedAccessLogName, ltsvAccessLogName), err
}
