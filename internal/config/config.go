// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package config

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	uuidLibrary "github.com/nginx/agent/v3/pkg/uuid"
	"github.com/spf13/cobra"
	flag "github.com/spf13/pflag"
	"github.com/spf13/viper"
)

const (
	ConfigFileName = "nginx-agent.conf"
	EnvPrefix      = "NGINX_AGENT"
	KeyDelimiter   = "."
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
	configDir := viperInstance.GetString(ConfigDirectoriesKey)
	allowedDirs := make([]string, 0)
	for _, dir := range strings.Split(configDir, ":") {
		if dir != "" && filepath.IsAbs(dir) {
			allowedDirs = append(allowedDirs, dir)
		} else {
			slog.Warn("Invalid directory: ", "dir", dir)
		}
	}

	// Collect all parsing errors before returning the error, so the user sees all issues with config
	// in one error message.
	var err error
	metrics, metricsErr := resolveMetrics(allowedDirs)
	err = errors.Join(err, metricsErr)
	if err != nil {
		return nil, err
	}

	config := &Config{
		UUID:               viperInstance.GetString(UUIDKey),
		Version:            viperInstance.GetString(VersionKey),
		Path:               viperInstance.GetString(ConfigPathKey),
		Log:                resolveLog(),
		DataPlaneConfig:    resolveDataPlaneConfig(),
		Client:             resolveClient(),
		ConfigDir:          configDir,
		AllowedDirectories: allowedDirs,
		Metrics:            metrics,
		Command:            resolveCommand(),
		Common:             resolveCommon(),
		Watchers:           resolveWatchers(),
	}

	slog.Debug("Agent config", "config", config)

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
		`The desired verbosity level for logging messages from nginx-agent. 
		Available options, in order of severity from highest to lowest, are: 
		panic, fatal, error, info, debug, and trace.`,
	)
	fs.String(
		LogPathKey,
		"",
		`The path to output log messages to. 
		If the default path doesn't exist, log messages are output to stdout/stderr.`,
	)

	fs.Duration(
		DataPlaneConfigNginxReloadMonitoringPeriodKey,
		DefaultDataPlaneConfigNginxReloadMonitoringPeriod,
		"The amount of time to monitor NGINX after a reload of configuration.",
	)
	fs.Bool(
		DataPlaneConfigNginxTreatWarningsAsErrorsKey,
		true,
		"Warning messages in the NGINX errors logs after a NGINX reload will be treated as an error.",
	)

	fs.Duration(ClientTimeoutKey, time.Minute, "Client timeout")
	fs.String(ConfigDirectoriesKey, "/etc/nginx:/usr/local/etc/nginx:/usr/share/nginx/modules",
		"Defines the paths that you want to grant NGINX Agent read/write access to."+
			" This key is formatted as a string and follows Unix PATH format")

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
		"Testing only. SkipVerify controls client verification of a server's certificate chain and host name.",
	)

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
	fs.String(
		MetricsOTLPExportURLKey,
		DefOTLPExportURL,
		"The OTLP metrics exporter's gRPC URL for the NGINX Agent OTel Collector.",
	)
	fs.String(
		MetricsCollectorConfigPathKey,
		DefCollectorConfigPath,
		"The path to the Opentelemetry Collector configuration file.",
	)
	fs.String(
		MetricsOTLPReceiverURLKey,
		DefOTLPReceiverURL,
		"The OTLP metrics receiver's gRPC URL for the NGINX Agent OTel Collector.",
	)
	fs.StringArray(
		MetricsCollectorReceiversKey,
		DefCollectorReceivers,
		"Metrics receiver names for the NGINX Agent OTel Collector.",
	)

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
		slog.Warn("Unable to determine process's current directory")
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

func resolveDataPlaneConfig() *DataPlaneConfig {
	return &DataPlaneConfig{
		Nginx: &NginxDataPlaneConfig{
			ReloadMonitoringPeriod: viperInstance.GetDuration(DataPlaneConfigNginxReloadMonitoringPeriodKey),
			TreatWarningsAsError:   viperInstance.GetBool(DataPlaneConfigNginxTreatWarningsAsErrorsKey),
		},
	}
}

func resolveClient() *Client {
	return &Client{
		Timeout:             viperInstance.GetDuration(ClientTimeoutKey),
		Time:                viperInstance.GetDuration(ClientTimeKey),
		PermitWithoutStream: viperInstance.GetBool(ClientPermitWithoutStreamKey),
	}
}

func resolveMetrics(allowedDirs []string) (*Metrics, error) {
	// We do not want to return a sentinel error because we are joining all returned errors
	// from config resolution and returning them without pattern matching.
	// nolint: nilnil
	if !viperInstance.IsSet(MetricsRootKey) {
		return nil, nil
	}

	strReceivers := viperInstance.GetStringSlice(MetricsCollectorReceiversKey)
	enumReceivers := make([]OTelReceiver, 0, len(strReceivers))
	for _, rec := range strReceivers {
		rec := toOTelReceiver(strings.ToLower(rec))
		// A OTLP receiver is always automatically added.
		if rec != Unsupported && rec != OTLP {
			enumReceivers = append(enumReceivers, rec)
		}
	}

	var err error
	otelColConfPath, pathErr := filePathKey(MetricsCollectorConfigPathKey, allowedDirs)
	err = errors.Join(pathErr, pathErr)
	if err != nil {
		return nil, fmt.Errorf("invalid metrics: %w", err)
	}

	return &Metrics{
		Collector:           viperInstance.GetBool(MetricsCollectorKey),
		OTLPExportURL:       viperInstance.GetString(MetricsOTLPExportURLKey),
		OTLPReceiverURL:     viperInstance.GetString(MetricsOTLPReceiverURLKey),
		CollectorConfigPath: otelColConfPath,
		CollectorReceivers:  enumReceivers,
	}, nil
}

func resolveCommand() *Command {
	if !viperInstance.IsSet(CommandRootKey) {
		return nil
	}
	command := &Command{}

	if viperInstance.IsSet(CommandServerKey) {
		serverType, ok := parseServerType(viperInstance.GetString(CommandServerTypeKey))
		if !ok {
			serverType = Grpc
			slog.Error(
				"Invalid value for command server type, defaulting to gRPC server type",
				"server_type", viperInstance.GetString(CommandServerTypeKey),
			)
		}

		command.Server = &ServerConfig{
			Host: viperInstance.GetString(CommandServerHostKey),
			Port: viperInstance.GetInt(CommandServerPortKey),
			Type: serverType,
		}
	}

	if viperInstance.IsSet(CommandAuthKey) {
		command.Auth = &AuthConfig{
			Token: viperInstance.GetString(CommandAuthTokenKey),
		}
	}

	if viperInstance.IsSet(CommandTLSKey) {
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

func resolveCommon() *CommonSettings {
	return &CommonSettings{
		InitialInterval:     DefBackoffInitialInterval,
		MaxInterval:         DefBackoffMaxInterval,
		MaxElapsedTime:      DefBackoffMaxElapsedTime,
		RandomizationFactor: DefBackoffRandomizationFactor,
		Multiplier:          DefBackoffMultiplier,
	}
}

func resolveWatchers() *Watchers {
	return &Watchers{
		InstanceWatcher: InstanceWatcher{
			MonitoringFrequency: DefInstanceWatcherMonitoringFrequency,
		},
		InstanceHealthWatcher: InstanceHealthWatcher{
			MonitoringFrequency: DefInstanceHealthWatcherMonitoringFrequency,
		},
	}
}

// Resolves the given `configKey` from viper and validates it.
func filePathKey(configKey string, allowedDirPaths []string) (string, error) {
	inputPath := viperInstance.GetString(configKey)
	cleaned := filepath.Clean(inputPath)

	for _, path := range allowedDirPaths {
		if strings.HasPrefix(cleaned, path) {
			return cleaned, nil
		}
	}

	return "", fmt.Errorf("%s: path %s not allowed", configKey, cleaned)
}
