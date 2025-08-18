// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.
package collector

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"os"
	"strings"
	"sync"
	"time"

	pkgConfig "github.com/nginx/agent/v3/pkg/config"

	"github.com/nginx/agent/v3/api/grpc/mpi/v1"
	"github.com/nginx/agent/v3/internal/backoff"
	"github.com/nginx/agent/v3/internal/bus"
	"github.com/nginx/agent/v3/internal/collector/types"
	"github.com/nginx/agent/v3/internal/config"
	"github.com/nginx/agent/v3/internal/model"
	"go.opentelemetry.io/collector/otelcol"
)

const (
	maxTimeToWaitForShutdown  = 30 * time.Second
	defaultCollectionInterval = 1 * time.Minute
	filePermission            = 0o600
	// To conform to the rfc3164 spec the timestamp in the logs need to be formatted correctly.
	// Here are some examples of what the timestamp conversions look like.
	// Notice how if the day begins with a zero that the zero is replaced with an empty space.

	// 2024-11-06T17:19:24+00:00 ---> Nov  6 17:19:24
	// 2024-11-16T17:19:24+00:00 ---> Nov 16 17:19:24
	timestampConversionExpression = `'EXPR(let timestamp = split(split(body, ">")[1], " ")[0]; ` +
		`let newTimestamp = ` +
		`timestamp matches "(\\d{4})-(\\d{2})-(0\\d{1})T(\\d{2}):(\\d{2}):(\\d{2})([+-]\\d{2}:\\d{2}|Z)" ` +
		`? (let utcTime = ` +
		`date(timestamp).UTC(); utcTime.Format("Jan  2 15:04:05")) : date(timestamp).Format("Jan 02 15:04:05"); ` +
		`split(body, ">")[0] + ">" + newTimestamp + " " + split(body, " ", 2)[1])'`
)

type (
	// Collector The OTel collector plugin start an embedded OTel collector for metrics collection in the OTel format.
	Collector struct {
		service                 types.CollectorInterface
		config                  *config.Config
		mu                      *sync.Mutex
		cancel                  context.CancelFunc
		previousNAPSysLogServer string
		stopped                 bool
	}
)

var (
	_         bus.Plugin = (*Collector)(nil)
	initMutex            = &sync.Mutex{}
)

// NewCollector is the constructor for the Collector plugin.
func NewCollector(conf *config.Config) (*Collector, error) {
	initMutex.Lock()

	defer initMutex.Unlock()
	if conf == nil {
		return nil, errors.New("nil agent config")
	}

	if conf.Collector == nil {
		return nil, errors.New("nil collector config")
	}

	if conf.Collector.Log != nil && conf.Collector.Log.Path != "" && conf.Collector.Log.Path != "stdout" {
		err := os.WriteFile(conf.Collector.Log.Path, []byte{}, filePermission)
		if err != nil {
			return nil, err
		}
	}

	settings := OTelCollectorSettings(conf)
	oTelCollector, err := otelcol.NewCollector(settings)
	if err != nil {
		return nil, err
	}

	return &Collector{
		config:                  conf,
		service:                 oTelCollector,
		stopped:                 true,
		mu:                      &sync.Mutex{},
		previousNAPSysLogServer: "",
	}, nil
}

func (oc *Collector) State() otelcol.State {
	oc.mu.Lock()
	defer oc.mu.Unlock()

	return oc.service.GetState()
}

// Init initializes and starts the plugin
func (oc *Collector) Init(ctx context.Context, mp bus.MessagePipeInterface) error {
	slog.InfoContext(ctx, "Starting OTel Collector plugin")

	var runCtx context.Context
	runCtx, oc.cancel = context.WithCancel(ctx)

	if !oc.config.AreReceiversConfigured() {
		slog.InfoContext(runCtx, "No receivers configured for OTel Collector. "+
			"Waiting to discover a receiver before starting OTel collector.")

		return nil
	}

	err := writeCollectorConfig(oc.config.Collector)
	if err != nil {
		return fmt.Errorf("write OTel Collector config: %w", err)
	}

	if oc.config.Collector.Receivers.OtlpReceivers != nil {
		oc.processReceivers(ctx, oc.config.Collector.Receivers.OtlpReceivers)
	}

	if !oc.stopped {
		return errors.New("OTel collector already running")
	}

	slog.InfoContext(ctx, "Starting OTel collector")
	bootErr := oc.bootup(runCtx)
	if bootErr != nil {
		slog.ErrorContext(runCtx, "Unable to start OTel Collector", "error", bootErr)
	}

	return nil
}

// Info the plugin.
func (oc *Collector) Info() *bus.Info {
	return &bus.Info{
		Name: "collector",
	}
}

// Close the plugin.
func (oc *Collector) Close(ctx context.Context) error {
	slog.InfoContext(ctx, "Closing OTel Collector plugin")

	if !oc.stopped {
		slog.InfoContext(ctx, "Shutting down OTel Collector", "state", oc.service.GetState())
		oc.service.Shutdown()
		oc.cancel()

		settings := *oc.config.Client.Backoff
		settings.MaxElapsedTime = maxTimeToWaitForShutdown
		err := backoff.WaitUntil(ctx, &settings, func() error {
			if oc.service.GetState() == otelcol.StateClosed {
				return nil
			}

			return errors.New("OTel Collector not in a closed state yet")
		})

		if err != nil {
			slog.ErrorContext(ctx, "Failed to shutdown OTel Collector", "error", err, "state", oc.service.GetState())
		} else {
			slog.InfoContext(ctx, "OTel Collector shutdown", "state", oc.service.GetState())
			oc.stopped = true
		}
	}

	return nil
}

// Process an incoming Message Bus message in the plugin
func (oc *Collector) Process(ctx context.Context, msg *bus.Message) {
	switch msg.Topic {
	case bus.NginxConfigUpdateTopic:
		oc.handleNginxConfigUpdate(ctx, msg)
	case bus.ResourceUpdateTopic:
		oc.handleResourceUpdate(ctx, msg)
	default:
		slog.DebugContext(ctx, "OTel collector plugin unknown topic", "topic", msg.Topic)
	}
}

// Subscriptions returns the list of topics the plugin is subscribed to
func (oc *Collector) Subscriptions() []string {
	return []string{
		bus.ResourceUpdateTopic,
		bus.NginxConfigUpdateTopic,
	}
}

// Process receivers and log warning for sub-optimal configurations
func (oc *Collector) processReceivers(ctx context.Context, receivers map[string]*config.OtlpReceiver) {
	for _, receiver := range receivers {
		if receiver.OtlpTLSConfig == nil {
			slog.WarnContext(ctx, "OTel receiver is configured without TLS. Connections are unencrypted.")
			continue
		}

		if receiver.OtlpTLSConfig.GenerateSelfSignedCert {
			slog.WarnContext(ctx,
				"Self-signed certificate for OTel receiver requested, "+
					"this is not recommended for production environments.",
			)

			if receiver.OtlpTLSConfig.ExistingCert {
				slog.WarnContext(ctx,
					"Certificate file already exists, skipping self-signed certificate generation",
				)
			}
		} else {
			slog.WarnContext(ctx, "OTel receiver is configured without TLS. Connections are unencrypted.")
		}
	}
}

// nolint: revive, cyclop
func (oc *Collector) bootup(ctx context.Context) error {
	errChan := make(chan error)

	go func() {
		if oc.service == nil {
			errChan <- errors.New("unable to start OTel collector: service is nil")
			return
		}

		appErr := oc.service.Run(ctx)
		if appErr != nil {
			errChan <- appErr
		}
		slog.InfoContext(ctx, "OTel collector run finished")
	}()

	for {
		select {
		case err := <-errChan:
			return err
		default:
			if oc.service == nil {
				return errors.New("unable to start otel collector: service is nil")
			}

			state := oc.service.GetState()
			switch state {
			case otelcol.StateStarting:
				// NoOp
				continue
			case otelcol.StateRunning:
				oc.stopped = false
				return nil
			case otelcol.StateClosing:
			case otelcol.StateClosed:
				oc.stopped = true
			default:
				return fmt.Errorf("unable to start, otelcol state is %s", state)
			}
		}
	}
}

func (oc *Collector) handleNginxConfigUpdate(ctx context.Context, msg *bus.Message) {
	slog.DebugContext(ctx, "OTel collector plugin received nginx config update message")
	oc.mu.Lock()
	defer oc.mu.Unlock()

	nginxConfigContext, ok := msg.Data.(*model.NginxConfigContext)
	if !ok {
		slog.ErrorContext(ctx, "Unable to cast message payload to *model.NginxConfigContext", "payload", msg.Data)
		return
	}

	reloadCollector := oc.checkForNewReceivers(ctx, nginxConfigContext)

	if reloadCollector {
		slog.InfoContext(ctx, "Reloading OTel collector config, nginx config updated")
		err := writeCollectorConfig(oc.config.Collector)
		if err != nil {
			slog.ErrorContext(ctx, "Failed to write OTel Collector config", "error", err)
			return
		}

		oc.restartCollector(ctx)
	}
}

func (oc *Collector) handleResourceUpdate(ctx context.Context, msg *bus.Message) {
	slog.DebugContext(ctx, "OTel collector plugin received resource update message")
	oc.mu.Lock()
	defer oc.mu.Unlock()

	resourceUpdateContext, ok := msg.Data.(*v1.Resource)
	if !ok {
		slog.ErrorContext(ctx, "Unable to cast message payload to *v1.Resource", "payload", msg.Data)
		return
	}

	resourceProcessorUpdated := oc.updateResourceProcessor(resourceUpdateContext)
	headersSetterExtensionUpdated := oc.updateHeadersSetterExtension(ctx, resourceUpdateContext)

	if resourceProcessorUpdated || headersSetterExtensionUpdated {
		slog.InfoContext(ctx, "Reloading OTel collector config, resource updated")
		err := writeCollectorConfig(oc.config.Collector)
		if err != nil {
			slog.ErrorContext(ctx, "Failed to write OTel Collector config", "error", err)
			return
		}

		oc.restartCollector(ctx)
	}
}

func (oc *Collector) updateResourceProcessor(resourceUpdateContext *v1.Resource) bool {
	resourceProcessorUpdated := false

	if oc.config.Collector.Processors.Resource == nil {
		oc.config.Collector.Processors.Resource = make(map[string]*config.Resource)
		oc.config.Collector.Processors.Resource["default"] = &config.Resource{
			Attributes: make([]config.ResourceAttribute, 0),
		}
	}

	if oc.config.Collector.Processors.Resource["default"] != nil &&
		resourceUpdateContext.GetResourceId() != "" {
		resourceProcessorUpdated = oc.updateResourceAttributes(
			[]config.ResourceAttribute{
				{
					Key:    "resource.id",
					Action: "insert",
					Value:  resourceUpdateContext.GetResourceId(),
				},
			},
		)
	}

	return resourceProcessorUpdated
}

func (oc *Collector) updateHeadersSetterExtension(
	ctx context.Context,
	resourceUpdateContext *v1.Resource,
) bool {
	headersSetterExtensionUpdated := false

	if oc.config.Collector.Extensions.HeadersSetter != nil &&
		oc.config.Collector.Extensions.HeadersSetter.Headers != nil {
		isUUIDHeaderSet := false
		for _, header := range oc.config.Collector.Extensions.HeadersSetter.Headers {
			if header.Key == "uuid" {
				isUUIDHeaderSet = true
				break
			}
		}

		if !isUUIDHeaderSet {
			slog.DebugContext(
				ctx, "Adding uuid header to OTel collector",
				"uuid", resourceUpdateContext.GetResourceId(),
			)
			oc.config.Collector.Extensions.HeadersSetter.Headers = append(
				oc.config.Collector.Extensions.HeadersSetter.Headers,
				config.Header{
					Action: "insert",
					Key:    "uuid",
					Value:  resourceUpdateContext.GetResourceId(),
				},
			)

			headersSetterExtensionUpdated = true
		}
	}

	return headersSetterExtensionUpdated
}

func (oc *Collector) restartCollector(ctx context.Context) {
	err := oc.Close(ctx)
	if err != nil {
		slog.ErrorContext(ctx, "Failed to shutdown OTel Collector", "error", err)
		return
	}

	settings := OTelCollectorSettings(oc.config)
	oTelCollector, err := otelcol.NewCollector(settings)
	if err != nil {
		slog.ErrorContext(ctx, "Failed to create OTel Collector", "error", err)
		return
	}
	oc.service = oTelCollector

	var runCtx context.Context
	runCtx, oc.cancel = context.WithCancel(ctx)

	if !oc.stopped {
		slog.ErrorContext(ctx, "Unable to restart OTel collector, failed to stop collector")
		return
	}

	slog.InfoContext(ctx, "Restarting OTel collector")
	bootErr := oc.bootup(runCtx)
	if bootErr != nil {
		slog.ErrorContext(runCtx, "Unable to start OTel Collector", "error", bootErr)
	}
}

func (oc *Collector) checkForNewReceivers(ctx context.Context, nginxConfigContext *model.NginxConfigContext) bool {
	nginxReceiverFound, reloadCollector := oc.updateExistingNginxPlusReceiver(nginxConfigContext)

	if !nginxReceiverFound && nginxConfigContext.PlusAPI.URL != "" {
		slog.DebugContext(ctx, "Adding new NGINX Plus receiver", "url", nginxConfigContext.PlusAPI.URL)
		oc.config.Collector.Receivers.NginxPlusReceivers = append(
			oc.config.Collector.Receivers.NginxPlusReceivers,
			config.NginxPlusReceiver{
				InstanceID: nginxConfigContext.InstanceID,
				PlusAPI: config.APIDetails{
					URL:      nginxConfigContext.PlusAPI.URL,
					Listen:   nginxConfigContext.PlusAPI.Listen,
					Location: nginxConfigContext.PlusAPI.Location,
					Ca:       nginxConfigContext.PlusAPI.Ca,
				},
				CollectionInterval: defaultCollectionInterval,
			},
		)
		slog.DebugContext(ctx, "NGINX Plus API found, NGINX Plus receiver enabled to scrape metrics")

		reloadCollector = true
	} else if nginxConfigContext.PlusAPI.URL == "" {
		slog.WarnContext(ctx, "NGINX Plus API is not configured, searching for stub status endpoint")
		reloadCollector = oc.addNginxOssReceiver(ctx, nginxConfigContext)
	}

	if oc.config.IsFeatureEnabled(pkgConfig.FeatureLogsNap) {
		tcplogReceiversFound := oc.updateNginxAppProtectTcplogReceivers(nginxConfigContext)
		if tcplogReceiversFound {
			reloadCollector = true
		}
	} else {
		slog.DebugContext(ctx, "NAP logs feature disabled", "enabled_features", oc.config.Features)
	}

	return reloadCollector
}

func (oc *Collector) addNginxOssReceiver(ctx context.Context, nginxConfigContext *model.NginxConfigContext) bool {
	nginxReceiverFound, reloadCollector := oc.updateExistingNginxOSSReceiver(nginxConfigContext)

	if !nginxReceiverFound && nginxConfigContext.StubStatus.URL != "" {
		slog.DebugContext(ctx, "Adding new NGINX OSS receiver", "url", nginxConfigContext.StubStatus.URL)
		oc.config.Collector.Receivers.NginxReceivers = append(
			oc.config.Collector.Receivers.NginxReceivers,
			config.NginxReceiver{
				InstanceID: nginxConfigContext.InstanceID,
				StubStatus: config.APIDetails{
					URL:      nginxConfigContext.StubStatus.URL,
					Listen:   nginxConfigContext.StubStatus.Listen,
					Location: nginxConfigContext.StubStatus.Location,
				},
				AccessLogs:         toConfigAccessLog(nginxConfigContext.AccessLogs),
				CollectionInterval: defaultCollectionInterval,
			},
		)
		slog.DebugContext(ctx, "Stub status endpoint found, OSS receiver enabled to scrape metrics")

		reloadCollector = true
	} else if nginxConfigContext.StubStatus.URL == "" {
		slog.WarnContext(ctx, "Stub status endpoint not found, NGINX metrics not available")
	}

	return reloadCollector
}

func (oc *Collector) updateExistingNginxPlusReceiver(
	nginxConfigContext *model.NginxConfigContext,
) (nginxReceiverFound, reloadCollector bool) {
	for index, nginxPlusReceiver := range oc.config.Collector.Receivers.NginxPlusReceivers {
		if nginxPlusReceiver.InstanceID == nginxConfigContext.InstanceID {
			nginxReceiverFound = true

			if nginxPlusReceiver.PlusAPI.URL != nginxConfigContext.PlusAPI.URL {
				oc.config.Collector.Receivers.NginxPlusReceivers = append(
					oc.config.Collector.Receivers.NginxPlusReceivers[:index],
					oc.config.Collector.Receivers.NginxPlusReceivers[index+1:]...,
				)
				if nginxConfigContext.PlusAPI.URL != "" {
					slog.Debug("Updating existing NGINX Plus receiver", "url",
						nginxConfigContext.PlusAPI.URL)
					nginxPlusReceiver.PlusAPI.URL = nginxConfigContext.PlusAPI.URL
					oc.config.Collector.Receivers.NginxPlusReceivers = append(
						oc.config.Collector.Receivers.NginxPlusReceivers,
						nginxPlusReceiver,
					)
				}

				reloadCollector = true
				nginxReceiverFound = true
			}

			return nginxReceiverFound, reloadCollector
		}
	}

	return nginxReceiverFound, reloadCollector
}

func (oc *Collector) updateExistingNginxOSSReceiver(
	nginxConfigContext *model.NginxConfigContext,
) (nginxReceiverFound, reloadCollector bool) {
	for index, nginxReceiver := range oc.config.Collector.Receivers.NginxReceivers {
		if nginxReceiver.InstanceID == nginxConfigContext.InstanceID {
			nginxReceiverFound = true

			if isOSSReceiverChanged(nginxReceiver, nginxConfigContext) {
				oc.config.Collector.Receivers.NginxReceivers = append(
					oc.config.Collector.Receivers.NginxReceivers[:index],
					oc.config.Collector.Receivers.NginxReceivers[index+1:]...,
				)
				if nginxConfigContext.StubStatus.URL != "" {
					slog.Debug("Updating existing NGINX OSS receiver", "url",
						nginxConfigContext.StubStatus.URL)
					nginxReceiver.StubStatus = config.APIDetails{
						URL:      nginxConfigContext.StubStatus.URL,
						Listen:   nginxConfigContext.StubStatus.Listen,
						Location: nginxConfigContext.StubStatus.Location,
					}
					nginxReceiver.AccessLogs = toConfigAccessLog(nginxConfigContext.AccessLogs)
					oc.config.Collector.Receivers.NginxReceivers = append(
						oc.config.Collector.Receivers.NginxReceivers,
						nginxReceiver,
					)
				}

				reloadCollector = true
				nginxReceiverFound = true
			}

			return nginxReceiverFound, reloadCollector
		}
	}

	return nginxReceiverFound, reloadCollector
}

func (oc *Collector) updateNginxAppProtectTcplogReceivers(nginxConfigContext *model.NginxConfigContext) bool {
	newTcplogReceiverAdded := false

	if oc.config.Collector.Receivers.TcplogReceivers == nil {
		oc.config.Collector.Receivers.TcplogReceivers = make(map[string]*config.TcplogReceiver)
	}

	napSysLogServer := oc.findAvailableSyslogServers(nginxConfigContext.NAPSysLogServers)

	if napSysLogServer != "" {
		if !oc.doesTcplogReceiverAlreadyExist(napSysLogServer) {
			oc.config.Collector.Receivers.TcplogReceivers["nginx_app_protect"] = &config.TcplogReceiver{
				ListenAddress: napSysLogServer,
				Operators: []config.Operator{
					// regex captures the priority number from the log line
					{
						Type: "regex_parser",
						Fields: map[string]string{
							"regex":      "^<(?P<priority>\\d+)>",
							"parse_from": "body",
							"parse_to":   "attributes",
						},
					},
					// filter drops all logs that have a severity above 4
					// https://docs.secureauth.com/0902/en/how-to-read-a-syslog-message.html#severity-code-table
					{
						Type: "filter",
						Fields: map[string]string{
							"expr":       "'int(attributes.priority) % 8 > 4'",
							"drop_ratio": "1.0",
						},
					},
					{
						Type: "add",
						Fields: map[string]string{
							"field": "body",
							"value": timestampConversionExpression,
						},
					},
					{
						Type: "syslog_parser",
						Fields: map[string]string{
							"protocol": "rfc3164",
						},
					},
					{
						Type: "remove",
						Fields: map[string]string{
							"field": "attributes.message",
						},
					},
					{
						Type: "add",
						Fields: map[string]string{
							"field": "resource[\"instance.id\"]",
							"value": nginxConfigContext.InstanceID,
						},
					},
				},
			}

			newTcplogReceiverAdded = true
		}
	}

	tcplogReceiverDeleted := oc.areNapReceiversDeleted(napSysLogServer)

	return newTcplogReceiverAdded || tcplogReceiverDeleted
}

func (oc *Collector) areNapReceiversDeleted(napSysLogServer string) bool {
	listenAddressesToBeDeleted := oc.configDeletedNapReceivers(napSysLogServer)
	if len(listenAddressesToBeDeleted) != 0 {
		delete(oc.config.Collector.Receivers.TcplogReceivers, "nginx_app_protect")
		return true
	}

	return false
}

func (oc *Collector) configDeletedNapReceivers(napSysLogServer string) map[string]bool {
	elements := make(map[string]bool)

	for _, tcplogReceiver := range oc.config.Collector.Receivers.TcplogReceivers {
		elements[tcplogReceiver.ListenAddress] = true
	}

	if napSysLogServer != "" {
		addressesToDelete := make(map[string]bool)
		if !elements[napSysLogServer] {
			addressesToDelete[napSysLogServer] = true
		}

		return addressesToDelete
	}

	return elements
}

func (oc *Collector) doesTcplogReceiverAlreadyExist(listenAddress string) bool {
	for _, tcplogReceiver := range oc.config.Collector.Receivers.TcplogReceivers {
		if listenAddress == tcplogReceiver.ListenAddress {
			return true
		}
	}

	return false
}

// nolint: revive
func (oc *Collector) updateResourceAttributes(
	attributesToAdd []config.ResourceAttribute,
) (actionUpdated bool) {
	actionUpdated = false

	if oc.config.Collector.Processors.Resource["default"].Attributes != nil {
	OUTER:
		for _, toAdd := range attributesToAdd {
			for _, action := range oc.config.Collector.Processors.Resource["default"].Attributes {
				if action.Key == toAdd.Key {
					continue OUTER
				}
			}
			oc.config.Collector.Processors.Resource["default"].Attributes = append(
				oc.config.Collector.Processors.Resource["default"].Attributes,
				toAdd,
			)
			actionUpdated = true
		}
	}

	return actionUpdated
}

func (oc *Collector) findAvailableSyslogServers(napSyslogServers []string) string {
	napSyslogServersMap := make(map[string]bool)
	for _, server := range napSyslogServers {
		napSyslogServersMap[server] = true
	}

	if oc.previousNAPSysLogServer != "" {
		if _, ok := napSyslogServersMap[oc.previousNAPSysLogServer]; ok {
			return oc.previousNAPSysLogServer
		}
	}

	for _, napSyslogServer := range napSyslogServers {
		ln, err := net.Listen("tcp", napSyslogServer)
		if err != nil {
			slog.Debug("NAP syslog server is not reachable", "address", napSyslogServer,
				"error", err)

			continue
		}
		closeError := ln.Close()
		if closeError != nil {
			slog.Debug("Failed to close syslog server", "address", napSyslogServer, "error", closeError)
		}

		slog.Debug("Found valid NAP syslog server", "address", napSyslogServer)

		return napSyslogServer
	}

	return ""
}

func isOSSReceiverChanged(nginxReceiver config.NginxReceiver, nginxConfigContext *model.NginxConfigContext) bool {
	return nginxReceiver.StubStatus.URL != nginxConfigContext.StubStatus.URL ||
		len(nginxReceiver.AccessLogs) != len(nginxConfigContext.AccessLogs)
}

func toConfigAccessLog(al []*model.AccessLog) []config.AccessLog {
	if al == nil {
		return nil
	}

	results := make([]config.AccessLog, 0, len(al))
	for _, ctxAccessLog := range al {
		results = append(results, config.AccessLog{
			LogFormat: escapeString(ctxAccessLog.Format),
			FilePath:  ctxAccessLog.Name,
		})
	}

	return results
}

func escapeString(input string) string {
	output := strings.ReplaceAll(input, "$", "$$")
	output = strings.ReplaceAll(output, "\"", "\\\"")

	return output
}
