/**
 * Copyright (c) F5, Inc.
 *
 * This source code is licensed under the Apache License, Version 2.0 license found in the
 * LICENSE file in the root directory of this source tree.
 */

package client

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestControllerContext(t *testing.T) {
	ctx, cncl := context.WithTimeout(context.Background(), 100*time.Millisecond)
	controller := NewClientController()
	controller.WithContext(ctx)

	assert.Equal(t, ctx, controller.Context())

	t.Cleanup(func() {
		controller.Close()
		cncl()
	})
}

func TestControllerConnect(t *testing.T) {
	ctx, cncl := context.WithTimeout(context.Background(), 100*time.Millisecond)
	commanderClient := NewMockCommandClient()
	commanderClient.On("Connect", mock.Anything).Return(nil)
	commanderClient.On("Close", mock.Anything).Return(nil)

	metricsReportClient := NewMockMetricsReportClient()
	metricsReportClient.On("Connect", mock.Anything).Return(nil)
	metricsReportClient.On("Close", mock.Anything).Return(nil)

	controller := NewClientController()
	controller.WithClient(commanderClient)
	controller.WithClient(metricsReportClient)
	controller.WithContext(ctx)

	err := controller.Connect()
	assert.Nil(t, err)

	commanderClient.AssertNumberOfCalls(t, "Connect", 1)
	metricsReportClient.AssertNumberOfCalls(t, "Connect", 1)

	t.Cleanup(func() {
		controller.Close()
		cncl()
	})
}

func TestControllerConnect_error(t *testing.T) {
	ctx, cncl := context.WithTimeout(context.Background(), 100*time.Millisecond)

	commanderClient := NewMockCommandClient()
	commanderClient.On("Connect", mock.Anything).Return(fmt.Errorf("Error connecting"))
	commanderClient.On("Server").Return("127.0.0.1")
	commanderClient.On("Close", mock.Anything).Return(nil)

	metricsReportClient := NewMockMetricsReportClient()
	metricsReportClient.On("Connect", mock.Anything).Return(fmt.Errorf("Error connecting"))
	metricsReportClient.On("Server").Return("127.0.0.1")
	metricsReportClient.On("Close", mock.Anything).Return(nil)

	controller := NewClientController()
	controller.WithClient(commanderClient)
	controller.WithClient(metricsReportClient)
	controller.WithContext(ctx)

	err := controller.Connect()
	assert.NotNil(t, err)

	commanderClient.AssertNumberOfCalls(t, "Connect", 1)
	metricsReportClient.AssertNumberOfCalls(t, "Connect", 1)

	t.Cleanup(func() {
		controller.Close()
		cncl()
	})
}

func TestControllerClose(t *testing.T) {
	ctx, cncl := context.WithTimeout(context.Background(), 100*time.Millisecond)

	commanderClient := NewMockCommandClient()
	commanderClient.On("Close").Return(nil)

	metricsReportClient := NewMockMetricsReportClient()
	metricsReportClient.On("Close").Return(nil)

	controller := NewClientController()
	controller.WithClient(commanderClient)
	controller.WithClient(metricsReportClient)
	controller.WithContext(ctx)

	err := controller.Close()
	assert.Nil(t, err)

	commanderClient.AssertNumberOfCalls(t, "Close", 1)
	metricsReportClient.AssertNumberOfCalls(t, "Close", 1)

	t.Cleanup(func() {
		cncl()
	})
}

func TestControllerClose_error(t *testing.T) {
	ctx, cncl := context.WithTimeout(context.Background(), 100*time.Millisecond)

	commanderClient := NewMockCommandClient()
	commanderClient.On("Close").Return(fmt.Errorf("Error closing"))
	commanderClient.On("Server").Return("127.0.0.1")

	metricsReportClient := NewMockMetricsReportClient()
	metricsReportClient.On("Close").Return(fmt.Errorf("Error closing"))
	metricsReportClient.On("Server").Return("127.0.0.1")

	controller := NewClientController()
	controller.WithClient(commanderClient)
	controller.WithClient(metricsReportClient)
	controller.WithContext(ctx)

	err := controller.Close()
	assert.NotNil(t, err)

	commanderClient.AssertNumberOfCalls(t, "Close", 1)
	metricsReportClient.AssertNumberOfCalls(t, "Close", 1)

	t.Cleanup(func() {
		controller.Close()
		cncl()
	})
}
