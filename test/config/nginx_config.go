// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package config

import (
	_ "embed"
	"fmt"
)

//go:embed nginx/nginx.conf
var embedNginxConf string

//go:embed nginx/nginx-with-test-location.conf
var embedNginxConfWithTestLocation string

//go:embed nginx/nginx-with-multiple-access-logs.conf
var embedNginxConfWithMultipleAccessLogs string

func GetNginxConfig() string {
	return embedNginxConf
}

func GetNginxConfWithTestLocation() string {
	return embedNginxConfWithTestLocation
}

func GetNginxConfigWithMultipleAccessLogs(
	errorLogName,
	accessLogName,
	combinedAccessLogName,
	ltsvAccessLogName string,
) string {
	return fmt.Sprintf(
		embedNginxConfWithMultipleAccessLogs,
		errorLogName,
		accessLogName,
		combinedAccessLogName,
		ltsvAccessLogName,
	)
}
