package config

import (
	"os"
	"path"
	"testing"
	"time"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
)

func TestRegisterConfigFile(t *testing.T) {
	viperInstance = viper.NewWithOptions(viper.KeyDelimiter("_"))
	file, err := os.Create("nginx-agent.conf")
	defer os.Remove(file.Name())
	assert.NoError(t, err)

	currentDirectory, err := os.Getwd()
	assert.NoError(t, err)

	err = RegisterConfigFile()

	assert.NoError(t, err)
	assert.Equal(t, path.Join(currentDirectory, "nginx-agent.conf"), viperInstance.GetString(ConfigPathKey))
}

func TestGetConfig(t *testing.T) {
	viperInstance = viper.NewWithOptions(viper.KeyDelimiter("_"))
	err := loadPropertiesFromFile("./testdata/nginx-agent.conf")
	assert.NoError(t, err)

	result := GetConfig()

	assert.Equal(t, "debug", result.Log.Level)
	assert.Equal(t, "./", result.Log.Path)

	assert.Equal(t, "127.0.0.1", result.DataplaneAPI.Host)
	assert.Equal(t, 8038, result.DataplaneAPI.Port)

	assert.Equal(t, 30*time.Second, result.ProcessMonitor.MonitoringFrequency)
}

func TestSetVersion(t *testing.T) {
	viperInstance = viper.NewWithOptions(viper.KeyDelimiter("_"))
	setVersion("v1.2.3", "asdf1234")

	assert.Equal(t, "v1.2.3", viperInstance.GetString(VersionConfigKey))
}

func TestSetDefaults(t *testing.T) {
	viperInstance = viper.NewWithOptions(viper.KeyDelimiter("_"))
	setDefaults()

	assert.Equal(t, "info", viperInstance.GetString(LogLevelConfigKey))
	assert.Equal(t, "", viperInstance.GetString(LogPathConfigKey))
	assert.Equal(t, time.Minute, viperInstance.GetDuration(ProcessMonitorMonitoringFrequencyConfigKey))
	assert.Equal(t, "", viperInstance.GetString(DataplaneAPIHostConfigKey))
	assert.Equal(t, 0, viperInstance.GetInt(DataplaneAPIPortConfigKey))
}

func TestRegisterFlags(t *testing.T) {
	viperInstance = viper.NewWithOptions(viper.KeyDelimiter("_"))
	os.Setenv("NGINX_AGENT_LOG_LEVEL", "warn")
	os.Setenv("NGINX_AGENT_LOG_PATH", "/var/log/test/agent.log")
	os.Setenv("NGINX_AGENT_PROCESS_MONITOR_MONITORING_FREQUENCY", "10s")
	os.Setenv("NGINX_AGENT_DATAPLANE_API_HOST", "example.com")
	os.Setenv("NGINX_AGENT_DATAPLANE_API_PORT", "9090")
	registerFlags()

	assert.Equal(t, "warn", viperInstance.GetString(LogLevelConfigKey))
	assert.Equal(t, "/var/log/test/agent.log", viperInstance.GetString(LogPathConfigKey))
	assert.Equal(t, 10*time.Second, viperInstance.GetDuration(ProcessMonitorMonitoringFrequencyConfigKey))
	assert.Equal(t, "example.com", viperInstance.GetString(DataplaneAPIHostConfigKey))
	assert.Equal(t, 9090, viperInstance.GetInt(DataplaneAPIPortConfigKey))
}

func TestSeekFileInPaths(t *testing.T) {
	viperInstance = viper.NewWithOptions(viper.KeyDelimiter("_"))
	result, err := seekFileInPaths("nginx-agent.conf", []string{"./", "./testdata"}...)

	assert.NoError(t, err)
	assert.Equal(t, "testdata/nginx-agent.conf", result)

	_, err = seekFileInPaths("nginx-agent.conf", []string{"./"}...)
	assert.Error(t, err)
}

func TestGetConfigFilePaths(t *testing.T) {
	viperInstance = viper.NewWithOptions(viper.KeyDelimiter("_"))
	currentDirectory, err := os.Getwd()
	assert.NoError(t, err)

	result := getConfigFilePaths()

	assert.Equal(t, 2, len(result))
	assert.Equal(t, "/etc/nginx-agent/", result[0])
	assert.Equal(t, currentDirectory, result[1])
}

func TestLoadPropertiesFromFile(t *testing.T) {
	viperInstance = viper.NewWithOptions(viper.KeyDelimiter("_"))
	err := loadPropertiesFromFile("./testdata/nginx-agent.conf")
	assert.NoError(t, err)

	assert.Equal(t, "debug", viperInstance.GetString(LogLevelConfigKey))
	assert.Equal(t, "./", viperInstance.GetString(LogPathConfigKey))

	assert.Equal(t, "127.0.0.1", viperInstance.GetString(DataplaneAPIHostConfigKey))
	assert.Equal(t, 8038, viperInstance.GetInt(DataplaneAPIPortConfigKey))

	assert.Equal(t, 30*time.Second, viperInstance.GetDuration(ProcessMonitorMonitoringFrequencyConfigKey))

	err = loadPropertiesFromFile("./testdata/unknown.conf")
	assert.Error(t, err)
}

func TestNormalizeFunc(t *testing.T) {
	viperInstance = viper.NewWithOptions(viper.KeyDelimiter("_"))
	var expected pflag.NormalizedName = "test-flag-name"
	result := normalizeFunc(&pflag.FlagSet{}, "test_flag.name")
	assert.Equal(t, expected, result)
}

func TestGetLog(t *testing.T) {
	viperInstance = viper.NewWithOptions(viper.KeyDelimiter("_"))
	viperInstance.Set(LogLevelConfigKey, "error")
	viperInstance.Set(LogPathConfigKey, "/var/log/test/test.log")

	result := getLog()
	assert.Equal(t, "error", result.Level)
	assert.Equal(t, "/var/log/test/test.log", result.Path)
}

func TestGetProcessMonitor(t *testing.T) {
	viperInstance = viper.NewWithOptions(viper.KeyDelimiter("_"))
	viperInstance.Set(ProcessMonitorMonitoringFrequencyConfigKey, time.Hour)

	result := getProcessMonitor()
	assert.Equal(t, time.Hour, result.MonitoringFrequency)
}

func TestGetDataplaneAPI(t *testing.T) {
	viperInstance = viper.NewWithOptions(viper.KeyDelimiter("_"))
	viperInstance.Set(DataplaneAPIHostConfigKey, "testhost")
	viperInstance.Set(DataplaneAPIPortConfigKey, 9091)

	result := getDataplaneAPI()
	assert.Equal(t, "testhost", result.Host)
	assert.Equal(t, 9091, result.Port)
}
