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
			"/etc/nginx", "/usr/local/etc/nginx", "/var/run/nginx", "/usr/share/nginx/modules", "/var/log/nginx",
		},
		DataPlaneConfig: &DataPlaneConfig{
			Nginx: &NginxDataPlaneConfig{
				ExcludeLogs:            []string{"/var/log/nginx/error.log", "/var/log/nginx/access.log"},
				ReloadMonitoringPeriod: 30 * time.Second,
				TreatWarningsAsErrors:  true,
			},
		},
		Collector: &Collector{
			ConfigPath: "/etc/nginx-agent/nginx-agent-otelcol.yaml",
			Exporters: Exporters{
				OtlpExporters: []OtlpExporter{
					{
						Server: &ServerConfig{
							Host: "127.0.0.1",
							Port: 5643,
							Type: 0,
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
						Type: 0,
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
				Batch: &Batch{
					SendBatchMaxSize: 1,
					SendBatchSize:    8199,
					Timeout:          30 * time.Second,
				},
				Attribute: &Attribute{
					Actions: []Action{
						{
							Key:    "test",
							Action: "insert",
							Value:  "value",
						},
					},
				},
			},
			Receivers: Receivers{
				OtlpReceivers: []OtlpReceiver{
					{
						Server: &ServerConfig{
							Host: "127.0.0.1",
							Port: 4317,
							Type: 0,
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
				NginxReceivers: []NginxReceiver{
					{
						InstanceID: "cd7b8911-c2c5-4daf-b311-dbead151d938",
						AccessLogs: []AccessLog{
							{
								LogFormat: "$remote_addr - $remote_user [$time_local] \"$request\"" +
									" $status $body_bytes_sent \"$http_referer\" \"$http_user_agent\" " +
									"\"$http_x_forwarded_for\"",
								FilePath: "/var/log/nginx/access-custom.conf",
							},
						},
					},
				},
				NginxPlusReceivers: []NginxPlusReceiver{
					{
						InstanceID: "cd7b8911-c2c5-4daf-b311-dbead151d939",
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
						Type: 0,
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
					},
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
		Watchers: &Watchers{
			InstanceWatcher: InstanceWatcher{
				MonitoringFrequency: 10 * time.Second,
			},
			InstanceHealthWatcher: InstanceHealthWatcher{
				MonitoringFrequency: 10 * time.Second,
			},
			FileWatcher: FileWatcher{
				MonitoringFrequency: 10 * time.Second,
			},
		},
	}
}
