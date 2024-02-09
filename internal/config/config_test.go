// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package config

import (
	"os"
	"path"
	"testing"
	"time"

	helpers "github.com/nginx/agent/v3/test"

	"github.com/stretchr/testify/require"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
)

const (
	viperKeyDeliDelimiter = "_"
)

func TestRegisterConfigFile(t *testing.T) {
	viperInstance = viper.NewWithOptions(viper.KeyDelimiter(viperKeyDeliDelimiter))
	file, err := os.Create("nginx-agent.conf")
	defer helpers.RemoveFileWithErrorCheck(t, file.Name())
	require.NoError(t, err)

	currentDirectory, err := os.Getwd()
	require.NoError(t, err)

	err = RegisterConfigFile()

	require.NoError(t, err)
	assert.Equal(t, path.Join(currentDirectory, "nginx-agent.conf"), viperInstance.GetString(ConfigPathKey))
}

func TestGetConfig(t *testing.T) {
	viperInstance = viper.NewWithOptions(viper.KeyDelimiter(viperKeyDeliDelimiter))
	err := loadPropertiesFromFile("./testdata/nginx-agent.conf")
	allowedDir := []string{"/etc/nginx", "/usr/local/etc/nginx", "/usr/share/nginx/modules"}
	require.NoError(t, err)

	result := GetConfig()

	assert.Equal(t, "debug", result.Log.Level)
	assert.Equal(t, "./", result.Log.Path)

	assert.Equal(t, "127.0.0.1", result.DataplaneAPI.Host)
	assert.Equal(t, 8038, result.DataplaneAPI.Port)

	assert.Equal(t, 30*time.Second, result.ProcessMonitor.MonitoringFrequency)
	assert.Equal(t, 10*time.Second, result.Client.Timeout)

	assert.Equal(t, "/etc/nginx:/usr/local/etc/nginx:/usr/share/nginx/modules", result.ConfigDir)
	assert.Equal(t, allowedDir, result.AllowedDirectories)
}

func TestSetVersion(t *testing.T) {
	viperInstance = viper.NewWithOptions(viper.KeyDelimiter(viperKeyDeliDelimiter))
	setVersion("v1.2.3", "asdf1234")

	assert.Equal(t, "v1.2.3", viperInstance.GetString(VersionConfigKey))
}

func TestRegisterFlags(t *testing.T) {
	viperInstance = viper.NewWithOptions(viper.KeyDelimiter(viperKeyDeliDelimiter))
	t.Setenv("NGINX_AGENT_LOG_LEVEL", "warn")
	t.Setenv("NGINX_AGENT_LOG_PATH", "/var/log/test/agent.log")
	t.Setenv("NGINX_AGENT_PROCESS_MONITOR_MONITORING_FREQUENCY", "10s")
	t.Setenv("NGINX_AGENT_DATAPLANE_API_HOST", "example.com")
	t.Setenv("NGINX_AGENT_DATAPLANE_API_PORT", "9090")
	t.Setenv("NGINX_AGENT_CLIENT_TIMEOUT", "10s")
	registerFlags()

	assert.Equal(t, "warn", viperInstance.GetString(LogLevelConfigKey))
	assert.Equal(t, "/var/log/test/agent.log", viperInstance.GetString(LogPathConfigKey))
	assert.Equal(t, 10*time.Second, viperInstance.GetDuration(ProcessMonitorMonitoringFrequencyConfigKey))
	assert.Equal(t, "example.com", viperInstance.GetString(DataplaneAPIHostConfigKey))
	assert.Equal(t, 9090, viperInstance.GetInt(DataplaneAPIPortConfigKey))
	assert.Equal(t, 10*time.Second, viperInstance.GetDuration(ClientTimeoutConfigKey))
}

func TestSeekFileInPaths(t *testing.T) {
	viperInstance = viper.NewWithOptions(viper.KeyDelimiter(viperKeyDeliDelimiter))
	result, err := seekFileInPaths("nginx-agent.conf", []string{"./", "./testdata"}...)

	require.NoError(t, err)
	assert.Equal(t, "testdata/nginx-agent.conf", result)

	_, err = seekFileInPaths("nginx-agent.conf", []string{"./"}...)
	require.Error(t, err)
}

func TestGetConfigFilePaths(t *testing.T) {
	viperInstance = viper.NewWithOptions(viper.KeyDelimiter(viperKeyDeliDelimiter))
	currentDirectory, err := os.Getwd()
	require.NoError(t, err)

	result := getConfigFilePaths()

	assert.Len(t, result, 2)
	assert.Equal(t, "/etc/nginx-agent/", result[0])
	assert.Equal(t, currentDirectory, result[1])
}

func TestLoadPropertiesFromFile(t *testing.T) {
	viperInstance = viper.NewWithOptions(viper.KeyDelimiter(viperKeyDeliDelimiter))
	err := loadPropertiesFromFile("./testdata/nginx-agent.conf")
	require.NoError(t, err)

	assert.Equal(t, "debug", viperInstance.GetString(LogLevelConfigKey))
	assert.Equal(t, "./", viperInstance.GetString(LogPathConfigKey))

	assert.Equal(t, "127.0.0.1", viperInstance.GetString(DataplaneAPIHostConfigKey))
	assert.Equal(t, 8038, viperInstance.GetInt(DataplaneAPIPortConfigKey))

	assert.Equal(t, 30*time.Second, viperInstance.GetDuration(ProcessMonitorMonitoringFrequencyConfigKey))

	assert.Equal(t, 10*time.Second, viperInstance.GetDuration(ClientTimeoutConfigKey))

	err = loadPropertiesFromFile("./testdata/unknown.conf")
	require.Error(t, err)
}

func TestNormalizeFunc(t *testing.T) {
	viperInstance = viper.NewWithOptions(viper.KeyDelimiter(viperKeyDeliDelimiter))
	var expected pflag.NormalizedName = "test-flag-name"
	result := normalizeFunc(&pflag.FlagSet{}, "test_flag.name")
	assert.Equal(t, expected, result)
}

func TestGetLog(t *testing.T) {
	viperInstance = viper.NewWithOptions(viper.KeyDelimiter(viperKeyDeliDelimiter))
	viperInstance.Set(LogLevelConfigKey, "error")
	viperInstance.Set(LogPathConfigKey, "/var/log/test/test.log")

	result := getLog()
	assert.Equal(t, "error", result.Level)
	assert.Equal(t, "/var/log/test/test.log", result.Path)
}

func TestGetProcessMonitor(t *testing.T) {
	viperInstance = viper.NewWithOptions(viper.KeyDelimiter(viperKeyDeliDelimiter))
	viperInstance.Set(ProcessMonitorMonitoringFrequencyConfigKey, time.Hour)

	result := getProcessMonitor()
	assert.Equal(t, time.Hour, result.MonitoringFrequency)
}

func TestGetDataplaneAPI(t *testing.T) {
	viperInstance = viper.NewWithOptions(viper.KeyDelimiter(viperKeyDeliDelimiter))
	viperInstance.Set(DataplaneAPIHostConfigKey, "testhost")
	viperInstance.Set(DataplaneAPIPortConfigKey, 9091)

	result := getDataplaneAPI()
	assert.Equal(t, "testhost", result.Host)
	assert.Equal(t, 9091, result.Port)
}

func TestGetClient(t *testing.T) {
	viperInstance = viper.NewWithOptions(viper.KeyDelimiter(viperKeyDeliDelimiter))
	viperInstance.Set(ClientTimeoutConfigKey, time.Hour)

	result := getClient()
	assert.Equal(t, time.Hour, result.Timeout)
}

func TestGetAllowedDirectories(t *testing.T) {
	viperInstance = viper.NewWithOptions(viper.KeyDelimiter(viperKeyDeliDelimiter))
	viperInstance.Set(ConfigDirectoriesConfigKey, "/etc/nginx:/usr/local/etc/nginx:/usr/share/nginx/modules")

	result := getConfigDir()
	assert.Equal(t, "/etc/nginx:/usr/local/etc/nginx:/usr/share/nginx/modules", result)
}
