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
	KeyDelimiter   = "_"
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
		ConfigDir:          configDir,
		AllowedDirectories: allowedDirs,
		Collector:          collector,
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
		panic, fatal, error, info and debug.`,
	)
	fs.String(
		LogPathKey,
		"",
		`The path to output log messages to. 
		If the default path doesn't exist, log messages are output to stdout/stderr.`,
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

	fs.Duration(ClientTimeoutKey, time.Minute, "Client timeout")
	fs.String(ConfigDirectoriesKey,
		DefConfigDirectories,
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

	fs.Duration(
		FileWatcherMonitoringFrequencyKey,
		DefFileWatcherMonitoringFrequency,
		"How often the NGINX Agent will check for file changes.",
	)

	fs.String(
		CollectorConfigPathKey,
		DefCollectorConfigPath,
		"The path to the Opentelemetry Collector configuration file.",
	)

	fs.String(
		CollectorLogLevelKey,
		DefCollectorLogLevel,
		`The desired verbosity level for logging messages from nginx-agent OTel collector. 
		Available options, in order of severity from highest to lowest, are: 
		ERROR, WARN, INFO and DEBUG.`,
	)

	fs.String(
		CollectorLogPathKey,
		DefCollectorLogPath,
		`The path to output OTel collector log messages to. 
		If the default path doesn't exist, log messages are output to stdout/stderr.`,
	)

	fs.Int(
		ClientMaxMessageSizeKey,
		DefMaxMessageSize,
		"The value used, if not 0, for both max_message_send_size and max_message_receive_size",
	)

	fs.Int(
		ClientMaxMessageRecieveSizeKey,
		DefMaxMessageRecieveSize,
		"Updates the client grpc setting MaxRecvMsgSize with the specific value in MB.",
	)

	fs.Int(
		ClientMaxMessageSendSizeKey,
		DefMaxMessageSendSize,
		"Updates the client grpc setting MaxSendMsgSize with the specific value in MB.",
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
			ReloadMonitoringPeriod: viperInstance.GetDuration(NginxReloadMonitoringPeriodKey),
			TreatWarningsAsError:   viperInstance.GetBool(NginxTreatWarningsAsErrorsKey),
		},
	}
}

func resolveClient() *Client {
	return &Client{
		Timeout:               viperInstance.GetDuration(ClientTimeoutKey),
		Time:                  viperInstance.GetDuration(ClientTimeKey),
		PermitWithoutStream:   viperInstance.GetBool(ClientPermitWithoutStreamKey),
		MaxMessageSize:        viperInstance.GetInt(ClientMaxMessageSizeKey),
		MaxMessageRecieveSize: viperInstance.GetInt(ClientMaxMessageRecieveSizeKey),
		MaxMessageSendSize:    viperInstance.GetInt(ClientMaxMessageSendSizeKey),
	}
}

func resolveCollector(allowedDirs []string) (*Collector, error) {
	// We do not want to return a sentinel error because we are joining all returned errors
	// from config resolution and returning them without pattern matching.
	// nolint: nilnil
	if !viperInstance.IsSet(CollectorRootKey) {
		return nil, nil
	}

	var (
		err         error
		exporters   []Exporter
		processors  []Processor
		receivers   Receivers
		healthCheck ServerConfig
		log         Log
	)

	err = errors.Join(
		err,
		resolveMapStructure(CollectorExportersKey, &exporters),
		resolveMapStructure(CollectorProcessorsKey, &processors),
		resolveMapStructure(CollectorReceiversKey, &receivers),
		resolveMapStructure(CollectorHealthKey, &healthCheck),
		resolveMapStructure(CollectorLogKey, &log),
	)
	if err != nil {
		return nil, fmt.Errorf("unmarshal collector config: %w", err)
	}

	if log.Level == "" {
		log.Level = DefCollectorLogLevel
	}

	if log.Path == "" {
		log.Path = DefCollectorLogPath
	}

	col := &Collector{
		ConfigPath: viperInstance.GetString(CollectorConfigPathKey),
		Exporters:  exporters,
		Processors: processors,
		Receivers:  receivers,
		Health:     &healthCheck,
		Log:        &log,
	}

	err = col.Validate(allowedDirs)
	if err != nil {
		return nil, fmt.Errorf("collector config: %w", err)
	}

	return col, nil
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
		FileWatcher: FileWatcher{
			MonitoringFrequency: DefFileWatcherMonitoringFrequency,
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
