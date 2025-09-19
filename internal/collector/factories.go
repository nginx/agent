// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package collector

import (
	"github.com/nginx/agent/v3/internal/collector/containermetricsreceiver"
	"github.com/nginx/agent/v3/internal/collector/logsgzipprocessor"
	nginxreceiver "github.com/nginx/agent/v3/internal/collector/nginxossreceiver"
	"github.com/nginx/agent/v3/internal/collector/nginxplusreceiver"
	"github.com/nginx/agent/v3/internal/collector/syslogprocessor"

	"github.com/open-telemetry/opentelemetry-collector-contrib/exporter/prometheusexporter"
	"github.com/open-telemetry/opentelemetry-collector-contrib/extension/headerssetterextension"
	"github.com/open-telemetry/opentelemetry-collector-contrib/extension/healthcheckextension"
	"github.com/open-telemetry/opentelemetry-collector-contrib/extension/pprofextension"
	"github.com/open-telemetry/opentelemetry-collector-contrib/processor/attributesprocessor"
	"github.com/open-telemetry/opentelemetry-collector-contrib/processor/deltatorateprocessor"
	"github.com/open-telemetry/opentelemetry-collector-contrib/processor/filterprocessor"
	"github.com/open-telemetry/opentelemetry-collector-contrib/processor/redactionprocessor"
	"github.com/open-telemetry/opentelemetry-collector-contrib/processor/resourceprocessor"
	"github.com/open-telemetry/opentelemetry-collector-contrib/processor/transformprocessor"
	"github.com/open-telemetry/opentelemetry-collector-contrib/receiver/hostmetricsreceiver"
	"github.com/open-telemetry/opentelemetry-collector-contrib/receiver/tcplogreceiver"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/connector"
	"go.opentelemetry.io/collector/exporter"
	"go.opentelemetry.io/collector/exporter/debugexporter"
	"go.opentelemetry.io/collector/exporter/otlpexporter"
	"go.opentelemetry.io/collector/exporter/otlphttpexporter"
	"go.opentelemetry.io/collector/extension"
	"go.opentelemetry.io/collector/otelcol"
	"go.opentelemetry.io/collector/processor"
	"go.opentelemetry.io/collector/processor/batchprocessor"
	"go.opentelemetry.io/collector/processor/memorylimiterprocessor"
	"go.opentelemetry.io/collector/receiver"
	"go.opentelemetry.io/collector/receiver/otlpreceiver"
)

// OTelComponentFactories returns all the OTel collector components supported
// based on https://github.com/DataDog/datadog-agent/blob/main/comp/otelcol/collector-contrib/impl/collectorcontrib.go
func OTelComponentFactories() (otelcol.Factories, error) {
	connectors := createConnectorFactories()
	extensions := createExtensionFactories()
	receivers := createReceiverFactories()
	processors := createProcessorFactories()
	exporters := createExporterFactories()

	factories := otelcol.Factories{
		Connectors: connectors,
		Extensions: extensions,
		Receivers:  receivers,
		Processors: processors,
		Exporters:  exporters,
	}

	return factories, nil
}

func createConnectorFactories() map[component.Type]connector.Factory {
	return make(map[component.Type]connector.Factory)
}

func createExtensionFactories() map[component.Type]extension.Factory {
	extensionsList := []extension.Factory{
		headerssetterextension.NewFactory(),
		healthcheckextension.NewFactory(),
		pprofextension.NewFactory(),
	}

	extensions := make(map[component.Type]extension.Factory)
	for _, extensionFactory := range extensionsList {
		extensions[extensionFactory.Type()] = extensionFactory
	}

	return extensions
}

func createReceiverFactories() map[component.Type]receiver.Factory {
	receiverList := []receiver.Factory{
		otlpreceiver.NewFactory(),
		containermetricsreceiver.NewFactory(),
		hostmetricsreceiver.NewFactory(),
		nginxreceiver.NewFactory(),
		nginxplusreceiver.NewFactory(),
		tcplogreceiver.NewFactory(),
	}

	receivers := make(map[component.Type]receiver.Factory)
	for _, receiverFactory := range receiverList {
		receivers[receiverFactory.Type()] = receiverFactory
	}

	return receivers
}

func createProcessorFactories() map[component.Type]processor.Factory {
	processorList := []processor.Factory{
		attributesprocessor.NewFactory(),
		batchprocessor.NewFactory(),
		deltatorateprocessor.NewFactory(),
		filterprocessor.NewFactory(),
		memorylimiterprocessor.NewFactory(),
		redactionprocessor.NewFactory(),
		resourceprocessor.NewFactory(),
		syslogprocessor.NewFactory(),
		transformprocessor.NewFactory(),
		logsgzipprocessor.NewFactory(),
	}

	processors := make(map[component.Type]processor.Factory)
	for _, processorFactory := range processorList {
		processors[processorFactory.Type()] = processorFactory
	}

	return processors
}

func createExporterFactories() map[component.Type]exporter.Factory {
	exporterList := []exporter.Factory{
		debugexporter.NewFactory(),
		prometheusexporter.NewFactory(),
		otlpexporter.NewFactory(),
		otlphttpexporter.NewFactory(),
	}

	exporters := make(map[component.Type]exporter.Factory)
	for _, exporterFactory := range exporterList {
		exporters[exporterFactory.Type()] = exporterFactory
	}

	return exporters
}
