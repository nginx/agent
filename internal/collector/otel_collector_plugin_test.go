// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.
package collector

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/otelcol"

	"github.com/nginx/agent/v3/internal/bus"
	"github.com/nginx/agent/v3/internal/config"
	"github.com/nginx/agent/v3/internal/model"
	"github.com/nginx/agent/v3/test/helpers"
	"github.com/nginx/agent/v3/test/types"
)

func TestCollector_New(t *testing.T) {
	conf := types.OTelConfig(t)
	_, err := New(conf)
	require.NoError(t, err, "NewCollector should not return an error with valid config")
}

func TestCollector_InitAndClose(t *testing.T) {
	conf := types.OTelConfig(t)
	collector, err := New(conf)
	require.NoError(t, err, "NewCollector should not return an error with valid config")

	ctx := context.Background()
	messagePipe := bus.NewMessagePipe(10)
	err = messagePipe.Register(10, []bus.Plugin{collector})

	require.NoError(t, err)
	require.NoError(t, collector.Init(ctx, messagePipe), "Init should not return an error")

	assert.Eventually(
		t,
		func() bool { return collector.service.GetState() == otelcol.StateRunning },
		2*time.Second,
		100*time.Millisecond,
	)

	require.NoError(t, collector.Close(ctx), "Close should not return an error")

	assert.Eventually(
		t,
		func() bool { return collector.service.GetState() == otelcol.StateClosed },
		2*time.Second,
		100*time.Millisecond,
	)
}

// nolint: revive
func TestCollector_Process(t *testing.T) {
	nginxPlusMock := helpers.NewMockNGINXPlusAPIServer(t)
	defer nginxPlusMock.Close()

	conf := types.OTelConfig(t)

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
					PlusAPI:    fmt.Sprintf("%s/api", nginxPlusMock.URL),
				},
			},
			receivers: config.Receivers{
				HostMetrics: config.HostMetrics{
					CollectionInterval: time.Minute,
					InitialDelay:       time.Second,
				},
				OtlpReceivers: []config.OtlpReceiver{
					{
						Server: &config.ServerConfig{
							Host: "localhost",
							Port: 4321,
							Type: 0,
						},
						Auth: &config.AuthConfig{
							Token: "even-secreter-token",
						},
					},
				},
				NginxPlusReceivers: []config.NginxPlusReceiver{
					{
						InstanceID: "123",
						PlusAPI:    fmt.Sprintf("%s/api", nginxPlusMock.URL),
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
					StubStatus: "http://test.com:8080/stub_status",
				},
			},
			receivers: config.Receivers{
				HostMetrics: config.HostMetrics{
					CollectionInterval: time.Minute,
					InitialDelay:       time.Second,
				},
				OtlpReceivers: []config.OtlpReceiver{
					{
						Server: &config.ServerConfig{
							Host: "localhost",
							Port: 4321,
							Type: 0,
						},
						Auth: &config.AuthConfig{
							Token: "even-secreter-token",
						},
					},
				},
				NginxReceivers: []config.NginxReceiver{
					{
						InstanceID: "123",
						StubStatus: "http://test.com:8080/stub_status",
					},
				},
				NginxPlusReceivers: []config.NginxPlusReceiver{},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(tt *testing.T) {
			collector, err := New(conf)
			require.NoError(tt, err, "NewCollector should not return an error with valid config")

			ctx := context.Background()
			messagePipe := bus.NewMessagePipe(10)
			err = messagePipe.Register(10, []bus.Plugin{collector})

			require.NoError(tt, err)
			require.NoError(tt, collector.Init(ctx, messagePipe), "Init should not return an error")
			defer collector.Close(ctx)

			assert.Eventually(
				tt,
				func() bool { return collector.service.GetState() == otelcol.StateRunning },
				5*time.Second,
				100*time.Millisecond,
			)

			collector.Process(ctx, test.message)

			assert.Eventually(
				tt,
				func() bool { return collector.service.GetState() == otelcol.StateRunning },
				5*time.Second,
				100*time.Millisecond,
			)

			if len(test.receivers.NginxPlusReceivers) > 0 {
				assert.Eventually(
					tt,
					func() bool { return len(collector.config.Collector.Receivers.NginxPlusReceivers) > 0 },
					5*time.Second,
					100*time.Millisecond,
				)
			}

			if len(test.receivers.NginxReceivers) > 0 {
				assert.Eventually(
					tt,
					func() bool { return len(collector.config.Collector.Receivers.NginxReceivers) > 0 },
					5*time.Second,
					100*time.Millisecond,
				)
			}

			assert.Equal(tt, test.receivers, collector.config.Collector.Receivers)
		})
	}
}

// nolint: dupl
func TestCollector_updateExistingNginxOSSReceiver(t *testing.T) {
	conf := types.OTelConfig(t)

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
				NginxReceivers: []config.NginxReceiver{
					{
						InstanceID: "123",
						StubStatus: "http://new-test-host:8080/api",
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
