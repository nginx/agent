// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.
package collector

import (
	_ "embed"
	"log/slog"
	"os"
	"path/filepath"
	"text/template"

	"github.com/nginx/agent/v3/internal/config"
	"go.opentelemetry.io/collector/confmap"
	"go.opentelemetry.io/collector/confmap/provider/envprovider"
	"go.opentelemetry.io/collector/confmap/provider/fileprovider"
	"go.opentelemetry.io/collector/confmap/provider/httpprovider"
	"go.opentelemetry.io/collector/confmap/provider/httpsprovider"
	"go.opentelemetry.io/collector/confmap/provider/yamlprovider"
	"go.opentelemetry.io/collector/otelcol"
)

const (
	otelTemplatePath     = "otelcol.tmpl"
	configFilePermission = 0o600
)

//go:embed otelcol.tmpl
var otelcolTemplate string

func OTelCollectorSettings(cfg *config.Config) otelcol.CollectorSettings {
	return otelcol.CollectorSettings{
		Factories:               OTelComponentFactories,
		BuildInfo:               BuildInfo(cfg),
		DisableGracefulShutdown: false,
		ConfigProviderSettings:  ConfigProviderSettings(cfg),
		LoggingOptions:          nil,
		SkipSettingGRPCLogger:   false,
	}
}

// ConfigProviderSettings are the settings to configure the behavior of the ConfigProvider.
func ConfigProviderSettings(cfg *config.Config) otelcol.ConfigProviderSettings {
	return otelcol.ConfigProviderSettings{
		ResolverSettings: confmap.ResolverSettings{
			ProviderFactories:  createProviderFactories(),
			ConverterFactories: createConverterFactories(),
			URIs:               createURIs(cfg),
		},
	}
}

func createProviderFactories() []confmap.ProviderFactory {
	providerConfig := []confmap.ProviderFactory{
		envprovider.NewFactory(),
		fileprovider.NewFactory(),
		httpprovider.NewFactory(),
		httpsprovider.NewFactory(),
		yamlprovider.NewFactory(),
	}

	return providerConfig
}

func createConverterFactories() []confmap.ConverterFactory {
	converterConfig := []confmap.ConverterFactory{}

	return converterConfig
}

func createURIs(cfg *config.Config) []string {
	return []string{cfg.Collector.ConfigPath}
}

// Generates a OTel Collector config to a file by injecting the Metrics Config to a Go template.
func writeCollectorConfig(conf *config.Collector) error {
	otelcolTemplate, err := template.New(otelTemplatePath).Parse(otelcolTemplate)
	if err != nil {
		return err
	}

	confPath := filepath.Clean(conf.ConfigPath)

	// Check if file exists.
	_, err = os.Stat(confPath)
	if err != nil {
		if !os.IsNotExist(err) {
			return err
		}

		// Create if doesn't exist.
		_, createErr := os.Create(confPath)
		if createErr != nil {
			return createErr
		}
	}

	file, err := os.OpenFile(confPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, configFilePermission)
	defer func() {
		err = file.Close()
		if err != nil {
			slog.Warn("Failed to close file", "file_path", confPath)
		}
	}()
	if err != nil {
		return err
	}

	err = otelcolTemplate.Execute(file, conf)
	if err != nil {
		return err
	}

	return nil
}
