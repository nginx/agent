// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package config

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"os"
	re "regexp"
	"strconv"
	"strings"

	"github.com/nginx/agent/v3/internal/client"

	"github.com/nginx/agent/v3/pkg/files"
	"github.com/nginx/agent/v3/internal/config"
	writer "github.com/nginx/agent/v3/internal/datasource/config"

	"github.com/nginx/agent/v3/api/grpc/mpi/v1"
	"github.com/nginx/agent/v3/internal/model"
	crossplane "github.com/nginxinc/nginx-go-crossplane"

	"github.com/nginx/agent/v3/internal/datasource/host/exec"
	"github.com/nginx/agent/v3/internal/datasource/nginx"
)

const (
	predefinedAccessLogFormat = "$remote_addr - $remote_user [$time_local]" +
		" \"$request\" $status $body_bytes_sent \"$http_referer\" \"$http_user_agent\""
	ltsvArg                           = "ltsv"
	defaultNumberOfDirectiveArguments = 2
	logTailerChannelSize              = 1024
	plusAPIDirective                  = "api"
	stubStatusAPIDirective            = "stub_status"
	apiFormat                         = "http://%s%s"
	locationDirective                 = "location"
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
	crossplaneTraverseCallback    = func(ctx context.Context, parent, current *crossplane.Directive) error
	crossplaneTraverseCallbackStr = func(ctx context.Context, parent, current *crossplane.Directive) string
)

type Nginx struct {
	executor      exec.ExecInterface
	configContext *model.NginxConfigContext
	configWriter  writer.ConfigWriterInterface
	fileCache     writer.FileCacheInterface
	instance      *v1.Instance
	agentConfig   *config.Config
}

func NewNginx(ctx context.Context, instance *v1.Instance, agentConfig *config.Config,
	configClient client.ConfigClient,
) *Nginx {
	fileCache := writer.NewFileCache(instance.GetInstanceMeta().GetInstanceId())
	cache, err := fileCache.ReadFileCache(ctx)
	if err != nil {
		err = fileCache.UpdateFileCache(ctx, cache)
		if err != nil {
			slog.DebugContext(ctx, "Error updating file cache", "error", err)
		}
	}

	configWriter, err := writer.NewConfigWriter(agentConfig, fileCache, configClient)
	if err != nil {
		slog.ErrorContext(
			ctx,
			"Failed to create new config writer",
			"instance_id", instance.GetInstanceMeta().GetInstanceId(),
			"error", err,
		)
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

func (n *Nginx) Write(ctx context.Context, request *v1.ManagementPlaneRequest_ConfigApplyRequest) (
	skippedFiles writer.CacheContent,
	err error,
) {
	return n.configWriter.Write(ctx, request)
}

func (n *Nginx) Complete(ctx context.Context) error {
	return n.configWriter.Complete(ctx)
}

func (n *Nginx) Rollback(ctx context.Context, skippedFiles writer.CacheContent,
	request *v1.ManagementPlaneRequest_ConfigApplyRequest,
) error {
	err := n.configWriter.Rollback(ctx, skippedFiles, request)
	return err
}

func (n *Nginx) SetConfigWriter(configWriter writer.ConfigWriterInterface) {
	n.configWriter = configWriter
}

func (n *Nginx) ParseConfig(ctx context.Context) (any, error) {
	var (
		plusAPI string
		plusErr error
	)

	payload, err := crossplane.Parse(n.instance.GetInstanceRuntime().GetConfigPath(),
		&crossplane.ParseOptions{
			IgnoreDirectives:   []string{},
			SingleFile:         false,
			StopParsingOnError: true,
		},
	)
	if err != nil {
		return nil, fmt.Errorf(
			"error reading config from %s, error: %w",
			n.instance.GetInstanceRuntime().GetConfigPath(),
			err,
		)
	}

	accessLogs, errorLogs, err := n.logs(ctx, payload)
	if err != nil {
		return nil, err
	}

	stubStatus, err := n.stubStatus(ctx, payload)
	if err != nil {
		slog.WarnContext(ctx, "Unable to get Stub Status API from NGINX configuration", "error", err)
	}

	if n.instance.GetInstanceRuntime().GetNginxPlusRuntimeInfo() != nil {
		plusAPI, plusErr = n.plusAPI(ctx, payload)
		if plusErr != nil {
			slog.WarnContext(ctx, "Unable to get Plus API from NGINX configuration ", "error", err)
		}
	}

	return &model.NginxConfigContext{
		AccessLogs: accessLogs,
		ErrorLogs:  errorLogs,
		StubStatus: stubStatus,
		PlusAPI:    plusAPI,
		InstanceID: n.instance.GetInstanceMeta().GetInstanceId(),
	}, nil
}

func (n *Nginx) SetConfigContext(configContext any) {
	if newConfigContext, ok := configContext.(*model.NginxConfigContext); ok {
		n.configContext = newConfigContext
	}
}

func (n *Nginx) stubStatus(ctx context.Context, payload *crossplane.Payload) (string, error) {
	for _, xpConf := range payload.Config {
		stubStatusAPIURL := n.crossplaneConfigTraverseStr(ctx, &xpConf, n.stubStatusAPICallback)
		if stubStatusAPIURL != "" {
			return stubStatusAPIURL, nil
		}
	}

	return "", errors.New("no stub status api reachable from the agent found")
}

func (n *Nginx) stubStatusAPICallback(ctx context.Context, parent, current *crossplane.Directive) string {
	urls := n.urlsForLocationDirective(parent, current, stubStatusAPIDirective)

	for _, url := range urls {
		if n.pingStubStatusAPIEndpoint(ctx, url) {
			slog.DebugContext(ctx, "Stub_status found", "url", url)
			return url
		}
		slog.DebugContext(ctx, "Stub_status is not reachable", "url", url)
	}

	return ""
}

func (n *Nginx) pingStubStatusAPIEndpoint(ctx context.Context, statusAPI string) bool {
	httpClient := http.Client{Timeout: n.agentConfig.Client.Timeout}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, statusAPI, nil)
	if err != nil {
		slog.WarnContext(ctx, "Unable to create Stub Status API GET request", "error", err)
		return false
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		slog.WarnContext(ctx, "Unable to GET Stub Status from API request", "error", err)
		return false
	}

	if resp.StatusCode != http.StatusOK {
		slog.DebugContext(ctx, "Stub Status API responded with unexpected status code", "status_code",
			resp.StatusCode, "expected", http.StatusOK)

		return false
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		slog.WarnContext(ctx, "Unable to read Stub Status API response body", "error", err)
		return false
	}

	// Expecting API to return data like this:
	//
	// Active connections: 2
	// server accepts handled requests
	//  18 18 3266
	// Reading: 0 Writing: 1 Waiting: 1
	body := string(bodyBytes)
	defer resp.Body.Close()

	return strings.Contains(body, "Active connections") && strings.Contains(body, "server accepts handled requests")
}

func (n *Nginx) plusAPI(ctx context.Context, payload *crossplane.Payload) (string, error) {
	for _, xpConfig := range payload.Config {
		plusAPIURL := n.crossplaneConfigTraverseStr(ctx, &xpConfig, n.plusAPICallback)
		if plusAPIURL != "" {
			return plusAPIURL, nil
		}
	}

	return "", errors.New("no plus api reachable from the agent found")
}

func (n *Nginx) plusAPICallback(ctx context.Context, parent, current *crossplane.Directive) string {
	urls := n.urlsForLocationDirective(parent, current, plusAPIDirective)

	for _, url := range urls {
		if n.pingPlusAPIEndpoint(ctx, url) {
			slog.DebugContext(ctx, "Plus API found", "url", url)
			return url
		}
		slog.DebugContext(ctx, "Plus API is not reachable", "url", url)
	}

	return ""
}

func (n *Nginx) pingPlusAPIEndpoint(ctx context.Context, statusAPI string) bool {
	httpClient := http.Client{Timeout: n.agentConfig.Client.Timeout}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, statusAPI, nil)
	if err != nil {
		slog.WarnContext(ctx, "Unable to create NGINX Plus API GET request", "error", err)
		return false
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		slog.WarnContext(ctx, "Unable to GET NGINX Plus API from API request", "error", err)
		return false
	}

	if resp.StatusCode != http.StatusOK {
		slog.DebugContext(ctx, "NGINX Plus API responded with unexpected status code", "status_code",
			resp.StatusCode, "expected", http.StatusOK)

		return false
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		slog.WarnContext(ctx, "Unable to read NGINX Plus API response body", "error", err)
		return false
	}

	// Expecting API to return the API versions in an array of positive integers
	// subset example: [ ... 6,7,8,9 ...]
	var responseBody []int
	err = json.Unmarshal(bodyBytes, &responseBody)
	defer resp.Body.Close()
	if err != nil {
		slog.DebugContext(ctx, "Unable to unmarshal NGINX Plus API response body", "error", err)
		return false
	}

	return true
}

func (n *Nginx) urlsForLocationDirective(parent, current *crossplane.Directive, locationDirectiveName string) []string {
	var urls []string
	// process from the location block
	if current.Directive != locationDirective {
		return urls
	}

	for _, locChild := range current.Block {
		if locChild.Directive != plusAPIDirective && locChild.Directive != stubStatusAPIDirective {
			continue
		}

		addresses := n.parseAddressesFromServerDirective(parent)

		for _, address := range addresses {
			path := n.parsePathFromLocationDirective(current)

			if locChild.Directive == locationDirectiveName {
				urls = append(urls, fmt.Sprintf(apiFormat, address, path))
			}
		}
	}

	return urls
}

func (n *Nginx) parsePathFromLocationDirective(location *crossplane.Directive) string {
	path := "/"
	if len(location.Args) > 0 {
		if location.Args[0] != "=" {
			path = location.Args[0]
		} else {
			path = location.Args[1]
		}
	}

	return path
}

func (n *Nginx) parseAddressesFromServerDirective(parent *crossplane.Directive) []string {
	foundHosts := []string{}
	port := "80"

	if parent == nil {
		return []string{}
	}

	for _, dir := range parent.Block {
		var hostname string

		switch dir.Directive {
		case "listen":
			listenHost, listenPort, err := net.SplitHostPort(dir.Args[0])
			if err == nil {
				hostname, port = n.parseListenHostAndPort(listenHost, listenPort)
			} else {
				hostname, port = n.parseListenDirective(dir, "127.0.0.1", port)
			}
			foundHosts = append(foundHosts, hostname)
		case "server_name":
			if dir.Args[0] == "_" {
				// default server
				continue
			}
			hostname = dir.Args[0]
			foundHosts = append(foundHosts, hostname)
		}
	}

	return n.formatAddresses(foundHosts, port)
}

func (n *Nginx) formatAddresses(foundHosts []string, port string) []string {
	addresses := []string{}
	for _, foundHost := range foundHosts {
		addresses = append(addresses, fmt.Sprintf("%s:%s", foundHost, port))
	}

	return addresses
}

func (n *Nginx) parseListenDirective(
	dir *crossplane.Directive,
	hostname, port string,
) (directiveHost, directivePort string) {
	directiveHost = hostname
	directivePort = port
	if n.isPort(dir.Args[0]) {
		directivePort = dir.Args[0]
	} else {
		directiveHost = dir.Args[0]
	}

	return directiveHost, directivePort
}

func (n *Nginx) parseListenHostAndPort(listenHost, listenPort string) (hostname, port string) {
	if listenHost == "*" || listenHost == "" {
		hostname = "127.0.0.1"
	} else if listenHost == "::" || listenHost == "::1" {
		hostname = "[::1]"
	} else {
		hostname = listenHost
	}
	port = listenPort

	return hostname, port
}

func (n *Nginx) isPort(value string) bool {
	port, err := strconv.Atoi(value)

	return err == nil && port >= 1 && port <= 65535
}

func (n *Nginx) logs(ctx context.Context, payload *crossplane.Payload) ([]*model.AccessLog, []*model.ErrorLog, error) {
	accessLogs := []*model.AccessLog{}
	errorLogs := []*model.ErrorLog{}
	for index := range payload.Config {
		formatMap := make(map[string]string)
		err := n.crossplaneConfigTraverse(ctx, &payload.Config[index],
			func(ctx context.Context, parent, directive *crossplane.Directive) error {
				switch directive.Directive {
				case "log_format":
					formatMap = n.formatMap(directive)
				case "access_log":
					accessLog := n.accessLog(directive.Args[0], n.accessLogDirectiveFormat(directive), formatMap)
					accessLogs = append(accessLogs, accessLog)
				case "error_log":
					errorLog := n.errorLog(directive.Args[0], n.errorLogDirectiveLevel(directive))
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

func (n *Nginx) Validate(ctx context.Context) error {
	slog.DebugContext(ctx, "Validating NGINX config")
	exePath := n.instance.GetInstanceRuntime().GetBinaryPath()

	out, err := n.executor.RunCmd(ctx, exePath, "-t")
	if err != nil {
		return fmt.Errorf("NGINX config test failed %w: %s", err, out)
	}

	err = n.validateConfigCheckResponse(out.Bytes())
	if err != nil {
		return err
	}

	slog.InfoContext(ctx, "NGINX config tested", "output", out)

	return nil
}

func (n *Nginx) Apply(ctx context.Context) error {
	slog.DebugContext(ctx, "Applying NGINX config")
	var errorsFound error

	errorLogs := n.configContext.ErrorLogs

	logErrorChannel := make(chan error, len(errorLogs))
	defer close(logErrorChannel)

	go n.monitorLogs(ctx, errorLogs, logErrorChannel)

	processID := n.instance.GetInstanceRuntime().GetProcessId()
	err := n.executor.KillProcess(processID)
	if err != nil {
		return fmt.Errorf("failed to reload NGINX, %w", err)
	}

	slog.InfoContext(ctx, "NGINX reloaded", "process_id", processID)

	numberOfExpectedMessages := len(errorLogs)

	for i := 0; i < numberOfExpectedMessages; i++ {
		err = <-logErrorChannel
		slog.DebugContext(ctx, "Message received in logErrorChannel", "error", err)
		if err != nil {
			errorsFound = errors.Join(errorsFound, err)
		}
	}

	slog.InfoContext(ctx, "Finished monitoring post reload")

	if errorsFound != nil {
		return errorsFound
	}

	return nil
}

func (n *Nginx) monitorLogs(ctx context.Context, errorLogs []*model.ErrorLog, errorChannel chan error) {
	if len(errorLogs) == 0 {
		slog.InfoContext(ctx, "No NGINX error logs found to monitor")
		return
	}

	for _, errorLog := range errorLogs {
		go n.tailLog(ctx, errorLog.Name, errorChannel)
	}
}

func (n *Nginx) tailLog(ctx context.Context, logFile string, errorChannel chan error) {
	t, err := nginx.NewTailer(logFile)
	if err != nil {
		slog.ErrorContext(ctx, "Unable to tail error log after NGINX reload", "log_file", logFile, "error", err)
		// this is not an error in the logs, ignoring tailing
		errorChannel <- nil

		return
	}

	ctxWithTimeout, cancel := context.WithTimeout(ctx, n.agentConfig.DataPlaneConfig.Nginx.ReloadMonitoringPeriod)
	defer cancel()

	slog.DebugContext(ctxWithTimeout, "Monitoring NGINX error log file for any errors", "file", logFile)

	data := make(chan string, logTailerChannelSize)
	go t.Tail(ctxWithTimeout, data)

	for {
		select {
		case d := <-data:
			if n.doesLogLineContainError(d) {
				errorChannel <- fmt.Errorf(d)
				return
			}
		case <-ctxWithTimeout.Done():
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

func (n *Nginx) validateConfigCheckResponse(out []byte) error {
	if bytes.Contains(out, []byte("[emerg]")) ||
		bytes.Contains(out, []byte("[alert]")) ||
		bytes.Contains(out, []byte("[crit]")) {
		return fmt.Errorf("error running nginx -t -c:\n%s", out)
	}

	return nil
}

func (n *Nginx) formatMap(directive *crossplane.Directive) map[string]string {
	formatMap := make(map[string]string)

	if n.hasAdditionArguments(directive.Args) {
		if directive.Args[0] == ltsvArg {
			formatMap[directive.Args[0]] = ltsvArg
		} else {
			formatMap[directive.Args[0]] = strings.Join(directive.Args[1:], "")
		}
	}

	return formatMap
}

func (n *Nginx) accessLog(file, format string, formatMap map[string]string) *model.AccessLog {
	accessLog := &model.AccessLog{
		Name:     file,
		Readable: false,
	}

	info, err := os.Stat(file)
	if err == nil {
		accessLog.Readable = true
		accessLog.Permissions = files.GetPermissions(info.Mode())
	}

	accessLog = n.updateLogFormat(format, formatMap, accessLog)

	return accessLog
}

func (n *Nginx) updateLogFormat(
	format string,
	formatMap map[string]string,
	accessLog *model.AccessLog,
) *model.AccessLog {
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

func (n *Nginx) errorLog(file, level string) *model.ErrorLog {
	errorLog := &model.ErrorLog{
		Name:     file,
		LogLevel: level,
		Readable: false,
	}
	info, err := os.Stat(file)
	if err == nil {
		errorLog.Permissions = files.GetPermissions(info.Mode())
		errorLog.Readable = true
	}

	return errorLog
}

func (n *Nginx) accessLogDirectiveFormat(directive *crossplane.Directive) string {
	if n.hasAdditionArguments(directive.Args) {
		return strings.ReplaceAll(directive.Args[1], "$", "")
	}

	return ""
}

func (n *Nginx) errorLogDirectiveLevel(directive *crossplane.Directive) string {
	if n.hasAdditionArguments(directive.Args) {
		return directive.Args[1]
	}

	return ""
}

func (n *Nginx) crossplaneConfigTraverse(
	ctx context.Context,
	root *crossplane.Config,
	callback crossplaneTraverseCallback,
) error {
	for _, dir := range root.Parsed {
		err := callback(ctx, nil, dir)
		if err != nil {
			return err
		}

		err = n.traverse(ctx, dir, callback)
		if err != nil {
			return err
		}
	}

	return nil
}

func (n *Nginx) crossplaneConfigTraverseStr(
	ctx context.Context,
	root *crossplane.Config,
	callback crossplaneTraverseCallbackStr,
) string {
	stop := false
	response := ""
	for _, dir := range root.Parsed {
		response = callback(ctx, nil, dir)
		if response != "" {
			return response
		}
		response = traverseStr(ctx, dir, callback, &stop)
		if response != "" {
			return response
		}
	}

	return response
}

func traverseStr(
	ctx context.Context,
	root *crossplane.Directive,
	callback crossplaneTraverseCallbackStr,
	stop *bool,
) string {
	response := ""
	if *stop {
		return ""
	}
	for _, child := range root.Block {
		response = callback(ctx, root, child)
		if response != "" {
			*stop = true
			return response
		}
		response = traverseStr(ctx, child, callback, stop)
		if *stop {
			return response
		}
	}

	return response
}

func (n *Nginx) traverse(ctx context.Context, root *crossplane.Directive, callback crossplaneTraverseCallback) error {
	for _, child := range root.Block {
		err := callback(ctx, root, child)
		if err != nil {
			return err
		}

		err = n.traverse(ctx, child, callback)
		if err != nil {
			return err
		}
	}

	return nil
}

func (n *Nginx) hasAdditionArguments(args []string) bool {
	return len(args) >= defaultNumberOfDirectiveArguments
}
