package utils

import (
	"context"
	"time"

	"github.com/nginx/agent/sdk/v2"
	"github.com/nginx/agent/sdk/v2/client"
	"github.com/nginx/agent/sdk/v2/interceptors"
	"github.com/stretchr/testify/mock"
	"google.golang.org/grpc"
)

type MockMetricsReportClient struct {
	mock.Mock
}

func NewMockMetricsReportClient() *MockMetricsReportClient {
	return &MockMetricsReportClient{}
}

var _ client.MetricReporter = NewMockMetricsReportClient()

func (m *MockMetricsReportClient) Server() string {
	args := m.Called()

	return args.String(0)
}

func (m *MockMetricsReportClient) WithServer(s string) client.Client {
	m.Called(s)

	return m
}

func (m *MockMetricsReportClient) DialOptions() []grpc.DialOption {
	args := m.Called()

	return args.Get(0).([]grpc.DialOption)
}

func (m *MockMetricsReportClient) WithDialOptions(options ...grpc.DialOption) client.Client {
	m.Called(options)

	return m
}

func (m *MockMetricsReportClient) WithInterceptor(interceptor interceptors.Interceptor) client.Client {
	m.Called(interceptor)

	return m
}

func (m *MockMetricsReportClient) WithClientInterceptor(interceptor interceptors.ClientInterceptor) client.Client {
	m.Called(interceptor)

	return m
}

func (m *MockMetricsReportClient) WithConnWaitDuration(d time.Duration) client.Client {
	m.Called(d)

	return m
}

func (m *MockMetricsReportClient) WithBackoffSettings(backoffSettings sdk.BackoffSettings) client.Client {
	m.Called(backoffSettings)

	return m
}

func (m *MockMetricsReportClient) Connect(ctx context.Context) error {
	args := m.Called(ctx)

	return args.Error(0)
}

func (m *MockMetricsReportClient) Send(ctx context.Context, message client.Message) error {
	args := m.Called(ctx, message)

	return args.Error(0)
}

func (m *MockMetricsReportClient) Close() error {
	args := m.Called()

	return args.Error(0)
}
