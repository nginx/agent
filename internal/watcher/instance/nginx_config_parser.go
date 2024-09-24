// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package instance

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	mpi "github.com/nginx/agent/v3/api/grpc/mpi/v1"
	"github.com/nginx/agent/v3/internal/config"
	"github.com/nginx/agent/v3/internal/model"
	"github.com/nginx/agent/v3/pkg/files"
	crossplane "github.com/nginxinc/nginx-go-crossplane"
)

const (
	predefinedAccessLogFormat = "$remote_addr - $remote_user [$time_local]" +
		" \"$request\" $status $body_bytes_sent \"$http_referer\" \"$http_user_agent\""
	ltsvArg                           = "ltsv"
	defaultNumberOfDirectiveArguments = 2
	plusAPIDirective                  = "api"
	stubStatusAPIDirective            = "stub_status"
	apiFormat                         = "http://%s%s"
	locationDirective                 = "location"
)

type (
	NginxConfigParser struct {
		agentConfig *config.Config
	}
)

var _ nginxConfigParser = (*NginxConfigParser)(nil)

type (
	crossplaneTraverseCallback    = func(ctx context.Context, parent, current *crossplane.Directive) error
	crossplaneTraverseCallbackStr = func(ctx context.Context, parent, current *crossplane.Directive) string
)

func NewNginxConfigParser(agentConfig *config.Config) *NginxConfigParser {
	return &NginxConfigParser{
		agentConfig: agentConfig,
	}
}

func (ncp *NginxConfigParser) Parse(ctx context.Context, instance *mpi.Instance) (*model.NginxConfigContext, error) {
	configPath := instance.GetInstanceRuntime().GetConfigPath()

	if !ncp.agentConfig.IsDirectoryAllowed(configPath) {
		return nil, fmt.Errorf("config path %s is not in allowed directories", configPath)
	}

	slog.DebugContext(
		ctx,
		"Parsing NGINX config",
		"file_path", configPath,
		"instance_id", instance.GetInstanceMeta().GetInstanceId(),
	)

	payload, err := crossplane.Parse(configPath,
		&crossplane.ParseOptions{
			SingleFile:         false,
			StopParsingOnError: true,
		},
	)
	if err != nil {
		return nil, err
	}

	return ncp.createNginxConfigContext(ctx, instance, payload)
}

// nolint: cyclop,revive,gocognit
func (ncp *NginxConfigParser) createNginxConfigContext(
	ctx context.Context,
	instance *mpi.Instance,
	payload *crossplane.Payload,
) (*model.NginxConfigContext, error) {
	nginxConfigContext := &model.NginxConfigContext{
		InstanceID: instance.GetInstanceMeta().GetInstanceId(),
	}

	rootDir := filepath.Dir(instance.GetInstanceRuntime().GetConfigPath())

	for _, conf := range payload.Config {
		formatMap := make(map[string]string)
		err := ncp.crossplaneConfigTraverse(ctx, &conf,
			func(ctx context.Context, parent, directive *crossplane.Directive) error {
				switch directive.Directive {
				case "log_format":
					formatMap = ncp.formatMap(directive)
				case "access_log":
					if !ncp.ignoreLog(directive.Args[0]) {
						accessLog := ncp.accessLog(directive.Args[0], ncp.accessLogDirectiveFormat(directive),
							formatMap)
						nginxConfigContext.AccessLogs = append(nginxConfigContext.AccessLogs, accessLog)
					}
				case "error_log":
					if !ncp.ignoreLog(directive.Args[0]) {
						errorLog := ncp.errorLog(directive.Args[0], ncp.errorLogDirectiveLevel(directive))
						nginxConfigContext.ErrorLogs = append(nginxConfigContext.ErrorLogs, errorLog)
					}
				case "root":
					rootFiles := ncp.rootFiles(ctx, directive.Args[0])
					nginxConfigContext.Files = append(nginxConfigContext.Files, rootFiles...)
				case "ssl_certificate", "proxy_ssl_certificate", "ssl_client_certificate", "ssl_trusted_certificate":
					sslCertFile := ncp.sslCert(ctx, directive.Args[0], rootDir)
					nginxConfigContext.Files = append(nginxConfigContext.Files, sslCertFile)
				}

				return nil
			},
		)
		if err != nil {
			return nginxConfigContext, fmt.Errorf("traverse nginx config: %w", err)
		}

		stubStatus := ncp.crossplaneConfigTraverseStr(ctx, &conf, ncp.stubStatusAPICallback)
		if stubStatus != "" {
			nginxConfigContext.StubStatus = stubStatus
		}

		if instance.GetInstanceMeta().GetInstanceType() == mpi.InstanceMeta_INSTANCE_TYPE_NGINX_PLUS {
			plusAPIURL := ncp.crossplaneConfigTraverseStr(ctx, &conf, ncp.plusAPICallback)
			if plusAPIURL != "" {
				nginxConfigContext.PlusAPI = plusAPIURL
			}
		}

		fileMeta, err := files.FileMeta(conf.File)
		if err != nil {
			slog.WarnContext(ctx, "Unable to get file metadata", "file_name", conf.File, "error", err)
		} else {
			nginxConfigContext.Files = append(nginxConfigContext.Files, &mpi.File{FileMeta: fileMeta})
		}
	}

	return nginxConfigContext, nil
}

func (ncp *NginxConfigParser) ignoreLog(logPath string) bool {
	logLower := strings.ToLower(logPath)
	ignoreLogs := []string{"off", "/dev/stderr", "/dev/stdout", "/dev/null"}

	if strings.HasPrefix(logLower, "syslog:") || slices.Contains(ignoreLogs, logLower) {
		return true
	}

	for _, path := range strings.Split(ncp.agentConfig.DataPlaneConfig.Nginx.ExcludeLogs, ":") {
		ok, err := filepath.Match(path, logPath)
		if err != nil {
			slog.Error("Invalid path for excluding log", "log_path", path)
		} else if ok {
			slog.Info("Excluding log as specified in config", "log_path", logPath)
			return true
		}
	}

	if !ncp.agentConfig.IsDirectoryAllowed(logLower) {
		slog.Warn("Log being read is outside of allowed directories", "log_path", logPath)
	}

	return false
}

func (ncp *NginxConfigParser) formatMap(directive *crossplane.Directive) map[string]string {
	formatMap := make(map[string]string)

	if ncp.hasAdditionArguments(directive.Args) {
		if directive.Args[0] == ltsvArg {
			formatMap[directive.Args[0]] = ltsvArg
		} else {
			formatMap[directive.Args[0]] = strings.Join(directive.Args[1:], "")
		}
	}

	return formatMap
}

func (ncp *NginxConfigParser) accessLog(file, format string, formatMap map[string]string) *model.AccessLog {
	accessLog := &model.AccessLog{
		Name:     file,
		Readable: false,
	}

	info, err := os.Stat(file)
	if err == nil {
		accessLog.Readable = true
		accessLog.Permissions = files.Permissions(info.Mode())
	}

	accessLog = ncp.updateLogFormat(format, formatMap, accessLog)

	return accessLog
}

func (ncp *NginxConfigParser) updateLogFormat(
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

func (ncp *NginxConfigParser) errorLog(file, level string) *model.ErrorLog {
	errorLog := &model.ErrorLog{
		Name:     file,
		LogLevel: level,
		Readable: false,
	}
	info, err := os.Stat(file)
	if err == nil {
		errorLog.Permissions = files.Permissions(info.Mode())
		errorLog.Readable = true
	}

	return errorLog
}

func (ncp *NginxConfigParser) accessLogDirectiveFormat(directive *crossplane.Directive) string {
	if ncp.hasAdditionArguments(directive.Args) {
		return strings.ReplaceAll(directive.Args[1], "$", "")
	}

	return ""
}

func (ncp *NginxConfigParser) errorLogDirectiveLevel(directive *crossplane.Directive) string {
	if ncp.hasAdditionArguments(directive.Args) {
		return directive.Args[1]
	}

	return ""
}

func (ncp *NginxConfigParser) rootFiles(ctx context.Context, rootDir string) (rootFiles []*mpi.File) {
	if !ncp.agentConfig.IsDirectoryAllowed(rootDir) {
		slog.DebugContext(ctx, "Root directory not in allowed directories", "root_directory", rootDir)
		return rootFiles
	}

	err := filepath.WalkDir(rootDir,
		func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}

			if d.IsDir() {
				return nil
			}

			rootFileMeta, fileMetaErr := files.FileMeta(path)
			if fileMetaErr != nil {
				return fileMetaErr
			}

			rootFiles = append(rootFiles, &mpi.File{FileMeta: rootFileMeta})

			return nil
		},
	)
	if err != nil {
		slog.WarnContext(ctx, "Unable to walk root directory", "root_directory", rootDir)
	}

	return rootFiles
}

func (ncp *NginxConfigParser) sslCert(ctx context.Context, file, rootDir string) (sslCertFile *mpi.File) {
	if strings.Contains(file, "$") {
		// cannot process any filepath with variables
		return nil
	}

	if !filepath.IsAbs(file) {
		file = filepath.Join(rootDir, file)
	}

	if !ncp.agentConfig.IsDirectoryAllowed(file) {
		slog.DebugContext(ctx, "File not in allowed directories", "file", file)
	} else {
		sslCertFileMeta, fileMetaErr := files.FileMeta(file)
		if fileMetaErr != nil {
			slog.ErrorContext(ctx, "Unable to get file metadata", "file", file, "error", fileMetaErr)
		} else {
			sslCertFile = &mpi.File{FileMeta: sslCertFileMeta}
		}
	}

	return sslCertFile
}

func (ncp *NginxConfigParser) crossplaneConfigTraverse(
	ctx context.Context,
	root *crossplane.Config,
	callback crossplaneTraverseCallback,
) error {
	for _, dir := range root.Parsed {
		err := callback(ctx, nil, dir)
		if err != nil {
			return err
		}

		err = ncp.traverse(ctx, dir, callback)
		if err != nil {
			return err
		}
	}

	return nil
}

func (ncp *NginxConfigParser) crossplaneConfigTraverseStr(
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

func (ncp *NginxConfigParser) traverse(
	ctx context.Context,
	root *crossplane.Directive,
	callback crossplaneTraverseCallback,
) error {
	for _, child := range root.Block {
		err := callback(ctx, root, child)
		if err != nil {
			return err
		}

		err = ncp.traverse(ctx, child, callback)
		if err != nil {
			return err
		}
	}

	return nil
}

func (ncp *NginxConfigParser) hasAdditionArguments(args []string) bool {
	return len(args) >= defaultNumberOfDirectiveArguments
}

func (ncp *NginxConfigParser) stubStatusAPICallback(ctx context.Context, parent, current *crossplane.Directive) string {
	urls := ncp.urlsForLocationDirective(parent, current, stubStatusAPIDirective)
	if len(urls) > 0 {
		slog.DebugContext(ctx, "Potential stub_status urls", "urls", urls)
	}

	for _, url := range urls {
		if ncp.pingStubStatusAPIEndpoint(ctx, url) {
			slog.DebugContext(ctx, "Stub_status found", "url", url)
			return url
		}
		slog.DebugContext(ctx, "Stub_status is not reachable", "url", url)
	}

	return ""
}

func (ncp *NginxConfigParser) pingStubStatusAPIEndpoint(ctx context.Context, statusAPI string) bool {
	httpClient := http.Client{Timeout: ncp.agentConfig.Client.Timeout}
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

func (ncp *NginxConfigParser) plusAPICallback(ctx context.Context, parent, current *crossplane.Directive) string {
	urls := ncp.urlsForLocationDirective(parent, current, plusAPIDirective)
	if len(urls) > 0 {
		slog.DebugContext(ctx, "Potential Plus API urls", "urls", urls)
	}

	for _, url := range urls {
		if ncp.pingPlusAPIEndpoint(ctx, url) {
			slog.DebugContext(ctx, "Plus API found", "url", url)
			return url
		}
		slog.DebugContext(ctx, "Plus API is not reachable", "url", url)
	}

	return ""
}

func (ncp *NginxConfigParser) pingPlusAPIEndpoint(ctx context.Context, statusAPI string) bool {
	httpClient := http.Client{Timeout: ncp.agentConfig.Client.Timeout}
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

func (ncp *NginxConfigParser) urlsForLocationDirective(
	parent, current *crossplane.Directive,
	locationDirectiveName string,
) []string {
	var urls []string
	// process from the location block
	if current.Directive != locationDirective {
		return urls
	}

	for _, locChild := range current.Block {
		if locChild.Directive != plusAPIDirective && locChild.Directive != stubStatusAPIDirective {
			continue
		}

		addresses := ncp.parseAddressesFromServerDirective(parent)

		for _, address := range addresses {
			path := ncp.parsePathFromLocationDirective(current)

			if locChild.Directive == locationDirectiveName {
				urls = append(urls, fmt.Sprintf(apiFormat, address, path))
			}
		}
	}

	return urls
}

func (ncp *NginxConfigParser) parsePathFromLocationDirective(location *crossplane.Directive) string {
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

func (ncp *NginxConfigParser) parseAddressesFromServerDirective(parent *crossplane.Directive) []string {
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
				hostname, port = ncp.parseListenHostAndPort(listenHost, listenPort)
			} else {
				hostname, port = ncp.parseListenDirective(dir, "127.0.0.1", port)
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

	return ncp.formatAddresses(foundHosts, port)
}

func (ncp *NginxConfigParser) formatAddresses(foundHosts []string, port string) []string {
	addresses := []string{}
	for _, foundHost := range foundHosts {
		addresses = append(addresses, fmt.Sprintf("%s:%s", foundHost, port))
	}

	return addresses
}

func (ncp *NginxConfigParser) parseListenDirective(
	dir *crossplane.Directive,
	hostname, port string,
) (directiveHost, directivePort string) {
	directiveHost = hostname
	directivePort = port
	if ncp.isPort(dir.Args[0]) {
		directivePort = dir.Args[0]
	} else {
		directiveHost = dir.Args[0]
	}

	return directiveHost, directivePort
}

func (ncp *NginxConfigParser) parseListenHostAndPort(listenHost, listenPort string) (hostname, port string) {
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

func (ncp *NginxConfigParser) isPort(value string) bool {
	port, err := strconv.Atoi(value)

	return err == nil && port >= 1 && port <= 65535
}
