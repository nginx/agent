// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.
package collector

import (
	"errors"

	nginxreceiver "github.com/nginx/agent/v3/internal/collector/nginxossreceiver"
	"github.com/nginx/agent/v3/internal/collector/nginxplusreceiver"
	"github.com/open-telemetry/opentelemetry-collector-contrib/connector/countconnector"
	"github.com/open-telemetry/opentelemetry-collector-contrib/connector/exceptionsconnector"
	"github.com/open-telemetry/opentelemetry-collector-contrib/connector/grafanacloudconnector"
	"github.com/open-telemetry/opentelemetry-collector-contrib/connector/routingconnector"
	"github.com/open-telemetry/opentelemetry-collector-contrib/connector/servicegraphconnector"
	"github.com/open-telemetry/opentelemetry-collector-contrib/connector/spanmetricsconnector"
	"github.com/open-telemetry/opentelemetry-collector-contrib/exporter/prometheusexporter"
	"github.com/open-telemetry/opentelemetry-collector-contrib/extension/ackextension"
	"github.com/open-telemetry/opentelemetry-collector-contrib/extension/asapauthextension"
	"github.com/open-telemetry/opentelemetry-collector-contrib/extension/awsproxy"
	"github.com/open-telemetry/opentelemetry-collector-contrib/extension/basicauthextension"
	"github.com/open-telemetry/opentelemetry-collector-contrib/extension/bearertokenauthextension"
	"github.com/open-telemetry/opentelemetry-collector-contrib/extension/headerssetterextension"
	"github.com/open-telemetry/opentelemetry-collector-contrib/extension/healthcheckextension"
	"github.com/open-telemetry/opentelemetry-collector-contrib/extension/httpforwarderextension"
	"github.com/open-telemetry/opentelemetry-collector-contrib/extension/oauth2clientauthextension"
	"github.com/open-telemetry/opentelemetry-collector-contrib/extension/observer/dockerobserver"
	"github.com/open-telemetry/opentelemetry-collector-contrib/extension/observer/hostobserver"
	"github.com/open-telemetry/opentelemetry-collector-contrib/extension/observer/k8sobserver"
	"github.com/open-telemetry/opentelemetry-collector-contrib/extension/oidcauthextension"
	"github.com/open-telemetry/opentelemetry-collector-contrib/extension/pprofextension"
	"github.com/open-telemetry/opentelemetry-collector-contrib/extension/sigv4authextension"
	"github.com/open-telemetry/opentelemetry-collector-contrib/processor/attributesprocessor"
	"github.com/open-telemetry/opentelemetry-collector-contrib/processor/cumulativetodeltaprocessor"
	"github.com/open-telemetry/opentelemetry-collector-contrib/processor/deltatorateprocessor"
	"github.com/open-telemetry/opentelemetry-collector-contrib/processor/filterprocessor"
	"github.com/open-telemetry/opentelemetry-collector-contrib/processor/groupbyattrsprocessor"
	"github.com/open-telemetry/opentelemetry-collector-contrib/processor/groupbytraceprocessor"
	"github.com/open-telemetry/opentelemetry-collector-contrib/processor/k8sattributesprocessor"
	"github.com/open-telemetry/opentelemetry-collector-contrib/processor/metricsgenerationprocessor"
	"github.com/open-telemetry/opentelemetry-collector-contrib/processor/metricstransformprocessor"
	"github.com/open-telemetry/opentelemetry-collector-contrib/processor/probabilisticsamplerprocessor"
	"github.com/open-telemetry/opentelemetry-collector-contrib/processor/redactionprocessor"
	"github.com/open-telemetry/opentelemetry-collector-contrib/processor/remotetapprocessor"
	"github.com/open-telemetry/opentelemetry-collector-contrib/processor/resourcedetectionprocessor"
	"github.com/open-telemetry/opentelemetry-collector-contrib/processor/resourceprocessor"
	"github.com/open-telemetry/opentelemetry-collector-contrib/processor/routingprocessor"
	"github.com/open-telemetry/opentelemetry-collector-contrib/processor/spanprocessor"
	"github.com/open-telemetry/opentelemetry-collector-contrib/processor/tailsamplingprocessor"
	"github.com/open-telemetry/opentelemetry-collector-contrib/processor/transformprocessor"
	"github.com/open-telemetry/opentelemetry-collector-contrib/receiver/bigipreceiver"
	"github.com/open-telemetry/opentelemetry-collector-contrib/receiver/dockerstatsreceiver"
	"github.com/open-telemetry/opentelemetry-collector-contrib/receiver/filelogreceiver"
	"github.com/open-telemetry/opentelemetry-collector-contrib/receiver/hostmetricsreceiver"
	"github.com/open-telemetry/opentelemetry-collector-contrib/receiver/httpcheckreceiver"
	"github.com/open-telemetry/opentelemetry-collector-contrib/receiver/k8sclusterreceiver"
	"github.com/open-telemetry/opentelemetry-collector-contrib/receiver/k8seventsreceiver"
	"github.com/open-telemetry/opentelemetry-collector-contrib/receiver/k8sobjectsreceiver"
	"github.com/open-telemetry/opentelemetry-collector-contrib/receiver/kubeletstatsreceiver"
	"github.com/open-telemetry/opentelemetry-collector-contrib/receiver/podmanreceiver"
	"github.com/open-telemetry/opentelemetry-collector-contrib/receiver/prometheusreceiver"
	"github.com/open-telemetry/opentelemetry-collector-contrib/receiver/simpleprometheusreceiver"
	"github.com/open-telemetry/opentelemetry-collector-contrib/receiver/syslogreceiver"
	"github.com/open-telemetry/opentelemetry-collector-contrib/receiver/tcplogreceiver"
	"github.com/open-telemetry/opentelemetry-collector-contrib/receiver/udplogreceiver"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/connector"
	"go.opentelemetry.io/collector/connector/forwardconnector"
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
	"go.opentelemetry.io/collector/receiver/nopreceiver"
	"go.opentelemetry.io/collector/receiver/otlpreceiver"
)

// OTelComponentFactories returns all the OTel collector components supported
// based on https://github.com/DataDog/datadog-agent/blob/main/comp/otelcol/collector-contrib/impl/collectorcontrib.go
func OTelComponentFactories() (otelcol.Factories, error) {
	var errs error

	connectors, err := createConnectorFactories()
	if err != nil {
		errs = errors.Join(errs, err)
	}

	extensions, err := createExtensionFactories()
	if err != nil {
		errs = errors.Join(errs, err)
	}

	receivers, err := createReceiverFactories()
	if err != nil {
		errs = errors.Join(errs, err)
	}

	processors, err := createProcessorFactories()
	if err != nil {
		errs = errors.Join(errs, err)
	}

	exporters, err := createExporterFactories()
	if err != nil {
		errs = errors.Join(errs, err)
	}

	factories := otelcol.Factories{
		Connectors: connectors,
		Extensions: extensions,
		Receivers:  receivers,
		Processors: processors,
		Exporters:  exporters,
	}

	return factories, errs
}

func createConnectorFactories() (map[component.Type]connector.Factory, error) {
	connectorsList := []connector.Factory{
		forwardconnector.NewFactory(),
		countconnector.NewFactory(),
		exceptionsconnector.NewFactory(),
		grafanacloudconnector.NewFactory(),
		routingconnector.NewFactory(),
		servicegraphconnector.NewFactory(),
		spanmetricsconnector.NewFactory(),
	}

	return connector.MakeFactoryMap(connectorsList...)
}

func createExtensionFactories() (map[component.Type]extension.Factory, error) {
	extensionsList := []extension.Factory{
		ackextension.NewFactory(),
		asapauthextension.NewFactory(),
		awsproxy.NewFactory(),
		basicauthextension.NewFactory(),
		bearertokenauthextension.NewFactory(),
		dockerobserver.NewFactory(),
		headerssetterextension.NewFactory(),
		healthcheckextension.NewFactory(),
		hostobserver.NewFactory(),
		httpforwarderextension.NewFactory(),
		k8sobserver.NewFactory(),
		oauth2clientauthextension.NewFactory(),
		oidcauthextension.NewFactory(),
		pprofextension.NewFactory(),
		sigv4authextension.NewFactory(),
	}

	return extension.MakeFactoryMap(extensionsList...)
}

func createReceiverFactories() (map[component.Type]receiver.Factory, error) {
	receiverList := []receiver.Factory{
		nopreceiver.NewFactory(),
		otlpreceiver.NewFactory(),
		bigipreceiver.NewFactory(),
		dockerstatsreceiver.NewFactory(),
		filelogreceiver.NewFactory(),
		hostmetricsreceiver.NewFactory(),
		httpcheckreceiver.NewFactory(),
		k8sclusterreceiver.NewFactory(),
		k8seventsreceiver.NewFactory(),
		k8sobjectsreceiver.NewFactory(),
		kubeletstatsreceiver.NewFactory(),
		nginxreceiver.NewFactory(),
		podmanreceiver.NewFactory(),
		prometheusreceiver.NewFactory(),
		simpleprometheusreceiver.NewFactory(),
		syslogreceiver.NewFactory(),
		tcplogreceiver.NewFactory(),
		udplogreceiver.NewFactory(),
		nginxplusreceiver.NewFactory(),
	}

	return receiver.MakeFactoryMap(receiverList...)
}

func createProcessorFactories() (map[component.Type]processor.Factory, error) {
	processorList := []processor.Factory{
		attributesprocessor.NewFactory(),
		batchprocessor.NewFactory(),
		cumulativetodeltaprocessor.NewFactory(),
		deltatorateprocessor.NewFactory(),
		filterprocessor.NewFactory(),
		groupbyattrsprocessor.NewFactory(),
		groupbytraceprocessor.NewFactory(),
		k8sattributesprocessor.NewFactory(),
		memorylimiterprocessor.NewFactory(),
		metricsgenerationprocessor.NewFactory(),
		metricstransformprocessor.NewFactory(),
		probabilisticsamplerprocessor.NewFactory(),
		redactionprocessor.NewFactory(),
		remotetapprocessor.NewFactory(),
		resourcedetectionprocessor.NewFactory(),
		resourceprocessor.NewFactory(),
		routingprocessor.NewFactory(),
		spanprocessor.NewFactory(),
		tailsamplingprocessor.NewFactory(),
		transformprocessor.NewFactory(),
	}

	return processor.MakeFactoryMap(processorList...)
}

func createExporterFactories() (map[component.Type]exporter.Factory, error) {
	exporterList := []exporter.Factory{
		debugexporter.NewFactory(),
		prometheusexporter.NewFactory(),
		otlpexporter.NewFactory(),
		otlphttpexporter.NewFactory(),
	}

	return exporter.MakeFactoryMap(exporterList...)
}
