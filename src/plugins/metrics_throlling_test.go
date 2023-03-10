/**
 * Copyright (c) F5, Inc.
 *
 * This source code is licensed under the Apache License, Version 2.0 license found in the
 * LICENSE file in the root directory of this source tree.
 */

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

func TestMetricsThrottle_Process(t *testing.T) {
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
			name: "flush buffer of metrics streaming",
			msgs: []*core.Message{
				core.NewMessage(core.MetricReport, &proto.MetricsReport{}),
				core.NewMessage(core.MetricReport, &proto.MetricsReport{}),
				core.NewMessage(core.MetricReport, &proto.MetricsReport{}),
			},
			msgTopics: []string{
				core.MetricReport,
				core.MetricReport,
				core.MetricReport,
				core.CommMetrics,
				core.CommMetrics,
				core.CommMetrics,
			},
			config: &config.Config{
				ClientID: "12345",
				Tags:     tutils.InitialConfTags,
				AgentMetrics: config.AgentMetrics{
					BulkSize:           1,
					ReportInterval:     5,
					CollectionInterval: 1,
					Mode:               "streaming",
				},
			},
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
			throttlePlugin := NewMetricsThrottle(test.config, &tutils.MockEnvironment{})
			ctx := context.Background()
			messagePipe := core.SetupMockMessagePipe(t, ctx, []core.Plugin{throttlePlugin}, []core.ExtensionPlugin{})

			messagePipe.Process(test.msgs...)
			messagePipe.Run()

			core.ValidateMessages(t, messagePipe, test.msgTopics)

			ctx.Done()
			throttlePlugin.Close()
		})
	}
}

func TestMetricsThrottle_Subscriptions(t *testing.T) {
	subs := []string{core.MetricReport, core.AgentConfigChanged, core.LoggerLevel}
	pluginUnderTest := NewMetricsThrottle(tutils.GetMockAgentConfig(), &tutils.MockEnvironment{})

	assert.Equal(t, subs, pluginUnderTest.Subscriptions())
}

func TestMetricsThrottle_Info(t *testing.T) {
	pluginUnderTest := NewMetricsThrottle(tutils.GetMockAgentConfig(), &tutils.MockEnvironment{})

	assert.Equal(t, "MetricsThrottle", pluginUnderTest.Info().Name())
}
