// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package config

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	uuidLibrary "github.com/nginx/agent/v3/pkg/id"
	selfsignedcerts "github.com/nginx/agent/v3/pkg/tls"
	"github.com/spf13/cobra"
	flag "github.com/spf13/pflag"
	"github.com/spf13/viper"
)

const (
	ConfigFileName = "nginx-agent.conf"
	EnvPrefix      = "NGINX_AGENT"
	KeyDelimiter   = "_"
	KeyValueNumber = 2
	AgentDirName   = "/etc/nginx-agent/"
)

var viperInstance = viper.NewWithOptions(viper.KeyDelimiter(KeyDelimiter))

func RegisterRunner(r func(cmd *cobra.Command, args []string)) {
	RootCommand.Run = r
}

func Execute(ctx context.Context) error {
	RootCommand.AddCommand(CompletionCommand)
	return RootCommand.ExecuteContext(ctx)
}

func Init(version, commit string) {
	setVersion(version, commit)
	registerFlags()
}

func RegisterConfigFile() error {
	configPath, err := seekFileInPaths(ConfigFileName, getConfigFilePaths()...)
	if err != nil {
		return err
	}

	if err = loadPropertiesFromFile(configPath); err != nil {
		return err
	}

	slog.Debug("Configuration file loaded", "config_path", configPath)
	viperInstance.Set(ConfigPathKey, configPath)

	exePath, err := os.Executable()
	if err != nil {
		return err
	}

	viperInstance.Set(UUIDKey, uuidLibrary.Generate(exePath, configPath))

	return nil
}

func ResolveConfig() (*Config, error) {
	// Collect allowed directories, so that paths in the config can be validated.
	directories := viperInstance.GetStringSlice(AllowedDirectoriesKey)
	allowedDirs := []string{AgentDirName}

	// Check directories in allowed_directories are valid
	for _, dir := range directories {
		if dir == "" || !filepath.IsAbs(dir) {
			slog.Warn("Invalid directory: ", "dir", dir)
			continue
		}

		if !strings.HasSuffix(dir, "/") {
			dir += "/"
		}
		allowedDirs = append(allowedDirs, dir)
		slog.Info("Configured allowed directories", "allowed_directories", allowedDirs)
	}

	// Collect all parsing errors before returning the error, so the user sees all issues with config
	// in one error message.
	var err error
	collector, otelcolErr := resolveCollector(allowedDirs)
	err = errors.Join(err, otelcolErr)
	if err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	config := &Config{
		UUID:               viperInstance.GetString(UUIDKey),
		Version:            viperInstance.GetString(VersionKey),
		Path:               viperInstance.GetString(ConfigPathKey),
		Log:                resolveLog(),
		DataPlaneConfig:    resolveDataPlaneConfig(),
		Client:             resolveClient(),
		AllowedDirectories: allowedDirs,
		Collector:          collector,
		Command:            resolveCommand(),
		Watchers:           resolveWatchers(),
		Features:           viperInstance.GetStringSlice(FeaturesKey),
		Labels:             resolveLabels(),
	}

	slog.Debug("Agent config", "config", config)
	slog.Info("Enabled features", "features", config.Features)
	slog.Info("Excluded files from being watched for file changes", "exclude_files",
		config.Watchers.FileWatcher.ExcludeFiles)

	return config, nil
}

func setVersion(version, commit string) {
	RootCommand.Version = version + "-" + commit
	viperInstance.SetDefault(VersionKey, version)
}

func registerFlags() {
	viperInstance.SetEnvPrefix(EnvPrefix)
	viperInstance.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	viperInstance.AutomaticEnv()

	fs := RootCommand.Flags()

	fs.String(
		LogLevelKey,
		"info",
		"The desired verbosity level for logging messages from nginx-agent. "+
			"Available options, in order of severity from highest to lowest, are: "+
			"panic, fatal, error, info and debug.",
	)
	fs.String(
		LogPathKey,
		"",
		"The path to output log messages to. "+
			"If the default path doesn't exist, log messages are output to stdout/stderr.",
	)

	fs.Duration(
		NginxReloadMonitoringPeriodKey,
		DefNginxReloadMonitoringPeriod,
		"The amount of time to monitor NGINX after a reload of configuration.",
	)
	fs.Bool(
		NginxTreatWarningsAsErrorsKey,
		DefTreatErrorsAsWarnings,
		"Warning messages in the NGINX errors logs after a NGINX reload will be treated as an error.",
	)

	fs.StringSlice(
		NginxExcludeLogsKey, []string{},
		"A comma-separated list of one or more NGINX log paths that you want to exclude from metrics "+
			"collection or error monitoring. This includes absolute paths or regex patterns",
	)

	fs.StringSlice(AllowedDirectoriesKey,
		DefaultAllowedDirectories(),
		"A comma-separated list of paths that you want to grant NGINX Agent read/write access to. Allowed "+
			"directories are case sensitive")

	fs.Duration(
		InstanceWatcherMonitoringFrequencyKey,
		DefInstanceWatcherMonitoringFrequency,
		"How often the NGINX Agent will check for instance changes.",
	)

	fs.Duration(
		InstanceHealthWatcherMonitoringFrequencyKey,
		DefInstanceHealthWatcherMonitoringFrequency,
		"How often the NGINX Agent will check for instance health changes.",
	)

	fs.Duration(
		FileWatcherMonitoringFrequencyKey,
		DefFileWatcherMonitoringFrequency,
		"How often the NGINX Agent will check for file changes.",
	)

	fs.StringSlice(
		NginxExcludeFilesKey, DefaultExcludedFiles(),
		"A comma-separated list of one or more file paths that you want to exclude from file monitoring. "+
			"This includes absolute paths or regex patterns",
	)

	fs.StringSlice(
		FeaturesKey,
		DefaultFeatures(),
		"A comma-separated list of features enabled for the agent.",
	)

	registerCommonFlags(fs)
	registerCommandFlags(fs)
	registerCollectorFlags(fs)
	registerClientFlags(fs)

	fs.SetNormalizeFunc(normalizeFunc)

	fs.VisitAll(func(flag *flag.Flag) {
		if err := viperInstance.BindPFlag(strings.ReplaceAll(flag.Name, "-", "_"), fs.Lookup(flag.Name)); err != nil {
			return
		}
		err := viperInstance.BindEnv(flag.Name)
		if err != nil {
			slog.Warn("Error occurred binding env", "env", flag.Name, "error", err)
		}
	})
}

func registerCommonFlags(fs *flag.FlagSet) {
	fs.StringToString(
		LabelsRootKey,
		DefaultLabels(),
		"A list of labels associated with these instances",
	)
}

func registerClientFlags(fs *flag.FlagSet) {
	// HTTP Flags
	fs.Duration(
		ClientHTTPTimeoutKey,
		DefHTTPTimeout,
		"The client HTTP Timeout, value in seconds")

	// Backoff Flags
	fs.Duration(
		ClientBackoffInitialIntervalKey,
		DefBackoffInitialInterval,
		"The client backoff initial interval, value in seconds")

	fs.Duration(
		ClientBackoffMaxIntervalKey,
		DefBackoffMaxInterval,
		"The client backoff max interval, value in seconds")

	fs.Duration(
		ClientBackoffMaxElapsedTimeKey,
		DefBackoffMaxElapsedTime,
		"The client backoff max elapsed time, value in seconds")

	fs.Float64(
		ClientBackoffRandomizationFactorKey,
		DefBackoffRandomizationFactor,
		"The client backoff randomization factor, value float")

	fs.Float64(
		ClientBackoffMultiplierKey,
		DefBackoffMultiplier,
		"The client backoff multiplier, value float")

	// GRPC Flags
	fs.Duration(
		ClientKeepAliveTimeoutKey,
		DefGRPCKeepAliveTimeout,
		"Updates the client grpc setting, KeepAlive Timeout with the specific value in seconds.",
	)

	fs.Duration(
		ClientKeepAliveTimeKey,
		DefGRPCKeepAliveTime,
		"Updates the client grpc setting, KeepAlive Time with the specific value in seconds.",
	)

	fs.Bool(
		ClientKeepAlivePermitWithoutStreamKey,
		DefGRPCKeepAlivePermitWithoutStream,
		"Update the client grpc setting, KeepAlive PermitWithoutStream value")

	fs.Int(
		ClientGRPCMaxMessageSizeKey,
		DefMaxMessageSize,
		"The value used, if not 0, for both max_message_send_size and max_message_receive_size",
	)

	fs.Int(
		ClientGRPCMaxMessageReceiveSizeKey,
		DefMaxMessageRecieveSize,
		"Updates the client grpc setting MaxRecvMsgSize with the specific value in MB.",
	)

	fs.Int(
		ClientGRPCMaxMessageSendSizeKey,
		DefMaxMessageSendSize,
		"Updates the client grpc setting MaxSendMsgSize with the specific value in MB.",
	)
}

func registerCommandFlags(fs *flag.FlagSet) {
	fs.String(
		CommandServerHostKey,
		DefCommandServerHostKey,
		"The target hostname of the command server endpoint for command and control.",
	)
	fs.Int32(
		CommandServerPortKey,
		DefCommandServerPortKey,
		"The target port of the command server endpoint for command and control.",
	)
	fs.String(
		CommandServerTypeKey,
		DefCommandServerTypeKey,
		"The target protocol (gRPC or HTTP) the command server endpoint for command and control.",
	)
	fs.String(
		CommandAuthTokenKey,
		DefCommandAuthTokenKey,
		"The token used in the authentication handshake with the command server endpoint for command and control.",
	)
	fs.String(
		CommandAuthTokenPathKey,
		DefCommandAuthTokenPathKey,
		"The path to the file containing the token used in the authentication handshake with "+
			"the command server endpoint for command and control.",
	)
	fs.String(
		CommandTLSCertKey,
		DefCommandTLSCertKey,
		"The path to the certificate file to use for TLS communication with the command server.",
	)
	fs.String(
		CommandTLSKeyKey,
		DefCommandTLSKeyKey,
		"The path to the certificate key file to use for TLS communication with the command server.",
	)
	fs.String(
		CommandTLSCaKey,
		DefCommandTLSCaKey,
		"The path to CA certificate file to use for TLS communication with the command server.",
	)
	fs.Bool(
		CommandTLSSkipVerifyKey,
		DefCommandTLSSkipVerifyKey,
		"Testing only. Skip verify controls client verification of a server's certificate chain and host name.",
	)
	fs.String(
		CommandTLSServerNameKey,
		DefCommandTLServerNameKey,
		"Specifies the name of the server sent in the TLS configuration.",
	)
}

func registerCollectorFlags(fs *flag.FlagSet) {
	fs.String(
		CollectorConfigPathKey,
		DefCollectorConfigPath,
		"The path to the Opentelemetry Collector configuration file.",
	)

	fs.String(
		CollectorLogLevelKey,
		DefCollectorLogLevel,
		"The desired verbosity level for logging messages from nginx-agent OTel collector. "+
			"Available options, in order of severity from highest to lowest, are: "+
			"ERROR, WARN, INFO and DEBUG.",
	)

	fs.String(
		CollectorLogPathKey,
		DefCollectorLogPath,
		"The path to output OTel collector log messages to. "+
			"If the default path doesn't exist, log messages are output to stdout/stderr.",
	)

	fs.Uint32(
		CollectorBatchProcessorSendBatchSizeKey,
		DefCollectorBatchProcessorSendBatchSize,
		`Number of metric data points after which a batch will be sent regardless of the timeout.`,
	)

	fs.Uint32(
		CollectorBatchProcessorSendBatchMaxSizeKey,
		DefCollectorBatchProcessorSendBatchMaxSize,
		`The upper limit of the batch size.`,
	)

	fs.Duration(
		CollectorBatchProcessorTimeoutKey,
		DefCollectorBatchProcessorTimeout,
		`Time duration after which a batch will be sent regardless of size.`,
	)

	fs.String(
		CollectorExtensionsHealthServerHostKey,
		DefCollectorExtensionsHealthServerHost,
		`The hostname of the address to publish the OTel collector health check status.`,
	)

	fs.Int32(
		CollectorExtensionsHealthServerPortKey,
		DefCollectorExtensionsHealthServerPort,
		`The port of the address to publish the OTel collector health check status.`,
	)

	fs.String(
		CollectorExtensionsHealthPathKey,
		DefCollectorExtensionsHealthPath,
		`The path to be configured for the OTel collector health check server`,
	)

	fs.String(
		CollectorExtensionsHealthTLSCertKey,
		DefCollectorExtensionsHealthTLSCertPath,
		"The path to the certificate file to use for TLS communication with the OTel collector health check server.",
	)
	fs.String(
		CollectorExtensionsHealthTLSKeyKey,
		DefCollectorExtensionsHealthTLSKeyPath,
		"The path to the certificate key file to use for TLS communication "+
			"with the OTel collector health check server.",
	)
	fs.String(
		CollectorExtensionsHealthTLSCaKey,
		DefCollectorExtensionsHealthTLSCAPath,
		"The path to CA certificate file to use for TLS communication with the OTel collector health check server.",
	)
	fs.Bool(
		CollectorExtensionsHealthTLSSkipVerifyKey,
		DefCollectorExtensionsHealthTLSSkipVerify,
		"Testing only. Skip verify controls client verification of a server's certificate chain and host name.",
	)
	fs.String(
		CollectorExtensionsHealthTLSServerNameKey,
		DefCollectorExtensionsHealthTLServerNameKey,
		"Specifies the name of the server sent in the TLS configuration.",
	)
}

func seekFileInPaths(fileName string, directories ...string) (string, error) {
	for _, directory := range directories {
		f := filepath.Join(directory, fileName)
		if _, err := os.Stat(f); err == nil {
			return f, nil
		}
	}

	return "", fmt.Errorf("a valid configuration has not been found in any of the search paths")
}

func getConfigFilePaths() []string {
	paths := []string{
		"/etc/nginx-agent/",
	}

	path, err := os.Getwd()
	if err == nil {
		paths = append(paths, path)
	} else {
		slog.Warn("Unable to determine process's current directory", "error", err)
	}

	return paths
}

func loadPropertiesFromFile(cfg string) error {
	viperInstance.SetConfigFile(cfg)
	viperInstance.SetConfigType("yaml")
	err := viperInstance.MergeInConfig()
	if err != nil {
		return fmt.Errorf("error loading config file %s: %w", cfg, err)
	}

	return nil
}

func normalizeFunc(f *flag.FlagSet, name string) flag.NormalizedName {
	from := []string{"_", "."}
	to := "-"
	for _, sep := range from {
		name = strings.ReplaceAll(name, sep, to)
	}

	return flag.NormalizedName(name)
}

func resolveLog() *Log {
	return &Log{
		Level: viperInstance.GetString(LogLevelKey),
		Path:  viperInstance.GetString(LogPathKey),
	}
}

func resolveLabels() map[string]interface{} {
	input := viperInstance.GetStringMapString(LabelsRootKey)

	// Parsing the environment variable for labels needs to be done differently
	// by parsing the environment variable as a string.
	envLabels := resolveEnvironmentVariableLabels()

	if len(envLabels) > 0 {
		input = envLabels
	}

	result := make(map[string]interface{})

	for key, value := range input {
		trimmedKey := strings.TrimSpace(key)
		trimmedValue := strings.TrimSpace(value)

		switch {
		case trimmedValue == "" || trimmedValue == "nil": // Handle empty values as nil
			result[trimmedKey] = nil

		case parseInt(trimmedValue) != nil: // Integer
			result[trimmedKey] = parseInt(trimmedValue)

		case parseFloat(trimmedValue) != nil: // Float
			result[trimmedKey] = parseFloat(trimmedValue)

		case parseBool(trimmedValue) != nil: // Boolean
			result[trimmedKey] = parseBool(trimmedValue)

		case parseJSON(trimmedValue) != nil: // JSON object/array
			result[trimmedKey] = parseJSON(trimmedValue)

		default: // String
			result[trimmedKey] = trimmedValue
		}
	}

	slog.Info("Configured labels", "labels", result)

	return result
}

func resolveEnvironmentVariableLabels() map[string]string {
	envLabels := make(map[string]string)
	envInput := viperInstance.GetString(LabelsRootKey)

	labels := strings.Split(envInput, ",")
	if len(labels) > 0 && labels[0] != "" {
		for _, label := range labels {
			splitLabel := strings.Split(label, "=")
			if len(splitLabel) == KeyValueNumber {
				envLabels[splitLabel[0]] = splitLabel[1]
			} else {
				slog.Warn("Unable to parse label ", "label", label)
			}
		}
	}

	return envLabels
}

// Parsing helper functions return the parsed value or nil if parsing fails
func parseInt(value string) interface{} {
	if intValue, err := strconv.Atoi(value); err == nil {
		return intValue
	}

	return nil
}

func parseFloat(value string) interface{} {
	if floatValue, err := strconv.ParseFloat(value, 64); err == nil {
		return floatValue
	}

	return nil
}

func parseBool(value string) interface{} {
	if boolValue, err := strconv.ParseBool(value); err == nil {
		return boolValue
	}

	return nil
}

func parseJSON(value string) interface{} {
	var jsonValue interface{}
	if err := json.Unmarshal([]byte(value), &jsonValue); err == nil {
		return jsonValue
	}

	return nil
}

func resolveDataPlaneConfig() *DataPlaneConfig {
	return &DataPlaneConfig{
		Nginx: &NginxDataPlaneConfig{
			ReloadMonitoringPeriod: viperInstance.GetDuration(NginxReloadMonitoringPeriodKey),
			TreatWarningsAsErrors:  viperInstance.GetBool(NginxTreatWarningsAsErrorsKey),
			ExcludeLogs:            viperInstance.GetStringSlice(NginxExcludeLogsKey),
		},
	}
}

func resolveClient() *Client {
	return &Client{
		HTTP: &HTTP{
			Timeout: viperInstance.GetDuration(ClientHTTPTimeoutKey),
		},
		Grpc: &GRPC{
			KeepAlive: &KeepAlive{
				Timeout:             viperInstance.GetDuration(ClientKeepAliveTimeoutKey),
				Time:                viperInstance.GetDuration(ClientKeepAliveTimeKey),
				PermitWithoutStream: viperInstance.GetBool(ClientKeepAlivePermitWithoutStreamKey),
			},
			MaxMessageSize:        viperInstance.GetInt(ClientGRPCMaxMessageSizeKey),
			MaxMessageReceiveSize: viperInstance.GetInt(ClientGRPCMaxMessageReceiveSizeKey),
			MaxMessageSendSize:    viperInstance.GetInt(ClientGRPCMaxMessageSendSizeKey),
		},
		Backoff: &BackOff{
			InitialInterval:     viperInstance.GetDuration(ClientBackoffInitialIntervalKey),
			MaxInterval:         viperInstance.GetDuration(ClientBackoffMaxIntervalKey),
			MaxElapsedTime:      viperInstance.GetDuration(ClientBackoffMaxElapsedTimeKey),
			RandomizationFactor: viperInstance.GetFloat64(ClientBackoffRandomizationFactorKey),
			Multiplier:          viperInstance.GetFloat64(ClientBackoffMultiplierKey),
		},
	}
}

func resolveCollector(allowedDirs []string) (*Collector, error) {
	var receivers Receivers

	err := resolveMapStructure(CollectorReceiversKey, &receivers)
	if err != nil {
		return nil, fmt.Errorf("unmarshal collector receivers config: %w", err)
	}

	exporters, err := resolveExporters()
	if err != nil {
		return nil, fmt.Errorf("unmarshal collector exporters config: %w", err)
	}

	//slog.Info("Resolved collector receivers", "receivers", receivers)

	col := &Collector{
		ConfigPath: viperInstance.GetString(CollectorConfigPathKey),
		Exporters:  exporters,
		Processors: resolveProcessors(),
		Receivers:  receivers,
		Extensions: resolveExtensions(),
		Log:        resolveCollectorLog(),
	}

	// Check for self-signed certificate true in Agent conf
	if err = handleSelfSignedCertificates(col); err != nil {
		return nil, err
	}

	err = col.Validate(allowedDirs)
	if err != nil {
		return nil, fmt.Errorf("collector config: %w", err)
	}

	return col, nil
}

func resolveExporters() (Exporters, error) {
	var otlpExporters []OtlpExporter
	exporters := Exporters{}

	if viperInstance.IsSet(CollectorDebugExporterKey) {
		exporters.Debug = &DebugExporter{}
	}

	if isPrometheusExporterSet() {
		exporters.PrometheusExporter = &PrometheusExporter{
			Server: &ServerConfig{
				Host: viperInstance.GetString(CollectorPrometheusExporterServerHostKey),
				Port: viperInstance.GetInt(CollectorPrometheusExporterServerPortKey),
			},
		}

		if arePrometheusExportTLSSettingsSet() {
			exporters.PrometheusExporter.TLS = &TLSConfig{
				Cert:       viperInstance.GetString(CollectorPrometheusExporterTLSCertKey),
				Key:        viperInstance.GetString(CollectorPrometheusExporterTLSKeyKey),
				Ca:         viperInstance.GetString(CollectorPrometheusExporterTLSCaKey),
				SkipVerify: viperInstance.GetBool(CollectorPrometheusExporterTLSSkipVerifyKey),
				ServerName: viperInstance.GetString(CollectorPrometheusExporterTLSServerNameKey),
			}
		}
	}

	err := resolveMapStructure(CollectorOtlpExportersKey, &otlpExporters)
	if err != nil {
		return exporters, err
	}

	exporters.OtlpExporters = otlpExporters

	return exporters, nil
}

func isPrometheusExporterSet() bool {
	return viperInstance.IsSet(CollectorPrometheusExporterKey) ||
		(viperInstance.IsSet(CollectorPrometheusExporterServerHostKey) &&
			viperInstance.IsSet(CollectorPrometheusExporterServerPortKey))
}

func resolveProcessors() Processors {
	processors := Processors{
		Batch: &Batch{
			SendBatchSize:    viperInstance.GetUint32(CollectorBatchProcessorSendBatchSizeKey),
			SendBatchMaxSize: viperInstance.GetUint32(CollectorBatchProcessorSendBatchMaxSizeKey),
			Timeout:          viperInstance.GetDuration(CollectorBatchProcessorTimeoutKey),
		},
	}

	if viperInstance.IsSet(CollectorAttributeProcessorKey) {
		err := resolveMapStructure(CollectorAttributeProcessorKey, &processors.Attribute)
		if err != nil {
			return processors
		}
	}

	return processors
}

// generate self-signed certificate for OTel receiver
// nolint: revive
func handleSelfSignedCertificates(col *Collector) error {
	if col.Receivers.OtlpReceivers != nil {
		for _, receiver := range col.Receivers.OtlpReceivers {
			if receiver.OtlpTLSConfig != nil && receiver.OtlpTLSConfig.GenerateSelfSignedCert {
				err := processOtlpReceivers(receiver.OtlpTLSConfig)
				if err != nil {
					return fmt.Errorf("failed to generate self-signed certificate: %w", err)
				}
			}
		}
	}

	return nil
}

func processOtlpReceivers(tlsConfig *OtlpTLSConfig) error {
	sanNames := strings.Split(DefCollectorTLSSANNames, ",")

	if tlsConfig.Ca == "" {
		tlsConfig.Ca = DefCollectorTLSCAPath
	}
	if tlsConfig.Cert == "" {
		tlsConfig.Cert = DefCollectorTLSCertPath
	}
	if tlsConfig.Key == "" {
		tlsConfig.Key = DefCollectorTLSKeyPath
	}

	if !slices.Contains(sanNames, tlsConfig.ServerName) {
		sanNames = append(sanNames, tlsConfig.ServerName)
	}
	if len(sanNames) > 0 {
		existingCert, err := selfsignedcerts.GenerateServerCerts(
			sanNames,
			tlsConfig.Ca,
			tlsConfig.Cert,
			tlsConfig.Key,
		)
		if err != nil {
			return fmt.Errorf("failed to generate self-signed certificate: %w", err)
		}
		if existingCert {
			tlsConfig.ExistingCert = true
		}
	}

	return nil
}

func resolveExtensions() Extensions {
	var health *Health
	var headersSetter *HeadersSetter

	if isHealthExtensionSet() {
		health = &Health{
			Server: &ServerConfig{
				Host: viperInstance.GetString(CollectorExtensionsHealthServerHostKey),
				Port: viperInstance.GetInt(CollectorExtensionsHealthServerPortKey),
			},
			Path: viperInstance.GetString(CollectorExtensionsHealthPathKey),
		}

		if areHealthExtensionTLSSettingsSet() {
			health.TLS = &TLSConfig{
				Cert:       viperInstance.GetString(CollectorExtensionsHealthTLSCertKey),
				Key:        viperInstance.GetString(CollectorExtensionsHealthTLSKeyKey),
				Ca:         viperInstance.GetString(CollectorExtensionsHealthTLSCaKey),
				SkipVerify: viperInstance.GetBool(CollectorExtensionsHealthTLSSkipVerifyKey),
				ServerName: viperInstance.GetString(CollectorExtensionsHealthTLSServerNameKey),
			}
		}
	}

	if viperInstance.IsSet(CollectorExtensionsHeadersSetterKey) {
		err := resolveMapStructure(CollectorExtensionsHeadersSetterKey, &headersSetter)
		if err != nil {
			headersSetter = nil
		}
	}

	return Extensions{
		Health:        health,
		HeadersSetter: headersSetter,
	}
}

func isHealthExtensionSet() bool {
	return viperInstance.IsSet(CollectorExtensionsHealthKey) ||
		(viperInstance.IsSet(CollectorExtensionsHealthServerHostKey) &&
			viperInstance.IsSet(CollectorExtensionsHealthServerPortKey))
}

func resolveCollectorLog() *Log {
	return &Log{
		Level: viperInstance.GetString(CollectorLogLevelKey),
		Path:  viperInstance.GetString(CollectorLogPathKey),
	}
}

func resolveCommand() *Command {
	serverType, ok := parseServerType(viperInstance.GetString(CommandServerTypeKey))
	if !ok {
		serverType = Grpc
		slog.Error(
			"Invalid value for command server type, defaulting to gRPC server type",
			"server_type", viperInstance.GetString(CommandServerTypeKey),
		)
	}

	command := &Command{
		Server: &ServerConfig{
			Host: viperInstance.GetString(CommandServerHostKey),
			Port: viperInstance.GetInt(CommandServerPortKey),
			Type: serverType,
		},
	}

	if areAuthSettingsSet() {
		command.Auth = &AuthConfig{
			Token:     viperInstance.GetString(CommandAuthTokenKey),
			TokenPath: viperInstance.GetString(CommandAuthTokenPathKey),
		}
	}

	if areTLSSettingsSet() {
		command.TLS = &TLSConfig{
			Cert:       viperInstance.GetString(CommandTLSCertKey),
			Key:        viperInstance.GetString(CommandTLSKeyKey),
			Ca:         viperInstance.GetString(CommandTLSCaKey),
			SkipVerify: viperInstance.GetBool(CommandTLSSkipVerifyKey),
			ServerName: viperInstance.GetString(CommandTLSServerNameKey),
		}
	}

	return command
}

func areAuthSettingsSet() bool {
	return viperInstance.IsSet(CommandAuthTokenKey) ||
		viperInstance.IsSet(CommandAuthTokenPathKey)
}

func areTLSSettingsSet() bool {
	return viperInstance.IsSet(CommandTLSCertKey) ||
		viperInstance.IsSet(CommandTLSKeyKey) ||
		viperInstance.IsSet(CommandTLSCaKey) ||
		viperInstance.IsSet(CommandTLSSkipVerifyKey) ||
		viperInstance.IsSet(CommandTLSServerNameKey)
}

func areHealthExtensionTLSSettingsSet() bool {
	return viperInstance.IsSet(CollectorExtensionsHealthTLSCertKey) ||
		viperInstance.IsSet(CollectorExtensionsHealthTLSKeyKey) ||
		viperInstance.IsSet(CollectorExtensionsHealthTLSCaKey) ||
		viperInstance.IsSet(CollectorExtensionsHealthTLSSkipVerifyKey) ||
		viperInstance.IsSet(CollectorExtensionsHealthTLSServerNameKey)
}

func arePrometheusExportTLSSettingsSet() bool {
	return viperInstance.IsSet(CollectorPrometheusExporterTLSCertKey) ||
		viperInstance.IsSet(CollectorPrometheusExporterTLSKeyKey) ||
		viperInstance.IsSet(CollectorPrometheusExporterTLSCaKey) ||
		viperInstance.IsSet(CollectorPrometheusExporterTLSSkipVerifyKey) ||
		viperInstance.IsSet(CollectorPrometheusExporterTLSServerNameKey)
}

func resolveWatchers() *Watchers {
	return &Watchers{
		InstanceWatcher: InstanceWatcher{
			MonitoringFrequency: viperInstance.GetDuration(InstanceWatcherMonitoringFrequencyKey),
		},
		InstanceHealthWatcher: InstanceHealthWatcher{
			MonitoringFrequency: viperInstance.GetDuration(InstanceHealthWatcherMonitoringFrequencyKey),
		},
		FileWatcher: FileWatcher{
			MonitoringFrequency: viperInstance.GetDuration(FileWatcherMonitoringFrequencyKey),
			ExcludeFiles:        viperInstance.GetStringSlice(NginxExcludeFilesKey),
		},
	}
}

// Wrapper needed for more detailed error message.
func resolveMapStructure(key string, object any) error {
	err := viperInstance.UnmarshalKey(key, &object)
	if err != nil {
		return fmt.Errorf("resolve config %s: %w", key, err)
	}

	return nil
}
