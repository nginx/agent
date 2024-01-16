/**
 * Copyright (c) F5, Inc.
 *
 * This source code is licensed under the Apache License, Version 2.0 license found in the
 * LICENSE file in the root directory of this source tree.
 */

package config

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	flag "github.com/spf13/pflag"
	"github.com/spf13/viper"
)

const (
	ConfigFileName = "nginx-agent.conf"
	EnvPrefix      = "NGINX_AGENT"
)

var viperInstance = viper.NewWithOptions(viper.KeyDelimiter("_"))

func RegisterRunner(r func(cmd *cobra.Command, args []string)) {
	ROOT_COMMAND.Run = r
}

func Execute() error {
	ROOT_COMMAND.AddCommand(COMPLETION_COMMAND)
	return ROOT_COMMAND.Execute()
}

func Init(version, commit string) {
	setVersion(version, commit)
	setDefaults()
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

	slog.Debug("configuration file loaded", "configPath", configPath)
	viperInstance.Set(ConfigPathKey, configPath)

	return nil
}

func GetConfig() *Config {
	config := &Config{
		Version:        viperInstance.GetString(VersionConfigKey),
		Log:            getLog(),
		ProcessMonitor: getProcessMonitor(),
		DataplaneAPI:   getDataplaneAPI(),
	}

	slog.Debug("agent config", "config", config)
	return config
}

func setVersion(version, commit string) {
	ROOT_COMMAND.Version = version + "-" + commit
	viperInstance.SetDefault(VersionConfigKey, version)
}

func setDefaults() {
	for _, agentFlag := range agentFlags {
		switch agentFlag := agentFlag.(type) {
		case *StringFlag:
			viperInstance.SetDefault(agentFlag.Name, agentFlag.DefaultValue)
		case *IntFlag:
			viperInstance.SetDefault(agentFlag.Name, agentFlag.DefaultValue)
		case *DurationFlag:
			viperInstance.SetDefault(agentFlag.Name, agentFlag.DefaultValue)
		}
	}
}

func registerFlags() {
	viperInstance.SetEnvPrefix(EnvPrefix)
	viperInstance.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	viperInstance.AutomaticEnv()

	fs := ROOT_COMMAND.Flags()
	for _, f := range agentFlags {
		f.register(fs)
	}

	fs.SetNormalizeFunc(normalizeFunc)

	fs.VisitAll(func(flag *flag.Flag) {
		if err := viperInstance.BindPFlag(strings.ReplaceAll(flag.Name, "-", "_"), fs.Lookup(flag.Name)); err != nil {
			return
		}
		err := viperInstance.BindEnv(flag.Name)
		if err != nil {
			slog.Warn("error occurred binding env", "env", flag.Name, "error", err)
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
		slog.Warn("unable to determine process's current directory")
	}

	return paths
}

func loadPropertiesFromFile(cfg string) error {
	viperInstance.SetConfigFile(cfg)
	viperInstance.SetConfigType("yaml")
	err := viperInstance.MergeInConfig()
	if err != nil {
		return fmt.Errorf("error loading config file %s: %v", cfg, err)
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
