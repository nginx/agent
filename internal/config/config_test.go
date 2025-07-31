// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.
package config

import (
	_ "embed"
	"errors"
	"os"
	"path"
	"sort"
	"strings"
	"testing"
	"time"

	conf "github.com/nginx/agent/v3/test/config"

	"github.com/nginx/agent/v3/pkg/config"

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

	_, err = file.WriteString("log:")
	require.NoError(t, err)

	currentDirectory, err := os.Getwd()
	require.NoError(t, err)

	err = RegisterConfigFile()

	require.NoError(t, err)
	assert.Equal(t, path.Join(currentDirectory, "nginx-agent.conf"), viperInstance.GetString(ConfigPathKey))
	assert.NotEmpty(t, viperInstance.GetString(UUIDKey))
}

func TestResolveConfig(t *testing.T) {
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
	sort.Slice(actual.Collector.Extensions.HeadersSetter.Headers, func(i, j int) bool {
		headers := actual.Collector.Extensions.HeadersSetter.Headers
		return headers[i].Key < headers[j].Key
	})
	assert.Equal(t, createConfig(), actual)
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
	assert.Equal(t, DefFileChunkSize, viperInstance.GetUint32(ClientGRPCFileChunkSizeKey))
	assert.Equal(t, DefMaxFileSize, viperInstance.GetUint32(ClientGRPCMaxFileSizeKey))
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

	result := configFilePaths()

	assert.Len(t, result, 2)
	assert.Equal(t, "/etc/nginx-agent/", result[0])
	assert.Equal(t, currentDirectory, result[1])
}

func TestLoadPropertiesFromFile(t *testing.T) {
	viperInstance = viper.NewWithOptions(viper.KeyDelimiter(KeyDelimiter))
	err := loadPropertiesFromFile("./testdata/nginx-agent.conf")
	require.NoError(t, err)

	assert.Equal(t, "debug", viperInstance.GetString(LogLevelKey))
	assert.Equal(t, "./test-path", viperInstance.GetString(LogPathKey))

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

func TestResolveAllowedDirectories(t *testing.T) {
	tests := []struct {
		name           string
		configuredDirs []string
		expected       []string
	}{
		{
			name:           "Test 1: Empty path",
			configuredDirs: []string{""},
			expected:       []string{"/etc/nginx-agent"},
		},
		{
			name:           "Test 2: Absolute path",
			configuredDirs: []string{"/etc/agent/"},
			expected:       []string{"/etc/nginx-agent", "/etc/agent"},
		},
		{
			name:           "Test 3: Absolute paths",
			configuredDirs: []string{"/etc/nginx/"},
			expected:       []string{"/etc/nginx-agent", "/etc/nginx"},
		},
		{
			name:           "Test 4: Absolute path with multiple slashes",
			configuredDirs: []string{"/etc///////////nginx-agent/"},
			expected:       []string{"/etc/nginx-agent"},
		},
		{
			name:           "Test 5: Absolute path with directory traversal",
			configuredDirs: []string{"/etc/nginx/../nginx-agent"},
			expected:       []string{"/etc/nginx-agent"},
		},
		{
			name:           "Test 6: Absolute path with repeat directory traversal",
			configuredDirs: []string{"/etc/nginx-agent/../../../../../nginx-agent"},
			expected:       []string{"/etc/nginx-agent"},
		},
		{
			name:           "Test 7: Absolute path with control characters",
			configuredDirs: []string{"/etc/nginx-agent/\\x08../tmp/"},
			expected:       []string{"/etc/nginx-agent"},
		},
		{
			name:           "Test 8: Absolute path with invisible characters",
			configuredDirs: []string{"/etc/nginx-agent/ㅤㅤㅤ/tmp/"},
			expected:       []string{"/etc/nginx-agent"},
		},
		{
			name:           "Test 9: Absolute path with escaped invisible characters",
			configuredDirs: []string{"/etc/nginx-agent/\\\\ㅤ/tmp/"},
			expected:       []string{"/etc/nginx-agent"},
		},
		{
			name: "Test 10: Mixed paths",
			configuredDirs: []string{
				"nginx-agent",
				"",
				"..",
				"/",
				"\\/",
				".",
				"/etc/nginx/",
			},
			expected: []string{"/etc/nginx-agent", "/etc/nginx"},
		},
		{
			name:           "Test 11: Relative path",
			configuredDirs: []string{"nginx-agent"},
			expected:       []string{"/etc/nginx-agent"},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			allowed := resolveAllowedDirectories(test.configuredDirs)
			assert.Equal(t, test.expected, allowed)
		})
	}
}

func TestResolveLog(t *testing.T) {
	viperInstance = viper.NewWithOptions(viper.KeyDelimiter(KeyDelimiter))

	tests := []struct {
		name             string
		logLevel         string
		logPath          string
		expectedLogPath  string
		expectedLogLevel string
	}{
		{
			name:             "Test 1: Log level set to info",
			logLevel:         "info",
			logPath:          "/var/log/test/test.log",
			expectedLogPath:  "/var/log/test/test.log",
			expectedLogLevel: "info",
		},
		{
			name:             "Test 2: Invalid log level set",
			logLevel:         "trace",
			logPath:          "/var/log/test/test.log",
			expectedLogPath:  "/var/log/test/test.log",
			expectedLogLevel: "info",
		},
		{
			name:             "Test 3: Log level set to debug",
			logLevel:         "debug",
			logPath:          "/var/log/test/test.log",
			expectedLogPath:  "/var/log/test/test.log",
			expectedLogLevel: "debug",
		},
		{
			name:             "Test 4: Log level set with capitalization",
			logLevel:         "DEBUG",
			logPath:          "./logs/nginx.log",
			expectedLogPath:  "./logs/nginx.log",
			expectedLogLevel: "DEBUG",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			viperInstance.Set(LogLevelKey, test.logLevel)
			viperInstance.Set(LogPathKey, test.logPath)

			result := resolveLog()
			assert.Equal(t, test.expectedLogLevel, result.Level)
			assert.Equal(t, test.expectedLogPath, result.Path)
		})
	}
}

func TestResolveClient(t *testing.T) {
	viperInstance = viper.NewWithOptions(viper.KeyDelimiter(KeyDelimiter))
	viperInstance.Set(ClientKeepAliveTimeoutKey, time.Hour)

	result := resolveClient()
	assert.Equal(t, time.Hour, result.Grpc.KeepAlive.Timeout)
}

func TestResolveCollector(t *testing.T) {
	testDefault := agentConfig()

	t.Run("Test 1: Happy path", func(t *testing.T) {
		expected := testDefault.Collector

		viperInstance = viper.NewWithOptions(viper.KeyDelimiter(KeyDelimiter))
		viperInstance.Set(CollectorConfigPathKey, expected.ConfigPath)
		viperInstance.Set(CollectorLogPathKey, expected.Log.Path)
		viperInstance.Set(CollectorLogLevelKey, expected.Log.Level)
		viperInstance.Set(CollectorReceiversKey, expected.Receivers)
		viperInstance.Set(CollectorProcessorsKey, expected.Processors)
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

func TestResolveCollectorLog(t *testing.T) {
	tests := []struct {
		name             string
		logLevel         string
		logPath          string
		agentLogLevel    string
		expectedLogPath  string
		expectedLogLevel string
	}{
		{
			name:             "Test 1: OTel Log Level Set In Config",
			logLevel:         "",
			logPath:          "/tmp/collector.log",
			agentLogLevel:    "debug",
			expectedLogPath:  "/tmp/collector.log",
			expectedLogLevel: "DEBUG",
		},
		{
			name:             "Test 2: Agent Log Level is Warn",
			logLevel:         "",
			logPath:          "/tmp/collector.log",
			agentLogLevel:    "warn",
			expectedLogPath:  "/tmp/collector.log",
			expectedLogLevel: "WARN",
		},
		{
			name:             "Test 3: OTel Log Level Set In Config",
			logLevel:         "INFO",
			logPath:          "/tmp/collector.log",
			agentLogLevel:    "debug",
			expectedLogPath:  "/tmp/collector.log",
			expectedLogLevel: "INFO",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			viperInstance = viper.NewWithOptions(viper.KeyDelimiter(KeyDelimiter))
			viperInstance.Set(CollectorLogPathKey, test.logPath)
			viperInstance.Set(LogLevelKey, test.agentLogLevel)

			if test.logLevel != "" {
				viperInstance.Set(CollectorLogLevelKey, test.logLevel)
			}

			log := resolveCollectorLog()

			assert.Equal(t, test.expectedLogLevel, log.Level)
			assert.Equal(t, test.expectedLogPath, log.Path)
		})
	}
}

func TestCommand(t *testing.T) {
	viperInstance = viper.NewWithOptions(viper.KeyDelimiter(KeyDelimiter))
	expected := agentConfig().Command

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

	expected := agentConfig().Command

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
	expected := agentConfig().Client

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

func TestValidateYamlFile(t *testing.T) {
	tests := []struct {
		expected error
		name     string
		input    string
	}{
		{
			name:     "Test 1: Valid NGINX Agent config file",
			input:    "testdata/nginx-agent.conf",
			expected: nil,
		},
		{
			name:     "Test 2: Invalid format NGINX Agent config file",
			input:    "testdata/invalid-format-nginx-agent.conf",
			expected: errors.New("[2:1] unknown field \"level\""),
		},
		{
			name:     "Test 3: Unknown field in NGINX Agent config file",
			input:    "testdata/unknown-field-nginx-agent.conf",
			expected: errors.New("[5:1] unknown field \"unknown_field\""),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := validateYamlFile(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestResolveExtensions(t *testing.T) {
	tests := []struct {
		name     string
		value    string
		value2   string
		path     string
		path2    string
		expected []string
	}{
		{
			name:     "Test 1: User includes a single value header only",
			value:    "super-secret-token",
			path:     "",
			expected: []string{"super-secret-token"},
		},
		{
			name:     "Test 2: User includes a single filepath header only",
			value:    "",
			path:     "testdata/nginx-token.crt",
			expected: []string{"super-secret-token"},
		},
		{
			name:     "Test 3: User includes both a single token and a single filepath header",
			value:    "very-secret-token",
			path:     "testdata/nginx-token.crt",
			expected: []string{"very-secret-token"},
		},
		{
			name:     "Test 4: User includes neither token nor filepath header",
			value:    "",
			path:     "",
			expected: []string{""},
		},
		{
			name:     "Test 5: User includes multiple headers",
			value:    "super-secret-token",
			value2:   "very-secret-token",
			path:     "",
			path2:    "",
			expected: []string{"super-secret-token", "very-secret-token"},
		},
	}

	viperInstance = viper.NewWithOptions(viper.KeyDelimiter(KeyDelimiter))
	tempDir := t.TempDir()
	var confContent []byte

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempFile := helpers.CreateFileWithErrorCheck(t, tempDir, "nginx-agent.conf")
			defer helpers.RemoveFileWithErrorCheck(t, tempFile.Name())

			if len(tt.expected) == 1 {
				confContent = []byte(conf.AgentConfigWithToken(tt.value, tt.path))
			} else {
				confContent = []byte(conf.AgentConfigWithMultipleHeaders(tt.value, tt.path, tt.value2, tt.path2))
			}

			_, writeErr := tempFile.Write(confContent)
			require.NoError(t, writeErr)

			err := loadPropertiesFromFile(tempFile.Name())
			require.NoError(t, err)

			extension := resolveExtensions()
			require.NotNil(t, extension)

			var result []string
			for _, header := range extension.HeadersSetter.Headers {
				result = append(result, header.Value)
			}

			assert.Equal(t, tt.expected, result)

			err = tempFile.Close()
			require.NoError(t, err)
		})
	}
}

func TestResolveExtensions_MultipleHeaders(t *testing.T) {
	tests := []struct {
		name     string
		token    string
		token2   string
		path     string
		path2    string
		expected string
	}{
		{
			name:     "Test 1: User includes a single value header only",
			token:    "super-secret-token",
			path:     "",
			expected: "super-secret-token",
		},
	}

	viperInstance = viper.NewWithOptions(viper.KeyDelimiter(KeyDelimiter))
	tempDir := t.TempDir()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempFile := helpers.CreateFileWithErrorCheck(t, tempDir, "nginx-agent.conf")
			defer helpers.RemoveFileWithErrorCheck(t, tempFile.Name())

			confContent := []byte(conf.AgentConfigWithToken(tt.token, tt.path))
			_, writeErr := tempFile.Write(confContent)
			require.NoError(t, writeErr)

			err := loadPropertiesFromFile(tempFile.Name())
			require.NoError(t, err)

			extension := resolveExtensions()
			require.NotNil(t, extension)
			assert.Equal(t, tt.expected, extension.HeadersSetter.Headers[0].Value)

			err = tempFile.Close()
			require.NoError(t, err)
		})
	}
}

func TestAddDefaultOtlpExporter(t *testing.T) {
	t.Run("Test 1: Command server only", func(t *testing.T) {
		collector := &Collector{}
		agentConfig := &Config{
			Command: &Command{
				Server: &ServerConfig{
					Host: "test.com",
					Port: 8080,
					Type: Grpc,
				},
				Auth: &AuthConfig{
					Token: "token",
				},
				TLS: &TLSConfig{
					SkipVerify: false,
				},
			},
		}

		addDefaultOtlpExporter(collector, agentConfig)

		assert.Equal(t, "test.com", collector.Exporters.OtlpExporters["default"].Server.Host)
		assert.Equal(t, 8080, collector.Exporters.OtlpExporters["default"].Server.Port)
		assert.False(t, collector.Exporters.OtlpExporters["default"].TLS.SkipVerify)
		assert.Equal(t, "headers_setter", collector.Exporters.OtlpExporters["default"].Authenticator)
		assert.Equal(t, "insert", collector.Extensions.HeadersSetter.Headers[0].Action)
		assert.Equal(t, "authorization", collector.Extensions.HeadersSetter.Headers[0].Key)
		assert.Equal(t, "token", collector.Extensions.HeadersSetter.Headers[0].Value)
	})

	t.Run("Test 2: Command and Auxiliary Command servers", func(t *testing.T) {
		collector := &Collector{}
		agentConfig := &Config{
			Command: &Command{
				Server: &ServerConfig{
					Host: "test.com",
					Port: 8080,
					Type: Grpc,
				},
				Auth: &AuthConfig{
					Token: "token",
				},
				TLS: &TLSConfig{
					SkipVerify: false,
				},
			},
			AuxiliaryCommand: &Command{
				Server: &ServerConfig{
					Host: "aux-test.com",
					Port: 9090,
					Type: Grpc,
				},
				Auth: &AuthConfig{
					Token: "aux-token",
				},
				TLS: &TLSConfig{
					SkipVerify: false,
				},
			},
		}

		addDefaultOtlpExporter(collector, agentConfig)

		assert.Equal(t, "aux-test.com", collector.Exporters.OtlpExporters["default"].Server.Host)
		assert.Equal(t, 9090, collector.Exporters.OtlpExporters["default"].Server.Port)
		assert.False(t, collector.Exporters.OtlpExporters["default"].TLS.SkipVerify)
		assert.Equal(t, "headers_setter", collector.Exporters.OtlpExporters["default"].Authenticator)
		assert.Equal(t, "insert", collector.Extensions.HeadersSetter.Headers[0].Action)
		assert.Equal(t, "authorization", collector.Extensions.HeadersSetter.Headers[0].Key)
		assert.Equal(t, "aux-token", collector.Extensions.HeadersSetter.Headers[0].Value)
	})
}

func agentConfig() *Config {
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
			"/etc/nginx/", "/etc/nginx-agent/", "/usr/local/etc/nginx/", "/var/run/nginx/", "/var/log/nginx/",
			"/usr/share/nginx/modules/", "/etc/app_protect/",
		},
		Collector: createDefaultCollectorConfig(),
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

func createConfig() *Config {
	return &Config{
		Log: &Log{
			Level: "debug",
			Path:  "./test-path",
		},
		Client: &Client{
			HTTP: &HTTP{
				Timeout: 15 * time.Second,
			},
			Grpc: &GRPC{
				KeepAlive: &KeepAlive{
					Timeout:             15 * time.Second,
					Time:                10 * time.Second,
					PermitWithoutStream: false,
				},
				MaxMessageSize:        1048575,
				MaxMessageReceiveSize: 1048575,
				MaxMessageSendSize:    1048575,
				MaxFileSize:           485753,
				FileChunkSize:         48575,
			},
			Backoff: &BackOff{
				InitialInterval:     200 * time.Millisecond,
				MaxInterval:         10 * time.Second,
				MaxElapsedTime:      25 * time.Second,
				RandomizationFactor: 1.5,
				Multiplier:          2.5,
			},
		},
		AllowedDirectories: []string{
			"/etc/nginx-agent", "/etc/nginx", "/usr/local/etc/nginx", "/var/run/nginx",
			"/usr/share/nginx/modules", "/var/log/nginx",
		},
		DataPlaneConfig: &DataPlaneConfig{
			Nginx: &NginxDataPlaneConfig{
				ExcludeLogs:            []string{"/var/log/nginx/error.log", "^/var/log/nginx/.*.log$"},
				ReloadMonitoringPeriod: 30 * time.Second,
				TreatWarningsAsErrors:  true,
			},
		},
		Collector: &Collector{
			ConfigPath: "/etc/nginx-agent/nginx-agent-otelcol.yaml",
			Exporters: Exporters{
				OtlpExporters: map[string]*OtlpExporter{
					"default": {
						Server: &ServerConfig{
							Host: "127.0.0.1",
							Port: 5643,
						},
						Authenticator: "test-saas-token",
						TLS: &TLSConfig{
							Cert:       "/path/to/server-cert.pem",
							Key:        "/path/to/server-key.pem",
							Ca:         "/path/to/server-cert.pem",
							SkipVerify: false,
							ServerName: "test-saas-server",
						},
					},
				},
				PrometheusExporter: &PrometheusExporter{
					Server: &ServerConfig{
						Host: "127.0.0.1",
						Port: 1235,
					},
					TLS: &TLSConfig{
						Cert:       "/path/to/server-cert.pem",
						Key:        "/path/to/server-key.pem",
						Ca:         "/path/to/server-cert.pem",
						SkipVerify: false,
						ServerName: "test-server",
					},
				},
				Debug: &DebugExporter{},
			},
			Processors: Processors{
				Batch: map[string]*Batch{
					"default": {
						SendBatchMaxSize: 1,
						SendBatchSize:    8199,
						Timeout:          30 * time.Second,
					},
					"default_metrics": {
						SendBatchMaxSize: 1000,
						SendBatchSize:    1000,
						Timeout:          30 * time.Second,
					},
					"default_logs": {
						SendBatchMaxSize: 100,
						SendBatchSize:    100,
						Timeout:          60 * time.Second,
					},
				},
				Attribute: map[string]*Attribute{
					"default": {
						Actions: []Action{
							{
								Key:    "test",
								Action: "insert",
								Value:  "value",
							},
						},
					},
				},
				LogsGzip: map[string]*LogsGzip{
					"default": {},
				},
			},
			Receivers: Receivers{
				OtlpReceivers: map[string]*OtlpReceiver{
					"default": {
						Server: &ServerConfig{
							Host: "127.0.0.1",
							Port: 4317,
						},
						Auth: &AuthConfig{
							Token: "secret-receiver-token",
						},
						OtlpTLSConfig: &OtlpTLSConfig{
							GenerateSelfSignedCert: false,
							Cert:                   "/tmp/cert.pem",
							Key:                    "/tmp/key.pem",
							Ca:                     "/tmp/ca.pem",
							SkipVerify:             true,
							ServerName:             "test-local-server",
						},
					},
				},
				HostMetrics: &HostMetrics{
					CollectionInterval: 10 * time.Second,
					InitialDelay:       2 * time.Second,
					Scrapers: &HostMetricsScrapers{
						CPU:        &CPUScraper{},
						Disk:       nil,
						Filesystem: nil,
						Memory:     nil,
						Network:    nil,
					},
				},
			},
			Extensions: Extensions{
				Health: &Health{
					Server: &ServerConfig{
						Host: "127.0.0.1",
						Port: 1337,
					},
					TLS: &TLSConfig{
						Cert:       "/path/to/server-cert.pem",
						Key:        "/path/to/server-key.pem",
						Ca:         "/path/to/server-ca.pem",
						SkipVerify: false,
						ServerName: "server-name",
					},
					Path: "/test",
				},
				HeadersSetter: &HeadersSetter{
					Headers: []Header{
						{
							Action: "action",
							Key:    "key",
							Value:  "value",
						},
						{
							Action: "insert",
							Key:    "label1",
							Value:  "label 1",
						},
						{
							Action: "insert",
							Key:    "label2",
							Value:  "new-value",
						},
					},
				},
			},
			Log: &Log{
				Level: "INFO",
				Path:  "/var/log/nginx-agent/opentelemetry-collector-agent.log",
			},
			Pipelines: Pipelines{
				Metrics: map[string]*Pipeline{
					"default": {
						Receivers:  []string{"host_metrics", "nginx_metrics"},
						Processors: []string{"batch/default_metrics"},
						Exporters:  []string{"otlp/default"},
					},
				},
				Logs: map[string]*Pipeline{
					"default": {
						Receivers:  []string{"tcplog/nginx_app_protect"},
						Processors: []string{"logsgzip/default", "batch/default_logs"},
						Exporters:  []string{"otlp/default"},
					},
				},
			},
		},
		Command: &Command{
			Server: &ServerConfig{
				Host: "127.0.0.1",
				Port: 8888,
				Type: Grpc,
			},
			Auth: &AuthConfig{
				Token:     "1234",
				TokenPath: "path/to/my_token",
			},
			TLS: &TLSConfig{
				Cert:       "some.cert",
				Key:        "some.key",
				Ca:         "some.ca",
				SkipVerify: false,
				ServerName: "server-name",
			},
		},
		AuxiliaryCommand: &Command{
			Server: &ServerConfig{
				Host: "second.management.plane",
				Port: 9999,
				Type: Grpc,
			},
			Auth: &AuthConfig{
				Token:     "1234",
				TokenPath: "path/to/my_token",
			},
			TLS: &TLSConfig{
				Cert:       "some.cert",
				Key:        "some.key",
				Ca:         "some.ca",
				SkipVerify: false,
				ServerName: "server-name",
			},
		},
		Watchers: &Watchers{
			InstanceWatcher: InstanceWatcher{
				MonitoringFrequency: 10 * time.Second,
			},
			InstanceHealthWatcher: InstanceHealthWatcher{
				MonitoringFrequency: 10 * time.Second,
			},
			FileWatcher: FileWatcher{
				MonitoringFrequency: 10 * time.Second,
				ExcludeFiles:        []string{"\\.*log$"},
			},
		},
		Labels: map[string]any{
			"label1": "label 1",
			"label2": "new-value",
			"label3": 123,
		},
		Features: []string{
			config.FeatureCertificates, config.FeatureFileWatcher, config.FeatureMetrics,
			config.FeatureAPIAction, config.FeatureLogsNap,
		},
	}
}

func createDefaultCollectorConfig() *Collector {
	return &Collector{
		ConfigPath: "/etc/nginx-agent/nginx-agent-otelcol.yaml",
		Exporters: Exporters{
			OtlpExporters: map[string]*OtlpExporter{
				"default": {
					Server: &ServerConfig{
						Host: "127.0.0.1",
						Port: 1234,
						Type: Grpc,
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
			Batch: map[string]*Batch{
				"default_logs": {
					SendBatchMaxSize: DefCollectorLogsBatchProcessorSendBatchMaxSize,
					SendBatchSize:    DefCollectorLogsBatchProcessorSendBatchSize,
					Timeout:          DefCollectorLogsBatchProcessorTimeout,
				},
			},
			LogsGzip: map[string]*LogsGzip{
				"default": {},
			},
		},
		Receivers: Receivers{
			OtlpReceivers: map[string]*OtlpReceiver{
				"default": {
					Server: &ServerConfig{
						Host: "localhost",
						Port: 4317,
						Type: Grpc,
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
				},
				Path: "/",
			},
		},
		Log: &Log{
			Level: "INFO",
			Path:  "/var/log/nginx-agent/opentelemetry-collector-agent.log",
		},
	}
}
