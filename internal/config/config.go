// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package config

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
	flag "github.com/spf13/pflag"
	"github.com/spf13/viper"
)

const (
	ConfigFileName = "nginx-agent.conf"
	EnvPrefix      = "NGINX_AGENT"
	keyDelimiter   = "_"
)

var viperInstance = viper.NewWithOptions(viper.KeyDelimiter(keyDelimiter))

func RegisterRunner(r func(cmd *cobra.Command, args []string)) {
	RootCommand.Run = r
}

func Execute() error {
	RootCommand.AddCommand(CompletionCommand)
	return RootCommand.Execute()
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

	if err := loadPropertiesFromFile(configPath); err != nil {
		return err
	}

	slog.Debug("Configuration file loaded", "config_path", configPath)
	viperInstance.Set(ConfigPathKey, configPath)

	return nil
}

func GetConfig() *Config {
	config := &Config{
		Version:            viperInstance.GetString(VersionKey),
		Log:                getLog(),
		ProcessMonitor:     getProcessMonitor(),
		DataPlaneAPI:       getDataPlaneAPI(),
		Client:             getClient(),
		ConfigDir:          getConfigDir(),
		AllowedDirectories: []string{},
		Metrics:            getMetrics(),
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
	fs.String(DataPlaneAPIHostKey, "", "The host used by the Dataplane API.")
	fs.Int(DataPlaneAPIPortKey, 0, "The desired port to use for NGINX Agent to expose for HTTP traffic.")
	fs.Duration(ClientTimeoutKey, time.Minute, "Client timeout")
	fs.String(ConfigDirectoriesKey, "/etc/nginx:/usr/local/etc/nginx:/usr/share/nginx/modules",
		"Defines the paths that you want to grant nginx-agent read/write access to."+
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

func getLog() Log {
	return Log{
		Level: viperInstance.GetString(LogLevelKey),
		Path:  viperInstance.GetString(LogPathKey),
	}
}

func getProcessMonitor() ProcessMonitor {
	return ProcessMonitor{
		MonitoringFrequency: viperInstance.GetDuration(ProcessMonitorMonitoringFrequencyKey),
	}
}

func getDataPlaneAPI() DataPlaneAPI {
	return DataPlaneAPI{
		Host: viperInstance.GetString(DataPlaneAPIHostKey),
		Port: viperInstance.GetInt(DataPlaneAPIPortKey),
	}
}

func getClient() Client {
	return Client{
		Timeout: viperInstance.GetDuration(ClientTimeoutKey),
	}
}

func getConfigDir() string {
	return viperInstance.GetString(ConfigDirectoriesKey)
}

func getMetrics() *Metrics {
	if !viperInstance.IsSet(MetricsRootKey) {
		return nil
	}

	m := &Metrics{
		ProduceInterval:  viperInstance.GetDuration(MetricsProduceIntervalKey),
		OTelExporter:     nil,
		PrometheusSource: nil,
	}

	if viperInstance.IsSet(MetricsOTelExporterKey) && viperInstance.IsSet(OTelGRPCKey) {
		// For some reason viperInstance.UnmarshalKey did not work here (maybe due to the nested structs?).
		otelExp := OTelExporter{
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
		m.OTelExporter = &otelExp
	}

	if viperInstance.IsSet(PrometheusSrcKey) {
		var prometheusSrc PrometheusSource
		err := viperInstance.UnmarshalKey(PrometheusSrcKey, &prometheusSrc)
		if err == nil {
			m.PrometheusSource = &prometheusSrc
		} else {
			slog.Error("metrics configuration: no Prometheus source configured", "error", err)
		}
	}

	return m
}
