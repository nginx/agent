// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.
package config

import (
	"os"
	"path"
	"strings"
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
	require.NoError(t, err)
	defer helpers.RemoveFileWithErrorCheck(t, file.Name())

	currentDirectory, err := os.Getwd()
	require.NoError(t, err)

	err = RegisterConfigFile()

	require.NoError(t, err)
	assert.Equal(t, path.Join(currentDirectory, "nginx-agent.conf"), viperInstance.GetString(ConfigPathKey))
	assert.NotEmpty(t, viperInstance.GetString(UUIDKey))
}

func TestResolveConfig(t *testing.T) {
	allowedDir := []string{
		"/etc/nginx", "/usr/local/etc/nginx", "/var/run/nginx",
		"/usr/share/nginx/modules", "/var/log/nginx",
	}
	viperInstance = viper.NewWithOptions(viper.KeyDelimiter(KeyDelimiter))
	err := loadPropertiesFromFile("./testdata/nginx-agent.conf")
	require.NoError(t, err)

	// Ensure viper instance has populated values based on config file before resolving to struct.
	assert.True(t, viperInstance.IsSet(CollectorRootKey))
	assert.True(t, viperInstance.IsSet(CollectorConfigPathKey))
	assert.True(t, viperInstance.IsSet(CollectorExportersKey))
	assert.True(t, viperInstance.IsSet(CollectorProcessorsKey))
	assert.True(t, viperInstance.IsSet(CollectorReceiversKey))
	assert.True(t, viperInstance.IsSet(CollectorExtensionsKey))

	actual, err := ResolveConfig()
	require.NoError(t, err)

	assert.Equal(t, "debug", actual.Log.Level)
	assert.Equal(t, "./", actual.Log.Path)

	assert.Equal(t, 30*time.Second, actual.DataPlaneConfig.Nginx.ReloadMonitoringPeriod)
	assert.False(t, actual.DataPlaneConfig.Nginx.TreatWarningsAsErrors)
	assert.Equal(t, []string{"/var/log/nginx/error.log", "/var/log/nginx/access.log"},
		actual.DataPlaneConfig.Nginx.ExcludeLogs)

	require.NotNil(t, actual.Collector)
	assert.Equal(t, "/etc/nginx-agent/nginx-agent-otelcol.yaml", actual.Collector.ConfigPath)
	assert.NotEmpty(t, actual.Collector.Receivers)
	assert.Equal(t, Processors{Batch: &Batch{}}, actual.Collector.Processors)
	assert.NotEmpty(t, actual.Collector.Exporters)
	assert.NotEmpty(t, actual.Collector.Extensions)

	// Client GRPC Settings
	assert.Equal(t, 15*time.Second, actual.Client.Grpc.KeepAlive.Timeout)
	assert.Equal(t, 10*time.Second, actual.Client.Grpc.KeepAlive.Time)
	assert.False(t, actual.Client.Grpc.KeepAlive.PermitWithoutStream)
	assert.Equal(t, 1048575, actual.Client.Grpc.MaxMessageSize)
	assert.Equal(t, 1048575, actual.Client.Grpc.MaxMessageReceiveSize)
	assert.Equal(t, 1048575, actual.Client.Grpc.MaxMessageSendSize)

	// Client HTTP Settings
	assert.Equal(t, 15*time.Second, actual.Client.HTTP.Timeout)

	// Client Backoff Settings
	assert.Equal(t, 200*time.Millisecond, actual.Client.Backoff.InitialInterval)
	assert.Equal(t, 10*time.Second, actual.Client.Backoff.MaxInterval)
	assert.Equal(t, 25*time.Second, actual.Client.Backoff.MaxElapsedTime)
	assert.InDelta(t, 1.5, actual.Client.Backoff.RandomizationFactor, 0.01)
	assert.InDelta(t, 2.5, actual.Client.Backoff.Multiplier, 0.01)

	assert.Equal(t,
		allowedDir,
		actual.AllowedDirectories,
	)

	assert.Equal(t, allowedDir, actual.AllowedDirectories)

	assert.Equal(t, 5*time.Second, actual.Watchers.InstanceWatcher.MonitoringFrequency)
	assert.Equal(t, 5*time.Second, actual.Watchers.InstanceHealthWatcher.MonitoringFrequency)
	assert.Equal(t, 5*time.Second, actual.Watchers.FileWatcher.MonitoringFrequency)
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
	t.Setenv("NGINX_AGENT_CLIENT_GRPC_KEEPALIVE_TIMEOUT", "10s")
	registerFlags()

	assert.Equal(t, "warn", viperInstance.GetString(LogLevelKey))
	assert.Equal(t, "/var/log/test/agent.log", viperInstance.GetString(LogPathKey))
	assert.Equal(t, 10*time.Second, viperInstance.GetDuration(ClientKeepAliveTimeoutKey))

	checkDefaultsClientValues(t, viperInstance)
}

func checkDefaultsClientValues(t *testing.T, viperInstance *viper.Viper) {
	t.Helper()

	assert.Equal(t, DefHTTPTimeout, viperInstance.GetDuration(ClientHTTPTimeoutKey))

	assert.Equal(t, DefBackoffInitialInterval, viperInstance.GetDuration(ClientBackoffInitialIntervalKey))
	assert.Equal(t, DefBackoffMaxInterval, viperInstance.GetDuration(ClientBackoffMaxIntervalKey))
	assert.InDelta(t, DefBackoffRandomizationFactor, viperInstance.GetFloat64(ClientBackoffRandomizationFactorKey),
		0.01)
	assert.InDelta(t, DefBackoffMultiplier, viperInstance.GetFloat64(ClientBackoffMultiplierKey), 0.01)
	assert.Equal(t, DefBackoffMaxElapsedTime, viperInstance.GetDuration(ClientBackoffMaxElapsedTimeKey))

	assert.Equal(t, DefGRPCKeepAliveTimeout, viperInstance.GetDuration(ClientKeepAliveTimeoutKey))
	assert.Equal(t, DefGRPCKeepAliveTime, viperInstance.GetDuration(ClientKeepAliveTimeKey))
	assert.Equal(t, DefGRPCKeepAlivePermitWithoutStream, viperInstance.GetBool(ClientKeepAlivePermitWithoutStreamKey))

	assert.Equal(t, DefMaxMessageSize, viperInstance.GetInt(ClientGRPCMaxMessageSizeKey))
	assert.Equal(t, DefMaxMessageRecieveSize, viperInstance.GetInt(ClientGRPCMaxMessageReceiveSizeKey))
	assert.Equal(t, DefMaxMessageSendSize, viperInstance.GetInt(ClientGRPCMaxMessageSendSizeKey))
	assert.Equal(t, make(map[string]string), viperInstance.GetStringMapString(LabelsRootKey))
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

	assert.Equal(t, 15*time.Second, viperInstance.GetDuration(ClientKeepAliveTimeoutKey))

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
	viperInstance.Set(ClientKeepAliveTimeoutKey, time.Hour)

	result := resolveClient()
	assert.Equal(t, time.Hour, result.Grpc.KeepAlive.Timeout)
}

func TestResolveCollector(t *testing.T) {
	testDefault := getAgentConfig()

	t.Run("Test 1: Happy path", func(t *testing.T) {
		expected := testDefault.Collector

		viperInstance = viper.NewWithOptions(viper.KeyDelimiter(KeyDelimiter))
		viperInstance.Set(CollectorConfigPathKey, expected.ConfigPath)
		viperInstance.Set(CollectorLogPathKey, expected.Log.Path)
		viperInstance.Set(CollectorLogLevelKey, expected.Log.Level)
		viperInstance.Set(CollectorReceiversKey, expected.Receivers)
		viperInstance.Set(CollectorBatchProcessorKey, expected.Processors.Batch)
		viperInstance.Set(CollectorBatchProcessorSendBatchSizeKey, expected.Processors.Batch.SendBatchSize)
		viperInstance.Set(CollectorBatchProcessorSendBatchMaxSizeKey, expected.Processors.Batch.SendBatchMaxSize)
		viperInstance.Set(CollectorBatchProcessorTimeoutKey, expected.Processors.Batch.Timeout)
		viperInstance.Set(CollectorExportersKey, expected.Exporters)
		viperInstance.Set(CollectorOtlpExportersKey, expected.Exporters.OtlpExporters)
		viperInstance.Set(CollectorExtensionsHealthServerHostKey, expected.Extensions.Health.Server.Host)
		viperInstance.Set(CollectorExtensionsHealthServerPortKey, expected.Extensions.Health.Server.Port)
		viperInstance.Set(CollectorExtensionsHealthPathKey, expected.Extensions.Health.Path)

		actual, err := resolveCollector(testDefault.AllowedDirectories)
		require.NoError(t, err)
		assert.Equal(t, expected, actual)
	})

	t.Run("Test 2: Non allowed path", func(t *testing.T) {
		expected := &Collector{
			ConfigPath: "/path/to/secret",
		}
		errMsg := "collector path /path/to/secret not allowed"

		viperInstance = viper.NewWithOptions(viper.KeyDelimiter(KeyDelimiter))
		viperInstance.Set(CollectorConfigPathKey, expected.ConfigPath)

		_, err := resolveCollector(testDefault.AllowedDirectories)

		require.Error(t, err)
		assert.Contains(t, err.Error(), errMsg)
	})
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

	viperInstance.Set(CommandServerHostKey, expected.Server.Host)
	viperInstance.Set(CommandServerPortKey, expected.Server.Port)
	viperInstance.Set(CommandServerTypeKey, expected.Server.Type)
	viperInstance.Set(CommandAuthTokenKey, expected.Auth.Token)

	assert.True(t, viperInstance.IsSet(CommandRootKey))
	assert.True(t, viperInstance.IsSet(CommandServerKey))
	assert.True(t, viperInstance.IsSet(CommandAuthKey))

	result := resolveCommand()
	assert.Equal(t, expected.Server, result.Server)
	assert.Equal(t, expected.Auth, result.Auth)
	assert.Nil(t, result.TLS)
}

func TestClient(t *testing.T) {
	viperInstance = viper.NewWithOptions(viper.KeyDelimiter(KeyDelimiter))
	expected := getAgentConfig().Client

	viperInstance.Set(ClientGRPCMaxMessageSizeKey, expected.Grpc.MaxMessageSize)
	viperInstance.Set(ClientKeepAlivePermitWithoutStreamKey, expected.Grpc.KeepAlive.PermitWithoutStream)
	viperInstance.Set(ClientKeepAliveTimeKey, expected.Grpc.KeepAlive.Time)
	viperInstance.Set(ClientKeepAliveTimeoutKey, expected.Grpc.KeepAlive.Timeout)

	viperInstance.Set(ClientHTTPTimeoutKey, expected.HTTP.Timeout)

	viperInstance.Set(ClientBackoffMaxIntervalKey, expected.Backoff.MaxInterval)
	viperInstance.Set(ClientBackoffMultiplierKey, expected.Backoff.Multiplier)
	viperInstance.Set(ClientBackoffMaxElapsedTimeKey, expected.Backoff.MaxElapsedTime)
	viperInstance.Set(ClientBackoffInitialIntervalKey, expected.Backoff.InitialInterval)
	viperInstance.Set(ClientBackoffRandomizationFactorKey, expected.Backoff.RandomizationFactor)

	// root keys for sections are set appropriately
	assert.True(t, viperInstance.IsSet(ClientGRPCMaxMessageSizeKey))
	assert.False(t, viperInstance.IsSet(ClientGRPCMaxMessageReceiveSizeKey))
	assert.False(t, viperInstance.IsSet(ClientGRPCMaxMessageSendSizeKey))

	viperInstance.Set(ClientGRPCMaxMessageReceiveSizeKey, expected.Grpc.MaxMessageReceiveSize)
	viperInstance.Set(ClientGRPCMaxMessageSendSizeKey, expected.Grpc.MaxMessageSendSize)

	result := resolveClient()

	assert.Equal(t, expected, result)
}

func TestResolveLabels(t *testing.T) {
	// Helper to set up the viper instance
	setupViper := func(input map[string]string) {
		viperInstance = viper.New() // Create a new viper instance for isolation
		viperInstance.Set(LabelsRootKey, input)
	}

	tests := []struct {
		input    map[string]string
		expected map[string]interface{}
		name     string
	}{
		{
			name: "Test 1: Integer values",
			input: map[string]string{
				"key1": "123",
				"key2": "456",
			},
			expected: map[string]interface{}{
				"key1": 123,
				"key2": 456,
			},
		},
		{
			name: "Test 2: Float values",
			input: map[string]string{
				"key1": "123.45",
				"key2": "678.90",
			},
			expected: map[string]interface{}{
				"key1": 123.45,
				"key2": 678.9,
			},
		},
		{
			name: "Test 3: Boolean values",
			input: map[string]string{
				"key1": "true",
				"key2": "false",
			},
			expected: map[string]interface{}{
				"key1": true,
				"key2": false,
			},
		},
		{
			name: "Test 4: Mixed types",
			input: map[string]string{
				"key1": "true",
				"key2": "123",
				"key3": "45.67",
				"key4": "hello",
			},
			expected: map[string]interface{}{
				"key1": true,
				"key2": 123,
				"key3": 45.67,
				"key4": "hello",
			},
		},
		{
			name: "Test 5: String values",
			input: map[string]string{
				"key1": "hello",
				"key2": "world",
			},
			expected: map[string]interface{}{
				"key1": "hello",
				"key2": "world",
			},
		},
		{
			name:     "Test 6: Empty input",
			input:    make(map[string]string),
			expected: make(map[string]interface{}),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup viper with test input
			setupViper(tt.input)

			// Call the function
			actual := resolveLabels()

			// Assert the results
			assert.Equal(t, tt.expected, actual)
		})
	}
}

func TestResolveLabelsWithYAML(t *testing.T) {
	tests := []struct {
		expected  map[string]interface{}
		name      string
		yamlInput string
	}{
		{
			name: "Test 1: Integer and Float Values",
			yamlInput: `
labels:
  key1: "123"
  key2: "45.67"
`,
			expected: map[string]interface{}{
				"key1": 123,
				"key2": 45.67,
			},
		},
		{
			name: "Test 2: Boolean Values",
			yamlInput: `
labels:
  key1: "true"
  key2: "false"
`,
			expected: map[string]interface{}{
				"key1": true,
				"key2": false,
			},
		},
		{
			name: "Test 3: Nil and Empty Values",
			yamlInput: `
labels:
  key1: "nil"
  key2: ""
`,
			expected: map[string]interface{}{
				"key1": nil,
				"key2": nil,
			},
		},
		{
			name: "Test 4: Array Values",
			yamlInput: `
labels:
  key1: "[1, 2, 3]"
`,
			expected: map[string]interface{}{
				"key1": []interface{}{float64(1), float64(2), float64(3)},
			},
		},
		{
			name: "Test 5: Nested JSON Object",
			yamlInput: `
labels:
  key1: '{"a": 1, "b": 2}'
`,
			expected: map[string]interface{}{
				"key1": map[string]interface{}{
					"a": float64(1),
					"b": float64(2),
				},
			},
		},
		{
			name: "Test 6: Plain Strings",
			yamlInput: `
labels:
  key1: "hello"
  key2: "world"
`,
			expected: map[string]interface{}{
				"key1": "hello",
				"key2": "world",
			},
		},
		{
			name: "Test 7: Specific Strings Example",
			yamlInput: `
labels:
  config-sync-group: "group1"
`,
			expected: map[string]interface{}{
				"config-sync-group": "group1",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set up viper with YAML input
			viperInstance = viper.New() // Create a new viper instance for isolation
			viperInstance.SetConfigType("yaml")

			err := viperInstance.ReadConfig(strings.NewReader(tt.yamlInput))
			require.NoError(t, err, "Error reading YAML input")

			// Call the function
			actual := resolveLabels()

			// Assert the results
			assert.Equal(t, tt.expected, actual)
		})
	}
}

func TestParseInt(t *testing.T) {
	tests := []struct {
		expected interface{}
		name     string
		input    string
	}{
		{name: "Test 1: Valid Integer", input: "123", expected: 123},
		{name: "Test 2: Negative Integer", input: "-456", expected: -456},
		{name: "Test 3: Zero", input: "0", expected: 0},
		{name: "Test 4: Invalid Integer", input: "abc", expected: nil},
		{name: "Test 5: Empty String", input: "", expected: nil},
		{name: "Test 6: Float String", input: "45.67", expected: nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseInt(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestParseFloat(t *testing.T) {
	tests := []struct {
		expected interface{}
		name     string
		input    string
	}{
		{name: "Test 1: Valid Float", input: "45.67", expected: 45.67},
		{name: "Test 2: Negative Float", input: "-123.45", expected: -123.45},
		{name: "Test 3: Valid Integer as Float", input: "123", expected: 123.0},
		{name: "Test 4: Invalid Float", input: "abc", expected: nil},
		{name: "Test 5: Empty String", input: "", expected: nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseFloat(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestParseBool(t *testing.T) {
	tests := []struct {
		expected interface{}
		name     string
		input    string
	}{
		{name: "Test 1: True (lowercase)", input: "true", expected: true},
		{name: "Test 2: False (lowercase)", input: "false", expected: false},
		{name: "Test 3: True (uppercase)", input: "TRUE", expected: true},
		{name: "Test 4: False (uppercase)", input: "FALSE", expected: false},
		{name: "Test 5: Numeric True", input: "1", expected: true},
		{name: "Test 6: Numeric False", input: "0", expected: false},
		{name: "Test 7: Invalid Boolean", input: "abc", expected: nil},
		{name: "Test 8: Empty String", input: "", expected: nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseBool(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestParseJSON(t *testing.T) {
	tests := []struct {
		expected interface{}
		name     string
		input    string
	}{
		{
			name:  "Test 1: Valid JSON Object",
			input: `{"a": 1, "b": "text"}`,
			expected: map[string]interface{}{
				"a": float64(1),
				"b": "text",
			},
		},
		{
			name:     "Test 2: Valid JSON Array",
			input:    `[1, 2, 3]`,
			expected: []interface{}{float64(1), float64(2), float64(3)},
		},
		{
			name:  "Test 3: Nested JSON",
			input: `{"a": {"b": [1, 2, 3]}}`,
			expected: map[string]interface{}{
				"a": map[string]interface{}{"b": []interface{}{float64(1), float64(2), float64(3)}},
			},
		},
		{name: "Test 4: Invalid JSON", input: `{"a": 1,`, expected: nil},
		{name: "Test 5: Empty String", input: "", expected: nil},
		{name: "Test 6: Plain String", input: `"hello"`, expected: "hello"},
		{name: "Test 7: Number as JSON", input: "123", expected: float64(123)},
		{name: "Test 8: Boolean as JSON", input: "true", expected: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseJSON(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func getAgentConfig() *Config {
	return &Config{
		UUID:    "",
		Version: "",
		Path:    "",
		Log:     &Log{},
		Client: &Client{
			HTTP: &HTTP{
				Timeout: 10 * time.Second,
			},
			Grpc: &GRPC{
				KeepAlive: &KeepAlive{
					Timeout:             5 * time.Second,
					Time:                4 * time.Second,
					PermitWithoutStream: true,
				},
				MaxMessageSize:        1,
				MaxMessageReceiveSize: 20,
				MaxMessageSendSize:    40,
			},
			Backoff: &BackOff{
				InitialInterval:     500 * time.Millisecond,
				MaxInterval:         5 * time.Second,
				MaxElapsedTime:      30 * time.Second,
				RandomizationFactor: 0.5,
				Multiplier:          1.5,
			},
		},
		AllowedDirectories: []string{
			"/etc/nginx", "/usr/local/etc/nginx", "/var/run/nginx", "/var/log/nginx", "/usr/share/nginx/modules",
		},
		Collector: &Collector{
			ConfigPath: "/etc/nginx-agent/nginx-agent-otelcol.yaml",
			Exporters: Exporters{
				OtlpExporters: []OtlpExporter{
					{
						Server: &ServerConfig{
							Host: "127.0.0.1",
							Port: 1234,
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
			},
			Processors: Processors{
				Batch: &Batch{
					SendBatchMaxSize: DefCollectorBatchProcessorSendBatchMaxSize,
					SendBatchSize:    DefCollectorBatchProcessorSendBatchSize,
					Timeout:          DefCollectorBatchProcessorTimeout,
				},
			},
			Receivers: Receivers{
				OtlpReceivers: []OtlpReceiver{
					{
						Server: &ServerConfig{
							Host: "localhost",
							Port: 4317,
							Type: 0,
						},
						Auth: &AuthConfig{
							Token: "even-secreter-token",
						},
						OtlpTLSConfig: &OtlpTLSConfig{
							GenerateSelfSignedCert: false,
							Cert:                   "/path/to/server-cert.pem",
							Key:                    "/path/to/server-cert.pem",
							Ca:                     "/path/to/server-cert.pem",
							SkipVerify:             true,
							ServerName:             "local-data-plane-server",
						},
					},
				},
				NginxReceivers: []NginxReceiver{
					{
						InstanceID: "cd7b8911-c2c5-4daf-b311-dbead151d938",
						StubStatus: APIDetails{
							URL:    "http://localhost:4321/status",
							Listen: "",
						},
						AccessLogs: []AccessLog{
							{
								LogFormat: accessLogFormat,
								FilePath:  "/var/log/nginx/access-custom.conf",
							},
						},
					},
				},
			},
			Extensions: Extensions{
				Health: &Health{
					Server: &ServerConfig{
						Host: "localhost",
						Port: 1337,
						Type: 0,
					},
					Path: "/",
				},
			},
			Log: &Log{
				Level: "INFO",
				Path:  "/var/log/nginx-agent/opentelemetry-collector-agent.log",
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
		Labels: make(map[string]any),
	}
}
