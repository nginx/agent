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

	ingesterClient := NewMockIngesterClient()
	ingesterClient.On("Connect", mock.Anything).Return(nil)

	controller := NewClientController()
	controller.WithClient(commanderClient)
	controller.WithClient(ingesterClient)

	err := controller.Connect()
	assert.Nil(t, err)

	commanderClient.AssertNumberOfCalls(t, "Connect", 1)
	ingesterClient.AssertNumberOfCalls(t, "Connect", 1)
}

func TestControllerConnect_error(t *testing.T) {
	commanderClient := NewMockCommandClient()
	commanderClient.On("Connect", mock.Anything).Return(fmt.Errorf("Error connecting"))
	commanderClient.On("Server").Return("127.0.0.1")

	ingesterClient := NewMockIngesterClient()
	ingesterClient.On("Connect", mock.Anything).Return(fmt.Errorf("Error connecting"))
	ingesterClient.On("Server").Return("127.0.0.1")

	controller := NewClientController()
	controller.WithClient(commanderClient)
	controller.WithClient(ingesterClient)

	err := controller.Connect()
	assert.NotNil(t, err)

	commanderClient.AssertNumberOfCalls(t, "Connect", 1)
	ingesterClient.AssertNumberOfCalls(t, "Connect", 1)
}

func TestControllerClose(t *testing.T) {
	commanderClient := NewMockCommandClient()
	commanderClient.On("Close").Return(nil)

	ingesterClient := NewMockIngesterClient()
	ingesterClient.On("Close").Return(nil)

	controller := NewClientController()
	controller.WithClient(commanderClient)
	controller.WithClient(ingesterClient)

	err := controller.Close()
	assert.Nil(t, err)

	commanderClient.AssertNumberOfCalls(t, "Close", 1)
	ingesterClient.AssertNumberOfCalls(t, "Close", 1)
}

func TestControllerClose_error(t *testing.T) {
	commanderClient := NewMockCommandClient()
	commanderClient.On("Close").Return(fmt.Errorf("Error closing"))
	commanderClient.On("Server").Return("127.0.0.1")

	ingesterClient := NewMockIngesterClient()
	ingesterClient.On("Close").Return(fmt.Errorf("Error closing"))
	ingesterClient.On("Server").Return("127.0.0.1")

	controller := NewClientController()
	controller.WithClient(commanderClient)
	controller.WithClient(ingesterClient)

	err := controller.Close()
	assert.NotNil(t, err)

	commanderClient.AssertNumberOfCalls(t, "Close", 1)
	ingesterClient.AssertNumberOfCalls(t, "Close", 1)
}
