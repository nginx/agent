// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package config

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"

	pkg "github.com/nginx/agent/v3/pkg/config"

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
	unixStubStatusFormat              = "http://config-status%s"
	unixPlusAPIFormat                 = "http://nginx-plus-api%s"
	locationDirective                 = "location"
)

var globFunction = func(path string) ([]string, error) {
	matches, err := filepath.Glob(path)
	if err != nil {
		return nil, err
	}

	// Exclude hidden files unless the glob pattern itself starts with a dot
	if !strings.HasPrefix(filepath.Base(path), ".") {
		filteredMatches := make([]string, 0)

		for _, match := range matches {
			base := filepath.Base(match)
			if !strings.HasPrefix(base, ".") {
				filteredMatches = append(filteredMatches, match)
			}
		}

		return filteredMatches, nil
	}

	return matches, nil
}

//go:generate go run github.com/maxbrunsfeld/counterfeiter/v6@v6.8.1 -generate
//counterfeiter:generate . ConfigParser

type (
	NginxConfigParser struct {
		agentConfig *config.Config
	}
)

type ConfigParser interface {
	Parse(ctx context.Context, instance *mpi.Instance) (*model.NginxConfigContext, error)
	FindStubStatusAPI(ctx context.Context, nginxConfigContext *model.NginxConfigContext) *model.APIDetails
	FindPlusAPI(ctx context.Context, nginxConfigContext *model.NginxConfigContext) *model.APIDetails
}

var _ ConfigParser = (*NginxConfigParser)(nil)

type (
	crossplaneTraverseCallback           = func(ctx context.Context, parent, current *crossplane.Directive) error
	crossplaneTraverseCallbackAPIDetails = func(ctx context.Context, parent,
		current *crossplane.Directive, apiType string) []*model.APIDetails
)

func NewNginxConfigParser(agentConfig *config.Config) *NginxConfigParser {
	return &NginxConfigParser{
		agentConfig: agentConfig,
	}
}

func (ncp *NginxConfigParser) Parse(ctx context.Context, instance *mpi.Instance) (*model.NginxConfigContext, error) {
	configPath, _ := filepath.Abs(instance.GetInstanceRuntime().GetConfigPath())

	if !ncp.agentConfig.IsDirectoryAllowed(configPath) {
		return nil, fmt.Errorf("config path %s is not in allowed directories", configPath)
	}

	slog.DebugContext(
		ctx,
		"Parsing NGINX config",
		"file_path", configPath,
		"instance_id", instance.GetInstanceMeta().GetInstanceId(),
	)

	lua := crossplane.Lua{}
	payload, err := crossplane.Parse(configPath,
		&crossplane.ParseOptions{
			SingleFile:         false,
			StopParsingOnError: true,
			LexOptions: crossplane.LexOptions{
				Lexers: []crossplane.RegisterLexer{lua.RegisterLexer()},
			},
			Glob: globFunction,
		},
	)
	if err != nil {
		return nil, err
	}

	return ncp.createNginxConfigContext(ctx, instance, payload, configPath)
}

func (ncp *NginxConfigParser) FindStubStatusAPI(
	ctx context.Context, nginxConfigContext *model.NginxConfigContext,
) *model.APIDetails {
	for _, stubStatus := range nginxConfigContext.StubStatuses {
		if stubStatus != nil && stubStatus.URL != "" {
			if ncp.pingAPIEndpoint(ctx, stubStatus, stubStatusAPIDirective) {
				slog.InfoContext(ctx, "Found NGINX stub status API", "url", stubStatus.URL)
				return stubStatus
			}
		}
	}

	return &model.APIDetails{
		URL:      "",
		Listen:   "",
		Location: "",
		Ca:       "",
	}
}

func (ncp *NginxConfigParser) FindPlusAPI(
	ctx context.Context, nginxConfigContext *model.NginxConfigContext,
) *model.APIDetails {
	for _, plusAPI := range nginxConfigContext.PlusAPIs {
		if plusAPI != nil && plusAPI.URL != "" {
			if ncp.pingAPIEndpoint(ctx, plusAPI, plusAPIDirective) {
				slog.InfoContext(ctx, "Found NGINX Plus API", "url", plusAPI.URL)
				return plusAPI
			}
		}
	}

	return &model.APIDetails{
		URL:      "",
		Listen:   "",
		Location: "",
		Ca:       "",
	}
}

//nolint:gocognit,gocyclo,revive,cyclop //  cognitive complexity is 51, cyclomatic complexity is 24
func (ncp *NginxConfigParser) createNginxConfigContext(
	ctx context.Context,
	instance *mpi.Instance,
	payload *crossplane.Payload,
	configPath string,
) (*model.NginxConfigContext, error) {
	napEnabled := false

	nginxConfigContext := &model.NginxConfigContext{
		InstanceID: instance.GetInstanceMeta().GetInstanceId(),
		ConfigPath: configPath,
		PlusAPI: &model.APIDetails{
			URL:      "",
			Listen:   "",
			Location: "",
		},
		StubStatus: &model.APIDetails{
			URL:      "",
			Listen:   "",
			Location: "",
		},
		NAPSysLogServer: "",
	}

	rootDir := filepath.Dir(instance.GetInstanceRuntime().GetConfigPath())

	for _, conf := range payload.Config {
		slog.DebugContext(ctx, "Traversing NGINX config file", "config", conf)
		if !ncp.agentConfig.IsDirectoryAllowed(conf.File) {
			slog.WarnContext(ctx, "File included in NGINX config is outside of allowed directories, "+
				"excluding from config",
				"file", conf.File)

			continue
		}

		formatMap := make(map[string]string)
		err := ncp.crossplaneConfigTraverse(ctx, &conf,
			func(ctx context.Context, parent, directive *crossplane.Directive) error {
				switch directive.Directive {
				case "include":
					include := ncp.parseIncludeDirective(directive, &conf)

					nginxConfigContext.Includes = append(nginxConfigContext.Includes, include)
				case "log_format":
					formatMap = ncp.formatMap(directive)
				case "access_log":
					if !ncp.ignoreLog(directive.Args[0]) {
						accessLog := ncp.accessLog(directive.Args[0], ncp.accessLogDirectiveFormat(directive),
							formatMap)
						nginxConfigContext.AccessLogs = ncp.addAccessLog(accessLog, nginxConfigContext.AccessLogs)
					}
				case "error_log":
					if !ncp.ignoreLog(directive.Args[0]) {
						errorLog := ncp.errorLog(directive.Args[0], ncp.errorLogDirectiveLevel(directive))
						nginxConfigContext.ErrorLogs = append(nginxConfigContext.ErrorLogs, errorLog)
					} else {
						slog.WarnContext(ctx, fmt.Sprintf("Currently error log outputs to %s. Log monitoring "+
							"is disabled while applying a config; "+"log errors to file to enable error monitoring",
							directive.Args[0]), "error_log", directive.Args[0])
					}
				case "ssl_certificate", "proxy_ssl_certificate", "ssl_client_certificate",
					"ssl_trusted_certificate":
					if ncp.agentConfig.IsFeatureEnabled(pkg.FeatureCertificates) {
						sslCertFile := ncp.sslCert(ctx, directive.Args[0], rootDir)
						if sslCertFile != nil && !ncp.isDuplicateFile(nginxConfigContext.Files, sslCertFile) {
							slog.DebugContext(ctx, "Adding SSL certificate file", "ssl_cert", sslCertFile)
							nginxConfigContext.Files = append(nginxConfigContext.Files, sslCertFile)
						}
					} else {
						slog.DebugContext(ctx, "Certificate feature is disabled, skipping cert",
							"enabled_features", ncp.agentConfig.Features)
					}
				case "app_protect_security_log":
					if len(directive.Args) > 1 {
						napEnabled = true
						sysLogServer := ncp.findLocalSysLogServers(directive.Args[1])
						if sysLogServer != "" {
							nginxConfigContext.NAPSysLogServer = sysLogServer
							slog.DebugContext(ctx, "Found NAP syslog server", "address", sysLogServer)
						}
					}
				}

				return nil
			},
		)
		if err != nil {
			return nginxConfigContext, fmt.Errorf("traverse nginx config: %w", err)
		}

		stubStatuses := ncp.crossplaneConfigTraverseAPIDetails(
			ctx, &conf, ncp.apiCallback, stubStatusAPIDirective,
		)
		if stubStatuses != nil {
			nginxConfigContext.StubStatuses = append(nginxConfigContext.StubStatuses, stubStatuses...)
		}

		plusAPIs := ncp.crossplaneConfigTraverseAPIDetails(
			ctx, &conf, ncp.apiCallback, plusAPIDirective,
		)
		if plusAPIs != nil {
			nginxConfigContext.PlusAPIs = append(nginxConfigContext.PlusAPIs, plusAPIs...)
		}

		fileMeta, err := files.FileMeta(conf.File)
		if err != nil {
			slog.WarnContext(ctx, "Unable to get file metadata", "file_name", conf.File, "error", err)
		} else {
			nginxConfigContext.Files = append(nginxConfigContext.Files, &mpi.File{FileMeta: fileMeta})
		}
	}

	if napEnabled && nginxConfigContext.NAPSysLogServer == "" {
		slog.WarnContext(ctx, fmt.Sprintf("Could not find available local NGINX App Protect syslog"+
			" server configured on port %s. Security violations will not be collected.",
			ncp.agentConfig.SyslogServer.Port))
	} else if napEnabled && nginxConfigContext.NAPSysLogServer != "" {
		slog.InfoContext(ctx, fmt.Sprintf("Found available local NGINX App Protect syslog"+
			"server configured on port %s", ncp.agentConfig.SyslogServer.Port))
	}

	nginxConfigContext.StubStatus = ncp.FindStubStatusAPI(ctx, nginxConfigContext)
	nginxConfigContext.PlusAPI = ncp.FindPlusAPI(ctx, nginxConfigContext)

	return nginxConfigContext, nil
}

func (ncp *NginxConfigParser) findLocalSysLogServers(sysLogServer string) string {
	re := regexp.MustCompile(`syslog:server=([\S]+)`)
	matches := re.FindStringSubmatch(sysLogServer)
	if len(matches) > 1 {
		host, port, err := net.SplitHostPort(matches[1])
		if err != nil {
			return ""
		}

		if port != ncp.agentConfig.SyslogServer.Port {
			return ""
		}

		ip := net.ParseIP(host)
		if ip.IsLoopback() || strings.EqualFold(host, "localhost") {
			return matches[1]
		}
	}

	return ""
}

func (ncp *NginxConfigParser) parseIncludeDirective(
	directive *crossplane.Directive,
	configFile *crossplane.Config,
) string {
	var include string
	if filepath.IsAbs(directive.Args[0]) {
		include = directive.Args[0]
	} else {
		include = filepath.Join(filepath.Dir(configFile.File), directive.Args[0])
	}

	return include
}

func (ncp *NginxConfigParser) addAccessLog(accessLog *model.AccessLog,
	accessLogs []*model.AccessLog,
) []*model.AccessLog {
	for i, log := range accessLogs {
		if accessLog.Name == log.Name {
			if accessLog.Format != log.Format {
				slog.Warn("Found multiple log_format directives for the same access log. Multiple log formats "+
					"are not supported in the same access log, metrics from this access log "+
					"will not be collected", "access_log", accessLog.Name)

				return append(accessLogs[:i], accessLogs[i+1:]...)
			}
			slog.Debug("Found duplicate access log, skipping", "access_log", accessLog.Name)

			return accessLogs
		}
	}

	slog.Debug("Found valid access log", "access_log", accessLog.Name)

	return append(accessLogs, accessLog)
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

func (ncp *NginxConfigParser) crossplaneConfigTraverseAPIDetails(
	ctx context.Context,
	root *crossplane.Config,
	callback crossplaneTraverseCallbackAPIDetails,
	apiType string,
) []*model.APIDetails {
	stop := false
	var responses []*model.APIDetails

	for _, dir := range root.Parsed {
		response := callback(ctx, nil, dir, apiType)
		if response != nil {
			responses = append(responses, response...)
			continue
		}
		response = traverseAPIDetails(ctx, dir, callback, &stop, apiType)
		if response != nil {
			responses = append(responses, response...)
		}
	}

	return responses
}

func traverseAPIDetails(
	ctx context.Context,
	root *crossplane.Directive,
	callback crossplaneTraverseCallbackAPIDetails,
	stop *bool,
	apiType string,
) (response []*model.APIDetails) {
	if *stop {
		return nil
	}

	for _, child := range root.Block {
		response = callback(ctx, root, child, apiType)
		if len(response) > 0 {
			*stop = true
			return response
		}
		response = traverseAPIDetails(ctx, child, callback, stop, apiType)
		if *stop {
			return response
		}
	}

	return response
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

func (ncp *NginxConfigParser) hasAdditionArguments(args []string) bool {
	return len(args) >= defaultNumberOfDirectiveArguments
}

func (ncp *NginxConfigParser) ignoreLog(logPath string) bool {
	ignoreLogs := []string{"off", "/dev/stderr", "/dev/stdout", "/dev/null", "stderr", "stdout"}

	if strings.HasPrefix(logPath, "syslog:") || slices.Contains(ignoreLogs, logPath) {
		return true
	}

	if ncp.isExcludeLog(logPath) {
		return true
	}

	if !ncp.agentConfig.IsDirectoryAllowed(logPath) {
		slog.Warn("Log being read is outside of allowed directories", "log_path", logPath)
	}

	return false
}

func (ncp *NginxConfigParser) isExcludeLog(path string) bool {
	for _, pattern := range ncp.agentConfig.DataPlaneConfig.Nginx.ExcludeLogs {
		_, compileErr := regexp.Compile(pattern)
		if compileErr != nil {
			slog.Error("Invalid path for excluding log", "log_path", pattern)
			continue
		}

		ok, err := regexp.MatchString(pattern, path)
		if err != nil {
			slog.Error("Invalid path for excluding log", "file_path", pattern)
			continue
		} else if ok {
			slog.Info("Excluding log as specified in config", "log_path", path)

			return true
		}
	}

	return false
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

func (ncp *NginxConfigParser) sslCert(ctx context.Context, file, rootDir string) (sslCertFile *mpi.File) {
	if strings.Contains(file, "$") {
		slog.DebugContext(ctx, "Cannot process SSL certificate file path with variables", "file", file)
		return nil
	}

	if !filepath.IsAbs(file) {
		file = filepath.Join(rootDir, file)
	}

	if !ncp.agentConfig.IsDirectoryAllowed(file) {
		slog.DebugContext(ctx, "File not in allowed directories", "file", file)
	} else {
		sslCertFileMeta, fileMetaErr := files.FileMetaWithCertificate(file)
		if fileMetaErr != nil {
			slog.ErrorContext(ctx, "Unable to get file metadata", "file", file, "error", fileMetaErr)
		} else {
			sslCertFile = &mpi.File{FileMeta: sslCertFileMeta}
		}
	}

	return sslCertFile
}

func (ncp *NginxConfigParser) apiCallback(
	ctx context.Context, parent, current *crossplane.Directive, apiType string,
) (details []*model.APIDetails) {
	details = append(details, ncp.apiDetailsFromLocationDirective(ctx, parent, current, apiType)...)
	if len(details) > 0 {
		slog.DebugContext(ctx, "Found "+apiType, "api_details", details)
	}

	return details
}

func (ncp *NginxConfigParser) pingAPIEndpoint(ctx context.Context, statusAPIDetail *model.APIDetails,
	apiType string,
) bool {
	httpClient, clientError := ncp.prepareHTTPClient(ctx)
	if clientError != nil {
		slog.ErrorContext(ctx, "Failed to prepare HTTP client", "error", clientError)
		return false
	}
	listen := statusAPIDetail.Listen
	statusAPI := statusAPIDetail.URL

	if strings.HasPrefix(listen, "unix:") {
		httpClient = ncp.socketClient(strings.TrimPrefix(listen, "unix:"))
	} else {
		httpClient.Timeout = ncp.agentConfig.Client.HTTP.Timeout
	}
	req, requestError := http.NewRequestWithContext(ctx, http.MethodGet, statusAPI, nil)
	if requestError != nil {
		slog.WarnContext(
			ctx, fmt.Sprintf("Unable to create %s API GET request", apiType),
			"error", requestError,
		)

		return false
	}

	slog.DebugContext(ctx, "Calling "+apiType+" API endpoint", "url", req.URL.String())

	resp, reqErr := httpClient.Do(req)
	if reqErr != nil {
		slog.WarnContext(ctx, fmt.Sprintf("Unable to ping %s API", apiType), "error", reqErr)
		return false
	}
	defer func() {
		closeErr := resp.Body.Close()
		if closeErr != nil {
			slog.WarnContext(
				ctx, fmt.Sprintf("Unable to close body from %s API response", apiType),
				"error", closeErr,
			)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		err := errors.New(apiType + " API responded with unexpected status code " + strconv.Itoa(resp.StatusCode))
		slog.WarnContext(ctx, fmt.Sprintf("Unable to ping %s API", apiType), "error", err)

		return false
	}

	bodyBytes, readErr := io.ReadAll(resp.Body)
	if readErr != nil {
		slog.WarnContext(ctx, fmt.Sprintf("Unable to ping %s API", apiType), "error", readErr)
		return false
	}

	validationError := validateAPIResponse(apiType, bodyBytes)
	if validationError != nil {
		slog.WarnContext(ctx, fmt.Sprintf("Unable to ping %s API", apiType), "error", validationError)
		return false
	}

	return true
}

func validateAPIResponse(apiType string, bodyBytes []byte) error {
	if apiType == stubStatusAPIDirective {
		// Expecting API to return data like this:
		//
		// Active connections: 2
		// server accepts handled requests
		//  18 18 3266
		// Reading: 0 Writing: 1 Waiting: 1

		body := string(bodyBytes)

		if !strings.Contains(body, "Active connections") &&
			!strings.Contains(body, "server accepts handled requests") {
			return errors.New("Unable to GET " + apiType + " API responded with unexpected response body")
		}
	} else {
		// Expecting API to return the API versions in an array of positive integers
		// subset example: [ ... 6,7,8,9 ...]

		var responseBody []int
		err := json.Unmarshal(bodyBytes, &responseBody)
		if err != nil {
			return errors.Join(errors.New("unable to unmarshal NGINX Plus API response body"), err)
		}
	}

	return nil
}

func (ncp *NginxConfigParser) apiDetailsFromLocationDirective(
	ctx context.Context, parent, current *crossplane.Directive,
	locationDirectiveName string,
) (details []*model.APIDetails) {
	// Check if SSL is enabled in the server block
	isSSL := ncp.isSSLEnabled(parent)

	// If SSl is enabled, check if CA cert is provided and the location is allowed
	var caCertLocation string
	if isSSL {
		caCertLocation = ncp.selfSignedCACertLocation(ctx)
	}

	if current.Directive != locationDirective {
		return nil
	}

	for _, locChild := range current.Block {
		if locChild.Directive != plusAPIDirective && locChild.Directive != stubStatusAPIDirective {
			continue
		}

		addresses := ncp.parseAddressFromServerDirective(parent)
		path := ncp.parsePathFromLocationDirective(current)

		if locChild.Directive == locationDirectiveName {
			for _, address := range addresses {
				details = append(
					details,
					ncp.createAPIDetails(locationDirectiveName, address, path, caCertLocation, isSSL),
				)
			}
		}
	}

	return details
}

func (ncp *NginxConfigParser) createAPIDetails(
	locationDirectiveName, address, path, caCertLocation string, isSSL bool,
) (details *model.APIDetails) {
	if strings.HasPrefix(address, "unix:") {
		format := unixStubStatusFormat

		if locationDirectiveName == plusAPIDirective {
			format = unixPlusAPIFormat
		}

		details = &model.APIDetails{
			URL:      fmt.Sprintf(format, path),
			Listen:   address,
			Location: path,
			Ca:       caCertLocation,
		}
	} else {
		details = &model.APIDetails{
			URL: fmt.Sprintf("%s://%s%s", map[bool]string{true: "https", false: "http"}[isSSL],
				address, path),
			Listen:   address,
			Location: path,
			Ca:       caCertLocation,
		}
	}

	return details
}

func (ncp *NginxConfigParser) parseAddressFromServerDirective(parent *crossplane.Directive) (addresses []string) {
	port := "80"
	hosts := []string{"localhost", "127.0.0.1"}

	if parent == nil {
		return addresses
	}

	for _, dir := range parent.Block {
		if dir.Directive == "listen" {
			for _, host := range hosts {
				port, host = ncp.parseListenDirectiveAddress(dir, port, host)
				addresses = append(addresses, host+":"+port)
			}
		}
	}

	return addresses
}

func (ncp *NginxConfigParser) parseListenDirectiveAddress(
	dir *crossplane.Directive, port, host string,
) (updatedPort, updatedHost string) {
	listenHost, listenPort, err := net.SplitHostPort(dir.Args[0])
	if err == nil {
		port = listenPort
		if listenHost == "unix" {
			host = listenHost
		}
	} else if ncp.isPort(dir.Args[0]) {
		port = dir.Args[0]
	}

	return port, host
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

func (ncp *NginxConfigParser) isPort(value string) bool {
	port, err := strconv.Atoi(value)

	return err == nil && port >= 1 && port <= 65535
}

// checks if any of the arguments contain "ssl".
func (ncp *NginxConfigParser) hasSSLArgument(args []string) bool {
	for i := 1; i < len(args); i++ {
		if args[i] == "ssl" {
			return true
		}
	}

	return false
}

// checks if a directive is a listen directive with ssl enabled.
func (ncp *NginxConfigParser) isSSLListenDirective(dir *crossplane.Directive) bool {
	return dir.Directive == "listen" && ncp.hasSSLArgument(dir.Args)
}

// checks if SSL is enabled for a given server block.
func (ncp *NginxConfigParser) isSSLEnabled(serverBlock *crossplane.Directive) bool {
	if serverBlock == nil {
		return false
	}

	for _, dir := range serverBlock.Block {
		if ncp.isSSLListenDirective(dir) {
			return true
		}
	}

	return false
}

func (ncp *NginxConfigParser) socketClient(socketPath string) *http.Client {
	return &http.Client{
		Timeout: ncp.agentConfig.Client.Grpc.KeepAlive.Timeout,
		Transport: &http.Transport{
			DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
				dialer := &net.Dialer{}
				return dialer.DialContext(ctx, "unix", socketPath)
			},
		},
	}
}

// prepareHTTPClient handles TLS config
func (ncp *NginxConfigParser) prepareHTTPClient(ctx context.Context) (*http.Client, error) {
	httpClient := http.DefaultClient
	caCertLocation := ncp.agentConfig.DataPlaneConfig.Nginx.APITls.Ca

	if caCertLocation != "" && ncp.agentConfig.IsDirectoryAllowed(caCertLocation) {
		slog.DebugContext(ctx, "Reading CA certificate", "file_path", caCertLocation)
		caCert, err := os.ReadFile(caCertLocation)
		if err != nil {
			return nil, err
		}
		caCertPool := x509.NewCertPool()
		caCertPool.AppendCertsFromPEM(caCert)

		httpClient = &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					RootCAs:    caCertPool,
					MinVersion: tls.VersionTLS13,
				},
			},
		}
	}

	return httpClient, nil
}

// Populate the CA cert location based ondirectory allowance.
func (ncp *NginxConfigParser) selfSignedCACertLocation(ctx context.Context) string {
	caCertLocation := ncp.agentConfig.DataPlaneConfig.Nginx.APITls.Ca

	if caCertLocation != "" && !ncp.agentConfig.IsDirectoryAllowed(caCertLocation) {
		// If SSL is enabled but CA cert is provided and not allowed, treat it as if no CA cert
		slog.WarnContext(ctx, "CA certificate location is not allowed, treating as if no CA cert provided.")
		return ""
	}

	return caCertLocation
}

func (ncp *NginxConfigParser) isDuplicateFile(nginxConfigContextFiles []*mpi.File, newFile *mpi.File) bool {
	for _, nginxConfigContextFile := range nginxConfigContextFiles {
		if nginxConfigContextFile.GetFileMeta().GetName() == newFile.GetFileMeta().GetName() {
			return true
		}
	}

	return false
}
