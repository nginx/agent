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
	"strings"

	"github.com/nginx/agent/v3/api/grpc/instances"
	datasource_os "github.com/nginx/agent/v3/internal/datasource/os"
	"github.com/nginx/agent/v3/internal/model"
	crossplane "github.com/nginxinc/nginx-go-crossplane"
)

const (
	predefinedAccessLogFormat = "$remote_addr - $remote_user [$time_local] \"$request\" $status $body_bytes_sent \"$http_referer\" \"$http_user_agent\""
)

type (
	crossplaneTraverseCallback = func(parent *crossplane.Directive, current *crossplane.Directive) (bool, error)
)

type Nginx struct{}

func NewNginx() *Nginx {
	return &Nginx{}
}

func (*Nginx) ParseConfig(instance *instances.Instance) (any, error) {
	payload, err := crossplane.Parse(instance.Meta.GetNginxMeta().GetConfigPath(),
		&crossplane.ParseOptions{
			IgnoreDirectives:   []string{},
			SingleFile:         false,
			StopParsingOnError: true,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("error reading config from %s, error: %s", instance.Meta.GetNginxMeta().GetConfigPath(), err)
	}

	accessLogs := []*model.AccessLog{}
	errorLogs := []*model.ErrorLog{}

	for _, xpConf := range payload.Config {
		formatMap := map[string]string{}

		err := crossplaneConfigTraverse(&xpConf,
			func(parent *crossplane.Directive, directive *crossplane.Directive) (bool, error) {
				switch directive.Directive {
				case "log_format":
					formatMap = getFormatMap(directive)
				case "access_log":
					accessLog := getAccessLog(directive.Args[0], getAccessLogDirectiveFormat(directive), formatMap)
					accessLogs = append(accessLogs, accessLog)
				case "error_log":
					errorLog := getErrorLog(directive.Args[0], getErrorLogDirectiveLevel(directive))
					errorLogs = append(errorLogs, errorLog)
				}
				return true, nil
			})
		if err != nil {
			return nil, fmt.Errorf("failed to traverse nginx config: %s", err)
		}
	}

	return &model.NginxConfigContext{
		AccessLogs: accessLogs,
		ErrorLogs:  errorLogs,
	}, nil
}

func getFormatMap(directive *crossplane.Directive) map[string]string {
	formatMap := map[string]string{}

	if len(directive.Args) >= 2 {
		if directive.Args[0] == "ltsv" {
			formatMap[directive.Args[0]] = "ltsv"
		} else {
			formatMap[directive.Args[0]] = strings.Join(directive.Args[1:], "")
		}
	}

	return formatMap
}

func getAccessLog(file string, format string, formatMap map[string]string) *model.AccessLog {
	accessLog := &model.AccessLog{
		Name:     file,
		Readable: false,
	}

	info, err := os.Stat(file)
	if err == nil {
		accessLog.Readable = true
		accessLog.Permissions = datasource_os.GetPermissions(info.Mode())
	}

	if formatMap[format] != "" {
		accessLog.Format = formatMap[format]
	} else if format == "" || format == "combined" {
		accessLog.Format = predefinedAccessLogFormat
	} else if format == "ltsv" {
		accessLog.Format = format
	} else {
		accessLog.Format = ""
	}

	return accessLog
}

func getErrorLog(file string, level string) *model.ErrorLog {
	errorLog := &model.ErrorLog{
		Name:     file,
		LogLevel: level,
		Readable: false,
	}
	info, err := os.Stat(file)
	if err == nil {
		errorLog.Permissions = datasource_os.GetPermissions(info.Mode())
		errorLog.Readable = true
	}

	return errorLog
}

func getAccessLogDirectiveFormat(directive *crossplane.Directive) string {
	if len(directive.Args) >= 2 {
		return strings.ReplaceAll(directive.Args[1], "$", "")
	}
	return ""
}

func getErrorLogDirectiveLevel(directive *crossplane.Directive) string {
	if len(directive.Args) >= 2 {
		return directive.Args[1]
	}
	return ""
}

func crossplaneConfigTraverse(root *crossplane.Config, callback crossplaneTraverseCallback) error {
	stop := false
	for _, dir := range root.Parsed {
		result, err := callback(nil, dir)
		if err != nil {
			return err
		}

		if !result {
			return nil
		}

		err = traverse(dir, callback, &stop)

		if err != nil {
			return err
		}
	}
	return nil
}

func traverse(root *crossplane.Directive, callback crossplaneTraverseCallback, stop *bool) error {
	if *stop {
		return nil
	}
	for _, child := range root.Block {
		result, err := callback(root, child)
		if err != nil {
			return err
		}

		if !result {
			*stop = true
			return nil
		}

		err = traverse(child, callback, stop)

		if err != nil {
			return err
		}

		if *stop {
			return nil
		}
	}
	return nil
}
