/**
 * Copyright (c) F5, Inc.
 *
 * This source code is licensed under the Apache License, Version 2.0 license found in the
 * LICENSE file in the root directory of this source tree.
 */

package plugins

import (
	"context"
	"fmt"

	"github.com/gogo/protobuf/types"
	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"

	agent_config "github.com/nginx/agent/sdk/v2/agent/config"
	"github.com/nginx/agent/sdk/v2/agent/events"
	"github.com/nginx/agent/sdk/v2/proto"
	eventsProto "github.com/nginx/agent/sdk/v2/proto/events"
	"github.com/nginx/agent/v2/src/core"
	"github.com/nginx/agent/v2/src/core/config"
)

const (
	NGINX_FOUND_MESSAGE             = "nginx-v%s master process was found with a pid %s"
	NGINX_STOP_MESSAGE              = "nginx-v%s master process (pid: %s) stopped"
	NGINX_RELOAD_SUCCESS_MESSAGE    = "nginx-v%s master process (pid: %s) reloaded successfully"
	NGINX_RELOAD_FAILED_MESSAGE     = "nginx-v%s master process (pid: %s) failed to reload"
	NGINX_WORKER_START_MESSAGE      = "new worker process started with pid %s for nginx-v%s process (pid: %s)"
	NGINX_WORKER_STOP_MESSAGE       = "worker process with pid %s is shutting down for nginx-v%s process (pid: %s)"
	CONFIG_APPLY_SUCCESS_MESSAGE    = "successfully applied config on %s"
	CONFIG_APPLY_FAILURE_MESSAGE    = "failed to apply nginx config on %s"
	CONFIG_ROLLBACK_SUCCESS_MESSAGE = "nginx config was rolled back on %s"
	CONFIG_ROLLBACK_FAILURE_MESSAGE = "failed to rollback nginx config on %s"
)

type Events struct {
	pipeline        core.MessagePipeInterface
	ctx             context.Context
	conf            *config.Config
	env             core.Environment
	meta            *proto.Metadata
	nginxBinary     core.NginxBinary
	agentEventsMeta *events.AgentEventMeta
}

func NewEvents(conf *config.Config, env core.Environment, meta *proto.Metadata, nginxBinary core.NginxBinary, agentEventsMeta *events.AgentEventMeta) *Events {
	return &Events{
		conf:            conf,
		env:             env,
		meta:            meta,
		nginxBinary:     nginxBinary,
		agentEventsMeta: agentEventsMeta,
	}
}

func (a *Events) Init(pipeline core.MessagePipeInterface) {
	log.Info("Events initializing")
	a.pipeline = pipeline
	a.ctx = pipeline.Context()
}

func (a *Events) Close() {
	log.Info("Events is wrapping up")
}

func (a *Events) Process(msg *core.Message) {
	log.Debugf("Process function in the events.go, %s %v", msg.Topic(), msg.Data())

	switch {
	case msg.Exact(core.AgentStarted):
		a.sendAgentStartedEvent(msg)
	case msg.Exact(core.NginxInstancesFound):
		a.sendNginxFoundEvent(msg)
	case msg.Exact(core.NginxReloadComplete):
		a.sendNginxReloadEvent(msg)
	case msg.Exact(core.CommResponse):
		a.sendConfigApplyEvent(msg)
	case msg.Exact(core.ConfigRollbackResponse):
		a.sendConfigRollbackEvent(msg)
	case msg.Exact(core.NginxMasterProcCreated):
		a.sendNginxStartEvent(msg)
	case msg.Exact(core.NginxMasterProcKilled):
		a.sendNginxStopEvent(msg)
	case msg.Exact(core.NginxWorkerProcCreated):
		a.sendNginxWorkerStartEvent(msg)
	case msg.Exact(core.NginxWorkerProcKilled):
		a.sendNginxWorkerStopEvent(msg)
	}
}

func (a *Events) Info() *core.Info {
	return core.NewInfo(agent_config.FeatureActivityEvents, "v0.0.1")
}

func (a *Events) Subscriptions() []string {
	return []string{
		core.AgentStarted,
		core.NginxInstancesFound,
		core.NginxReloadComplete,
		core.CommResponse,
		core.ConfigRollbackResponse,
		core.NginxMasterProcCreated,
		core.NginxMasterProcKilled,
		core.NginxWorkerProcCreated,
		core.NginxWorkerProcKilled,
	}
}

func (a *Events) sendAgentStartedEvent(msg *core.Message) {
	agentEventMeta, ok := msg.Data().(*events.AgentEventMeta)
	if !ok {
		log.Warnf("Invalid message received, %T, for topic, %s", msg.Data(), msg.Topic())
		return
	}

	event := agentEventMeta.GenerateAgentStartEventCommand()

	log.Debugf("Created event: %v", event)
	a.pipeline.Process(core.NewMessage(core.Events, event))
}

func (a *Events) sendNginxFoundEvent(msg *core.Message) {
	nginxDetailsMap, ok := msg.Data().(map[string]*proto.NginxDetails)
	if !ok {
		log.Warnf("Invalid message received, %T, for topic, %s", msg.Data(), msg.Topic())
		return
	}

	protoEvents := []*eventsProto.Event{}

	for _, nginxDetail := range nginxDetailsMap {
		event := a.createNginxEvent(
			nginxDetail.GetNginxId(),
			&types.Timestamp{Seconds: nginxDetail.GetStartTime() / 1000, Nanos: int32(nginxDetail.GetStartTime() % 1000)},
			events.INFO_EVENT_LEVEL,
			fmt.Sprintf(NGINX_FOUND_MESSAGE, nginxDetail.GetVersion(), nginxDetail.GetProcessId()),
			uuid.NewString(),
		)

		log.Debugf("Created event: %v", event)
		protoEvents = append(protoEvents, event)
	}

	a.pipeline.Process(core.NewMessage(core.Events, &proto.Command{
		Meta: a.meta,
		Type: proto.Command_NORMAL,
		Data: &proto.Command_EventReport{
			EventReport: &eventsProto.EventReport{
				Events: protoEvents,
			},
		},
	}))
}

func (a *Events) sendNginxReloadEvent(msg *core.Message) {
	nginxReload, ok := msg.Data().(NginxReloadResponse)
	if !ok {
		log.Warnf("Invalid message received, %T, for topic, %s", msg.Data(), msg.Topic())
		return
	}

	var event *eventsProto.Event
	if nginxReload.succeeded {
		event = a.createNginxEvent(
			nginxReload.nginxDetails.GetNginxId(),
			nginxReload.timestamp,
			events.WARN_EVENT_LEVEL,
			fmt.Sprintf(NGINX_RELOAD_SUCCESS_MESSAGE, nginxReload.nginxDetails.GetVersion(), nginxReload.nginxDetails.GetProcessId()),
			nginxReload.correlationId,
		)
	} else {
		event = a.createNginxEvent(
			nginxReload.nginxDetails.GetNginxId(),
			nginxReload.timestamp,
			events.ERROR_EVENT_LEVEL,
			fmt.Sprintf(NGINX_RELOAD_FAILED_MESSAGE, nginxReload.nginxDetails.GetVersion(), nginxReload.nginxDetails.GetProcessId()),
			nginxReload.correlationId,
		)
	}

	log.Debugf("Created event: %v", event)
	a.pipeline.Process(core.NewMessage(core.Events, &proto.Command{
		Meta: a.meta,
		Type: proto.Command_NORMAL,
		Data: &proto.Command_EventReport{
			EventReport: &eventsProto.EventReport{
				Events: []*eventsProto.Event{event},
			},
		},
	}))
}

func (a *Events) sendConfigApplyEvent(msg *core.Message) {
	command, ok := msg.Data().(*proto.Command)
	if !ok {
		log.Warnf("Invalid message received, %T, for topic, %s", msg.Data(), msg.Topic())
		return
	}

	nginxConfigResponse := command.GetNginxConfigResponse()

	log.Debugf("nginxConfigResponse: %v", nginxConfigResponse)
	log.Debugf("nginxConfigResponse.GetConfigData(): %v", nginxConfigResponse.GetConfigData())

	switch action := nginxConfigResponse.GetAction(); action {
	case proto.NginxConfigAction_ROLLBACK, proto.NginxConfigAction_RETURN, proto.NginxConfigAction_UNKNOWN, proto.NginxConfigAction_TEST:
		return
	}

	var event *eventsProto.Event

	if nginxConfigResponse.Status.Status == proto.CommandStatusResponse_CMD_OK {
		event = a.createConfigApplyEvent(
			nginxConfigResponse.GetConfigData().GetNginxId(),
			command.GetMeta().GetTimestamp(),
			events.INFO_EVENT_LEVEL,
			fmt.Sprintf(CONFIG_APPLY_SUCCESS_MESSAGE, a.env.GetHostname()),
			command.Meta.GetMessageId(),
		)
	} else if nginxConfigResponse.Status.Status == proto.CommandStatusResponse_CMD_ERROR {
		event = a.createConfigApplyEvent(
			nginxConfigResponse.GetConfigData().GetNginxId(),
			command.GetMeta().GetTimestamp(),
			events.ERROR_EVENT_LEVEL,
			fmt.Sprintf(CONFIG_APPLY_FAILURE_MESSAGE, a.env.GetHostname()),
			command.Meta.GetMessageId(),
		)
	} else {
		return
	}

	log.Debugf("Created event: %v", event)

	a.pipeline.Process(core.NewMessage(core.Events, &proto.Command{
		Meta: a.meta,
		Type: proto.Command_NORMAL,
		Data: &proto.Command_EventReport{
			EventReport: &eventsProto.EventReport{
				Events: []*eventsProto.Event{event},
			},
		},
	}))
}

func (a *Events) sendConfigRollbackEvent(msg *core.Message) {
	configRollbackResponse, ok := msg.Data().(ConfigRollbackResponse)
	if !ok {
		log.Warnf("Invalid message received, %T, for topic, %s", msg.Data(), msg.Topic())
		return
	}

	var event *eventsProto.Event

	if configRollbackResponse.succeeded {
		event = a.createConfigApplyEvent(
			configRollbackResponse.nginxDetails.GetNginxId(),
			configRollbackResponse.timestamp,
			events.WARN_EVENT_LEVEL,
			fmt.Sprintf(CONFIG_ROLLBACK_SUCCESS_MESSAGE, a.env.GetHostname()),
			configRollbackResponse.correlationId,
		)
	} else {
		event = a.createConfigApplyEvent(
			configRollbackResponse.nginxDetails.GetNginxId(),
			configRollbackResponse.timestamp,
			events.ERROR_EVENT_LEVEL,
			fmt.Sprintf(CONFIG_ROLLBACK_FAILURE_MESSAGE, a.env.GetHostname()),
			configRollbackResponse.correlationId,
		)
	}

	log.Debugf("Created event: %v", event)

	a.pipeline.Process(core.NewMessage(core.Events, &proto.Command{
		Meta: a.meta,
		Type: proto.Command_NORMAL,
		Data: &proto.Command_EventReport{
			EventReport: &eventsProto.EventReport{
				Events: []*eventsProto.Event{event},
			},
		},
	}))
}

func (a *Events) sendNginxStartEvent(msg *core.Message) {
	nginxDetails, ok := msg.Data().(*proto.NginxDetails)
	if !ok {
		log.Warnf("Invalid message received, %T, for topic, %s", msg.Data(), msg.Topic())
		return
	}

	event := a.createNginxEvent(
		nginxDetails.GetNginxId(),
		&types.Timestamp{Seconds: nginxDetails.GetStartTime() / 1000, Nanos: int32(nginxDetails.GetStartTime() % 1000)},
		events.INFO_EVENT_LEVEL,
		fmt.Sprintf(NGINX_FOUND_MESSAGE, nginxDetails.GetVersion(), nginxDetails.GetProcessId()),
		uuid.NewString(),
	)

	log.Debugf("Created event: %v", event)
	a.pipeline.Process(core.NewMessage(core.Events, &proto.Command{
		Meta: a.meta,
		Type: proto.Command_NORMAL,
		Data: &proto.Command_EventReport{
			EventReport: &eventsProto.EventReport{
				Events: []*eventsProto.Event{event},
			},
		},
	}))
}

func (a *Events) sendNginxStopEvent(msg *core.Message) {
	nginxDetails, ok := msg.Data().(*proto.NginxDetails)
	if !ok {
		log.Warnf("Invalid message received, %T, for topic, %s", msg.Data(), msg.Topic())
		return
	}

	event := a.createNginxEvent(
		nginxDetails.GetNginxId(),
		types.TimestampNow(),
		events.WARN_EVENT_LEVEL,
		fmt.Sprintf(NGINX_STOP_MESSAGE, nginxDetails.GetVersion(), nginxDetails.GetProcessId()),
		uuid.NewString(),
	)

	log.Debugf("Created event: %v", event)
	a.pipeline.Process(core.NewMessage(core.Events, &proto.Command{
		Meta: a.meta,
		Type: proto.Command_NORMAL,
		Data: &proto.Command_EventReport{
			EventReport: &eventsProto.EventReport{
				Events: []*eventsProto.Event{event},
			},
		},
	}))
}

func (a *Events) sendNginxWorkerStartEvent(msg *core.Message) {
	nginxDetails, ok := msg.Data().(*proto.NginxDetails)
	if !ok {
		log.Warnf("Invalid message received, %T, for topic, %s", msg.Data(), msg.Topic())
		return
	}

	event := a.createNginxEvent(
		nginxDetails.GetNginxId(),
		&types.Timestamp{Seconds: nginxDetails.GetStartTime() / 1000, Nanos: int32(nginxDetails.GetStartTime() % 1000)},
		events.INFO_EVENT_LEVEL,
		fmt.Sprintf(NGINX_WORKER_START_MESSAGE, nginxDetails.GetProcessId(), nginxDetails.GetVersion(), nginxDetails.GetProcessId()),
		uuid.NewString(),
	)

	log.Debugf("Created event: %v", event)
	a.pipeline.Process(core.NewMessage(core.Events, &proto.Command{
		Meta: a.meta,
		Type: proto.Command_NORMAL,
		Data: &proto.Command_EventReport{
			EventReport: &eventsProto.EventReport{
				Events: []*eventsProto.Event{event},
			},
		},
	}))
}

func (a *Events) sendNginxWorkerStopEvent(msg *core.Message) {
	nginxDetails, ok := msg.Data().(*proto.NginxDetails)
	if !ok {
		log.Warnf("Invalid message received, %T, for topic, %s", msg.Data(), msg.Topic())
		return
	}

	event := a.createNginxEvent(
		nginxDetails.GetNginxId(),
		types.TimestampNow(),
		events.INFO_EVENT_LEVEL,
		fmt.Sprintf(NGINX_WORKER_STOP_MESSAGE, nginxDetails.GetProcessId(), nginxDetails.GetVersion(), nginxDetails.GetProcessId()),
		uuid.NewString(),
	)

	log.Debugf("Created event: %v", event)
	a.pipeline.Process(core.NewMessage(core.Events, &proto.Command{
		Meta: a.meta,
		Type: proto.Command_NORMAL,
		Data: &proto.Command_EventReport{
			EventReport: &eventsProto.EventReport{
				Events: []*eventsProto.Event{event},
			},
		},
	}))
}

func (e *Events) createNginxEvent(nginxId string, timestamp *types.Timestamp, level string, message string, correlationId string) *eventsProto.Event {
	activityEvent := e.agentEventsMeta.CreateActivityEvent(message, nginxId)

	return &eventsProto.Event{
		Metadata: &eventsProto.Metadata{
			UUID:          uuid.NewString(),
			CorrelationID: correlationId,
			Module:        config.MODULE,
			Timestamp:     timestamp,
			EventLevel:    level,
			Type:          events.NGINX_EVENT_TYPE,
			Category:      events.STATUS_CATEGORY,
		},
		Data: &eventsProto.Event_ActivityEvent{
			ActivityEvent: activityEvent,
		},
	}
}

func (e *Events) createConfigApplyEvent(nginxId string, timestamp *types.Timestamp, level string, message string, correlationId string) *eventsProto.Event {
	activityEvent := e.agentEventsMeta.CreateActivityEvent(message, nginxId)

	return &eventsProto.Event{
		Metadata: &eventsProto.Metadata{
			UUID:          uuid.NewString(),
			CorrelationID: correlationId,
			Module:        config.MODULE,
			Timestamp:     timestamp,
			EventLevel:    level,
			Type:          events.AGENT_EVENT_TYPE,
			Category:      events.CONFIG_CATEGORY,
		},
		Data: &eventsProto.Event_ActivityEvent{
			ActivityEvent: activityEvent,
		},
	}
}
