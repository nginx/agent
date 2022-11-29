package plugins

import (
	"context"
	"fmt"
	"strings"

	"github.com/gogo/protobuf/types"
	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"

	sdkGRPC "github.com/nginx/agent/sdk/v2/grpc"
	"github.com/nginx/agent/sdk/v2/proto"
	commonProto "github.com/nginx/agent/sdk/v2/proto/common"
	eventsProto "github.com/nginx/agent/sdk/v2/proto/events"
	"github.com/nginx/agent/v2/src/core"
	"github.com/nginx/agent/v2/src/core/config"
)

const (
	MODULE = "NGINX-AGENT"

	AGENT_START_MESSAGE             = "nginx-agent %s started on %s with pid %s"
	AGENT_STOP_MESSAGE              = "nginx-agent %s (pid: %s) stopped on %s"
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

	// Types
	NGINX_EVENT_TYPE = "Nginx"
	AGENT_EVENT_TYPE = "Agent"

	// Categories
	STATUS_CATEGORY      = "Status"
	CONFIG_CATEGORY      = "Config"
	APP_PROTECT_CATEGORY = "AppProtect"

	// Event Levels
	INFO_EVENT_LEVEL     = "INFO"
	DEBUG_EVENT_LEVEL    = "DEBUG"
	WARN_EVENT_LEVEL     = "WARN"
	ERROR_EVENT_LEVEL    = "ERROR"
	CRITICAL_EVENT_LEVEL = "CRITICAL"
)

type Events struct {
	pipeline    core.MessagePipeInterface
	ctx         context.Context
	conf        *config.Config
	env         core.Environment
	meta        *proto.Metadata
	nginxBinary core.NginxBinary
}

func NewEvents(conf *config.Config, env core.Environment, meta *proto.Metadata, nginxBinary core.NginxBinary) *Events {
	return &Events{
		conf:        conf,
		env:         env,
		meta:        meta,
		nginxBinary: nginxBinary,
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
		a.sendNingxFoundEvent(msg)
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
	return core.NewInfo("Events", "v0.0.1")
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
	agentEventMeta, ok := msg.Data().(*AgentEventMeta)
	if !ok {
		log.Warnf("Invalid message received, %T, for topic, %s", msg.Data(), msg.Topic())
		return
	}

	event := a.createAgentEvent(
		types.TimestampNow(),
		INFO_EVENT_LEVEL,
		fmt.Sprintf(AGENT_START_MESSAGE, agentEventMeta.version, a.env.GetHostname(), agentEventMeta.pid),
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

func (a *Events) sendNingxFoundEvent(msg *core.Message) {
	nginxDetailsMap, ok := msg.Data().(map[string]*proto.NginxDetails)
	if !ok {
		log.Warnf("Invalid message received, %T, for topic, %s", msg.Data(), msg.Topic())
		return
	}

	events := []*eventsProto.Event{}

	for _, nginxDetail := range nginxDetailsMap {
		event := a.createNginxEvent(
			nginxDetail.GetNginxId(),
			&types.Timestamp{Seconds: nginxDetail.GetStartTime() / 1000, Nanos: int32(nginxDetail.GetStartTime() % 1000)},
			INFO_EVENT_LEVEL,
			fmt.Sprintf(NGINX_FOUND_MESSAGE, nginxDetail.GetVersion(), nginxDetail.GetProcessId()),
			uuid.NewString(),
		)

		log.Debugf("Created event: %v", event)
		events = append(events, event)
	}

	a.pipeline.Process(core.NewMessage(core.Events, &proto.Command{
		Meta: a.meta,
		Type: proto.Command_NORMAL,
		Data: &proto.Command_EventReport{
			EventReport: &eventsProto.EventReport{
				Events: events,
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
			WARN_EVENT_LEVEL,
			fmt.Sprintf(NGINX_RELOAD_SUCCESS_MESSAGE, nginxReload.nginxDetails.GetVersion(), nginxReload.nginxDetails.GetProcessId()),
			nginxReload.correlationId,
		)
	} else {
		event = a.createNginxEvent(
			nginxReload.nginxDetails.GetNginxId(),
			nginxReload.timestamp,
			ERROR_EVENT_LEVEL,
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

	if (nginxConfigResponse.GetAction() != proto.NginxConfigAction_APPLY) {
		return
	}

	var event *eventsProto.Event

	if nginxConfigResponse.Status.Status == proto.CommandStatusResponse_CMD_OK {
		event = a.createConfigApplyEvent(
			nginxConfigResponse.GetConfigData().NginxId,
			command.GetMeta().Timestamp,
			INFO_EVENT_LEVEL,
			fmt.Sprintf(CONFIG_APPLY_SUCCESS_MESSAGE, a.env.GetHostname()),
			command.Meta.GetMessageId(),
		)
	} else if nginxConfigResponse.Status.Status == proto.CommandStatusResponse_CMD_ERROR {
		event = a.createConfigApplyEvent(
			nginxConfigResponse.GetConfigData().NginxId,
			command.GetMeta().Timestamp,
			ERROR_EVENT_LEVEL,
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
			WARN_EVENT_LEVEL,
			fmt.Sprintf(CONFIG_ROLLBACK_SUCCESS_MESSAGE, a.env.GetHostname()),
			configRollbackResponse.correlationId,
		)
	} else {
		event = a.createConfigApplyEvent(
			configRollbackResponse.nginxDetails.GetNginxId(),
			configRollbackResponse.timestamp,
			ERROR_EVENT_LEVEL,
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
		INFO_EVENT_LEVEL,
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
		WARN_EVENT_LEVEL,
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
		INFO_EVENT_LEVEL,
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
		INFO_EVENT_LEVEL,
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
	activityEvent := e.createActivityEvent(message, nginxId)

	return &eventsProto.Event{
		Metadata: &eventsProto.Metadata{
			UUID:          uuid.NewString(),
			CorrelationID: correlationId,
			Module:        MODULE,
			Timestamp:     timestamp,
			EventLevel:    level,
			Type:          NGINX_EVENT_TYPE,
			Category:      STATUS_CATEGORY,
		},
		Data: &eventsProto.Event_ActivityEvent{
			ActivityEvent: activityEvent,
		},
	}
}

func (e *Events) createConfigApplyEvent(nginxId string, timestamp *types.Timestamp, level string, message string, correlationId string) *eventsProto.Event {
	activityEvent := e.createActivityEvent(message, nginxId)

	return &eventsProto.Event{
		Metadata: &eventsProto.Metadata{
			UUID:          uuid.NewString(),
			CorrelationID: correlationId,
			Module:        MODULE,
			Timestamp:     timestamp,
			EventLevel:    level,
			Type:          AGENT_EVENT_TYPE,
			Category:      CONFIG_CATEGORY,
		},
		Data: &eventsProto.Event_ActivityEvent{
			ActivityEvent: activityEvent,
		},
	}
}

func (e *Events) createAgentEvent(timestamp *types.Timestamp, level string, message string, correlationId string) *eventsProto.Event {
	activityEvent := e.createActivityEvent(message, "") // blank nginxId, this relates to agent not it's nginx instances

	return &eventsProto.Event{
		Metadata: &eventsProto.Metadata{
			UUID:          uuid.NewString(),
			CorrelationID: correlationId,
			Module:        MODULE,
			Timestamp:     timestamp,
			EventLevel:    level,
			Type:          AGENT_EVENT_TYPE,
			Category:      STATUS_CATEGORY,
		},
		Data: &eventsProto.Event_ActivityEvent{
			ActivityEvent: activityEvent,
		},
	}
}

func (e *Events) createActivityEvent(message string, nginxId string) *eventsProto.ActivityEvent {
	activityEvent := &eventsProto.ActivityEvent{
		Message: message,
		Dimensions: []*commonProto.Dimension{
			{
				Name:  "system_id",
				Value: e.env.GetSystemUUID(),
			},
			{
				Name:  "hostname",
				Value: e.env.GetHostname(),
			},
			{
				Name:  "instance_group",
				Value: e.conf.InstanceGroup,
			},
			{
				Name:  "system.tags",
				Value: strings.Join(e.conf.Tags, ","),
			},
		},
	}

	if nginxId != "" {
		nginxDim := []*commonProto.Dimension{{Name: "nginx_id", Value: nginxId}}
		activityEvent.Dimensions = append(nginxDim, activityEvent.Dimensions...)
	}

	return activityEvent
}

type AgentEventMeta struct {
	version string
	pid     string
}

func NewAgentEventMeta(version string, pid string) *AgentEventMeta {
	return &AgentEventMeta{
		version: version,
		pid:     pid,
	}
}

func GenerateAgentStopEventCommand(agentEvent *AgentEventMeta, conf *config.Config, env core.Environment) *proto.Command {
	activityEvent := &eventsProto.ActivityEvent{
		Message: fmt.Sprintf(AGENT_STOP_MESSAGE, agentEvent.version, agentEvent.pid, env.GetHostname()),
		Dimensions: []*commonProto.Dimension{
			{
				Name:  "system_id",
				Value: env.GetSystemUUID(),
			},
			{
				Name:  "hostname",
				Value: env.GetHostname(),
			},
			{
				Name:  "instance_group",
				Value: conf.InstanceGroup,
			},
			{
				Name:  "system.tags",
				Value: strings.Join(conf.Tags, ","),
			},
		},
	}

	event := &eventsProto.Event{
		Metadata: &eventsProto.Metadata{
			UUID:          uuid.NewString(),
			CorrelationID: uuid.NewString(),
			Module:        MODULE,
			Timestamp:     types.TimestampNow(),
			EventLevel:    WARN_EVENT_LEVEL,
			Type:          AGENT_EVENT_TYPE,
			Category:      STATUS_CATEGORY,
		},
		Data: &eventsProto.Event_ActivityEvent{
			ActivityEvent: activityEvent,
		},
	}

	return &proto.Command{
		Meta: sdkGRPC.NewMessageMeta(uuid.NewString()),
		Type: proto.Command_NORMAL,
		Data: &proto.Command_EventReport{
			EventReport: &eventsProto.EventReport{
				Events: []*eventsProto.Event{event},
			},
		},
	}
}
