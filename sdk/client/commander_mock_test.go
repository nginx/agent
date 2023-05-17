/**
 * Copyright (c) F5, Inc.
 *
 * This source code is licensed under the Apache License, Version 2.0 license found in the
 * LICENSE file in the root directory of this source tree.
 */

package client

import (
	"context"
	"time"

	"github.com/nginx/agent/sdk/v2/backoff"
	"github.com/nginx/agent/sdk/v2/interceptors"
	"github.com/nginx/agent/sdk/v2/proto"
	"github.com/stretchr/testify/mock"
	"google.golang.org/grpc"
)

type MockCommandClient struct {
	mock.Mock
}

func NewMockCommandClient() *MockCommandClient {
	return &MockCommandClient{}
}

var _ Commander = NewMockCommandClient()

func (m *MockCommandClient) Connect(ctx context.Context) error {
	args := m.Called(ctx)

	return args.Error(0)
}

func (m *MockCommandClient) Close() error {
	args := m.Called()

	return args.Error(0)
}

func (m *MockCommandClient) Server() string {
	args := m.Called()

	return args.String(0)
}

func (m *MockCommandClient) WithServer(s string) Client {
	m.Called(s)

	return m
}

func (m *MockCommandClient) DialOptions() []grpc.DialOption {
	args := m.Called()

	return args.Get(0).([]grpc.DialOption)
}

func (m *MockCommandClient) WithDialOptions(options ...grpc.DialOption) Client {
	m.Called(options)

	return m
}

func (m *MockCommandClient) ChunksSize() int {
	args := m.Called()

	return args.Int(0)
}

func (m *MockCommandClient) WithChunkSize(i int) Client {
	m.Called(i)

	return m
}

func (m *MockCommandClient) WithInterceptor(interceptor interceptors.Interceptor) Client {
	m.Called(interceptor)

	return m
}

func (m *MockCommandClient) WithClientInterceptor(interceptor interceptors.ClientInterceptor) Client {
	m.Called(interceptor)

	return m
}

func (m *MockCommandClient) WithConnWaitDuration(d time.Duration) Client {
	m.Called(d)

	return m
}

func (m *MockCommandClient) WithBackoffSettings(backoffSettings backoff.BackoffSettings) Client {
	m.Called(backoffSettings)

	return m
}

func (m *MockCommandClient) Send(ctx context.Context, message Message) error {
	m.Called(ctx, message)

	return nil
}

func (m *MockCommandClient) Recv() <-chan Message {
	args := m.Called()

	return args.Get(0).(<-chan Message)
}

func (m *MockCommandClient) Download(_ context.Context, meta *proto.Metadata) (*proto.NginxConfig, error) {
	args := m.Called(meta)
	cfg := args.Get(0).(*proto.NginxConfig)
	err := args.Error(1)

	return cfg, err
}

func (m *MockCommandClient) Upload(_ context.Context, cfg *proto.NginxConfig, messageId string) error {
	args := m.Called(cfg, messageId)
	return args.Error(0)
}
