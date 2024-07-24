// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package accesslog

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	crossplane "github.com/nginxinc/nginx-go-crossplane"
)

func logFormatFromNginxConf(nginxConfPath string) (string, error) {
	fi, err := os.Stat(nginxConfPath)
	if err != nil {
		return "", fmt.Errorf("NGINX config path [%s]: %w", nginxConfPath, err)
	}

	if fi.IsDir() {
		return "", errors.New("NGINX config path argument is a directory")
	}

	payload, err := crossplane.Parse(nginxConfPath, &crossplane.ParseOptions{
		SingleFile:         false,
		StopParsingOnError: true,
	})
	if err != nil {
		return "", fmt.Errorf("parse NGINX config: %w", err)
	}

	return extractLogFormat(payload, filepath.Base(nginxConfPath))
}

func extractLogFormat(payload *crossplane.Payload, fileName string) (string, error) {
	searchStrings := map[string]struct{}{
		"log_format": {},
	}

	results := make([]*crossplane.Directive, 0, len(searchStrings))
	for _, conf := range payload.Config {
		if strings.Contains(conf.File, fileName) {

			tmp := make([]*crossplane.Directive, 0, len(searchStrings))
			results = append(results, findDirectives(conf.Parsed, searchStrings, tmp)...)
			break
		}
	}

	if len(results) != 1 || results[0] == nil {
		return "", errors.New("no log_format directive found")
	}

	logFormatDirective := results[0]

	// The log_format directive will always have at least 2 arguments.
	return strings.Join(logFormatDirective.Args[1:], ""), nil
}

// Recursive function for finding directives based on their names.
func findDirectives(nodes crossplane.Directives, searchStrings map[string]struct{}, input []*crossplane.Directive) []*crossplane.Directive {
	// Copy to avoid operating on the original slice.
	res := make([]*crossplane.Directive, 0)
	copy(input, res)

	if len(nodes) == 0 {
		return res
	}

	for _, node := range nodes {
		if _, ok := searchStrings[node.Directive]; ok {
			res = append(res, node)
		}

		res = append(res, findDirectives(node.Block, searchStrings, res)...)
	}

	return res
}
