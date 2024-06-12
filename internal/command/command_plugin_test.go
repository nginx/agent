// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package command

import (
	"bytes"
	"context"
	"strings"
	"testing"
	"time"

	"github.com/nginx/agent/v3/internal/bus"
	"github.com/nginx/agent/v3/internal/command/commandfakes"
	"github.com/nginx/agent/v3/internal/grpc/grpcfakes"
	"github.com/nginx/agent/v3/test/protos"
	"github.com/nginx/agent/v3/test/stub"
	"github.com/nginx/agent/v3/test/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCommandPlugin_Info(t *testing.T) {
	commandPlugin := NewCommandPlugin(types.AgentConfig(), &grpcfakes.FakeGrpcConnectionInterface{})
	info := commandPlugin.Info()

	assert.Equal(t, "command", info.Name)
}

func TestCommandPlugin_Subscriptions(t *testing.T) {
	commandPlugin := NewCommandPlugin(types.AgentConfig(), &grpcfakes.FakeGrpcConnectionInterface{})
	subscriptions := commandPlugin.Subscriptions()

	assert.Equal(
		t,
		[]string{
			bus.ResourceUpdateTopic,
			bus.InstanceHealthTopic,
		},
		subscriptions,
	)
}

func TestCommandPlugin_Init(t *testing.T) {
	ctx := context.Background()
	messagePipe := bus.NewFakeMessagePipe()
	fakeCommandService := &commandfakes.FakeCommandService{}

	commandPlugin := NewCommandPlugin(types.AgentConfig(), &grpcfakes.FakeGrpcConnectionInterface{})
	err := commandPlugin.Init(ctx, messagePipe)
	require.NoError(t, err)

	require.NotNil(t, commandPlugin.messagePipe)
	require.NotNil(t, commandPlugin.commandService)

	commandPlugin.commandService = fakeCommandService

	closeError := commandPlugin.Close(ctx)
	require.NoError(t, closeError)
	require.Equal(t, 1, fakeCommandService.CancelSubscriptionCallCount())
}

func TestCommandPlugin_Process(t *testing.T) {
	ctx := context.Background()
	messagePipe := bus.NewFakeMessagePipe()
	fakeCommandService := &commandfakes.FakeCommandService{}

	commandPlugin := NewCommandPlugin(types.AgentConfig(), &grpcfakes.FakeGrpcConnectionInterface{})
	err := commandPlugin.Init(ctx, messagePipe)
	require.NoError(t, err)
	defer commandPlugin.Close(ctx)

	commandPlugin.commandService = fakeCommandService

	commandPlugin.Process(ctx, &bus.Message{Topic: bus.ResourceUpdateTopic, Data: protos.GetHostResource()})
	require.Equal(t, 1, fakeCommandService.UpdateDataPlaneStatusCallCount())

	commandPlugin.Process(ctx, &bus.Message{Topic: bus.InstanceHealthTopic, Data: protos.GetInstanceHealths()})
	require.Equal(t, 1, fakeCommandService.UpdateDataPlaneHealthCallCount())
}

func TestMonitorSubscribeChannel(t *testing.T) {
	ctx, cncl := context.WithCancel(context.Background())
	defer cncl()

	logBuf := &bytes.Buffer{}
	stub.StubLoggerWith(logBuf)

	cp := NewCommandPlugin(types.AgentConfig(), &grpcfakes.FakeGrpcConnectionInterface{})

	message := protos.CreateManagementPlaneRequest()

	// Run in a separate goroutine
	go cp.monitorSubscribeChannel(ctx)

	// Give some time to exit the goroutine
	time.Sleep(100 * time.Millisecond)

	cp.subscribeChannel <- message

	// Give some time to process the message
	time.Sleep(100 * time.Millisecond)

	cncl()

	time.Sleep(100 * time.Millisecond)

	// Verify the logger was called
	if s := logBuf.String(); !strings.Contains(s, "Received management plane request") {
		// defer wg.Done()
		t.Errorf("Unexpected log %s", s)
	}

	// Clear the log buffer
	logBuf.Reset()
}
