package plugins

import (
	"context"
	"testing"

	"github.com/nginx/agent/sdk/v2/proto"
	"github.com/stretchr/testify/assert"

	"github.com/nginx/agent/v2/src/core"
	"github.com/nginx/agent/v2/src/core/config"
	tutils "github.com/nginx/agent/v2/test/utils"
)

func TestMetricsAggregator_Process(t *testing.T) {
	tests := []struct {
		name         string
		metricBuffer []core.Payload
		msgs         []*core.Message
		msgTopics    []string
		config       *config.Config
	}{
		{
			name: "not enough metrics",
			msgs: []*core.Message{
				core.NewMessage(core.MetricReport, &proto.MetricsReport{}),
			},
			msgTopics: []string{
				core.MetricReport,
			},
			config: tutils.GetMockAgentConfig(),
		},
		{
			name: "flush buffer of metrics",
			msgs: []*core.Message{
				core.NewMessage(core.MetricReport, &proto.MetricsReport{}),
				core.NewMessage(core.MetricReport, &proto.MetricsReport{}),
				core.NewMessage(core.MetricReport, &proto.MetricsReport{}),
			},
			msgTopics: []string{
				core.MetricReport,
				core.MetricReport,
				core.MetricReport,
			},
			config: tutils.GetMockAgentConfig(),
		},
		{
			name: "config changed",
			msgs: []*core.Message{
				core.NewMessage(core.AgentConfigChanged, nil),
			},
			msgTopics: []string{
				core.AgentConfigChanged,
			},
			config: tutils.GetMockAgentConfig(),
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(tt *testing.T) {
			throttlePlugin := NewMetricsAggregator(test.config, &tutils.MockEnvironment{})
			ctx := context.Background()
			messagePipe := core.SetupMockMessagePipe(t, ctx, throttlePlugin)

			messagePipe.Process(test.msgs...)
			messagePipe.Run()

			core.ValidateMessages(t, messagePipe, test.msgTopics)

			ctx.Done()
			throttlePlugin.Close()
		})
	}
}

func TestMetricsAggregator_Subscriptions(t *testing.T) {
	subs := []string{core.MetricReport, core.AgentConfigChanged, core.LoggerLevel}
	pluginUnderTest := NewMetricsAggregator(tutils.GetMockAgentConfig(), &tutils.MockEnvironment{})

	assert.Equal(t, subs, pluginUnderTest.Subscriptions())
}

func TestMetricsAggregator_Info(t *testing.T) {
	pluginUnderTest := NewMetricsAggregator(tutils.GetMockAgentConfig(), &tutils.MockEnvironment{})

	assert.Equal(t, "MetricsAggregator", pluginUnderTest.Info().Name())
}
