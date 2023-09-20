package events

import (
	"fmt"
	"strings"
	"testing"

	"github.com/nginx/agent/sdk/v2/proto"
	commonProto "github.com/nginx/agent/sdk/v2/proto/common"
	eventsProto "github.com/nginx/agent/sdk/v2/proto/events"
	"github.com/stretchr/testify/assert"
)

func TestNewAgentEventMeta(t *testing.T) {
	// Create an instance of AgentEventMeta using the constructor
	module := "nginx-agent"
	version := "v1.0"
	pid := "12345"
	hostname := "example-host"
	systemUuid := "system-uuid"
	instanceGroup := "group1"
	tags := []string{"tag1", "tag2"}

	meta := NewAgentEventMeta(module, version, pid, hostname, systemUuid, instanceGroup, tags)

	assert.NotNil(t, meta)

	assert.Equal(t, version, meta.version)
	assert.Equal(t, pid, meta.pid)
	assert.Equal(t, hostname, meta.hostname)
	assert.Equal(t, systemUuid, meta.systemUuid)
	assert.Equal(t, instanceGroup, meta.instanceGroup)
	assert.Equal(t, tags, meta.tagsRaw)
	assert.Equal(t, strings.Join(tags, ","), meta.tags)
}

func TestGenerateAgentStartEventCommand(t *testing.T) {
	agentEvent := NewAgentEventMeta(
		"agent-module",
		"v2.0",
		"54321",
		"test-host",
		"test-uuid",
		"group2",
		[]string{"tag3", "tag4"},
	)

	expectedActivityEvent := &eventsProto.ActivityEvent{
		Message: fmt.Sprintf("%s %s started on %s with pid %s", "nginx-agent", "v2.0", "test-host", "54321"),
		Dimensions: []*commonProto.Dimension{
			{
				Name:  "system_id",
				Value: "test-uuid",
			},
			{
				Name:  "hostname",
				Value: "test-host",
			},
			{
				Name:  "instance_group",
				Value: "group2",
			},
			{
				Name:  "system.tags",
				Value: strings.Join([]string{"tag3", "tag4"}, ","),
			},
		},
	}

	expected := &eventsProto.EventReport{
		Events: []*eventsProto.Event{
			{
				Metadata: &eventsProto.Metadata{
					Module:     agentEvent.module,
					Type:       AGENT_EVENT_TYPE,
					Category:   CONFIG_CATEGORY,
					EventLevel: ERROR_EVENT_LEVEL,
				},
				Data: &eventsProto.Event_ActivityEvent{
					ActivityEvent: expectedActivityEvent,
				},
			},
		},
	}

	cmd := agentEvent.GenerateAgentStartEventCommand()
	assert.NotNil(t, cmd)
	assert.NotNil(t, cmd.Meta)
	assert.Equal(t, proto.Command_NORMAL, cmd.Type)
	assert.NotNil(t, cmd.GetData())

	assert.Equal(t, expected.GetEvents()[0].GetData(), cmd.GetData().(*proto.Command_EventReport).EventReport.GetEvents()[0].GetData())
}

func TestGenerateAgentStopEventCommand(t *testing.T) {
	agentEvent := NewAgentEventMeta(
		"agent-module",
		"v2.0",
		"54321",
		"test-host",
		"test-uuid",
		"group2",
		[]string{"tag3", "tag4"},
	)

	expectedActivityEvent := &eventsProto.ActivityEvent{
		Message: fmt.Sprintf("%s %s (pid: %s) stopped on %s", "nginx-agent", "v2.0", "54321", "test-host"),
		Dimensions: []*commonProto.Dimension{
			{
				Name:  "system_id",
				Value: "test-uuid",
			},
			{
				Name:  "hostname",
				Value: "test-host",
			},
			{
				Name:  "instance_group",
				Value: "group2",
			},
			{
				Name:  "system.tags",
				Value: strings.Join([]string{"tag3", "tag4"}, ","),
			},
		},
	}

	expected := &eventsProto.EventReport{
		Events: []*eventsProto.Event{
			{
				Metadata: &eventsProto.Metadata{
					Module:     agentEvent.module,
					Type:       AGENT_EVENT_TYPE,
					Category:   CONFIG_CATEGORY,
					EventLevel: ERROR_EVENT_LEVEL,
				},
				Data: &eventsProto.Event_ActivityEvent{
					ActivityEvent: expectedActivityEvent,
				},
			},
		},
	}

	cmd := agentEvent.GenerateAgentStopEventCommand()
	assert.NotNil(t, cmd)
	assert.NotNil(t, cmd.Meta)
	assert.Equal(t, proto.Command_NORMAL, cmd.Type)
	assert.NotNil(t, cmd.GetData())

	assert.Equal(t, expected.GetEvents()[0].GetData(), cmd.GetData().(*proto.Command_EventReport).EventReport.GetEvents()[0].GetData())
}
