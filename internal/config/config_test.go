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

const accessLogFormat = `$remote_addr - $remote_user [$time_local] \"$request\" $status $body_bytes_sent ` +
	`\"$http_referer\" \"$http_user_agent\" \"$http_x_forwarded_for\"\"$upstream_cache_status\"`

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

func TestResolveConfig(t *testing.T) {
	allowedDir := []string{"/etc/nginx", "/usr/local/etc/nginx", "/var/run/nginx",
		"/usr/share/nginx/modules", "/var/log/nginx"}
	viperInstance = viper.NewWithOptions(viper.KeyDelimiter(KeyDelimiter))
	err := loadPropertiesFromFile("./testdata/nginx-agent.conf")
	require.NoError(t, err)

	// Ensure viper instance has populated values based on config file before resolving to struct.
	assert.True(t, viperInstance.IsSet(CollectorRootKey))
	assert.True(t, viperInstance.IsSet(CollectorConfigPathKey))
	assert.True(t, viperInstance.IsSet(CollectorExportersKey))
	assert.True(t, viperInstance.IsSet(CollectorProcessorsKey))
	assert.True(t, viperInstance.IsSet(CollectorReceiversKey))
	assert.True(t, viperInstance.IsSet(CollectorHealthKey))

	actual, err := ResolveConfig()
	require.NoError(t, err)

	assert.Equal(t, "debug", actual.Log.Level)
	assert.Equal(t, "./", actual.Log.Path)

	assert.Equal(t, 30*time.Second, actual.DataPlaneConfig.Nginx.ReloadMonitoringPeriod)
	assert.False(t, actual.DataPlaneConfig.Nginx.TreatWarningsAsError)

	require.NotNil(t, actual.Collector)
	assert.Equal(t, "/etc/nginx-agent/nginx-agent-otelcol.yaml", actual.Collector.ConfigPath)
	assert.NotEmpty(t, actual.Collector.Receivers)
	assert.NotEmpty(t, actual.Collector.Processors)
	assert.NotEmpty(t, actual.Collector.Exporters)
	assert.NotEmpty(t, actual.Collector.Health)

	assert.Equal(t, 10*time.Second, actual.Client.Timeout)

	assert.Equal(t,
		"/etc/nginx:/usr/local/etc/nginx:/var/run/nginx:/usr/share/nginx/modules:/var/log/nginx:invalid/path",
		actual.ConfigDir,
	)

	assert.Equal(t, allowedDir, actual.AllowedDirectories)
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

func TestResolveConfigFilePaths(t *testing.T) {
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

func TestResolveLog(t *testing.T) {
	viperInstance = viper.NewWithOptions(viper.KeyDelimiter(KeyDelimiter))
	viperInstance.Set(LogLevelKey, "error")
	viperInstance.Set(LogPathKey, "/var/log/test/test.log")

	result := resolveLog()
	assert.Equal(t, "error", result.Level)
	assert.Equal(t, "/var/log/test/test.log", result.Path)
}

func TestResolveClient(t *testing.T) {
	viperInstance = viper.NewWithOptions(viper.KeyDelimiter(KeyDelimiter))
	viperInstance.Set(ClientTimeoutKey, time.Hour)

	result := resolveClient()
	assert.Equal(t, time.Hour, result.Timeout)
}

func TestResolveCollector(t *testing.T) {
	testDefault := getAgentConfig()
	tests := []struct {
		expected  *Collector
		name      string
		errMsg    string
		shouldErr bool
	}{
		{
			name:     "Test 1: Happy path",
			expected: testDefault.Collector,
		},
		{
			name: "Test 2: Non allowed path",
			expected: &Collector{
				ConfigPath: "/path/to/secret",
			},
			shouldErr: true,
			errMsg:    "collector path /path/to/secret not allowed",
		},
		{
			name: "Test 3: Unsupported Exporter",
			expected: &Collector{
				ConfigPath: testDefault.Collector.ConfigPath,
				Exporters: []Exporter{
					{
						Type: "not-allowed",
					},
				},
			},
			shouldErr: true,
			errMsg:    "unsupported exporter type: not-allowed",
		},
		{
			name: "Test 4: Unsupported Processor",
			expected: &Collector{
				ConfigPath: testDefault.Collector.ConfigPath,
				Exporters:  testDefault.Collector.Exporters,
				Processors: []Processor{
					{
						Type: "custom-processor",
					},
				},
			},
			shouldErr: true,
			errMsg:    "unsupported processor type: custom-processor",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			viperInstance = viper.NewWithOptions(viper.KeyDelimiter(KeyDelimiter))
			viperInstance.Set(CollectorRootKey, "set")
			viperInstance.Set(CollectorConfigPathKey, test.expected.ConfigPath)
			viperInstance.Set(CollectorReceiversKey, test.expected.Receivers)
			viperInstance.Set(CollectorProcessorsKey, test.expected.Processors)
			viperInstance.Set(CollectorExportersKey, test.expected.Exporters)
			viperInstance.Set(CollectorHealthKey, test.expected.Health)

			actual, err := resolveCollector(testDefault.AllowedDirectories)
			if test.shouldErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), test.errMsg)
			} else {
				require.NoError(t, err)
				assert.Equal(t, test.expected, actual)
			}
		})
	}
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

	result := resolveCommand()

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

	result := resolveCommand()
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
		ConfigDir: "",
		AllowedDirectories: []string{
			"/etc/nginx", "/usr/local/etc/nginx", "/var/run/nginx", "/var/log/nginx", "/usr/share/nginx/modules",
		},
		Collector: &Collector{
			ConfigPath: "/etc/nginx-agent/nginx-agent-otelcol.yaml",
			Exporters: []Exporter{
				{
					Type: "otlp",
					Server: &ServerConfig{
						Host: "127.0.0.1",
						Port: 1234,
						Type: 0,
					},
					Auth: &AuthConfig{
						Token: "super-secret-token",
					},
					TLS: &TLSConfig{
						Cert:       "/path/to/server-cert.pem",
						Key:        "/path/to/server-cert.pem",
						Ca:         "/path/to/server-cert.pem",
						SkipVerify: true,
						ServerName: "remote-saas-server",
					},
				},
			},
			Processors: []Processor{
				{
					Type: "batch",
				},
			},
			Receivers: Receivers{
				OtlpReceivers: []OtlpReceiver{
					{
						Server: &ServerConfig{
							Host: "localhost",
							Port: 4321,
							Type: 0,
						},
						Auth: &AuthConfig{
							Token: "even-secreter-token",
						},
						TLS: &TLSConfig{
							Cert:       "/path/to/server-cert.pem",
							Key:        "/path/to/server-cert.pem",
							Ca:         "/path/to/server-cert.pem",
							SkipVerify: true,
							ServerName: "local-dataa-plane-server",
						},
					},
				},
				NginxReceivers: []NginxReceiver{
					{
						InstanceID: "cd7b8911-c2c5-4daf-b311-dbead151d938",
						StubStatus: "http://localhost:4321/status",
						AccessLogs: []AccessLog{
							{
								LogFormat: accessLogFormat,
								FilePath:  "/var/log/nginx/access-custom.conf",
							},
						},
					},
				},
			},
			Health: &ServerConfig{
				Host: "localhost",
				Port: 1337,
				Type: 0,
			},
		},
		Command: &Command{
			Server: &ServerConfig{
				Host: "127.0.0.1",
				Port: 8888,
				Type: Grpc,
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
