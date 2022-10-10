package client

import (
	"context"
	"time"

	"github.com/nginx/agent/sdk/v2/interceptors"
	"github.com/stretchr/testify/mock"
	"google.golang.org/grpc"
)

type MockIngesterClient struct {
	mock.Mock
}

func NewMockIngesterClient() *MockIngesterClient {
	return &MockIngesterClient{}
}

var _ Ingester = NewMockIngesterClient()

func (m *MockIngesterClient) Server() string {
	args := m.Called()

	return args.String(0)
}

func (m *MockIngesterClient) WithServer(s string) Client {
	m.Called(s)

	return m
}

func (m *MockIngesterClient) DialOptions() []grpc.DialOption {
	args := m.Called()

	return args.Get(0).([]grpc.DialOption)
}

func (m *MockIngesterClient) WithDialOptions(options ...grpc.DialOption) Client {
	m.Called(options)

	return m
}

func (m *MockIngesterClient) WithInterceptor(interceptor interceptors.Interceptor) Client {
	m.Called(interceptor)

	return m
}

func (m *MockIngesterClient) WithClientInterceptor(interceptor interceptors.ClientInterceptor) Client {
	m.Called(interceptor)

	return m
}

func (m *MockIngesterClient) WithConnWaitDuration(d time.Duration) Client {
	m.Called(d)

	return m
}

func (m *MockIngesterClient) WithBackoffSettings(backoffSettings BackoffSettings) Client {
	m.Called(backoffSettings)

	return m
}

func (m *MockIngesterClient) Connect(ctx context.Context) error {
	args := m.Called(ctx)

	return args.Error(0)
}

func (m *MockIngesterClient) SendMetricsReport(ctx context.Context, message Message) error {
	args := m.Called(ctx, message)

	return args.Error(0)
}

func (m *MockIngesterClient) Close() error {
	args := m.Called()

	return args.Error(0)
}
