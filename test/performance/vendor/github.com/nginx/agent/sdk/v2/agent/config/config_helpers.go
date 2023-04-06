/**
 * Copyright (c) F5, Inc.
 *
 * This source code is licensed under the Apache License, Version 2.0 license found in the
 * LICENSE file in the root directory of this source tree.
 */

package config

const (
	KeyDelimiter = "_"

	// viper keys used in config
	FeaturesKey         = "features"
	FeatureRegistration = "registration"
	// Deprecated: use nginx-config-async instead
	FeatureNginxConfig      = "nginx-config"
	FeatureNginxConfigAsync = "nginx-config-async"
	FeatureNginxSSLConfig   = "nginx-ssl-config"
	FeatureNginxCounting    = "nginx-counting"
	FeatureMetrics          = "metrics"
	FeatureMetricsThrottle  = "metrics-throttle"
	FeatureDataPlaneStatus  = "dataplane-status"
	FeatureProcessWatcher   = "process-watcher"
	FeatureFileWatcher      = "file-watcher"
	FeatureActivityEvents   = "activity-events"
	FeatureAgentAPI         = "agent-api"
)

func GetDefaultFeatures() []string {
	return []string{
		FeatureRegistration,
		FeatureNginxConfigAsync,
		FeatureNginxSSLConfig,
		FeatureNginxCounting,
		FeatureMetrics,
		FeatureMetricsThrottle,
		FeatureDataPlaneStatus,
		FeatureProcessWatcher,
		FeatureFileWatcher,
		FeatureActivityEvents,
		FeatureAgentAPI,
	}
}
