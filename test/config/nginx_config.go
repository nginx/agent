// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package config

import (
	_ "embed"
	"fmt"
)

//go:embed nginx/nginx-with-multiple-access-logs.conf
var embedNginxConfWithMultipleAccessLogs string

//go:embed nginx/nginx-not-allowed-dir.conf
var embedNginxConfWithNotAllowedDir string

//go:embed nginx/nginx-ssl-certs-with-variables.conf
var embedNginxConfWithSSLCertsWithVariables string

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

func GetNginxConfigWithNotAllowedDir(errorLogFile, notAllowedFile, allowedFileDir, accessLogFile string) string {
	return fmt.Sprintf(embedNginxConfWithNotAllowedDir, errorLogFile, notAllowedFile, allowedFileDir, accessLogFile)
}

func GetNginxConfWithSSLCertsWithVariables() string {
	return embedNginxConfWithSSLCertsWithVariables
}
