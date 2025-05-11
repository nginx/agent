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
	KeyDelimiter = "_"

	// viper keys used in config
	FeaturesKey         = "features"
	FeatureRegistration = "registration"
	// Deprecated: use nginx-config-async instead
	FeatureNginxConfig         = "nginx-config"
	FeatureNginxConfigAsync    = "nginx-config-async"
	FeatureNginxSSLConfig      = "nginx-ssl-config"
	FeatureNginxCounting       = "nginx-counting"
	FeatureMetrics             = "metrics"
	FeatureMetricsCollection   = "metrics-collection"
	FeatureMetricsSender       = "metrics-sender"
	FeatureMetricsThrottle     = "metrics-throttle"
	FeatureDataPlaneStatus     = "dataplane-status"
	FeatureProcessWatcher      = "process-watcher"
	FeatureFileWatcher         = "file-watcher"
	FeatureFileWatcherThrottle = "file-watch-throttle"
	FeatureActivityEvents      = "activity-events"
	FeatureAgentAPI            = "agent-api"

	CommanderPlugin    = "commander"
	ConfigReaderPlugin = "config-reader-plugin"
	ExtensionPlugin    = "extension-plugin"
	FeaturesPlugin     = "features-plugin"
	NginxBinaryPlugin  = "nginx-binary"

	// Extensions
	ExtensionsKey                            = "extensions"
	AdvancedMetricsExtensionPlugin           = "advanced-metrics"
	NginxAppProtectExtensionPlugin           = "nginx-app-protect"
	NginxAppProtectMonitoringExtensionPlugin = "nap-monitoring"
	PhpFpmMetricsExtensionPlugin             = "php-fpm-metrics"

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
		// TODO : Enable php-fpm metric extension functionality
		// PhpFpmMetricsExtensionPlugin,
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
		FeatureNginxConfigAsync,
		FeatureNginxSSLConfig,
		FeatureNginxCounting,
		FeatureMetrics,
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
