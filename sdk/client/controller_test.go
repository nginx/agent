package client

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestControllerContext(t *testing.T) {
	ctx := context.TODO()
	controller := NewClientController()
	controller.WithContext(ctx)

	assert.Equal(t, ctx, controller.Context())
}

func TestControllerConnect(t *testing.T) {
	commanderClient := NewMockCommandClient()
	commanderClient.On("Connect", mock.Anything).Return(nil)

	metricsReportClient := NewMockMetricsReportClient()
	metricsReportClient.On("Connect", mock.Anything).Return(nil)

	controller := NewClientController()
	controller.WithClient(commanderClient)
	controller.WithClient(metricsReportClient)

	err := controller.Connect()
	assert.Nil(t, err)

	commanderClient.AssertNumberOfCalls(t, "Connect", 1)
	metricsReportClient.AssertNumberOfCalls(t, "Connect", 1)
}

func TestControllerConnect_error(t *testing.T) {
	commanderClient := NewMockCommandClient()
	commanderClient.On("Connect", mock.Anything).Return(fmt.Errorf("Error connecting"))
	commanderClient.On("Server").Return("127.0.0.1")

	metricsReportClient := NewMockMetricsReportClient()
	metricsReportClient.On("Connect", mock.Anything).Return(fmt.Errorf("Error connecting"))
	metricsReportClient.On("Server").Return("127.0.0.1")

	controller := NewClientController()
	controller.WithClient(commanderClient)
	controller.WithClient(metricsReportClient)

	err := controller.Connect()
	assert.NotNil(t, err)

	commanderClient.AssertNumberOfCalls(t, "Connect", 1)
	metricsReportClient.AssertNumberOfCalls(t, "Connect", 1)
}

func TestControllerClose(t *testing.T) {
	commanderClient := NewMockCommandClient()
	commanderClient.On("Close").Return(nil)

	metricsReportClient := NewMockMetricsReportClient()
	metricsReportClient.On("Close").Return(nil)

	controller := NewClientController()
	controller.WithClient(commanderClient)
	controller.WithClient(metricsReportClient)

	err := controller.Close()
	assert.Nil(t, err)

	commanderClient.AssertNumberOfCalls(t, "Close", 1)
	metricsReportClient.AssertNumberOfCalls(t, "Close", 1)
}

func TestControllerClose_error(t *testing.T) {
	commanderClient := NewMockCommandClient()
	commanderClient.On("Close").Return(fmt.Errorf("Error closing"))
	commanderClient.On("Server").Return("127.0.0.1")

	metricsReportClient := NewMockMetricsReportClient()
	metricsReportClient.On("Close").Return(fmt.Errorf("Error closing"))
	metricsReportClient.On("Server").Return("127.0.0.1")

	controller := NewClientController()
	controller.WithClient(commanderClient)
	controller.WithClient(metricsReportClient)

	err := controller.Close()
	assert.NotNil(t, err)

	commanderClient.AssertNumberOfCalls(t, "Close", 1)
	metricsReportClient.AssertNumberOfCalls(t, "Close", 1)
}
