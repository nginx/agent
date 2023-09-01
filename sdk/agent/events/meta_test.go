package events

import (
	"testing"

	"github.com/nginx/agent/sdk/v2/proto"
	eventsProto "github.com/nginx/agent/sdk/v2/proto/events"
	"github.com/stretchr/testify/assert"
)

func TestNewAgentEventMeta(t *testing.T) {
	// Create an instance of AgentEventMeta using the constructor
	module := "nginx-agent"
	version := "v1.0"
	pid := "12345"
	message := "Sample message: Version=%s, PID=%s, Hostname=%s"
	hostname := "example-host"
	systemUuid := "system-uuid"
	instanceGroup := "group1"
	tags := []string{"tag1", "tag2"}

	meta := NewAgentEventMeta(module, version, pid, message, hostname, systemUuid, instanceGroup, tags)

	assert.NotNil(t, meta)

	assert.Equal(t, version, meta.version)
	assert.Equal(t, pid, meta.pid)
	assert.Equal(t, message, meta.message)
	assert.Equal(t, hostname, meta.hostname)
	assert.Equal(t, systemUuid, meta.systemUuid)
	assert.Equal(t, instanceGroup, meta.instanceGroup)
	assert.Equal(t, tags, meta.tags)
}

func TestGenerateAgentStopEventCommand(t *testing.T) {
	// Create a mock AgentEventMeta object
	agentEvent := &AgentEventMeta{
		module:        "agent-module",
		version:       "v2.0",
		pid:           "54321",
		message:       "Sample message: Version=%s, PID=%s, Hostname=%s",
		hostname:      "test-host",
		systemUuid:    "test-uuid",
		instanceGroup: "group2",
		tags:          []string{"tag3", "tag4"},
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
					ActivityEvent: &eventsProto.ActivityEvent{
						Message:    "failed to rollback nginx config on test-host",
						Dimensions: nil,
					},
				},
			},
		},
	}

	// Generate the AgentStopEventCommand using the function
	cmd := GenerateAgentStopEventCommand(agentEvent)

	// Assert that the generated command is not nil
	assert.NotNil(t, cmd)

	// You can add more specific assertions based on the expected structure and values of the generated command.
	// For example, checking the UUIDs, message format, and other fields.
	assert.NotNil(t, cmd.Meta)
	assert.Equal(t, proto.Command_NORMAL, cmd.Type)
	assert.NotNil(t, cmd.Data)
	assert.Equal(t, expected, cmd.GetData())
}
