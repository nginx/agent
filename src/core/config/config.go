package config

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"time"

	agent_config "github.com/nginx/agent/sdk/v2/agent/config"
	advanced_metrics "github.com/nginx/agent/v2/src/extensions/advanced-metrics/pkg/advanced-metrics"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"

	log "github.com/sirupsen/logrus"
	flag "github.com/spf13/pflag"
)

const (
	dynamicConfigUsageComment = `#
# /etc/nginx-agent/dynamic-agent.conf
#
# Dynamic configuration file for NGINX Agent.
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

`
)

var (
	Viper = viper.NewWithOptions(viper.KeyDelimiter(agent_config.KeyDelimiter))
)

func SetVersion(version, commit string) {
	ROOT_COMMAND.Version = version + "-" + commit
}

func Execute() error {
	ROOT_COMMAND.AddCommand(COMPLETION_COMMAND)
	return ROOT_COMMAND.Execute()
}

func SetDefaults() {
	// CLOUDACCOUNTID DEFAULT
	Viper.SetDefault(CloudAccountIdKey, Defaults.CloudAccountID)

	// SERVER DEFAULTS
	Viper.SetDefault(ServerMetrics, Defaults.Server.Metrics)
	Viper.SetDefault(ServerCommand, Defaults.Server.Command)

	// DATAPLANE DEFAULTS
	Viper.SetDefault(DataplaneStatusPoll, Defaults.Dataplane.Status.PollInterval)

	// METRICS DEFAULTS
	Viper.SetDefault(MetricsBulkSize, Defaults.AgentMetrics.BulkSize)
	Viper.SetDefault(MetricsReportInterval, Defaults.AgentMetrics.ReportInterval)
	Viper.SetDefault(MetricsCollectionInterval, Defaults.AgentMetrics.CollectionInterval)

	// NGINX DEFAULTS
	Viper.SetDefault(NginxClientVersion, Defaults.Nginx.NginxClientVersion)
}

func SetNginxAppProtectDefaults() {
	Viper.SetDefault(NginxAppProtectReportInterval, Defaults.NginxAppProtect.ReportInterval)
}

func SetNAPMonitoringDefaults() {
	Viper.SetDefault(NAPMonitoringCollectorBufferSize, Defaults.NAPMonitoring.CollectorBufferSize)
	Viper.SetDefault(NAPMonitoringProcessorBufferSize, Defaults.NAPMonitoring.ProcessorBufferSize)
	Viper.SetDefault(NAPMonitoringSyslogIP, Defaults.NAPMonitoring.SyslogIP)
	Viper.SetDefault(NAPMonitoringSyslogPort, Defaults.NAPMonitoring.SyslogPort)
	Viper.SetDefault(NAPMonitoringReportInterval, Defaults.NAPMonitoring.ReportInterval)
	Viper.SetDefault(NAPMonitoringReportCount, Defaults.NAPMonitoring.ReportCount)
}

func setFlagDeprecated(name string, usageMessage string) {
	err := ROOT_COMMAND.Flags().MarkDeprecated(name, usageMessage)
	if err != nil {
		log.Warnf("error occurred deprecating flag %s: %v", name, err)
	}
}

func deprecateFlags() {
	setFlagDeprecated("api-token", "DEPRECATED. API Token is no longer set. No replacement command.")
	setFlagDeprecated("location", "DEPRECATED. Set through APIs. No replacement command.")
	setFlagDeprecated("metadata", "DEPRECATED. Use tags instead.")
	setFlagDeprecated("metrics-server", "DEPRECATED. Use server instead.")
	setFlagDeprecated("metrics-tls-ca", "DEPRECATED. metrics-tls-ca has been replaced by tls-ca")
	setFlagDeprecated("metrics-tls-cert", "DEPRECATED. metrics-tls-cert has been replaced by tls-cert")
	setFlagDeprecated("metrics-tls-enable", "DEPRECATED. metrics-tls-enable has been replaced by tls-enable")
	setFlagDeprecated("metrics-tls-key", "DEPRECATED. metrics-tls-key has been replaced by tls-key")
	setFlagDeprecated("nginx-bin-path", "DEPRECATED. nginx-bin-path is no longer used. The agent strives to discover the nginx instances on the dataplane")
	setFlagDeprecated("nginx-metrics-poll-interval", "DEPRECATED. nginx-metrics-poll-interval has been replaced by metrics-report-interval and metrics-collection-interval")
	setFlagDeprecated("nginx-pid-path", "DEPRECATED. nginx-pid-path is no longer used. The agent strives to discover the nginx instances on the dataplane")
	setFlagDeprecated("nginx-plus-api", "DEPRECATED. nginx-plus-api is no longer used. The agent strives to discover the nginx instances on the dataplane and read this from the configuration file")
	setFlagDeprecated("nginx-stub-status", "DEPRECATED. nginx-stub-status is no longer used. The agent strives to discover the nginx instances on the dataplane and read this from the configuration file")
	setFlagDeprecated("server", "DEPRECATED. server has been replaced with the new server structure")
	setFlagDeprecated("dataplane-sync-enable", "DEPRECATED. server has been replaced with the new server structure")
	setFlagDeprecated("dataplane-events-enable", "DEPRECATED. server has been replaced with the new server structure")
}

func RegisterFlags() {
	Viper.SetEnvPrefix(EnvPrefix)
	Viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	Viper.AutomaticEnv()

	fs := ROOT_COMMAND.Flags()
	for _, f := range append(agentFlags, deprecatedFlags...) {
		f.register(fs)
	}

	fs.SetNormalizeFunc(wordSepNormalizeFunc)
	deprecateFlags()

	fs.VisitAll(func(flag *flag.Flag) {
		if err := Viper.BindPFlag(strings.ReplaceAll(flag.Name, "-", "_"), fs.Lookup(flag.Name)); err != nil {
			return
		}
		err := Viper.BindEnv(flag.Name)
		if err != nil {
			log.Warnf("error occurred binding env %s: %v", flag.Name, err)
		}
	})
}

func RegisterConfigFile(dynamicConfFilePath string, confFileName string, confPaths ...string) (string, error) {
	cfg, err := SeekConfigFileInPaths(confFileName, confPaths...)
	if err != nil {
		return cfg, err
	}

	SetDynamicConfigFileAbsPath(dynamicConfFilePath)
	if err := LoadPropertiesFromFile(cfg); err != nil {
		log.Fatalf("Unable to load properties from config files (%s, %s) - %v", cfg, dynamicConfFilePath, err)
	}

	return cfg, nil
}

func RegisterRunner(r func(cmd *cobra.Command, args []string)) {
	ROOT_COMMAND.Run = r
}

func GetConfig(clientId string) (*Config, error) {
	config := &Config{
		Path:                  Viper.GetString(ConfigPathKey),
		DynamicConfigPath:     Viper.GetString(DynamicConfigPathKey),
		ClientID:              clientId,
		CloudAccountID:        Viper.GetString(CloudAccountIdKey),
		Server:                getServer(),
		AgentAPI:              getAgentAPI(),
		ConfigDirs:            Viper.GetString(ConfigDirsKey),
		Log:                   getLog(),
		TLS:                   getTLS(),
		Nginx:                 getNginx(),
		Dataplane:             getDataplane(),
		AgentMetrics:          getMetrics(),
		Features:              Viper.GetStringSlice(agent_config.FeaturesKey),
		Tags:                  Viper.GetStringSlice(TagsKey),
		Updated:               filePathUTime(Viper.GetString(DynamicConfigPathKey)),
		AllowedDirectoriesMap: map[string]struct{}{},
		DisplayName:           Viper.GetString(DisplayNameKey),
		InstanceGroup:         Viper.GetString(InstanceGroupKey),
		NginxAppProtect:       getNginxAppProtect(),
		NAPMonitoring:         getNAPMonitoring(),
		AdvancedMetrics:       getAdvancedMetrics(),
	}

	for _, dir := range strings.Split(config.ConfigDirs, ":") {
		if dir != "" {
			config.AllowedDirectoriesMap[dir] = struct{}{}
		}
	}
	config.Server.Target = fmt.Sprintf("%s:%d", config.Server.Host, config.Server.GrpcPort)

	log.Tracef("%v", config)
	return config, nil
}

// UpdateAgentConfig updates the Agent config on disk with the tags and features that are
// passed into it. A bool is returned indicating if the Agent config was
// overwritten or not.
func UpdateAgentConfig(systemId string, updateTags []string, updateFeatures []string) (bool, error) {
	// Get current config on disk
	config, err := GetConfig(systemId)
	if err != nil {
		log.Errorf("Failed to register config: %v", err)
		return false, err
	}

	// Update nil valued updateTags to empty slice for comparison
	if updateTags == nil {
		updateTags = []string{}
	}

	if updateFeatures == nil {
		updateFeatures = []string{}
	}

	// Sort tags and compare them
	sort.Strings(updateTags)
	sort.Strings(config.Tags)
	synchronizedTags := reflect.DeepEqual(updateTags, config.Tags)

	Viper.Set(TagsKey, updateTags)
	config.Tags = Viper.GetStringSlice(TagsKey)

	sort.Strings(updateFeatures)
	sort.Strings(config.Features)
	synchronizedFeatures := reflect.DeepEqual(updateFeatures, config.Features)

	Viper.Set(agent_config.FeaturesKey, updateFeatures)
	config.Features = Viper.GetStringSlice(agent_config.FeaturesKey)

	// If the features are already synchronized there is no need to overwrite
	if synchronizedTags && synchronizedFeatures {
		log.Debug("Manager and Local tags and features are already synchronized")
		return false, nil
	}

	// Get the dynamic config path and use default dynamic config path if it's not
	// already set.
	dynamicCfgPath := Viper.GetString(DynamicConfigPathKey)
	if dynamicCfgPath == "" {
		dynamicCfgPath = DynamicConfigFileAbsPath
	}

	// Overwrite existing nginx-agent.conf with updated config
	updatedConfBytes, err := yaml.Marshal(config)
	if err != nil {
		return false, err
	}

	updatedConfBytes = append([]byte(dynamicConfigUsageComment), updatedConfBytes...)

	err = ioutil.WriteFile(dynamicCfgPath, updatedConfBytes, 0)
	if err != nil {
		return false, err
	}

	config.Updated = filePathUTime(dynamicCfgPath)

	log.Infof("Successfully updated agent config (%s)", dynamicCfgPath)

	return true, nil
}

func getMetrics() AgentMetrics {
	return AgentMetrics{
		BulkSize:           Viper.GetInt(MetricsBulkSize),
		ReportInterval:     Viper.GetDuration(MetricsReportInterval),
		CollectionInterval: Viper.GetDuration(MetricsCollectionInterval),
		Mode:               Viper.GetString(MetricsMode),
	}
}

func getAdvancedMetrics() AdvancedMetrics {
	return AdvancedMetrics{
		SocketPath:        Viper.GetString(AdvancedMetricsSocketPath),
		AggregationPeriod: Viper.GetDuration(AdvancedMetricsAggregationPeriod),
		PublishingPeriod:  Viper.GetDuration(AdvancedMetricsPublishPeriod),
		TableSizesLimits: advanced_metrics.TableSizesLimits{
			StagingTableMaxSize:    Viper.GetInt(AdvancedMetricsTableSizesLimitsSTMS),
			StagingTableThreshold:  Viper.GetInt(AdvancedMetricsTableSizesLimitsSTT),
			PriorityTableMaxSize:   Viper.GetInt(AdvancedMetricsTableSizesLimitsPTMS),
			PriorityTableThreshold: Viper.GetInt(AdvancedMetricsTableSizesLimitsPTT),
		},
	}
}

func getLog() LogConfig {
	return LogConfig{
		Level: Viper.GetString(LogLevel),
		Path:  Viper.GetString(LogPath),
	}
}

func getDataplane() Dataplane {
	return Dataplane{
		Status: Status{
			PollInterval:   Viper.GetDuration(DataplaneStatusPoll),
			ReportInterval: Viper.GetDuration(DataplaneStatusReportInterval),
		},
	}
}

func getNginxAppProtect() NginxAppProtect {
	return NginxAppProtect{
		ReportInterval: Viper.GetDuration(NginxAppProtectReportInterval),
	}
}

func getNAPMonitoring() NAPMonitoring {
	return NAPMonitoring{
		CollectorBufferSize: Viper.GetInt(NAPMonitoringCollectorBufferSize),
		ProcessorBufferSize: Viper.GetInt(NAPMonitoringProcessorBufferSize),
		SyslogIP:            Viper.GetString(NAPMonitoringSyslogIP),
		SyslogPort:          Viper.GetInt(NAPMonitoringSyslogPort),
		ReportInterval:      Viper.GetDuration(NAPMonitoringReportInterval),
		ReportCount:         Viper.GetInt(NAPMonitoringReportCount),
	}
}

func getNginx() Nginx {
	return Nginx{
		ExcludeLogs:         Viper.GetString(NginxExcludeLogs),
		Debug:               Viper.GetBool(NginxDebug),
		NginxCountingSocket: Viper.GetString(NginxCountingSocket),
		NginxClientVersion:  Viper.GetInt(NginxClientVersion),
	}
}

func getServer() Server {
	return Server{
		Host:     Viper.GetString(ServerHost),
		GrpcPort: Viper.GetInt(ServerGrpcPort),
		Token:    Viper.GetString(ServerToken),
		Metrics:  Viper.GetString(ServerMetrics),
		Command:  Viper.GetString(ServerCommand),
	}
}

func getAgentAPI() AgentAPI {
	return AgentAPI{
		Port: viper.GetInt(AgentAPIPort),
	}
}

func getTLS() TLSConfig {
	return TLSConfig{
		Enable:     Viper.GetBool(TlsEnable),
		Cert:       Viper.GetString(TlsCert),
		Key:        Viper.GetString(TlsPrivateKey),
		Ca:         Viper.GetString(TlsCa),
		SkipVerify: Viper.GetBool(TlsSkipVerify),
	}
}

func LoadPropertiesFromFile(cfg string) error {
	Viper.SetConfigFile(cfg)
	Viper.SetConfigType(ConfigFileType)
	err := Viper.MergeInConfig()
	if err != nil {
		return fmt.Errorf("Error loading config file %s: %v", cfg, err)
	}

	// Get the dynamic config path and use default dynamic config path if it's not
	// already set.
	dynamicCfgPath := Viper.GetString(DynamicConfigPathKey)
	if dynamicCfgPath == "" {
		dynamicCfgPath = DynamicConfigFileAbsPath
	}
	dynamicCfgDir, dynamicCfgFile := filepath.Split(dynamicCfgPath)

	// Get dynamic file, if it doesn't exist create it.
	file, err := os.Stat(dynamicCfgPath)
	if err != nil {
		log.Warnf("Unable to read dynamic config (%s), got the following error: %v", dynamicCfgPath, err)
	}

	if file == nil {
		log.Infof("Writing the following file to disk: %s", dynamicCfgPath)
		err = os.MkdirAll(dynamicCfgDir, 0755)
		if err != nil {
			return fmt.Errorf("Error attempting to create directory for dynamic config (%s), got the following error: %v", dynamicCfgDir, err)
		}

		err = os.WriteFile(dynamicCfgPath, []byte(dynamicConfigUsageComment), 0644)
		if err != nil {
			return fmt.Errorf("Error attempting to create dynamic config (%s), got the following error: %v", dynamicCfgPath, err)
		}
	}

	// Load properties from existing file
	log.Debugf("Loading dynamic properties from file: %s", dynamicCfgPath)
	Viper.AddConfigPath(dynamicCfgDir)
	Viper.SetConfigName(dynamicCfgFile)
	err = Viper.MergeInConfig()
	if err != nil {
		return fmt.Errorf("Error loading file %s: %v", dynamicCfgPath, err)
	}

	return nil
}

func SetDynamicConfigFileAbsPath(dynamicCfgPath string) {
	Viper.Set(DynamicConfigPathKey, dynamicCfgPath)
	log.Debugf("Set dynamic agent config file: %s", dynamicCfgPath)
}

func wordSepNormalizeFunc(f *flag.FlagSet, name string) flag.NormalizedName {
	from := []string{"_", "."}
	to := "-"
	for _, sep := range from {
		name = strings.Replace(name, sep, to, -1)
	}
	return flag.NormalizedName(name)
}

func SeekConfigFileInPaths(configName string, searchPaths ...string) (string, error) {
	for _, p := range searchPaths {
		f := filepath.Join(p, configName)
		if _, err := os.Stat(f); err == nil {
			return f, nil
		}
	}
	return "", fmt.Errorf("A valid configuration has not been found in any of the search paths.")
}

func filePathUTime(path string) time.Time {
	s, err := os.Stat(path)
	if err != nil {
		log.Warnf("Unable to determine the modified time of %s: %s. Defaulting the value to Now.", path, err)
		return time.Now()
	}
	return s.ModTime()
}

func CheckAndSetDefault(attribute interface{}, defaultValue interface{}) {
	if value, ok := attribute.(*string); ok {
		if *value == "" {
			*value = defaultValue.(string)
		}
	} else if value, ok := attribute.(*time.Duration); ok {
		if *value == 0*time.Second {
			*value = defaultValue.(time.Duration)
		}
	} else if value, ok := attribute.(*int); ok {
		if *value == int(0) {
			*value = defaultValue.(int)
		}
	}
}
