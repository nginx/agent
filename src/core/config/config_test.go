/**
 * Copyright (c) F5, Inc.
 *
 * This source code is licensed under the Apache License, Version 2.0 license found in the
 * LICENSE file in the root directory of this source tree.
 */

package config

import (
	"bytes"
	"fmt"
	"os"
	"reflect"
	"sort"
	"strings"
	"testing"

	agent_config "github.com/nginx/agent/sdk/v2/agent/config"
	sysutils "github.com/nginx/agent/v2/test/utils/system"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	updatedServerHost         = "192.168.0.1"
	updatedServerGrpcPort     = 11000
	updatedAgentAPIPort       = 9010
	updatedLogLevel           = "fatal"
	updatedLogPath            = "./test-path"
	updatedConfigDirs         = "/usr/local/etc/nginx"
	testCfgDir                = "../../plugins/testdata/configs"
	updateCfgFile             = "updated.conf"
	updatedDynamicFile        = "updated-dynamic.conf"
	tempDynamicCfgFile        = "temp-agent-dynamic.conf"
	tempCfgFile               = "temp-nginx-agent.conf"
	updatedTempDynamicCfgFile = "updated-temp-agent-dynamic.conf"
	updatedTempCfgFile        = "updated-temp-nginx-agent.conf"
	emptyConfigFile           = "empty_config.conf"
)

var (
	updatedConfTags = []string{"updated-locally-tagged", "updated-tagged-locally"}
	searchPaths     = []string{
		".",
		testCfgDir,
	}
)

func TestSeekConfigFileInPaths(t *testing.T) {
	tests := []struct {
		searchPaths []string
	}{
		{
			searchPaths: searchPaths,
		},
	}

	for _, test := range tests {
		_, err := SeekConfigFileInPaths(ConfigFileName, test.searchPaths...)
		assert.NoError(t, err, "SeekConfigFileInPaths returned error on config file and %v paths", test.searchPaths)
	}
}

func TestSeekConfigFileInPathsFail(t *testing.T) {
	tests := []struct {
		searchPaths []string
	}{
		{
			searchPaths: []string{},
		},
		{
			searchPaths: []string{
				"missing.conf",
				"/etcy/nginx-agent",
			},
		},
	}

	for _, test := range tests {
		result, err := SeekConfigFileInPaths(ConfigFileName, test.searchPaths...)
		assert.Error(t, err, "SeekConfigFileInPaths didn't return an error on config file and %v paths", test.searchPaths)
		assert.Empty(t, result)
	}
}

func TestDefaultConfig(t *testing.T) {
	configPath := "../../../nginx-agent.conf"

	t.Run("parsing of default config with dynamic config dir and file creation", func(t *testing.T) {
		tmpDynConfigDir := t.TempDir() + "/defaultConfigTest"
		defer os.RemoveAll(tmpDynConfigDir)
		dynConfigPath := fmt.Sprintf("%s/%s", tmpDynConfigDir, DynamicConfigFileName)
		setDynamicConfigFileAbsPath(dynConfigPath)
		assert.NoError(t, LoadPropertiesFromFile(configPath))
		assert.FileExists(t, dynConfigPath)
	})

	t.Run("parsing of default config and existing dynamic config", func(t *testing.T) {
		dynConfigPath := fmt.Sprintf("%s/%s", testCfgDir, DynamicConfigFileName)
		setDynamicConfigFileAbsPath(dynConfigPath)
		assert.NoError(t, LoadPropertiesFromFile(configPath))
	})
}

func TestGetConfig(t *testing.T) {
	// Get current directory
	curDir, err := os.Getwd()
	require.NoError(t, err)

	t.Run("no config file, no passed flags, defaults used", func(t *testing.T) {
		// Copy empty config file to current directory
		tempConfDeleteFunc, err := sysutils.CopyFile(fmt.Sprintf("%s/%s", testCfgDir, emptyConfigFile), tempCfgFile)
		defer func() {
			err := tempConfDeleteFunc()
			require.NoError(t, err, "deletion of temp config file failed")
		}()
		require.NoError(t, err)

		// Copy empty dynamic config file to current directory
		tempDynamicDeleteFunc, err := sysutils.CopyFile(fmt.Sprintf("%s/%s", testCfgDir, emptyConfigFile), tempDynamicCfgFile)
		defer func() {
			err := tempDynamicDeleteFunc()
			if err != nil {
				require.NoError(t, err, "deletion of temp dynamic config file failed")
			}
		}()
		require.NoError(t, err)

		// Initialize environment with the empty configs
		cleanEnv(t, tempCfgFile, fmt.Sprintf("%s/%s", curDir, tempDynamicCfgFile))

		config, err := GetConfig("12345")
		require.NoError(t, err)

		assert.Equal(t, Defaults.CloudAccountID, config.CloudAccountID)

		assert.Equal(t, Defaults.Log.Level, config.Log.Level)
		assert.Equal(t, "", config.Log.Path)

		assert.Equal(t, Defaults.Server.Host, config.Server.Host)
		assert.Equal(t, Defaults.Server.GrpcPort, config.Server.GrpcPort)
		assert.Equal(t, Defaults.Server.Command, config.Server.Command)
		assert.Equal(t, Defaults.Server.Metrics, config.Server.Metrics)

		assert.Equal(t, Defaults.AgentAPI.Port, config.AgentAPI.Port)
		assert.Equal(t, Defaults.AgentAPI.Host, config.AgentAPI.Host)

		assert.Equal(t, Defaults.Nginx.ConfigReloadMonitoringPeriod, config.Nginx.ConfigReloadMonitoringPeriod)

		assert.True(t, len(config.AllowedDirectoriesMap) > 0)
		assert.Equal(t, Defaults.ConfigDirs, config.ConfigDirs)
		assert.Equal(t, Defaults.TLS.Enable, config.TLS.Enable)

		assert.Equal(t, Defaults.Dataplane.Status.PollInterval, config.Dataplane.Status.PollInterval)

		assert.Equal(t, Defaults.AgentMetrics.BulkSize, config.AgentMetrics.BulkSize)
		assert.Equal(t, Defaults.AgentMetrics.ReportInterval, config.AgentMetrics.ReportInterval)
		assert.Equal(t, Defaults.AgentMetrics.CollectionInterval, config.AgentMetrics.CollectionInterval)

		assert.Equal(t, []string{}, config.Tags)
		assert.Equal(t, Defaults.Features, config.Features)
		assert.Equal(t, []string{}, config.Extensions)
	})

	t.Run("test override defaults with flags", func(t *testing.T) {
		// Copy empty config file to current directory
		tempConfDeleteFunc, err := sysutils.CopyFile(fmt.Sprintf("%s/%s", testCfgDir, emptyConfigFile), tempCfgFile)
		defer func() {
			err := tempConfDeleteFunc()
			require.NoError(t, err, "deletion of temp config file failed")
		}()
		require.NoError(t, err)

		// Copy empty dynamic config file to current directory
		tempDynamicDeleteFunc, err := sysutils.CopyFile(fmt.Sprintf("%s/%s", testCfgDir, emptyConfigFile), tempDynamicCfgFile)
		defer func() {
			err := tempDynamicDeleteFunc()
			require.NoError(t, err, "deletion of temp dynamic config file failed")
		}()
		require.NoError(t, err)

		// Initialize environment with the empty configs
		cleanEnv(t, tempCfgFile, fmt.Sprintf("%s/%s", curDir, tempDynamicCfgFile))

		updatedTag := "updated-tag"
		updatedLogLevel := "fatal"

		Viper.Set(LogLevel, updatedLogLevel)
		Viper.Set(DisplayNameKey, updatedTag)
		Viper.Set(TagsKey, []string{updatedTag})
		Viper.Set(DynamicConfigPathKey, tempDynamicCfgFile)

		config, err := GetConfig("23456")
		require.NoError(t, err)

		// Check for updated values
		assert.Equal(t, updatedLogLevel, config.Log.Level)
		assert.Equal(t, updatedTag, config.DisplayName)
		assert.Equal(t, []string{updatedTag}, config.Tags)
		assert.Equal(t, tempDynamicCfgFile, config.DynamicConfigPath)

		// Everything else should still be default
		assert.Equal(t, Defaults.Server.Host, config.Server.Host)
		assert.Equal(t, Defaults.Server.GrpcPort, config.Server.GrpcPort)
		assert.Equal(t, Defaults.AgentAPI.Port, config.AgentAPI.Port)
	})

	t.Run("test override defaults with config file values", func(t *testing.T) {
		// Copy empty config file to current directory
		tempConfDeleteFunc, err := sysutils.CopyFile(fmt.Sprintf("%s/%s", testCfgDir, emptyConfigFile), tempCfgFile)
		defer func() {
			err := tempConfDeleteFunc()
			require.NoError(t, err, "deletion of temp config file failed")
		}()
		require.NoError(t, err)

		// Copy empty dynamic config file to current directory
		tempDynamicDeleteFunc, err := sysutils.CopyFile(fmt.Sprintf("%s/%s", testCfgDir, emptyConfigFile), tempDynamicCfgFile)
		defer func() {
			err := tempDynamicDeleteFunc()
			require.NoError(t, err, "deletion of temp dynamic config file failed")
		}()
		require.NoError(t, err)

		// Initialize environment with the empty configs
		cleanEnv(t, tempCfgFile, fmt.Sprintf("%s/%s", curDir, tempDynamicCfgFile))

		// Copy config file with updated values to current directory
		updatedTempConfDeleteFunc, err := sysutils.CopyFile(fmt.Sprintf("%s/%s", testCfgDir, updateCfgFile), updatedTempCfgFile)
		defer func() {
			err := updatedTempConfDeleteFunc()
			require.NoError(t, err, "deletion of updated temp config file failed")
		}()
		require.NoError(t, err)

		// Copy dynamic config file with updated values to current directory
		updatedTempDynamicDeleteFunc, err := sysutils.CopyFile(fmt.Sprintf("%s/%s", testCfgDir, updatedDynamicFile), updatedTempDynamicCfgFile)
		defer func() {
			err := updatedTempDynamicDeleteFunc()
			require.NoError(t, err, "deletion of updated temp dynamic config file failed")
		}()
		require.NoError(t, err)

		testDynamicCfg := fmt.Sprintf("%s/%s", curDir, updatedTempDynamicCfgFile)
		setDynamicConfigFileAbsPath(testDynamicCfg)
		err = LoadPropertiesFromFile(updatedTempCfgFile)
		require.NoError(t, err)

		config, err := GetConfig("7890")
		require.NoError(t, err)

		assert.Equal(t, updatedServerHost, config.Server.Host)
		assert.Equal(t, updatedServerGrpcPort, config.Server.GrpcPort)
		assert.Equal(t, updatedAgentAPIPort, config.AgentAPI.Port)
		assert.Equal(t, updatedConfTags, config.Tags)

		// Check for updated values
		assert.Equal(t, updatedConfigDirs, config.ConfigDirs)
		assert.Equal(t, updatedLogLevel, config.Log.Level)
		assert.Equal(t, updatedLogPath, config.Log.Path)

		// Check TLS defaults
		assert.Equal(t, false, config.TLS.Enable)
		assert.Equal(t, "", config.TLS.Ca)
		assert.Equal(t, "", config.TLS.Cert)
		assert.Equal(t, "", config.TLS.Key)

		// Everything else should still be default
		assert.Equal(t, Defaults.AgentMetrics.Mode, config.AgentMetrics.Mode)
		assert.Equal(t, Defaults.Features, config.Features)
		assert.Equal(t, []string{}, config.Extensions)
	})

	t.Run("test override config values with ENV variables", func(t *testing.T) {
		// Copy sample config file to current directory
		tempConfDeleteFunc, err := sysutils.CopyFile(fmt.Sprintf("%s/%s", testCfgDir, ConfigFileName), tempCfgFile)
		defer func() {
			err := tempConfDeleteFunc()
			require.NoError(t, err, "deletion of temp config file failed")
		}()
		require.NoError(t, err)

		// Copy sample dynamic config file to current directory
		tempDynamicDeleteFunc, err := sysutils.CopyFile(fmt.Sprintf("%s/%s", testCfgDir, DynamicConfigFileName), tempDynamicCfgFile)
		defer func() {
			err := tempDynamicDeleteFunc()
			require.NoError(t, err, "deletion of temp dynamic config file failed")
		}()
		require.NoError(t, err)

		// Initialize environment with specified configs
		cleanEnv(t, tempCfgFile, fmt.Sprintf("%s/%s", curDir, tempDynamicCfgFile))

		envTags := "env tags"
		setEnvVariable(t, ServerHost, updatedServerHost)
		setEnvVariable(t, LogLevel, updatedLogLevel)
		setEnvVariable(t, LogPath, updatedLogPath)
		setEnvVariable(t, TagsKey, envTags)

		config, err := GetConfig("5678")
		require.NoError(t, err)

		// Check for updated values
		assert.Equal(t, updatedLogLevel, config.Log.Level)
		assert.Equal(t, updatedLogPath, config.Log.Path)
		assert.Equal(t, updatedServerHost, config.Server.Host)
		assert.Equal(t, []string{"env", "tags"}, config.Tags)

		// Everything else should still be default
		assert.Equal(t, Defaults.ConfigDirs, config.ConfigDirs)
	})

	t.Run("test override default values with ENV variables", func(t *testing.T) {
		// Copy empty config file to current directory
		tempConfDeleteFunc, err := sysutils.CopyFile(fmt.Sprintf("%s/%s", testCfgDir, emptyConfigFile), tempCfgFile)
		defer func() {
			err := tempConfDeleteFunc()
			require.NoError(t, err, "deletion of temp config file failed")
		}()
		require.NoError(t, err)

		// Copy empty dynamic config file to current directory
		tempDynamicDeleteFunc, err := sysutils.CopyFile(fmt.Sprintf("%s/%s", testCfgDir, emptyConfigFile), tempDynamicCfgFile)
		defer func() {
			err := tempDynamicDeleteFunc()
			require.NoError(t, err, "deletion of temp dynamic config file failed")
		}()
		require.NoError(t, err)

		// Initialize environment with the empty configs
		cleanEnv(t, tempCfgFile, fmt.Sprintf("%s/%s", curDir, tempDynamicCfgFile))

		envTags := "env tags"
		setEnvVariable(t, ServerHost, updatedServerHost)
		setEnvVariable(t, LogLevel, updatedLogLevel)
		setEnvVariable(t, LogPath, updatedLogPath)
		setEnvVariable(t, TagsKey, envTags)

		config, err := GetConfig("5678")
		require.NoError(t, err)

		// Check for updated values
		assert.Equal(t, updatedLogLevel, config.Log.Level)
		assert.Equal(t, updatedLogPath, config.Log.Path)
		assert.Equal(t, updatedServerHost, config.Server.Host)
		assert.Equal(t, []string{"env", "tags"}, config.Tags)

		// Everything else should still be default
		assert.Equal(t, Defaults.ConfigDirs, config.ConfigDirs)
	})

	t.Run("test reading extensions from config file", func(t *testing.T) {
		configData := `
extensions:
  - advanced-metrics
  - unknown-extension`
		err := os.WriteFile(tempCfgFile, []byte(configData), 0o644)
		require.NoError(t, err)
		defer os.Remove(tempCfgFile)

		// Copy sample dynamic config file to current directory
		tempDynamicDeleteFunc, err := sysutils.CopyFile(fmt.Sprintf("%s/%s", testCfgDir, DynamicConfigFileName), tempDynamicCfgFile)
		defer func() {
			err := tempDynamicDeleteFunc()
			require.NoError(t, err, "deletion of temp dynamic config file failed")
		}()
		require.NoError(t, err)

		// Initialize environment with specified configs
		cleanEnv(t, tempCfgFile, fmt.Sprintf("%s/%s", curDir, tempDynamicCfgFile))

		config, err := GetConfig("5678")
		require.NoError(t, err)

		// Check extensions value
		assert.Equal(t, []string{"advanced-metrics"}, config.Extensions)
		assert.Equal(t, agent_config.AdvancedMetricsExtensionPlugin, config.Extensions[0])
	})
}

func TestUpdateAgentConfig(t *testing.T) {
	// Get current directory
	curDir, err := os.Getwd()
	require.NoError(t, err)

	// Copy initial dynamic config file to current directory
	tempDynamicDeleteFunc, err := sysutils.CopyFile(fmt.Sprintf("%s/%s", testCfgDir, DynamicConfigFileName), tempDynamicCfgFile)
	defer func() {
		err := tempDynamicDeleteFunc()
		require.NoError(t, err, "deletion of temp dynamic config file failed")
	}()
	require.NoError(t, err)

	cleanEnv(t, "empty_config.conf", fmt.Sprintf("%s/%s", curDir, tempDynamicCfgFile))

	// Get the current config so we can correctly set a few testcase variables
	curConf, err := GetConfig("12345")
	require.NoError(t, err, "failed to load config")

	testCases := []struct {
		testName            string
		updatedConfTags     []string
		updatedConfFeatures []string
		expConfTags         []string
		expConfFeatures     []string
		updatedConf         bool
	}{
		{
			testName:            "NoFieldsInConfToUpdate",
			updatedConfTags:     curConf.Tags,
			updatedConfFeatures: curConf.Features,
			expConfTags:         curConf.Tags,
			expConfFeatures:     curConf.Features,
			updatedConf:         false,
		},
		{
			testName:            "UpdatedTags",
			updatedConfTags:     []string{"new-tag1:One", "new-tag2:Two"},
			updatedConfFeatures: curConf.Features,
			expConfTags:         []string{"new-tag1:One", "new-tag2:Two"},
			expConfFeatures:     curConf.Features,
			updatedConf:         true,
		},
		{
			testName:            "RemoveAllTags",
			updatedConfTags:     []string{},
			updatedConfFeatures: curConf.Features,
			expConfTags:         []string{},
			expConfFeatures:     curConf.Features,
			updatedConf:         true,
		},
		{
			testName:            "UpdateFeatures",
			updatedConfTags:     curConf.Tags,
			updatedConfFeatures: []string{"registration", "nginx-config", "metrics"},
			expConfTags:         curConf.Tags,
			expConfFeatures:     []string{"registration", "nginx-config", "metrics"},
			updatedConf:         true,
		},
		{
			testName:            "RemoveAllFeatures",
			updatedConfTags:     curConf.Tags,
			updatedConfFeatures: []string{},
			expConfTags:         curConf.Tags,
			expConfFeatures:     []string{},
			updatedConf:         true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.testName, func(t *testing.T) {
			// Attempt update & check results
			updated, err := UpdateAgentConfig("12345", tc.updatedConfTags, tc.updatedConfFeatures)
			assert.NoError(t, err)
			assert.Equal(t, updated, tc.updatedConf)

			// Get potentially updated config
			updatedConf, err := GetConfig("12345")
			assert.NoError(t, err)
			if updated {
				assert.NotEqual(t, curConf, updatedConf)
			}

			// Sort tags before asserting
			sort.Strings(tc.expConfTags)
			sort.Strings(updatedConf.Tags)
			equalTags := reflect.DeepEqual(tc.expConfTags, updatedConf.Tags)

			assert.Equal(t, equalTags, true)
			// Sort features before asserting
			sort.Strings(tc.expConfFeatures)
			sort.Strings(updatedConf.Features)
			equalFeatures := reflect.DeepEqual(tc.expConfFeatures, updatedConf.Features)
			assert.Equal(t, equalFeatures, true)
		})
	}
}

func setEnvVariable(t *testing.T, name string, value string) {
	key := strings.ToUpper(EnvPrefix + agent_config.KeyDelimiter + name)
	err := os.Setenv(key, value)
	require.NoError(t, err)
}

func cleanEnv(t *testing.T, confFileName, dynamicConfFileAbsPath string) {
	os.Clearenv()
	ROOT_COMMAND.ResetFlags()
	ROOT_COMMAND.ResetCommands()
	Viper = viper.NewWithOptions(viper.KeyDelimiter(agent_config.KeyDelimiter))
	SetDefaults()
	RegisterFlags()

	cfg, err := RegisterConfigFile(dynamicConfFileAbsPath, confFileName, searchPaths...)
	require.NoError(t, err)

	err = LoadPropertiesFromFile(cfg)
	require.NoError(t, err)

	Viper.Set(ConfigPathKey, cfg)
}

func TestRemoveFeatures(t *testing.T) {
	tests := []struct {
		name               string
		input              string
		want               string
		wantFeaturesAreSet bool
	}{
		{
			name: "default dyn config. unchanged",
			input: `# Dynamic configuration file for NGINX Agent.
#
# The purpose of this file is to track agent configuration
# values that can be dynamically changed via the API and the agent install script.
# You may edit this file, but API calls that modify the tags on this system will
# overwrite the tag values in this file.
#
# The agent configuration values that API calls can modify are as follows:
#    - tags
#
# The agent configuration values that the agent install script can modify are as follows:
#    - instance_group`,
			want: `# Dynamic configuration file for NGINX Agent.
#
# The purpose of this file is to track agent configuration
# values that can be dynamically changed via the API and the agent install script.
# You may edit this file, but API calls that modify the tags on this system will
# overwrite the tag values in this file.
#
# The agent configuration values that API calls can modify are as follows:
#    - tags
#
# The agent configuration values that the agent install script can modify are as follows:
#    - instance_group
`,
			wantFeaturesAreSet: false,
		},
		{
			name: "dyn conf with features enabled",
			input: `# Dynamic configuration file for NGINX Agent.
#
# The purpose of this file is to track agent configuration
# values that can be dynamically changed via the API and the agent install script.
# You may edit this file, but API calls that modify the tags on this system will
# overwrite the tag values in this file.
#
# The agent configuration values that API calls can modify are as follows:
#    - tags
#
# The agent configuration values that the agent install script can modify are as follows:
#    - instance_group

features:
	- features_activity-events
	- features_agent-api
	- features_dataplane-status
	- features_file-watcher
	- features_metrics-throttle
	- features_nginx-config-async
	- features_nginx-counting
	- features_nginx-ssl-config
	- features_process-watcher
	- features_registration
`,
			want: `# Dynamic configuration file for NGINX Agent.
#
# The purpose of this file is to track agent configuration
# values that can be dynamically changed via the API and the agent install script.
# You may edit this file, but API calls that modify the tags on this system will
# overwrite the tag values in this file.
#
# The agent configuration values that API calls can modify are as follows:
#    - tags
#
# The agent configuration values that the agent install script can modify are as follows:
#    - instance_group


`,
			wantFeaturesAreSet: true,
		},
		{
			name: "dyn conf with features enabled and tags after",
			input: `# Dynamic configuration file for NGINX Agent.

features:
	- features_activity-events
	- features_process-watcher
	- features_registration

tags:
	- tag1
	- tag2
`,
			want: `# Dynamic configuration file for NGINX Agent.


tags:
	- tag1
	- tag2
`,
			wantFeaturesAreSet: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			inputBuf := bytes.NewBufferString(tt.input)

			featuresAreSet, got, err := removeFeatures(inputBuf)
			assert.Equal(t, tt.want, string(got))
			assert.Equal(t, tt.wantFeaturesAreSet, featuresAreSet)
			assert.NoError(t, err)
		})
	}
}
