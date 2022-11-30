/**
 * Copyright (c) F5, Inc.
 *
 * This source code is licensed under the Apache License, Version 2.0 license found in the
 * LICENSE file in the root directory of this source tree.
 */

package reader

import (
	"context"
	"errors"
	"fmt"
	"net"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/nginx/agent/v2/src/extensions/advanced-metrics/reader/mocks"
	"github.com/stretchr/testify/assert"
)

const address = "/tmp/advanced-metrics.sr"

func TestReaderShouldAcceptNewConnectionAndStartWorker(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	configMock := mocks.NewMockListenerConfig(ctrl)
	listenerMock := mocks.NewMockListener(ctrl)
	workerMock := mocks.NewMockWorker(ctrl)
	conn := mocks.NewMockConn(ctrl)
	newWorker := func(net.Conn, chan Frame) Worker {
		return workerMock
	}

	frameChannel := make(chan Frame)
	reader := newReader(address, configMock, frameChannel, newWorker)
	ctx, cancel := context.WithCancel(context.Background())

	configMock.EXPECT().Listen(gomock.Any(), "unix", address).Return(listenerMock, nil)

	listenerMock.EXPECT().Accept().Return(conn, nil).Do(func() {
		cancel()
	})
	listenerMock.EXPECT().Accept().Return(nil, net.ErrClosed)
	listenerMock.EXPECT().Close().Return(nil)

	workerMock.EXPECT().Run(gomock.Any()).Do(func(ctx context.Context) {
		<-ctx.Done()
	}).Return(nil)

	assert.NoError(t, reader.Run(ctx))

	_, ok := <-frameChannel
	assert.False(t, ok)
}

func TestReaderShouldStopWorkersOnAcceptError(t *testing.T) {
	const dummyError = "error"

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	configMock := mocks.NewMockListenerConfig(ctrl)
	listenerMock := mocks.NewMockListener(ctrl)
	conn := mocks.NewMockConn(ctrl)
	workerMock := mocks.NewMockWorker(ctrl)
	newWorker := func(net.Conn, chan Frame) Worker {
		return workerMock
	}

	frameChannel := make(chan Frame)
	reader := newReader(address, configMock, frameChannel, newWorker)
	ctx := context.Background()

	configMock.EXPECT().Listen(gomock.Any(), "unix", address).Return(listenerMock, nil)

	listenerMock.EXPECT().Accept().Return(conn, nil)
	listenerMock.EXPECT().Accept().Return(nil, errors.New(dummyError))
	listenerMock.EXPECT().Close().Return(nil)

	workerMock.EXPECT().Run(gomock.Any()).Do(func(ctx context.Context) {
		<-ctx.Done()
	})

	assert.EqualError(t, reader.Run(ctx), fmt.Sprintf("fail to accept new connection: %s", dummyError))

	_, ok := <-frameChannel
	assert.False(t, ok)
}
