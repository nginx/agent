// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package config

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	uuidLibrary "github.com/nginx/agent/v3/internal/uuid"
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

func GetConfig() *Config {
	config := &Config{
		UUID:               viperInstance.GetString(UUIDKey),
		Version:            viperInstance.GetString(VersionKey),
		Path:               viperInstance.GetString(ConfigPathKey),
		Log:                getLog(),
		ProcessMonitor:     getProcessMonitor(),
		DataPlaneConfig:    getDataPlaneConfig(),
		Client:             getClient(),
		ConfigDir:          getConfigDir(),
		AllowedDirectories: []string{},
		Metrics:            getMetrics(),
		Command:            getCommand(),
		Common:             getCommon(),
		Watchers:           getWatchers(),
	}

	for _, dir := range strings.Split(config.ConfigDir, ":") {
		if dir != "" && filepath.IsAbs(dir) {
			config.AllowedDirectories = append(config.AllowedDirectories, dir)
		} else {
			slog.Warn("Invalid directory: ", "dir", dir)
		}
	}
	slog.Debug("Agent config", "config", config)

	return config
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
		ProcessMonitorMonitoringFrequencyKey,
		time.Minute,
		"How often the NGINX Agent will check for process changes.",
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

	fs.Duration(
		MetricsProduceIntervalKey, DefMetricsProduceInterval,
		"The interval for how often NGINX Agent queries metrics from its sources.",
	)
	fs.Int(
		OTelExporterBufferLengthKey, DefOTelExporterBufferLength,
		"The length of the OTel Exporter's buffer for metrics.",
	)
	fs.Int(
		OTelExporterExportRetryCountKey, DefOTelExporterExportRetryCount,
		"How many times an OTel Export is retried in the event of failure.",
	)
	fs.Duration(
		OTelExporterExportIntervalKey, DefOTelExporterExportInterval,
		"The interval for how often NGINX Agent attempts to send the contents of its OTel Exporter's buffer.",
	)
	fs.String(
		OTelGRPCTargetKey, "", "The target URI for a gRPC OTel Collector.",
	)
	fs.Duration(
		OTelGRPCConnTimeoutKey, DefOTelGRPCConnTimeout,
		"The connection timeout for the gRPC connection to the OTel collector.",
	)
	fs.Duration(
		OTelGRPCMinConnTimeoutKey, DefOTelGRPCMinConnTimeout,
		"The minimum connection timeout for the gRPC connection to the OTel collector.",
	)
	fs.Duration(
		OTelGRPCBackoffDelayKey, DefOTelGRPCMBackoffDelay,
		"The maximum delay on the gRPC backoff strategy for retrying a failed connection.",
	)
	fs.StringArray(
		PrometheusTargetsKey, []string{}, "The target URI(s) of Prometheus endpoint(s) for metrics collection.",
	)
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

func getLog() *Log {
	return &Log{
		Level: viperInstance.GetString(LogLevelKey),
		Path:  viperInstance.GetString(LogPathKey),
	}
}

func getProcessMonitor() *ProcessMonitor {
	return &ProcessMonitor{
		MonitoringFrequency: viperInstance.GetDuration(ProcessMonitorMonitoringFrequencyKey),
	}
}

func getDataPlaneConfig() *DataPlaneConfig {
	return &DataPlaneConfig{
		Nginx: &NginxDataPlaneConfig{
			ReloadMonitoringPeriod: viperInstance.GetDuration(DataPlaneConfigNginxReloadMonitoringPeriodKey),
			TreatWarningsAsError:   viperInstance.GetBool(DataPlaneConfigNginxTreatWarningsAsErrorsKey),
		},
	}
}

func getClient() *Client {
	return &Client{
		Timeout:             viperInstance.GetDuration(ClientTimeoutKey),
		Time:                viperInstance.GetDuration(ClientTimeKey),
		PermitWithoutStream: viperInstance.GetBool(ClientPermitWithoutStreamKey),
	}
}

func getConfigDir() string {
	return viperInstance.GetString(ConfigDirectoriesKey)
}

func getMetrics() *Metrics {
	if !viperInstance.IsSet(MetricsRootKey) {
		return nil
	}

	metrics := &Metrics{
		ProduceInterval:  viperInstance.GetDuration(MetricsProduceIntervalKey),
		OTelExporter:     nil,
		PrometheusSource: nil,
		Collector:        viperInstance.GetBool(MetricsCollectorKey),
	}

	if viperInstance.IsSet(MetricsOTelExporterKey) && viperInstance.IsSet(OTelGRPCKey) {
		// For some reason viperInstance.UnmarshalKey did not work here (maybe due to the nested structs?).
		otelExp := &OTelExporter{
			BufferLength:     viperInstance.GetInt(OTelExporterBufferLengthKey),
			ExportRetryCount: viperInstance.GetInt(OTelExporterExportRetryCountKey),
			ExportInterval:   viperInstance.GetDuration(OTelExporterExportIntervalKey),
			GRPC: &GRPC{
				Target:         viperInstance.GetString(OTelGRPCTargetKey),
				ConnTimeout:    viperInstance.GetDuration(OTelGRPCConnTimeoutKey),
				MinConnTimeout: viperInstance.GetDuration(OTelGRPCMinConnTimeoutKey),
				BackoffDelay:   viperInstance.GetDuration(OTelGRPCBackoffDelayKey),
			},
		}
		metrics.OTelExporter = otelExp
	}

	if viperInstance.IsSet(PrometheusSrcKey) {
		var prometheusSrc PrometheusSource
		err := viperInstance.UnmarshalKey(PrometheusSrcKey, &prometheusSrc)
		if err == nil {
			metrics.PrometheusSource = &prometheusSrc
		} else {
			slog.Error("metrics configuration: no Prometheus source configured", "error", err)
		}
	}

	return metrics
}

func getCommand() *Command {
	if !viperInstance.IsSet(CommandRootKey) {
		return nil
	}
	command := &Command{}

	if viperInstance.IsSet(CommandServerKey) {
		command.Server = &ServerConfig{
			Host: viperInstance.GetString(CommandServerHostKey),
			Port: viperInstance.GetInt(CommandServerPortKey),
			Type: viperInstance.GetString(CommandServerTypeKey),
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

func getCommon() *CommonSettings {
	return &CommonSettings{
		InitialInterval:     DefBackoffInitalInterval,
		MaxInterval:         DefBackoffMaxInterval,
		MaxElapsedTime:      DefBackoffMaxElapsedTime,
		RandomizationFactor: DefBackoffRandomizationFactor,
		Multiplier:          DefBackoffMultiplier,
	}
}

func getWatchers() *Watchers {
	return &Watchers{
		InstanceWatcher: InstanceWatcher{
			MonitoringFrequency: DefInstanceWatcherMonitoringFrequency,
		},
	}
}
