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
	"os"
	"strings"
	"sync"
	"time"

	"github.com/nginx/agent/v3/api/grpc/mpi/v1"
	"github.com/nginx/agent/v3/internal/backoff"
	"github.com/nginx/agent/v3/internal/bus"
	"github.com/nginx/agent/v3/internal/config"
	"github.com/nginx/agent/v3/internal/model"
	"go.opentelemetry.io/collector/otelcol"
)

const (
	maxTimeToWaitForShutdown = 30 * time.Second
	filePermission           = 0o600
)

type (
	// Collector The OTel collector plugin start an embedded OTel collector for metrics collection in the OTel format.
	Collector struct {
		service *otelcol.Collector
		cancel  context.CancelFunc
		config  *config.Config
		mu      *sync.Mutex
		stopped bool
	}
)

var _ bus.Plugin = (*Collector)(nil)

// NewCollector is the constructor for the Collector plugin.
func New(conf *config.Config) (*Collector, error) {
	if conf == nil {
		return nil, errors.New("nil agent config")
	}

	if conf.Collector == nil {
		return nil, errors.New("nil collector config")
	}

	if conf.Collector.Log != nil && conf.Collector.Log.Path != "" {
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
		config:  conf,
		service: oTelCollector,
		stopped: true,
		mu:      &sync.Mutex{},
	}, nil
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

	bootErr := oc.bootup(runCtx)
	if bootErr != nil {
		slog.ErrorContext(runCtx, "Unable to start OTel Collector", "error", bootErr)
	}

	return nil
}

// Process receivers and log warning for sub-optimal configurations
func (oc *Collector) processReceivers(ctx context.Context, receivers []config.OtlpReceiver) {
	for _, receiver := range receivers {
		if receiver.OtlpTLSConfig == nil {
			slog.WarnContext(ctx, "OTEL receiver is configured without TLS. Connections are unencrypted.")
			continue
		}

		if receiver.OtlpTLSConfig.GenerateSelfSignedCert {
			slog.WarnContext(ctx,
				"Self-signed certificate for OTEL receiver requested, "+
					"this is not recommended for production environments.",
			)

			if receiver.OtlpTLSConfig.ExistingCert {
				slog.WarnContext(ctx,
					"Certificate file already exists, skipping self-signed certificate generation",
				)
			}
		} else {
			slog.WarnContext(ctx, "OTEL receiver is configured without TLS. Connections are unencrypted.")
		}
	}
}

func (oc *Collector) bootup(ctx context.Context) error {
	slog.InfoContext(ctx, "Starting OTel collector")
	errChan := make(chan error)

	go func() {
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

		settings := oc.config.Common
		settings.MaxElapsedTime = maxTimeToWaitForShutdown
		err := backoff.WaitUntil(ctx, oc.config.Common, func() error {
			if oc.service.GetState() == otelcol.StateClosed {
				return nil
			}

			return errors.New("OTel Collector not in a closed state yet")
		})

		if err != nil {
			slog.ErrorContext(ctx, "Failed to shutdown OTel Collector", "error", err, "state", oc.service.GetState())
		} else {
			slog.InfoContext(ctx, "OTel Collector shutdown", "state", oc.service.GetState())
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

func (oc *Collector) handleNginxConfigUpdate(ctx context.Context, msg *bus.Message) {
	nginxConfigContext, ok := msg.Data.(*model.NginxConfigContext)
	if !ok {
		slog.ErrorContext(ctx, "Unable to cast message payload to *model.NginxConfigContext", "payload", msg.Data)
		return
	}

	reloadCollector := oc.checkForNewNginxReceivers(nginxConfigContext)

	if reloadCollector {
		slog.InfoContext(ctx, "Reloading OTel collector config")
		err := writeCollectorConfig(oc.config.Collector)
		if err != nil {
			slog.ErrorContext(ctx, "Failed to write OTel Collector config", "error", err)
			return
		}

		oc.restartCollector(ctx)
	}
}

func (oc *Collector) handleResourceUpdate(ctx context.Context, msg *bus.Message) {
	var reloadCollector bool
	resourceUpdateContext, ok := msg.Data.(*v1.Resource)
	if !ok {
		slog.ErrorContext(ctx, "Unable to cast message payload to *v1.Resource", "payload", msg.Data)
		return
	}

	if oc.config.Collector.Processors.Attribute == nil {
		oc.config.Collector.Processors.Attribute = &config.Attribute{
			Actions: make([]config.Action, 0),
		}
	}

	if oc.config.Collector.Processors.Attribute != nil &&
		resourceUpdateContext.GetResourceId() != "" {
		reloadCollector = oc.updateAttributeActions(
			[]config.Action{
				{
					Key:    "resource.id",
					Action: "insert",
					Value:  resourceUpdateContext.GetResourceId(),
				},
			},
		)
	}

	if reloadCollector {
		slog.InfoContext(ctx, "Reloading OTel collector config")
		err := writeCollectorConfig(oc.config.Collector)
		if err != nil {
			slog.ErrorContext(ctx, "Failed to write OTel Collector config", "error", err)
			return
		}

		oc.restartCollector(ctx)
	}
}

func (oc *Collector) restartCollector(ctx context.Context) {
	oc.mu.Lock()
	defer oc.mu.Unlock()
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

	bootErr := oc.bootup(runCtx)
	if bootErr != nil {
		slog.ErrorContext(runCtx, "Unable to start OTel Collector", "error", bootErr)
	}
}

func (oc *Collector) checkForNewNginxReceivers(nginxConfigContext *model.NginxConfigContext) bool {
	nginxReceiverFound, reloadCollector := oc.updateExistingNginxPlusReceiver(nginxConfigContext)

	if !nginxReceiverFound && nginxConfigContext.PlusAPI != "" {
		oc.config.Collector.Receivers.NginxPlusReceivers = append(
			oc.config.Collector.Receivers.NginxPlusReceivers,
			config.NginxPlusReceiver{
				InstanceID: nginxConfigContext.InstanceID,
				PlusAPI:    nginxConfigContext.PlusAPI,
			},
		)

		reloadCollector = true
	} else if nginxConfigContext.PlusAPI == "" {
		nginxReceiverFound, reloadCollector = oc.updateExistingNginxOSSReceiver(nginxConfigContext)

		if !nginxReceiverFound && nginxConfigContext.StubStatus != "" {
			oc.config.Collector.Receivers.NginxReceivers = append(
				oc.config.Collector.Receivers.NginxReceivers,
				config.NginxReceiver{
					InstanceID: nginxConfigContext.InstanceID,
					StubStatus: nginxConfigContext.StubStatus,
					AccessLogs: toConfigAccessLog(nginxConfigContext.AccessLogs),
				},
			)

			reloadCollector = true
		}
	}

	return reloadCollector
}

func (oc *Collector) updateExistingNginxPlusReceiver(
	nginxConfigContext *model.NginxConfigContext,
) (nginxReceiverFound, reloadCollector bool) {
	for index, nginxPlusReceiver := range oc.config.Collector.Receivers.NginxPlusReceivers {
		if nginxPlusReceiver.InstanceID == nginxConfigContext.InstanceID {
			nginxReceiverFound = true

			if nginxPlusReceiver.PlusAPI != nginxConfigContext.PlusAPI {
				oc.config.Collector.Receivers.NginxPlusReceivers = append(
					oc.config.Collector.Receivers.NginxPlusReceivers[:index],
					oc.config.Collector.Receivers.NginxPlusReceivers[index+1:]...,
				)
				if nginxConfigContext.PlusAPI != "" {
					nginxPlusReceiver.PlusAPI = nginxConfigContext.PlusAPI
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
				if nginxConfigContext.StubStatus != "" {
					nginxReceiver.StubStatus = nginxConfigContext.StubStatus
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

// nolint: revive
func (oc *Collector) updateAttributeActions(
	actionsToAdd []config.Action,
) (reloadCollector bool) {
	reloadCollector = false

	if oc.config.Collector.Processors.Attribute.Actions != nil {
	OUTER:
		for _, toAdd := range actionsToAdd {
			for _, action := range oc.config.Collector.Processors.Attribute.Actions {
				if action.Key == toAdd.Key {
					continue OUTER
				}
			}
			oc.config.Collector.Processors.Attribute.Actions = append(
				oc.config.Collector.Processors.Attribute.Actions,
				toAdd,
			)
			reloadCollector = true
		}
	}

	return reloadCollector
}

func isOSSReceiverChanged(nginxReceiver config.NginxReceiver, nginxConfigContext *model.NginxConfigContext) bool {
	return nginxReceiver.StubStatus != nginxConfigContext.StubStatus ||
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
