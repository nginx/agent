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
	ConfigFileName                             = "nginx-agent.conf"
	EnvPrefix                                  = "NGINX_AGENT"
	ConfigPathKey                              = "path"
	VersionConfigKey                           = "version"
	LogLevelConfigKey                          = "log_level"
	LogPathConfigKey                           = "log_path"
	ProcessMonitorMonitoringFrequencyConfigKey = "process_monitor_monitoring_frequency"
	DataplaneAPIHostConfigKey                  = "dataplane_api_host"
	DataplaneAPIPortConfigKey                  = "dataplane_api_port"
	ClientTimeoutConfigKey                     = "client_timeout"
	ConfigDirectoriesConfigKey                 = "config_dirs"
)

var viperInstance = viper.NewWithOptions(viper.KeyDelimiter("_"))

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

	slog.Debug("Configuration file loaded", "configPath", configPath)
	viperInstance.Set(ConfigPathKey, configPath)

	return nil
}

func GetConfig() *Config {
	config := &Config{
		Version:            viperInstance.GetString(VersionConfigKey),
		Log:                getLog(),
		ProcessMonitor:     getProcessMonitor(),
		DataplaneAPI:       getDataplaneAPI(),
		Client:             getClient(),
		ConfigDir:          getConfigDir(),
		AllowedDirectories: []string{},
	}

	for _, dir := range strings.Split(config.ConfigDir, ":") {
		if dir != "" {
			config.AllowedDirectories = append(config.AllowedDirectories, dir)
		}
	}
	slog.Debug("Agent config", "config", config)

	return config
}

func setVersion(version, commit string) {
	RootCommand.Version = version + "-" + commit
	viperInstance.SetDefault(VersionConfigKey, version)
}

func registerFlags() {
	viperInstance.SetEnvPrefix(EnvPrefix)
	viperInstance.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	viperInstance.AutomaticEnv()

	fs := RootCommand.Flags()
	fs.String(
		LogLevelConfigKey,
		"info",
		`The desired verbosity level for logging messages from nginx-agent. 
		Available options, in order of severity from highest to lowest, are: 
		panic, fatal, error, info, debug, and trace.`,
	)
	fs.String(
		LogPathConfigKey,
		"",
		`The path to output log messages to. 
		If the default path doesn't exist, log messages are output to stdout/stderr.`,
	)
	fs.Duration(
		ProcessMonitorMonitoringFrequencyConfigKey,
		time.Minute,
		"How often the NGINX Agent will check for process changes.",
	)
	fs.String(DataplaneAPIHostConfigKey, "", "The host used by the Dataplane API.")
	fs.Int(DataplaneAPIPortConfigKey, 0, "The desired port to use for NGINX Agent to expose for HTTP traffic.")
	fs.Duration(ClientTimeoutConfigKey, time.Minute, "Client timeout")
	fs.String(ConfigDirectoriesConfigKey, "/etc/nginx:/usr/local/etc/nginx:/usr/share/nginx/modules",
		"Defines the paths that you want to grant nginx-agent read/write access to."+
			" This key is formatted as a string and follows Unix PATH format")

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
		Level: viperInstance.GetString(LogLevelConfigKey),
		Path:  viperInstance.GetString(LogPathConfigKey),
	}
}

func getProcessMonitor() ProcessMonitor {
	return ProcessMonitor{
		MonitoringFrequency: viperInstance.GetDuration(ProcessMonitorMonitoringFrequencyConfigKey),
	}
}

func getDataplaneAPI() DataplaneAPI {
	return DataplaneAPI{
		Host: viperInstance.GetString(DataplaneAPIHostConfigKey),
		Port: viperInstance.GetInt(DataplaneAPIPortConfigKey),
	}
}

func getClient() Client {
	return Client{
		Timeout: viperInstance.GetDuration(ClientTimeoutConfigKey),
	}
}

func getConfigDir() string {
	return viperInstance.GetString(ConfigDirectoriesConfigKey)
}
