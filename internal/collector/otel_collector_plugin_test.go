// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.
package collector

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/nginx/agent/v3/test/protos"
	"github.com/nginx/agent/v3/test/stub"

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
	conf.Collector.Log.Path = ""

	_, err := New(conf)
	require.NoError(t, err, "NewCollector should not return an error with valid config")
}

func TestCollector_Init(t *testing.T) {
	conf := types.OTelConfig(t)
	conf.Collector = &config.Collector{}

	logBuf := &bytes.Buffer{}
	stub.StubLoggerWith(logBuf)

	collector, err := New(conf)
	require.NoError(t, err, "NewCollector should not return an error with valid config")

	initError := collector.Init(context.Background(), nil)
	require.NoError(t, initError)

	if s := logBuf.String(); !strings.Contains(s, "No receivers configured for OTel Collector. "+
		"Waiting to discover a receiver before starting OTel collector.") {
		t.Errorf("Unexpected log %s", s)
	}

	assert.True(t, collector.stopped)
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
func TestCollector_ProcessNginxConfigUpdateTopic(t *testing.T) {
	nginxPlusMock := helpers.NewMockNGINXPlusAPIServer(t)
	defer nginxPlusMock.Close()

	conf := types.OTelConfig(t)
	conf.Collector.Log.Path = ""

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
				HostMetrics: &config.HostMetrics{
					CollectionInterval: time.Minute,
					InitialDelay:       time.Second,
					Scrapers: &config.HostMetricsScrapers{
						CPU:        &config.CPUScraper{},
						Disk:       &config.DiskScraper{},
						Filesystem: &config.FilesystemScraper{},
						Memory:     &config.MemoryScraper{},
						Network:    &config.NetworkScraper{},
					},
				},
				OtlpReceivers: types.OtlpReceivers(),
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
					AccessLogs: []*model.AccessLog{
						{
							Name:   "/var/log/nginx/access.log",
							Format: "$remote_addr - $remote_user [$time_local] \"$request\"",
						},
					},
				},
			},
			receivers: config.Receivers{
				HostMetrics: &config.HostMetrics{
					CollectionInterval: time.Minute,
					InitialDelay:       time.Second,
					Scrapers: &config.HostMetricsScrapers{
						CPU:        &config.CPUScraper{},
						Disk:       &config.DiskScraper{},
						Filesystem: &config.FilesystemScraper{},
						Memory:     &config.MemoryScraper{},
						Network:    &config.NetworkScraper{},
					},
				},
				OtlpReceivers: types.OtlpReceivers(),
				NginxReceivers: []config.NginxReceiver{
					{
						InstanceID: "123",
						StubStatus: "http://test.com:8080/stub_status",
						AccessLogs: []config.AccessLog{
							{
								FilePath:  "/var/log/nginx/access.log",
								LogFormat: "$$remote_addr - $$remote_user [$$time_local] \\\"$$request\\\"",
							},
						},
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

func TestCollector_ProcessResourceUpdateTopic(t *testing.T) {
	conf := types.OTelConfig(t)
	conf.Collector.Log.Path = ""
	conf.Collector.Processors.Batch = nil
	conf.Collector.Processors.Attribute = nil

	tests := []struct {
		message    *bus.Message
		processors config.Processors
		name       string
	}{
		{
			name: "Test 1: Resource update adds resource id action",
			message: &bus.Message{
				Topic: bus.ResourceUpdateTopic,
				Data:  protos.GetHostResource(),
			},
			processors: config.Processors{
				Attribute: &config.Attribute{
					Actions: []config.Action{
						{
							Key:    "resource.id",
							Action: "insert",
							Value:  "1234",
						},
					},
				},
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

			assert.Equal(tt, test.processors, collector.config.Collector.Processors)
		})
	}
}

func TestCollector_ProcessResourceUpdateTopicFails(t *testing.T) {
	conf := types.OTelConfig(t)
	conf.Collector.Log.Path = ""
	conf.Collector.Processors.Batch = nil
	conf.Collector.Processors.Attribute = nil

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

			assert.Equal(tt,
				config.Processors{
					Batch:     nil,
					Attribute: nil,
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

	tests := []struct {
		name                   string
		setupActions           []config.Action
		actions                []config.Action
		expectedAttribs        []config.Action
		expectedReloadRequired bool
	}{
		{
			name:                   "Test 1: No Actions returns false",
			setupActions:           []config.Action{},
			actions:                []config.Action{},
			expectedReloadRequired: false,
			expectedAttribs:        []config.Action{},
		},
		{
			name:                   "Test 2: Adding an action returns true",
			setupActions:           []config.Action{},
			actions:                []config.Action{{Key: "test", Action: "insert", Value: "test value"}},
			expectedReloadRequired: true,
			expectedAttribs:        []config.Action{{Key: "test", Action: "insert", Value: "test value"}},
		},
		{
			name:                   "Test 3: Adding a duplicate key doesn't append",
			setupActions:           []config.Action{{Key: "test", Action: "insert", Value: "test value 1"}},
			actions:                []config.Action{{Key: "test", Action: "insert", Value: "updated value 2"}},
			expectedReloadRequired: false,
			expectedAttribs:        []config.Action{{Key: "test", Action: "insert", Value: "test value 1"}},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(tt *testing.T) {
			collector, err := New(conf)
			require.NoError(tt, err, "NewCollector should not return an error with valid config")

			// set up Actions
			conf.Collector.Processors.Attribute = &config.Attribute{Actions: test.setupActions}

			reloadRequired := collector.updateAttributeActions(test.actions)
			assert.Equal(tt,
				test.expectedAttribs,
				conf.Collector.Processors.Attribute.Actions)
			assert.Equal(tt, test.expectedReloadRequired, reloadRequired)
		})
	}
}
