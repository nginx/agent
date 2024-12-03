// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.
package collector

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/nginx/agent/v3/test/protos"
	"github.com/nginx/agent/v3/test/stub"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/otelcol"

	"github.com/nginx/agent/v3/internal/bus"
	"github.com/nginx/agent/v3/internal/collector/types/typesfakes"
	"github.com/nginx/agent/v3/internal/config"
	"github.com/nginx/agent/v3/internal/model"
	"github.com/nginx/agent/v3/test/helpers"
	"github.com/nginx/agent/v3/test/types"
)

func TestCollector_New(t *testing.T) {
	tmpDir := t.TempDir()

	tests := []struct {
		config        *config.Config
		expectedError error
		name          string
	}{
		{
			name:          "Nil agent config",
			config:        nil,
			expectedError: errors.New("nil agent config"),
		},
		{
			name: "Nil collector config",
			config: &config.Config{
				Collector: nil,
			},
			expectedError: errors.New("nil collector config"),
		},
		{
			name: "File write error",
			config: &config.Config{
				Collector: &config.Collector{
					Log: &config.Log{Path: "/invalid/path"},
				},
			},
			expectedError: errors.New("open /invalid/path: no such file or directory"),
		},
		{
			name: "Successful initialization",
			config: &config.Config{
				Collector: &config.Collector{
					Log: &config.Log{Path: filepath.Join(tmpDir, "test.log")},
				},
			},
			expectedError: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			collector, err := New(tt.config)

			if tt.expectedError != nil {
				require.Error(t, err)
				assert.Equal(t, tt.expectedError.Error(), err.Error())
			} else {
				require.NoError(t, err)
				assert.NotNil(t, collector)
			}
		})
	}
}

func TestCollector_Init(t *testing.T) {
	tmpDir := t.TempDir()

	tests := []struct {
		name          string
		expectedLog   string
		expectedError bool
	}{
		{
			name:          "Default configured",
			expectedError: false,
			expectedLog:   "",
		},
		{
			name:          "No receivers set in config",
			expectedError: true,
			expectedLog:   "No receivers configured for OTel Collector",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conf := types.OTelConfig(t)

			var collector *Collector
			var err error
			logBuf := &bytes.Buffer{}
			stub.StubLoggerWith(logBuf)

			conf.Collector.Log = &config.Log{Path: filepath.Join(tmpDir, "test.log")}

			if tt.expectedError {
				conf.Collector.Receivers = config.Receivers{}
			}

			collector, err = New(conf)
			require.NoError(t, err, "NewCollector should not return an error with valid config")

			collector.service = createFakeCollector()

			initError := collector.Init(context.Background(), nil)
			require.NoError(t, initError)

			validateLog(t, tt.expectedLog, logBuf)

			require.NoError(t, collector.Close(context.TODO()))
		})
	}
}

func validateLog(t *testing.T, expectedLog string, logBuf *bytes.Buffer) {
	t.Helper()

	if expectedLog != "" {
		if !strings.Contains(logBuf.String(), expectedLog) {
			t.Errorf("Expected log to contain %q, but got %q", expectedLog, logBuf.String())
		}
	}
}

func TestCollector_InitAndClose(t *testing.T) {
	conf := types.OTelConfig(t)
	conf.Collector.Log.Path = ""

	collector, err := New(conf)
	require.NoError(t, err, "NewCollector should not return an error with valid config")

	ctx := context.Background()
	messagePipe := bus.NewMessagePipe(10)
	err = messagePipe.Register(10, []bus.Plugin{collector})

	require.NoError(t, err)
	require.NoError(t, collector.Init(ctx, messagePipe), "Init should not return an error")

	collector.service = createFakeCollector()

	assert.Equal(t, otelcol.StateRunning, collector.GetState())

	require.NoError(t, collector.Close(ctx), "Close should not return an error")

	assert.Equal(t, otelcol.StateClosed, collector.GetState())
}

// nolint: revive
func TestCollector_ProcessNginxConfigUpdateTopic(t *testing.T) {
	tests := []struct {
		name      string
		message   *bus.Message
		receivers config.Receivers
	}{
		{
			name: "Test 1: NGINX Plus receiver",
			message: &bus.Message{
				Topic: bus.NginxConfigUpdateTopic,
				Data: &model.NginxConfigContext{
					InstanceID: "123",
					PlusAPI:    "",
				},
			},
			receivers: config.Receivers{
				HostMetrics:   nil,
				OtlpReceivers: nil,
				NginxPlusReceivers: []config.NginxPlusReceiver{
					{
						InstanceID: "123",
						PlusAPI:    "",
					},
				},
			},
		},
		{
			name: "Test 2: NGINX receiver",
			message: &bus.Message{
				Topic: bus.NginxConfigUpdateTopic,
				Data: &model.NginxConfigContext{
					InstanceID: "123",
					StubStatus: "",
					AccessLogs: []*model.AccessLog{
						{
							Name:   "/var/log/nginx/access.log",
							Format: "$remote_addr - $remote_user [$time_local] \"$request\"",
						},
					},
				},
			},
			receivers: config.Receivers{
				HostMetrics:   nil,
				OtlpReceivers: nil,
				NginxReceivers: []config.NginxReceiver{
					{
						InstanceID: "123",
						StubStatus: "",
						AccessLogs: []config.AccessLog{
							{
								FilePath:  "/var/log/nginx/access.log",
								LogFormat: "$$remote_addr - $$remote_user [$$time_local] \\\"$$request\\\"",
							},
						},
					},
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(tt *testing.T) {
			nginxPlusMock := helpers.NewMockNGINXPlusAPIServer(t)
			defer nginxPlusMock.Close()

			conf := types.OTelConfig(t)

			conf.Command = nil

			conf.Collector.Log.Path = ""
			conf.Collector.Receivers.HostMetrics = nil
			conf.Collector.Receivers.OtlpReceivers = nil

			if len(test.receivers.NginxPlusReceivers) == 1 {
				url := fmt.Sprintf("%s/api", nginxPlusMock.URL)
				test.receivers.NginxPlusReceivers[0].PlusAPI = url

				model, ok := test.message.Data.(*model.NginxConfigContext)
				if !ok {
					t.Logf("Can't cast type")
					t.Fail()
				}

				model.PlusAPI = url
			} else {
				url := fmt.Sprintf("%s/stub_status", nginxPlusMock.URL)
				test.receivers.NginxReceivers[0].StubStatus = url

				model, ok := test.message.Data.(*model.NginxConfigContext)
				if !ok {
					t.Logf("Can't cast type")
					t.Fail()
				}

				model.StubStatus = url
			}

			conf.Collector.Processors.Batch = nil
			conf.Collector.Processors.Attribute = nil
			conf.Collector.Processors.Resource = nil
			conf.Collector.Extensions.Health = nil
			conf.Collector.Extensions.HeadersSetter = nil
			conf.Collector.Exporters.PrometheusExporter = nil

			collector, err := New(conf)
			require.NoError(tt, err, "NewCollector should not return an error with valid config")

			collector.service = createFakeCollector()

			ctx := context.Background()
			messagePipe := bus.NewMessagePipe(10)
			err = messagePipe.Register(10, []bus.Plugin{collector})

			require.NoError(tt, err)
			require.NoError(tt, collector.Init(ctx, messagePipe), "Init should not return an error")

			collector.Process(ctx, test.message)

			assert.Equal(tt, test.receivers, collector.config.Collector.Receivers)

			defer collector.Close(ctx)
		})
	}
}

func TestCollector_ProcessResourceUpdateTopic(t *testing.T) {
	conf := types.OTelConfig(t)
	conf.Collector.Log.Path = ""
	conf.Collector.Processors.Batch = nil
	conf.Collector.Processors.Attribute = nil
	conf.Collector.Processors.Resource = nil
	conf.Collector.Exporters.OtlpExporters = nil
	conf.Collector.Exporters.PrometheusExporter = &config.PrometheusExporter{
		Server: &config.ServerConfig{
			Host: "",
			Port: 0,
			Type: 0,
		},
		TLS: &config.TLSConfig{
			Cert:       "",
			Key:        "",
			Ca:         "",
			ServerName: "",
			SkipVerify: false,
		},
	}

	tests := []struct {
		message    *bus.Message
		processors config.Processors
		name       string
		headers    []config.Header
	}{
		{
			name: "Test 1: Resource update adds resource id attribute",
			message: &bus.Message{
				Topic: bus.ResourceUpdateTopic,
				Data:  protos.GetHostResource(),
			},
			processors: config.Processors{
				Resource: &config.Resource{
					Attributes: []config.ResourceAttribute{
						{
							Key:    "resource.id",
							Action: "insert",
							Value:  "1234",
						},
					},
				},
			},
			headers: []config.Header{
				{
					Action: "insert",
					Key:    "authorization",
					Value:  "fake-authorization",
				},
				{
					Action: "insert",
					Key:    "uuid",
					Value:  "1234",
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(tt *testing.T) {
			collector, err := New(conf)
			require.NoError(tt, err, "NewCollector should not return an error with valid config")

			collector.service = createFakeCollector()

			ctx := context.Background()
			messagePipe := bus.NewMessagePipe(10)
			err = messagePipe.Register(10, []bus.Plugin{collector})

			require.NoError(tt, err)
			require.NoError(tt, collector.Init(ctx, messagePipe), "Init should not return an error")

			collector.Process(ctx, test.message)

			assert.Equal(tt, test.processors, collector.config.Collector.Processors)
			assert.Equal(tt, test.headers, collector.config.Collector.Extensions.HeadersSetter.Headers)

			defer collector.Close(ctx)
		})
	}
}

func TestCollector_ProcessResourceUpdateTopicFails(t *testing.T) {
	conf := types.OTelConfig(t)
	conf.Collector.Log.Path = ""
	conf.Collector.Processors.Batch = nil
	conf.Collector.Processors.Attribute = nil
	conf.Collector.Processors.Resource = nil
	conf.Collector.Exporters.OtlpExporters = nil
	conf.Collector.Exporters.PrometheusExporter = &config.PrometheusExporter{
		Server: &config.ServerConfig{
			Host: "",
			Port: 0,
			Type: 0,
		},
		TLS: &config.TLSConfig{
			Cert:       "",
			Key:        "",
			Ca:         "",
			ServerName: "",
			SkipVerify: false,
		},
	}

	tests := []struct {
		message    *bus.Message
		processors config.Processors
		name       string
	}{
		{
			name: "Test 1: Message cannot be parsed to v1.Resource",
			message: &bus.Message{
				Topic: bus.ResourceUpdateTopic,
				Data:  struct{}{},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(tt *testing.T) {
			collector, err := New(conf)
			require.NoError(tt, err, "NewCollector should not return an error with valid config")

			collector.service = createFakeCollector()

			ctx := context.Background()
			messagePipe := bus.NewMessagePipe(10)
			err = messagePipe.Register(10, []bus.Plugin{collector})

			require.NoError(tt, err)
			require.NoError(tt, collector.Init(ctx, messagePipe), "Init should not return an error")
			defer collector.Close(ctx)

			collector.Process(ctx, test.message)

			assert.Equal(tt,
				config.Processors{
					Batch:     nil,
					Attribute: nil,
					Resource:  nil,
				},
				collector.config.Collector.Processors)
		})
	}
}

// nolint: dupl
func TestCollector_updateExistingNginxOSSReceiver(t *testing.T) {
	conf := types.OTelConfig(t)
	conf.Collector.Log.Path = ""

	tests := []struct {
		name               string
		nginxConfigContext *model.NginxConfigContext
		existingReceivers  config.Receivers
		expectedReceivers  config.Receivers
	}{
		{
			name: "Test 1: Existing NGINX Receiver",
			nginxConfigContext: &model.NginxConfigContext{
				InstanceID: "123",
				StubStatus: "http://new-test-host:8080/api",
				AccessLogs: []*model.AccessLog{
					{
						Name:   "/etc/nginx/test.log",
						Format: `$remote_addr [$time_local] "$request" $status`,
					},
				},
			},
			existingReceivers: config.Receivers{
				NginxReceivers: []config.NginxReceiver{
					{
						InstanceID: "123",
						StubStatus: "http://test.com:8080/api",
						AccessLogs: []config.AccessLog{
							{
								FilePath:  "/etc/nginx/existing.log",
								LogFormat: `$remote_addr [$time_local] "$request"`,
							},
						},
					},
				},
			},
			expectedReceivers: config.Receivers{
				NginxReceivers: []config.NginxReceiver{
					{
						InstanceID: "123",
						StubStatus: "http://new-test-host:8080/api",
						AccessLogs: []config.AccessLog{
							{
								FilePath:  "/etc/nginx/test.log",
								LogFormat: "$$remote_addr [$$time_local] \\\"$$request\\\" $$status",
							},
						},
					},
				},
			},
		},
		{
			name: "Test 2: Removing NGINX Receiver",
			nginxConfigContext: &model.NginxConfigContext{
				InstanceID: "123",
				StubStatus: "",
			},
			existingReceivers: config.Receivers{
				NginxReceivers: []config.NginxReceiver{
					{
						InstanceID: "123",
						StubStatus: "http://test.com:8080/api",
					},
				},
			},
			expectedReceivers: config.Receivers{
				NginxReceivers: []config.NginxReceiver{},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(tt *testing.T) {
			conf.Collector.Receivers = test.existingReceivers
			collector, err := New(conf)
			require.NoError(tt, err, "NewCollector should not return an error with valid config")

			nginxReceiverFound, reloadCollector := collector.updateExistingNginxOSSReceiver(test.nginxConfigContext)

			assert.True(tt, nginxReceiverFound)
			assert.True(tt, reloadCollector)
			assert.Equal(tt, test.expectedReceivers, collector.config.Collector.Receivers)
		})
	}
}

// nolint: dupl
func TestCollector_updateExistingNginxPlusReceiver(t *testing.T) {
	conf := types.OTelConfig(t)
	conf.Collector.Log.Path = ""

	tests := []struct {
		name               string
		nginxConfigContext *model.NginxConfigContext
		existingReceivers  config.Receivers
		expectedReceivers  config.Receivers
	}{
		{
			name: "Test 1: Existing NGINX Plus Receiver",
			nginxConfigContext: &model.NginxConfigContext{
				InstanceID: "123",
				PlusAPI:    "http://new-test-host:8080/api",
			},
			existingReceivers: config.Receivers{
				NginxPlusReceivers: []config.NginxPlusReceiver{
					{
						InstanceID: "123",
						PlusAPI:    "http://test.com:8080/api",
					},
				},
			},
			expectedReceivers: config.Receivers{
				NginxPlusReceivers: []config.NginxPlusReceiver{
					{
						InstanceID: "123",
						PlusAPI:    "http://new-test-host:8080/api",
					},
				},
			},
		},
		{
			name: "Test 2: Removing NGINX Plus Receiver",
			nginxConfigContext: &model.NginxConfigContext{
				InstanceID: "123",
				PlusAPI:    "",
			},
			existingReceivers: config.Receivers{
				NginxPlusReceivers: []config.NginxPlusReceiver{
					{
						InstanceID: "123",
						PlusAPI:    "http://test.com:8080/api",
					},
				},
			},
			expectedReceivers: config.Receivers{
				NginxPlusReceivers: []config.NginxPlusReceiver{},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(tt *testing.T) {
			conf.Collector.Receivers = test.existingReceivers
			collector, err := New(conf)
			require.NoError(tt, err, "NewCollector should not return an error with valid config")

			nginxReceiverFound, reloadCollector := collector.updateExistingNginxPlusReceiver(test.nginxConfigContext)

			assert.True(tt, nginxReceiverFound)
			assert.True(tt, reloadCollector)
			assert.Equal(tt, test.expectedReceivers, collector.config.Collector.Receivers)
		})
	}
}

func TestCollector_updateResourceAttributes(t *testing.T) {
	conf := types.OTelConfig(t)
	conf.Collector.Log.Path = ""
	conf.Collector.Processors.Batch = nil
	conf.Collector.Processors.Attribute = nil
	conf.Collector.Processors.Resource = nil

	tests := []struct {
		name                   string
		setup                  []config.ResourceAttribute
		attributes             []config.ResourceAttribute
		expectedAttribs        []config.ResourceAttribute
		expectedReloadRequired bool
	}{
		{
			name:                   "Test 1: No Actions returns false",
			setup:                  []config.ResourceAttribute{},
			attributes:             []config.ResourceAttribute{},
			expectedReloadRequired: false,
			expectedAttribs:        []config.ResourceAttribute{},
		},
		{
			name:                   "Test 2: Adding an action returns true",
			setup:                  []config.ResourceAttribute{},
			attributes:             []config.ResourceAttribute{{Key: "test", Action: "insert", Value: "test value"}},
			expectedReloadRequired: true,
			expectedAttribs:        []config.ResourceAttribute{{Key: "test", Action: "insert", Value: "test value"}},
		},
		{
			name:  "Test 3: Adding a duplicate key doesn't append",
			setup: []config.ResourceAttribute{{Key: "test", Action: "insert", Value: "test value 1"}},
			attributes: []config.ResourceAttribute{
				{Key: "test", Action: "insert", Value: "updated value 2"},
			},
			expectedReloadRequired: false,
			expectedAttribs:        []config.ResourceAttribute{{Key: "test", Action: "insert", Value: "test value 1"}},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(tt *testing.T) {
			collector, err := New(conf)
			require.NoError(tt, err, "NewCollector should not return an error with valid config")

			// set up Actions
			conf.Collector.Processors.Resource = &config.Resource{Attributes: test.setup}

			reloadRequired := collector.updateResourceAttributes(test.attributes)
			assert.Equal(tt,
				test.expectedAttribs,
				conf.Collector.Processors.Resource.Attributes)
			assert.Equal(tt, test.expectedReloadRequired, reloadRequired)
		})
	}
}

func TestCollector_updateTcplogReceivers(t *testing.T) {
	conf := types.OTelConfig(t)
	conf.Collector.Log.Path = ""
	conf.Collector.Processors.Batch = nil
	conf.Collector.Processors.Attribute = nil
	conf.Collector.Processors.Resource = nil

	collector, err := New(conf)
	require.NoError(t, err)

	nginxConfigContext := &model.NginxConfigContext{
		NAPSysLogServers: []string{
			"localhost:151",
		},
	}

	assert.Empty(t, conf.Collector.Receivers.TcplogReceivers)

	t.Run("Test 1: New TcplogReceiver added", func(tt *testing.T) {
		tcplogReceiverAdded := collector.updateTcplogReceivers(nginxConfigContext)

		assert.True(tt, tcplogReceiverAdded)
		assert.Len(tt, conf.Collector.Receivers.TcplogReceivers, 1)
		assert.Equal(tt, "localhost:151", conf.Collector.Receivers.TcplogReceivers[0].ListenAddress)
		assert.Len(tt, conf.Collector.Receivers.TcplogReceivers[0].Operators, 4)
	})

	// Calling updateTcplogReceivers shouldn't update the TcplogReceivers slice
	// since there is already a receiver with the same ListenAddress
	t.Run("Test 2: TcplogReceiver already exists", func(tt *testing.T) {
		tcplogReceiverAdded := collector.updateTcplogReceivers(nginxConfigContext)
		assert.False(t, tcplogReceiverAdded)
		assert.Len(t, conf.Collector.Receivers.TcplogReceivers, 1)
		assert.Equal(t, "localhost:151", conf.Collector.Receivers.TcplogReceivers[0].ListenAddress)
		assert.Len(t, conf.Collector.Receivers.TcplogReceivers[0].Operators, 4)
	})

	t.Run("Test 3: TcplogReceiver deleted", func(tt *testing.T) {
		tcplogReceiverDeleted := collector.updateTcplogReceivers(&model.NginxConfigContext{})
		assert.True(t, tcplogReceiverDeleted)
		assert.Empty(t, conf.Collector.Receivers.TcplogReceivers)
	})

	t.Run("Test 4: New tcplogReceiver added and deleted another", func(tt *testing.T) {
		tcplogReceiverDeleted := collector.updateTcplogReceivers(&model.NginxConfigContext{NAPSysLogServers: []string{
			"localhost:152",
		}})
		assert.True(t, tcplogReceiverDeleted)
		assert.Len(t, conf.Collector.Receivers.TcplogReceivers, 1)
		assert.Equal(t, "localhost:152", conf.Collector.Receivers.TcplogReceivers[0].ListenAddress)
		assert.Len(t, conf.Collector.Receivers.TcplogReceivers[0].Operators, 5)
	})
}

func createFakeCollector() *typesfakes.FakeCollectorInterface {
	fakeCollector := &typesfakes.FakeCollectorInterface{}
	fakeCollector.RunStub = func(ctx context.Context) error { return nil }
	fakeCollector.GetStateReturnsOnCall(0, otelcol.StateRunning)
	fakeCollector.GetStateReturnsOnCall(1, otelcol.StateClosing)
	fakeCollector.ShutdownCalls(func() {
		fakeCollector.GetStateReturns(otelcol.StateClosed)
	})

	return fakeCollector
}
