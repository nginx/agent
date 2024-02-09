// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package config

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"os"
	re "regexp"
	"strings"
	"time"

	"github.com/nginx/agent/v3/api/grpc/instances"
	"github.com/nginx/agent/v3/internal/model"
	crossplane "github.com/nginxinc/nginx-go-crossplane"

	"github.com/nginx/agent/v3/internal/datasource/host"
	"github.com/nginx/agent/v3/internal/datasource/host/exec"
	"github.com/nginx/agent/v3/internal/datasource/nginx"
)

const (
	predefinedAccessLogFormat = "$remote_addr - $remote_user [$time_local]" +
		" \"$request\" $status $body_bytes_sent \"$http_referer\" \"$http_user_agent\""
	ltsvArg                           = "ltsv"
	defaultNumberOfDirectiveArguments = 2
	logTailerChannelSize              = 1024
)

var (
	reloadErrorList = []*re.Regexp{
		re.MustCompile(`.*\[emerg\].*`),
		re.MustCompile(`.*\[alert\].*`),
		re.MustCompile(`.*\[crit\].*`),
	}
	warningRegex = re.MustCompile(`.*\[warn\].*`)
)

type (
	crossplaneTraverseCallback = func(parent, current *crossplane.Directive) error
)

type Nginx struct {
	executor      exec.ExecInterface
	configContext *model.NginxConfigContext
}

func NewNginx() *Nginx {
	return &Nginx{
		configContext: &model.NginxConfigContext{},
		executor:      &exec.Exec{},
	}
}

func (*Nginx) ParseConfig(instance *instances.Instance) (any, error) {
	payload, err := crossplane.Parse(instance.GetMeta().GetNginxMeta().GetConfigPath(),
		&crossplane.ParseOptions{
			IgnoreDirectives:   []string{},
			SingleFile:         false,
			StopParsingOnError: true,
		},
	)
	if err != nil {
		return nil, fmt.Errorf(
			"error reading config from %s, error: %w",
			instance.GetMeta().GetNginxMeta().GetConfigPath(),
			err,
		)
	}

	accessLogs, errorLogs, err := getLogs(payload)
	if err != nil {
		return nil, err
	}

	return &model.NginxConfigContext{
		AccessLogs: accessLogs,
		ErrorLogs:  errorLogs,
	}, nil
}

func (n *Nginx) SetConfigContext(configContext any) {
	if newConfigContext, ok := configContext.(*model.NginxConfigContext); ok {
		n.configContext = newConfigContext
	}
}

func getLogs(payload *crossplane.Payload) ([]*model.AccessLog, []*model.ErrorLog, error) {
	accessLogs := []*model.AccessLog{}
	errorLogs := []*model.ErrorLog{}
	for index := range payload.Config {
		formatMap := make(map[string]string)
		err := crossplaneConfigTraverse(&payload.Config[index],
			func(parent, directive *crossplane.Directive) error {
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

				return nil
			})
		if err != nil {
			return accessLogs, errorLogs, fmt.Errorf("failed to traverse nginx config: %w", err)
		}
	}

	return accessLogs, errorLogs, nil
}

func (n *Nginx) Validate(instance *instances.Instance) error {
	exePath := instance.GetMeta().GetNginxMeta().GetExePath()

	out, err := n.executor.RunCmd(exePath, "-t")
	if err != nil {
		return fmt.Errorf("NGINX config test failed %w: %s", err, out)
	}

	err = validateConfigCheckResponse(out.Bytes())
	if err != nil {
		return err
	}

	slog.Info("NGINX config tested", "output", out)

	return nil
}

func (n *Nginx) Reload(instance *instances.Instance) error {
	var errorsFound []string

	exePath := instance.GetMeta().GetNginxMeta().GetExePath()
	errorLogs := n.configContext.ErrorLogs

	logErrorChannel := make(chan string, len(errorLogs))
	defer close(logErrorChannel)

	go n.monitorLogs(errorLogs, logErrorChannel)

	out, err := n.executor.RunCmd(exePath, "-s", "reload")
	if err != nil {
		return fmt.Errorf("failed to reload NGINX %w: %s", err, out)
	}
	slog.Info("NGINX reloaded")

	numberOfExpectedMessages := len(errorLogs)

	for i := 0; i < numberOfExpectedMessages; i++ {
		err := <-logErrorChannel
		slog.Debug("Message received in logErrorChannel", "error", err)
		if err != "" {
			errorsFound = append(errorsFound, err)
		}
	}

	slog.Info("Finished monitoring post reload")

	if len(errorsFound) > 0 {
		return fmt.Errorf(errorsFound[0])
	}

	return nil
}

func (n *Nginx) monitorLogs(errorLogs []*model.ErrorLog, errorChannel chan string) {
	if len(errorLogs) == 0 {
		slog.Info("No NGINX error logs found to monitor")
		return
	}

	for _, errorLog := range errorLogs {
		go n.tailLog(errorLog.Name, errorChannel)
	}
}

func (n *Nginx) tailLog(logFile string, errorChannel chan string) {
	t, err := nginx.NewTailer(logFile)
	if err != nil {
		slog.Error("Unable to tail error log after NGINX reload", "logFile", logFile, "error", err)
		// this is not an error in the logs, ignoring tailing
		errorChannel <- ""

		return
	}

	// TODO: get monitoring period from configuration
	ctx, cncl := context.WithTimeout(context.Background(), 5*time.Second)
	defer cncl()

	slog.Debug("Monitoring NGINX error log file for any errors", "file", logFile)

	data := make(chan string, logTailerChannelSize)
	go t.Tail(ctx, data)

	for {
		select {
		case d := <-data:
			if n.doesLogLineContainError(d) {
				errorChannel <- d
				return
			}
		case <-ctx.Done():
			errorChannel <- ""
			return
		}
	}
}

func (n *Nginx) doesLogLineContainError(line string) bool {
	// TODO: get TreatWarningsAsErrors from configuration
	if warningRegex.MatchString(line) {
		return true
	}

	for _, errorRegex := range reloadErrorList {
		if errorRegex.MatchString(line) {
			return true
		}
	}

	return false
}

func validateConfigCheckResponse(out []byte) error {
	if bytes.Contains(out, []byte("[emerg]")) ||
		bytes.Contains(out, []byte("[alert]")) ||
		bytes.Contains(out, []byte("[crit]")) {
		return fmt.Errorf("error running nginx -t -c:\n%s", out)
	}

	return nil
}

func getFormatMap(directive *crossplane.Directive) map[string]string {
	formatMap := make(map[string]string)

	if hasAdditionArguments(directive.Args) {
		if directive.Args[0] == ltsvArg {
			formatMap[directive.Args[0]] = ltsvArg
		} else {
			formatMap[directive.Args[0]] = strings.Join(directive.Args[1:], "")
		}
	}

	return formatMap
}

func getAccessLog(file, format string, formatMap map[string]string) *model.AccessLog {
	accessLog := &model.AccessLog{
		Name:     file,
		Readable: false,
	}

	info, err := os.Stat(file)
	if err == nil {
		accessLog.Readable = true
		accessLog.Permissions = host.GetPermissions(info.Mode())
	}

	accessLog = Test(format, formatMap, accessLog)

	return accessLog
}

func Test(format string, formatMap map[string]string, accessLog *model.AccessLog) *model.AccessLog {
	if formatMap[format] != "" {
		accessLog.Format = formatMap[format]
	} else if format == "" || format == "combined" {
		accessLog.Format = predefinedAccessLogFormat
	} else if format == ltsvArg {
		accessLog.Format = format
	} else {
		accessLog.Format = ""
	}

	return accessLog
}

func getErrorLog(file, level string) *model.ErrorLog {
	errorLog := &model.ErrorLog{
		Name:     file,
		LogLevel: level,
		Readable: false,
	}
	info, err := os.Stat(file)
	if err == nil {
		errorLog.Permissions = host.GetPermissions(info.Mode())
		errorLog.Readable = true
	}

	return errorLog
}

func getAccessLogDirectiveFormat(directive *crossplane.Directive) string {
	if hasAdditionArguments(directive.Args) {
		return strings.ReplaceAll(directive.Args[1], "$", "")
	}

	return ""
}

func getErrorLogDirectiveLevel(directive *crossplane.Directive) string {
	if hasAdditionArguments(directive.Args) {
		return directive.Args[1]
	}

	return ""
}

func crossplaneConfigTraverse(root *crossplane.Config, callback crossplaneTraverseCallback) error {
	for _, dir := range root.Parsed {
		err := callback(nil, dir)
		if err != nil {
			return err
		}

		err = traverse(dir, callback)

		if err != nil {
			return err
		}
	}

	return nil
}

func traverse(root *crossplane.Directive, callback crossplaneTraverseCallback) error {
	for _, child := range root.Block {
		err := callback(root, child)
		if err != nil {
			return err
		}

		err = traverse(child, callback)

		if err != nil {
			return err
		}
	}

	return nil
}

func hasAdditionArguments(args []string) bool {
	return len(args) >= defaultNumberOfDirectiveArguments
}
