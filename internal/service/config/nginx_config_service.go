// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package config

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	re "regexp"
	"strings"

	"github.com/nginx/agent/v3/internal/config"
	writer "github.com/nginx/agent/v3/internal/datasource/config"

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
	configWriter  writer.ConfigWriterInterface
	fileCache     writer.FileCacheInterface
	instance      *instances.Instance
	agentConfig   *config.Config
}

func NewNginx(instance *instances.Instance, agentConfig *config.Config) *Nginx {
	fileCache := writer.NewFileCache(instance.GetInstanceId())
	cache, err := fileCache.ReadFileCache()
	// Will in future work check cache and if its nil upload file
	if err != nil {
		err = fileCache.UpdateFileCache(cache)
		if err != nil {
			slog.Debug("error updating file cache %w", err)
		}
	}

	configWriter, err := writer.NewConfigWriter(agentConfig, fileCache)
	if err != nil {
		slog.Error("failed to create new config writer for", "instance_id", instance.GetInstanceId(), "err", err)
	}

	return &Nginx{
		configContext: &model.NginxConfigContext{},
		executor:      &exec.Exec{},
		fileCache:     fileCache,
		configWriter:  configWriter,
		instance:      instance,
		agentConfig:   agentConfig,
	}
}

func (n *Nginx) Write(ctx context.Context, filesURL, tenantID string) (skippedFiles writer.CacheContent,
	err error,
) {
	return n.configWriter.Write(ctx, filesURL, tenantID, n.instance.GetInstanceId())
}

func (n *Nginx) Complete() error {
	return n.configWriter.Complete()
}

func (n *Nginx) Rollback(ctx context.Context, skippedFiles writer.CacheContent,
	filesURL, tenantID, instanceID string,
) error {
	err := n.configWriter.Rollback(ctx, skippedFiles, filesURL, tenantID, instanceID)
	return err
}

func (n *Nginx) SetConfigWriter(configWriter writer.ConfigWriterInterface) {
	n.configWriter = configWriter
}

func (n *Nginx) ParseConfig() (any, error) {
	payload, err := crossplane.Parse(n.instance.GetMeta().GetNginxMeta().GetConfigPath(),
		&crossplane.ParseOptions{
			IgnoreDirectives:   []string{},
			SingleFile:         false,
			StopParsingOnError: true,
		},
	)
	if err != nil {
		return nil, fmt.Errorf(
			"error reading config from %s, error: %w",
			n.instance.GetMeta().GetNginxMeta().GetConfigPath(),
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

func (n *Nginx) Validate() error {
	slog.Debug("Validating NGINX config")
	exePath := n.instance.GetMeta().GetNginxMeta().GetExePath()

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

func (n *Nginx) Apply() error {
	slog.Debug("Applying NGINX config")
	var errorsFound error

	errorLogs := n.configContext.ErrorLogs

	logErrorChannel := make(chan error, len(errorLogs))
	defer close(logErrorChannel)

	go n.monitorLogs(errorLogs, logErrorChannel)

	processID := n.instance.GetMeta().GetNginxMeta().GetProcessId()
	err := n.executor.KillProcess(processID)
	if err != nil {
		return fmt.Errorf("failed to reload NGINX, %w", err)
	}

	slog.Info("NGINX reloaded", "process_id", processID)

	numberOfExpectedMessages := len(errorLogs)

	for i := 0; i < numberOfExpectedMessages; i++ {
		err := <-logErrorChannel
		slog.Debug("Message received in logErrorChannel", "error", err)
		if err != nil {
			errorsFound = errors.Join(errorsFound, err)
		}
	}

	slog.Info("Finished monitoring post reload")

	if errorsFound != nil {
		return errorsFound
	}

	return nil
}

func (n *Nginx) monitorLogs(errorLogs []*model.ErrorLog, errorChannel chan error) {
	if len(errorLogs) == 0 {
		slog.Info("No NGINX error logs found to monitor")
		return
	}

	for _, errorLog := range errorLogs {
		go n.tailLog(errorLog.Name, errorChannel)
	}
}

func (n *Nginx) tailLog(logFile string, errorChannel chan error) {
	t, err := nginx.NewTailer(logFile)
	if err != nil {
		slog.Error("Unable to tail error log after NGINX reload", "log_file", logFile, "error", err)
		// this is not an error in the logs, ignoring tailing
		errorChannel <- nil

		return
	}

	ctx, cncl := context.WithTimeout(context.Background(), n.agentConfig.DataPlaneConfig.Nginx.ReloadMonitoringPeriod)
	defer cncl()

	slog.Debug("Monitoring NGINX error log file for any errors", "file", logFile)

	data := make(chan string, logTailerChannelSize)
	go t.Tail(ctx, data)

	for {
		select {
		case d := <-data:
			if n.doesLogLineContainError(d) {
				errorChannel <- fmt.Errorf(d)
				return
			}
		case <-ctx.Done():
			errorChannel <- nil
			return
		}
	}
}

func (n *Nginx) doesLogLineContainError(line string) bool {
	if n.agentConfig.DataPlaneConfig.Nginx.TreatWarningsAsError && warningRegex.MatchString(line) {
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
