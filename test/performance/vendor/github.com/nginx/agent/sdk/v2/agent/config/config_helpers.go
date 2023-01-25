/**
 * Copyright (c) F5, Inc.
 *
 * This source code is licensed under the Apache License, Version 2.0 license found in the
 * LICENSE file in the root directory of this source tree.
 */

package config

const (
	KeyDelimiter = "_"

	// Features
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

	// Extensions
	ExtensionsKey                  = "extensions"
	AdvancedMetricsExtensionPlugin = "advanced-metrics"

	// Configuration Keys
	AdvancedMetricsExtensionPluginConfigKey = "advanced_metrics"
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
