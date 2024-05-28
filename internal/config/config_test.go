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

	"github.com/nginx/agent/v3/test/helpers"

	"github.com/stretchr/testify/require"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
)

func TestRegisterConfigFile(t *testing.T) {
	viperInstance = viper.NewWithOptions(viper.KeyDelimiter(KeyDelimiter))
	file, err := os.Create("nginx-agent.conf")
	defer helpers.RemoveFileWithErrorCheck(t, file.Name())
	require.NoError(t, err)

	currentDirectory, err := os.Getwd()
	require.NoError(t, err)

	err = RegisterConfigFile()

	require.NoError(t, err)
	assert.Equal(t, path.Join(currentDirectory, "nginx-agent.conf"), viperInstance.GetString(ConfigPathKey))
	assert.NotEmpty(t, viperInstance.GetString(UUIDKey))
}

func TestGetConfig(t *testing.T) {
	viperInstance = viper.NewWithOptions(viper.KeyDelimiter(KeyDelimiter))
	err := loadPropertiesFromFile("./testdata/nginx-agent.conf")
	allowedDir := []string{"/etc/nginx", "/usr/local/etc/nginx", "/usr/share/nginx/modules"}
	require.NoError(t, err)

	result := GetConfig()

	assert.Equal(t, "debug", result.Log.Level)
	assert.Equal(t, "./", result.Log.Path)

	assert.Equal(t, 30*time.Second, result.DataPlaneConfig.Nginx.ReloadMonitoringPeriod)
	assert.False(t, result.DataPlaneConfig.Nginx.TreatWarningsAsError)

	assert.Equal(t, 10*time.Second, result.Client.Timeout)

	assert.Equal(t, "/etc/nginx:/usr/local/etc/nginx:/usr/share/nginx/modules:invalid/path", result.ConfigDir)

	assert.Equal(t, allowedDir, result.AllowedDirectories)
}

func TestSetVersion(t *testing.T) {
	viperInstance = viper.NewWithOptions(viper.KeyDelimiter(KeyDelimiter))
	setVersion("v1.2.3", "asdf1234")

	assert.Equal(t, "v1.2.3", viperInstance.GetString(VersionKey))
}

func TestRegisterFlags(t *testing.T) {
	viperInstance = viper.NewWithOptions(viper.KeyDelimiter(KeyDelimiter))
	t.Setenv("NGINX_AGENT_LOG_LEVEL", "warn")
	t.Setenv("NGINX_AGENT_LOG_PATH", "/var/log/test/agent.log")
	t.Setenv("NGINX_AGENT_PROCESS_MONITOR_MONITORING_FREQUENCY", "10s")
	t.Setenv("NGINX_AGENT_DATA_PLANE_API_HOST", "example.com")
	t.Setenv("NGINX_AGENT_DATA_PLANE_API_PORT", "9090")
	t.Setenv("NGINX_AGENT_CLIENT_TIMEOUT", "10s")
	registerFlags()

	assert.Equal(t, "warn", viperInstance.GetString(LogLevelKey))
	assert.Equal(t, "/var/log/test/agent.log", viperInstance.GetString(LogPathKey))
	assert.Equal(t, 10*time.Second, viperInstance.GetDuration(ClientTimeoutKey))
}

func TestSeekFileInPaths(t *testing.T) {
	viperInstance = viper.NewWithOptions(viper.KeyDelimiter(KeyDelimiter))
	result, err := seekFileInPaths("nginx-agent.conf", []string{"./", "./testdata"}...)

	require.NoError(t, err)
	assert.Equal(t, "testdata/nginx-agent.conf", result)

	_, err = seekFileInPaths("nginx-agent.conf", []string{"./"}...)
	require.Error(t, err)
}

func TestGetConfigFilePaths(t *testing.T) {
	viperInstance = viper.NewWithOptions(viper.KeyDelimiter(KeyDelimiter))
	currentDirectory, err := os.Getwd()
	require.NoError(t, err)

	result := getConfigFilePaths()

	assert.Len(t, result, 2)
	assert.Equal(t, "/etc/nginx-agent/", result[0])
	assert.Equal(t, currentDirectory, result[1])
}

func TestLoadPropertiesFromFile(t *testing.T) {
	viperInstance = viper.NewWithOptions(viper.KeyDelimiter(KeyDelimiter))
	err := loadPropertiesFromFile("./testdata/nginx-agent.conf")
	require.NoError(t, err)

	assert.Equal(t, "debug", viperInstance.GetString(LogLevelKey))
	assert.Equal(t, "./", viperInstance.GetString(LogPathKey))

	assert.Equal(t, 10*time.Second, viperInstance.GetDuration(ClientTimeoutKey))

	err = loadPropertiesFromFile("./testdata/unknown.conf")
	require.Error(t, err)
}

func TestNormalizeFunc(t *testing.T) {
	viperInstance = viper.NewWithOptions(viper.KeyDelimiter(KeyDelimiter))
	var expected pflag.NormalizedName = "test-flag-name"
	result := normalizeFunc(&pflag.FlagSet{}, "test_flag.name")
	assert.Equal(t, expected, result)
}

func TestGetLog(t *testing.T) {
	viperInstance = viper.NewWithOptions(viper.KeyDelimiter(KeyDelimiter))
	viperInstance.Set(LogLevelKey, "error")
	viperInstance.Set(LogPathKey, "/var/log/test/test.log")

	result := getLog()
	assert.Equal(t, "error", result.Level)
	assert.Equal(t, "/var/log/test/test.log", result.Path)
}

func TestGetClient(t *testing.T) {
	viperInstance = viper.NewWithOptions(viper.KeyDelimiter(KeyDelimiter))
	viperInstance.Set(ClientTimeoutKey, time.Hour)

	result := getClient()
	assert.Equal(t, time.Hour, result.Timeout)
}

func TestGetAllowedDirectories(t *testing.T) {
	viperInstance = viper.NewWithOptions(viper.KeyDelimiter(KeyDelimiter))
	viperInstance.Set(ConfigDirectoriesKey, "/etc/nginx:/usr/local/etc/nginx:/usr/share/nginx/modules")

	result := getConfigDir()
	assert.Equal(t, "/etc/nginx:/usr/local/etc/nginx:/usr/share/nginx/modules", result)
}

func TestCommand(t *testing.T) {
	viperInstance = viper.NewWithOptions(viper.KeyDelimiter(KeyDelimiter))
	expected := getAgentConfig().Command

	// Server
	viperInstance.Set(CommandServerHostKey, expected.Server.Host)
	viperInstance.Set(CommandServerPortKey, expected.Server.Port)
	viperInstance.Set(CommandServerTypeKey, expected.Server.Type)

	// Auth
	viperInstance.Set(CommandAuthTokenKey, expected.Auth.Token)

	// TLS
	viperInstance.Set(CommandTLSCertKey, expected.TLS.Cert)
	viperInstance.Set(CommandTLSKeyKey, expected.TLS.Key)
	viperInstance.Set(CommandTLSCaKey, expected.TLS.Ca)
	viperInstance.Set(CommandTLSSkipVerifyKey, expected.TLS.SkipVerify)
	viperInstance.Set(CommandTLSServerNameKey, expected.TLS.ServerName)

	// root keys for sections are set
	assert.True(t, viperInstance.IsSet(CommandRootKey))
	assert.True(t, viperInstance.IsSet(CommandServerKey))
	assert.True(t, viperInstance.IsSet(CommandAuthKey))
	assert.True(t, viperInstance.IsSet(CommandTLSKey))

	result := getCommand()

	assert.Equal(t, expected.Server, result.Server)
	assert.Equal(t, expected.Auth, result.Auth)
	assert.Equal(t, expected.TLS, result.TLS)
}

func TestMissingServerTLS(t *testing.T) {
	viperInstance = viper.NewWithOptions(viper.KeyDelimiter(KeyDelimiter))

	expected := getAgentConfig().Command
	expected.TLS = nil

	viperInstance.Set(CommandServerHostKey, expected.Server.Host)
	viperInstance.Set(CommandServerPortKey, expected.Server.Port)
	viperInstance.Set(CommandServerTypeKey, expected.Server.Type)
	viperInstance.Set(CommandAuthTokenKey, expected.Auth.Token)

	assert.True(t, viperInstance.IsSet(CommandRootKey))
	assert.True(t, viperInstance.IsSet(CommandServerKey))
	assert.True(t, viperInstance.IsSet(CommandAuthKey))
	assert.False(t, viperInstance.IsSet(CommandTLSKey))

	result := getCommand()
	assert.Equal(t, expected.Server, result.Server)
	assert.Equal(t, expected.Auth, result.Auth)
	assert.Nil(t, result.TLS)
}

func getAgentConfig() *Config {
	return &Config{
		UUID:    "",
		Version: "",
		Path:    "",
		Log:     &Log{},
		Client: &Client{
			Timeout: 5 * time.Second,
		},
		ConfigDir:          "",
		AllowedDirectories: []string{},
		Metrics:            &Metrics{},
		Command: &Command{
			Server: &ServerConfig{
				Host: "127.0.0.1",
				Port: 8888,
				Type: "grpc",
			},
			Auth: &AuthConfig{
				Token: "1234",
			},
			TLS: &TLSConfig{
				Cert:       "some.cert",
				Key:        "some.key",
				Ca:         "some.ca",
				SkipVerify: false,
				ServerName: "server-name",
			},
		},
	}
}
