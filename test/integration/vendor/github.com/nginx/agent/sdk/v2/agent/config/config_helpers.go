/**
 * Copyright (c) F5, Inc.
 *
 * This source code is licensed under the Apache License, Version 2.0 license found in the
 * LICENSE file in the root directory of this source tree.
 */

package config

import (
	"github.com/mitchellh/mapstructure"
)

const (
	DefaultPluginSize = 100
	KeyDelimiter      = "_"

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
	ExtensionsKey                            = "extensions"
	AdvancedMetricsExtensionPlugin           = "advanced-metrics"
	NginxAppProtectExtensionPlugin           = "nginx-app-protect"
	NginxAppProtectMonitoringExtensionPlugin = "nap-monitoring"

	// Configuration Keys
	AdvancedMetricsExtensionPluginConfigKey           = "advanced_metrics"
	NginxAppProtectExtensionPluginConfigKey           = "nginx_app_protect"
	NginxAppProtectMonitoringExtensionPluginConfigKey = "nap_monitoring"
)

func GetKnownExtensions() []string {
	return []string{
		AdvancedMetricsExtensionPlugin,
		NginxAppProtectExtensionPlugin,
		NginxAppProtectMonitoringExtensionPlugin,
	}
}

func IsKnownExtension(extension string) bool {
	for _, knownExtension := range GetKnownExtensions() {
		if knownExtension == extension {
			return true
		}
	}

	return false
}

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

func DecodeConfig[T interface{}](input interface{}) (output T, err error) {
	decoder, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		WeaklyTypedInput: true,
		DecodeHook:       mapstructure.ComposeDecodeHookFunc(mapstructure.StringToTimeDurationHookFunc()),
		Result:           &output,
	})

	if err != nil {
		return output, err
	}

	err = decoder.Decode(input)

	if err != nil {
		return output, err
	}

	return output, nil
}
