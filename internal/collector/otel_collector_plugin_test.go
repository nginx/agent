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
					PlusAPI: &model.APIDetails{
						URL:      fmt.Sprintf("%s/api", nginxPlusMock.URL),
						Location: "",
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
				NginxPlusReceivers: []config.NginxPlusReceiver{
					{
						InstanceID: "123",
						PlusAPI: config.APIDetails{
							URL:      fmt.Sprintf("%s/api", nginxPlusMock.URL),
							Location: "",
						},
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
					StubStatus: &model.APIDetails{
						URL:      "http://test.com:8080/stub_status",
						Location: "",
					},
					PlusAPI: &model.APIDetails{
						URL:      "",
						Location: "",
					},
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
						StubStatus: config.APIDetails{
							URL:      "http://test.com:8080/stub_status",
							Location: "",
						},
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

			ctx := context.Background()
			messagePipe := bus.NewMessagePipe(10)
			err = messagePipe.Register(10, []bus.Plugin{collector})

			require.NoError(tt, err)
			require.NoError(tt, collector.Init(ctx, messagePipe), "Init should not return an error")
			defer collector.Close(ctx)

			assert.Eventually(
				tt,
				func() bool {
					tt.Logf("Collector state is %+v", collector.service.GetState())
					return collector.service.GetState() == otelcol.StateRunning
				},
				5*time.Second,
				100*time.Millisecond,
			)

			collector.Process(ctx, test.message)

			assert.Eventually(
				tt,
				func() bool {
					tt.Logf("Collector state is %+v", collector.service.GetState())
					return collector.service.GetState() == otelcol.StateRunning
				},
				5*time.Second,
				100*time.Millisecond,
			)

			assert.Equal(tt, test.processors, collector.config.Collector.Processors)
			assert.Equal(tt, test.headers, collector.config.Collector.Extensions.HeadersSetter.Headers)
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

			ctx := context.Background()
			messagePipe := bus.NewMessagePipe(10)
			err = messagePipe.Register(10, []bus.Plugin{collector})

			require.NoError(tt, err)
			require.NoError(tt, collector.Init(ctx, messagePipe), "Init should not return an error")
			defer collector.Close(ctx)

			assert.Eventually(
				tt,
				func() bool {
					tt.Logf("Collector state is %+v", collector.service.GetState())
					return collector.service.GetState() == otelcol.StateRunning
				},
				5*time.Second,
				100*time.Millisecond,
			)

			collector.Process(ctx, test.message)

			assert.Eventually(
				tt,
				func() bool {
					tt.Logf("Collector state is %+v", collector.service.GetState())
					return collector.service.GetState() == otelcol.StateRunning
				},
				5*time.Second,
				100*time.Millisecond,
			)

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
				StubStatus: &model.APIDetails{
					URL:      "http://new-test-host:8080/api",
					Location: "",
				},
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
						StubStatus: config.APIDetails{
							URL:      "http://test.com:8080/api",
							Location: "",
						},
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
						StubStatus: config.APIDetails{
							URL:      "http://new-test-host:8080/api",
							Location: "",
						},
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
				StubStatus: &model.APIDetails{},
			},
			existingReceivers: config.Receivers{
				NginxReceivers: []config.NginxReceiver{
					{
						InstanceID: "123",
						StubStatus: config.APIDetails{
							URL:      "http://test.com:8080/api",
							Location: "",
						},
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
				PlusAPI: &model.APIDetails{
					URL:      "http://new-test-host:8080/api",
					Location: "",
				},
			},
			existingReceivers: config.Receivers{
				NginxPlusReceivers: []config.NginxPlusReceiver{
					{
						InstanceID: "123",
						PlusAPI: config.APIDetails{
							URL:      "http://test.com:8080/api",
							Location: "",
						},
					},
				},
			},
			expectedReceivers: config.Receivers{
				NginxPlusReceivers: []config.NginxPlusReceiver{
					{
						InstanceID: "123",
						PlusAPI: config.APIDetails{
							URL:      "http://new-test-host:8080/api",
							Location: "",
						},
					},
				},
			},
		},
		{
			name: "Test 2: Removing NGINX Plus Receiver",
			nginxConfigContext: &model.NginxConfigContext{
				InstanceID: "123",
				PlusAPI: &model.APIDetails{
					URL:      "",
					Location: "",
				},
			},
			existingReceivers: config.Receivers{
				NginxPlusReceivers: []config.NginxPlusReceiver{
					{
						InstanceID: "123",
						PlusAPI: config.APIDetails{
							URL:      "http://test.com:8080/api",
							Location: "",
						},
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
