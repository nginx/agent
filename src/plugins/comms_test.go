package plugins

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/gogo/protobuf/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/nginx/agent/sdk/v2/proto"
	"github.com/nginx/agent/v2/src/core"
	tutils "github.com/nginx/agent/v2/test/utils"
)

func TestCommsSendMetrics(t *testing.T) {
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
		t.Run(test.name, func(tt *testing.T) {

			ctx := context.TODO()
			mockMetricsReportClient := tutils.NewMockMetricsReportClient()
			mockMetricsReportClient.Mock.On("Send", ctx, mock.Anything).Return(test.err)
			pluginUnderTest := NewComms(mockMetricsReportClient)

			assert.False(t, pluginUnderTest.started.Load())
			assert.False(t, pluginUnderTest.readyToSend.Load())

			pluginUnderTest.Init(core.NewMockMessagePipe(ctx))

			assert.True(t, pluginUnderTest.started.Load())
			assert.False(t, pluginUnderTest.readyToSend.Load())

			pluginUnderTest.Process(core.NewMessage(core.RegistrationCompletedTopic, nil))

			assert.True(t, pluginUnderTest.readyToSend.Load())

			pluginUnderTest.Process(core.NewMessage(core.CommMetrics, []core.Payload{&proto.MetricsReport{
				Meta: &proto.Metadata{Timestamp: types.TimestampNow()},
				Type: proto.MetricsReport_INSTANCE,
				Data: make([]*proto.StatsEntity, 0, 1),
			}}))

			time.Sleep(1 * time.Second) // for the above call being asynchronous
			assert.True(t, mockMetricsReportClient.AssertExpectations(t))

			pluginUnderTest.Close()
			assert.False(t, pluginUnderTest.readyToSend.Load())
		})
	}
}

func TestCommsSubscriptions(t *testing.T) {
	pluginUnderTest := NewComms(tutils.NewMockMetricsReportClient())
	assert.Equal(t, []string{core.CommMetrics, core.RegistrationCompletedTopic}, pluginUnderTest.Subscriptions())
}
