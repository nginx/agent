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
	message       string
	hostname      string
	systemUuid    string
	instanceGroup string
	tags          []string
}

func NewAgentEventMeta(
	module, version, pid, message, hostname, systemUuid, instanceGroup string,
	tags []string,
) *AgentEventMeta {
	return &AgentEventMeta{
		module:        module,
		version:       version,
		pid:           pid,
		message:       message,
		hostname:      hostname,
		systemUuid:    systemUuid,
		instanceGroup: instanceGroup,
		tags:          tags,
	}
}

func (aem *AgentEventMeta) GetVersion() string {
	return aem.version
}

func (aem *AgentEventMeta) GetPid() string {
	return aem.pid
}

func (aem *AgentEventMeta) GenerateAgentStopEventCommand() *proto.Command {
	activityEvent := &eventsProto.ActivityEvent{
		Message: fmt.Sprintf(aem.message, aem.version, aem.pid, aem.hostname),
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
				Value: strings.Join(aem.tags, ","),
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
