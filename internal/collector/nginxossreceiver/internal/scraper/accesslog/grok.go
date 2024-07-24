// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package accesslog

import (
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/trivago/grok"
	"go.uber.org/zap"
)

var (
	formatVariables = map[string]string{
		"$remote_addr":              "%{IPORHOST:remote_addr}",
		"$remote_user":              "%{USERNAME:remote_user}",
		"$time_local":               `%{HTTPDATE:time_local}`,
		"$status":                   "%{INT:status}",
		"$body_bytes_sent":          "%{NUMBER:body_bytes_sent}",
		"$http_referer":             "%{DATA:http_referer}",
		"$http_user_agent":          "%{DATA:http_user_agent}",
		"$http_x_forwarded_for":     "%{DATA:http_x_forwarded_for}",
		"$bytes_sent":               "%{NUMBER:bytes_sent}",
		"$gzip_ratio":               "%{DATA:gzip_ratio}",
		"$server_protocol":          "%{DATA:server_protocol}",
		"$request_length":           "%{INT:request_length}",
		"$request_time":             "%{DATA:request_time}",
		"\"$request\"":              "\"%{DATA:request}\"",
		"$request ":                 "%{DATA:request} ",
		"$upstream_connect_time":    "%{DATA:upstream_connect_time}",
		"$upstream_header_time":     "%{DATA:upstream_header_time}",
		"$upstream_response_time":   "%{DATA:upstream_response_time}",
		"$upstream_response_length": "%{DATA:upstream_response_length}",
		"$upstream_status":          "%{DATA:upstream_status}",
		"$upstream_cache_status":    "%{DATA:upstream_cache_status}",
		"[":                         "\\[",
		"]":                         "\\]",
	}

	// Pattern to match all the variables that are mentioned in the access log format
	logVarRegex = regexp.MustCompile(`\$([a-zA-Z]+[_[a-zA-Z]+]*)`)
)

func newGrok(logFormat string, logger *zap.SugaredLogger) (*grok.CompiledGrok, error) {
	if logger == nil {
		return nil, errors.New("Logger cannot be nil")
	}

	// The log format can have trailing whitespace which will cause grok to NOT work, so the trim is important.
	grokPattern := strings.TrimSpace(logFormat)
	for key, value := range formatVariables {
		grokPattern = strings.ReplaceAll(grokPattern, key, value)
	}
	grokPattern = replaceCustomLogVars(grokPattern)
	logger.Debugf("Using Grok pattern: %s", grokPattern)

	g, err := grok.New(grok.Config{
		NamedCapturesOnly: false,
		Patterns:          map[string]string{"DEFAULT": grokPattern},
	})
	if err != nil {
		return nil, err
	}

	return g.Compile("%{DEFAULT}")
}

func replaceCustomLogVars(logPattern string) string {
	variables := logVarRegex.FindAllStringSubmatch(logPattern, -1)

	for _, match := range variables {
		variable := match[0]
		subMatch := match[1] // Excludes the leading $ in the var name

		replacement := fmt.Sprintf("%%{DATA:%s}", subMatch)
		logPattern = strings.Replace(logPattern, string(variable), replacement, 1)
	}
	return logPattern
}
