package config

const (
	KeyDelimiter = "_"

	// viper keys used in config
	FeaturesKey             = "features"
	FeatureRegistration     = FeaturesKey + KeyDelimiter + "registration"
	FeatureNginxConfig      = FeaturesKey + KeyDelimiter + "nginx-config"
	FeatureNginxConfigAsync = FeaturesKey + KeyDelimiter + "nginx-config-async"
	FeatureNginxSSLConfig   = FeaturesKey + KeyDelimiter + "nginx-ssl-config"
	FeatureNginxCounting    = FeaturesKey + KeyDelimiter + "nginx-counting"
	FeatureMetrics          = FeaturesKey + KeyDelimiter + "metrics"
	FeatureMetricsThrottle  = FeaturesKey + KeyDelimiter + "metrics-throttle"
	FeatureDataPlaneStatus  = FeaturesKey + KeyDelimiter + "dataplane-status"
	FeatureProcessWatcher   = FeaturesKey + KeyDelimiter + "process-watcher"
	FeatureFileWatcher      = FeaturesKey + KeyDelimiter + "file-watcher"
	FeatureActivityEvents   = FeaturesKey + KeyDelimiter + "activity-events"
	FeatureAgentAPI         = FeaturesKey + KeyDelimiter + "agent-api"
)

func GetDefaultFeatures() []string {
	return []string{
		FeatureRegistration,
		FeatureNginxConfig,
		FeatureNginxSSLConfig,
		FeatureNginxCounting,
		FeatureNginxConfigAsync,
		FeatureMetrics,
		FeatureMetricsThrottle,
		FeatureDataPlaneStatus,
		FeatureProcessWatcher,
		FeatureFileWatcher,
		FeatureActivityEvents,
		FeatureAgentAPI,
	}
}
