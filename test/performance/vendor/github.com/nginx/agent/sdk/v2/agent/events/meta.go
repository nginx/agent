/**
 * Copyright (c) F5, Inc.
 *
 * This source code is licensed under the Apache License, Version 2.0 license found in the
 * LICENSE file in the root directory of this source tree.
 */

package events

import (
	"fmt"
	"strings"

	"github.com/gogo/protobuf/types"
	"github.com/google/uuid"
	sdkGRPC "github.com/nginx/agent/sdk/v2/grpc"
	"github.com/nginx/agent/sdk/v2/proto"
	commonProto "github.com/nginx/agent/sdk/v2/proto/common"
	eventsProto "github.com/nginx/agent/sdk/v2/proto/events"
)

type AgentEventMeta struct {
	module        string
	version       string
	pid           string
	hostname      string
	systemUuid    string
	instanceGroup string
	tags          string
	tagsRaw       []string
}

func NewAgentEventMeta(
	module, version, pid, hostname, systemUuid, instanceGroup string,
	tags []string,
) *AgentEventMeta {
	return &AgentEventMeta{
		module:        module,
		version:       version,
		pid:           pid,
		hostname:      hostname,
		systemUuid:    systemUuid,
		instanceGroup: instanceGroup,
		tagsRaw:       tags,
		tags:          strings.Join(tags, ","),
	}
}

func (aem *AgentEventMeta) GetVersion() string {
	return aem.version
}

func (aem *AgentEventMeta) GetPid() string {
	return aem.pid
}

func (aem *AgentEventMeta) GenerateAgentStartEventCommand() *proto.Command {
	activityEvent := &eventsProto.ActivityEvent{
		Message: fmt.Sprintf(AGENT_START_MESSAGE, aem.version, aem.hostname, aem.pid),
		Dimensions: []*commonProto.Dimension{
			{
				Name:  "system_id",
				Value: aem.systemUuid,
			},
			{
				Name:  "hostname",
				Value: aem.hostname,
			},
			{
				Name:  "instance_group",
				Value: aem.instanceGroup,
			},
			{
				Name:  "system.tags",
				Value: aem.tags,
			},
		},
	}

	event := &eventsProto.Event{
		Metadata: &eventsProto.Metadata{
			UUID:          uuid.NewString(),
			CorrelationID: uuid.NewString(),
			Module:        aem.module,
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

func (aem *AgentEventMeta) GenerateAgentStopEventCommand() *proto.Command {
	activityEvent := &eventsProto.ActivityEvent{
		Message: fmt.Sprintf(AGENT_STOP_MESSAGE, aem.version, aem.pid, aem.hostname),
		Dimensions: []*commonProto.Dimension{
			{
				Name:  "system_id",
				Value: aem.systemUuid,
			},
			{
				Name:  "hostname",
				Value: aem.hostname,
			},
			{
				Name:  "instance_group",
				Value: aem.instanceGroup,
			},
			{
				Name:  "system.tags",
				Value: aem.tags,
			},
		},
	}

	event := &eventsProto.Event{
		Metadata: &eventsProto.Metadata{
			UUID:          uuid.NewString(),
			CorrelationID: uuid.NewString(),
			Module:        aem.module,
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

func (aem *AgentEventMeta) CreateAgentEvent(timestamp *types.Timestamp, level, message, correlationId, module string) *eventsProto.Event {
	activityEvent := aem.CreateActivityEvent(message, "") // blank nginxId, this relates to agent not it's nginx instances

	return &eventsProto.Event{
		Metadata: &eventsProto.Metadata{
			UUID:          uuid.NewString(),
			CorrelationID: correlationId,
			Module:        module,
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

func (aem *AgentEventMeta) CreateActivityEvent(message string, nginxId string) *eventsProto.ActivityEvent {
	activityEvent := &eventsProto.ActivityEvent{
		Message: message,
		Dimensions: []*commonProto.Dimension{
			{
				Name:  "system_id",
				Value: aem.systemUuid,
			},
			{
				Name:  "hostname",
				Value: aem.hostname,
			},
			{
				Name:  "instance_group",
				Value: aem.instanceGroup,
			},
			{
				Name:  "system.tags",
				Value: aem.tags,
			},
		},
	}

	if nginxId != "" {
		nginxDim := []*commonProto.Dimension{{Name: "nginx_id", Value: nginxId}}
		activityEvent.Dimensions = append(nginxDim, activityEvent.Dimensions...)
	}

	return activityEvent
}
