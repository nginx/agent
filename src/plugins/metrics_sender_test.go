/**
 * Copyright (c) F5, Inc.
 *
 * This source code is licensed under the Apache License, Version 2.0 license found in the
 * LICENSE file in the root directory of this source tree.
 */

package plugins

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/gogo/protobuf/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/nginx/agent/sdk/v2/backoff"
	"github.com/nginx/agent/sdk/v2/client"
	"github.com/nginx/agent/sdk/v2/proto"
	"github.com/nginx/agent/v2/src/core"
	tutils "github.com/nginx/agent/v2/test/utils"
)

func TestMetricsSenderSendMetrics(t *testing.T) {
	tests := []struct {
		name string
		err  error
	}{
		{
			name: "test send metrics no error",
			err:  nil,
		},
		{
			name: "test send metrics error",
			err:  errors.New("send err"),
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(_ *testing.T) {
			ctx := context.TODO()
			mockMetricsReportClient := tutils.NewMockMetricsReportClient()
			mockMetricsReportClient.Mock.On("Send", ctx, mock.Anything).Return(test.err)
			pluginUnderTest := NewMetricsSender(mockMetricsReportClient)

			assert.False(t, pluginUnderTest.started.Load())
			assert.False(t, pluginUnderTest.readyToSend.Load())

			pluginUnderTest.Init(core.NewMockMessagePipe(ctx))

			assert.True(t, pluginUnderTest.started.Load())
			assert.False(t, pluginUnderTest.readyToSend.Load())

			pluginUnderTest.Process(core.NewMessage(core.RegistrationCompletedTopic, nil))

			assert.True(t, pluginUnderTest.readyToSend.Load())

			metricData := make([]*proto.StatsEntity, 0, 1)
			metricData = append(metricData, &proto.StatsEntity{Simplemetrics: []*proto.SimpleMetric{{Name: "Metric A", Value: 5}}})

			pluginUnderTest.Process(core.NewMessage(core.CommMetrics, []core.Payload{&proto.MetricsReport{
				Meta: &proto.Metadata{Timestamp: types.TimestampNow()},
				Type: proto.MetricsReport_INSTANCE,
				Data: metricData,
			}}))

			time.Sleep(1 * time.Second) // for the above call being asynchronous
			assert.True(t, mockMetricsReportClient.AssertExpectations(t))

			pluginUnderTest.Close()
			assert.False(t, pluginUnderTest.readyToSend.Load())
		})
	}
}

func TestMetricsSenderBackoff(t *testing.T) {
	tests := []struct {
		name        string
		msg         *core.Message
		wantBackoff backoff.BackoffSettings
	}{
		{
			name: "test reporter client backoff setting as sent by server",
			msg: core.NewMessage(core.AgentConfig,
				&proto.Command{
					Data: &proto.Command_AgentConfig{
						AgentConfig: &proto.AgentConfig{
							Details: &proto.AgentDetails{
								Server: &proto.Server{
									Backoff: &proto.Backoff{
										InitialInterval:     900,
										RandomizationFactor: .5,
										Multiplier:          .5,
										MaxInterval:         900,
										MaxElapsedTime:      1800,
									},
								},
							},
						},
					},
				}),
			wantBackoff: backoff.BackoffSettings{
				InitialInterval: time.Duration(15 * time.Minute),
				Jitter:          .5,
				Multiplier:      .5,
				MaxInterval:     time.Duration(15 * time.Minute),
				MaxElapsedTime:  time.Duration(30 * time.Minute),
			},
		},
		{
			name: "test reporter client backoff setting as default",
			msg: core.NewMessage(core.AgentConfig,
				&proto.Command{
					Data: &proto.Command_AgentConfig{
						AgentConfig: &proto.AgentConfig{
							Details: &proto.AgentDetails{
								Server: &proto.Server{},
							},
						},
					},
				}),
			wantBackoff: client.DefaultBackoffSettings,
		},
		{
			name: "test reporter client backoff setting not updated",
			msg: core.NewMessage(core.AgentConfig,
				&proto.Command_AgentConfig{
					AgentConfig: &proto.AgentConfig{
						Details: &proto.AgentDetails{
							Server: &proto.Server{},
						},
					},
				}),
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(_ *testing.T) {
			ctx := context.TODO()
			mockMetricsReportClient := tutils.NewMockMetricsReportClient()
			pluginUnderTest := NewMetricsSender(mockMetricsReportClient)

			pluginUnderTest.Init(core.NewMockMessagePipe(ctx))
			pluginUnderTest.Process(core.NewMessage(core.RegistrationCompletedTopic, nil))

			if !reflect.ValueOf(test.wantBackoff).IsZero() {
				mockMetricsReportClient.On("WithBackoffSettings", test.wantBackoff)
			}

			pluginUnderTest.Process(test.msg)

			time.Sleep(1 * time.Second)
			assert.True(t, mockMetricsReportClient.AssertExpectations(t))

			pluginUnderTest.Close()
		})
	}
}

func TestMetricsSenderSubscriptions(t *testing.T) {
	pluginUnderTest := NewMetricsSender(tutils.NewMockMetricsReportClient())
	assert.Equal(t, []string{core.CommMetrics, core.RegistrationCompletedTopic, core.AgentConfig}, pluginUnderTest.Subscriptions())
}
