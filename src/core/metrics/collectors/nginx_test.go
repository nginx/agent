/**
 * Copyright (c) F5, Inc.
 *
 * This source code is licensed under the Apache License, Version 2.0 license found in the
 * LICENSE file in the root directory of this source tree.
 */

package collectors

import (
	"context"
	"reflect"
	"sync"
	"testing"

	"github.com/nginx/agent/sdk/v2/proto"
	"github.com/nginx/agent/v2/src/core"
	"github.com/nginx/agent/v2/src/core/config"
	"github.com/nginx/agent/v2/src/core/metrics"
	tutils "github.com/nginx/agent/v2/test/utils"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

var (
	nginxId  = "223344"
	nginxPid = "12345"

	detailsMap = map[string]*proto.NginxDetails{
		nginxPid: {
			ProcessPath: "/path/to/nginx",
			NginxId:     nginxId,
			Plus: &proto.NginxPlusMetaData{
				Enabled: true,
			},
		},
	}

	configuration = &config.Config{
		ClientID: "456789",
		Tags:     tutils.InitialConfTags,
		AgentMetrics: config.AgentMetrics{
			BulkSize:           100,
			ReportInterval:     10,
			CollectionInterval: 1,
			Mode:               "aggregated",
		},
		Features: config.Defaults.Features,
		Nginx: config.Nginx{
			Debug:                 false,
			NginxCountingSocket:   "unix:/var/run/nginx-agent/nginx.sock",
			NginxClientVersion:    9,
			TreatWarningsAsErrors: false,
		},
	}

	collectorConfigNoApi = &metrics.NginxCollectorConfig{
		BinPath:            "/path/to/nginx",
		NginxId:            nginxId,
		CollectionInterval: 1,
		AccessLogs:         []string{},
		ErrorLogs:          []string{},
	}

	collectorConfigStubStatusApi = &metrics.NginxCollectorConfig{
		BinPath:            "/path/to/nginx",
		NginxId:            nginxPid,
		CollectionInterval: 1,
		StubStatus:         "http://localhost:80/stub_status",
		AccessLogs:         []string{},
		ErrorLogs:          []string{},
	}

	collectorConfigPlusApi = &metrics.NginxCollectorConfig{
		BinPath:            "/path/to/nginx",
		NginxId:            nginxPid,
		CollectionInterval: 1,
		PlusAPI:            "http://localhost:80/api",
		AccessLogs:         []string{},
		ErrorLogs:          []string{},
	}
)

func TestNewNginxCollector(t *testing.T) {
	testCases := []struct {
		testName                string
		config                  *config.Config
		collectorConfig         *metrics.NginxCollectorConfig
		expectedSourceTypes     []string
		expectedCollectorConfig *metrics.NginxCollectorConfig
		expectedDimensions      *metrics.CommonDim
	}{
		{
			testName:                "NoNginxApiConfigured",
			config:                  configuration,
			collectorConfig:         collectorConfigNoApi,
			expectedSourceTypes:     []string{"*sources.NginxProcess", "*sources.NginxWorker", "*sources.NginxStatic"},
			expectedCollectorConfig: collectorConfigNoApi,
			expectedDimensions: &metrics.CommonDim{
				Hostname:            "test-host",
				InstanceTags:        "locally-tagged,tagged-locally",
				NginxId:             nginxId,
				NginxAccessLogPaths: []string{},
			},
		},
		{
			testName:                "StubStatusApiConfigured",
			config:                  configuration,
			collectorConfig:         collectorConfigStubStatusApi,
			expectedSourceTypes:     []string{"*sources.NginxProcess", "*sources.NginxWorker", "*sources.NginxOSS", "*sources.NginxAccessLog", "*sources.NginxErrorLog"},
			expectedCollectorConfig: collectorConfigStubStatusApi,
			expectedDimensions: &metrics.CommonDim{
				Hostname:            "test-host",
				InstanceTags:        "locally-tagged,tagged-locally",
				NginxId:             nginxPid,
				NginxAccessLogPaths: []string{},
			},
		},
		{
			testName:                "PlusApiConfigured",
			config:                  configuration,
			collectorConfig:         collectorConfigPlusApi,
			expectedSourceTypes:     []string{"*sources.NginxProcess", "*sources.NginxWorker", "*sources.NginxPlus", "*sources.NginxAccessLog", "*sources.NginxErrorLog"},
			expectedCollectorConfig: collectorConfigPlusApi,
			expectedDimensions: &metrics.CommonDim{
				Hostname:            "test-host",
				InstanceTags:        "locally-tagged,tagged-locally",
				NginxId:             nginxPid,
				NginxAccessLogPaths: []string{},
			},
		},
	}

	binary := tutils.NewMockNginxBinary()
	binary.On("GetAccessLogs").Return(map[string]string{})
	binary.On("GetErrorLogs").Return(map[string]string{})
	binary.On("GetNginxDetailsFromProcess", &core.Process{Name: nginxPid, IsMaster: true}).Return(detailsMap[nginxPid])

	for _, tc := range testCases {
		t.Run(tc.testName, func(t *testing.T) {
			_, _, cleanupFunc, err := tutils.CreateTestAgentConfigEnv()
			if err != nil {
				t.Fatalf(err.Error())
			}
			defer cleanupFunc()

			env := tutils.GetMockEnvWithHostAndProcess()
			nginxCollector := NewNginxCollector(tc.config, env, tc.collectorConfig, binary)

			sourceTypes := []string{}
			for _, nginxSource := range nginxCollector.sources {
				sourceTypes = append(sourceTypes, reflect.TypeOf(nginxSource).String())
			}

			assert.Equal(t, len(tc.expectedSourceTypes), len(nginxCollector.sources))
			assert.Equal(t, tc.expectedCollectorConfig, nginxCollector.collectorConf)
			assert.Equal(t, tc.expectedSourceTypes, sourceTypes)
			assert.Equal(t, tc.expectedDimensions, nginxCollector.dimensions)
		})
	}
}

func TestNginxCollector_Collect(t *testing.T) {
	mockNginxSource1 := GetNginxSourceMock()
	mockNginxSource2 := GetNginxSourceMock()

	nginxCollector := &NginxCollector{
		sources: []metrics.NginxSource{
			mockNginxSource1,
			mockNginxSource2,
		},
	}

	ctx := context.TODO()
	wg := &sync.WaitGroup{}
	wg.Add(1)
	go nginxCollector.Collect(ctx, wg, make(chan<- *metrics.StatsEntityWrapper))
	wg.Wait()

	mockNginxSource1.AssertExpectations(t)
	mockNginxSource2.AssertExpectations(t)
}

func TestNginxCollector_UpdateConfig(t *testing.T) {
	mockNginxSource1 := new(NginxSourceMock)
	mockNginxSource1.On("Update", mock.Anything, mock.Anything).Once()

	mockNginxSource2 := new(NginxSourceMock)
	mockNginxSource2.On("Update", mock.Anything, mock.Anything).Once()

	env := tutils.GetMockEnv()

	nginxCollector := &NginxCollector{
		sources: []metrics.NginxSource{
			mockNginxSource1,
			mockNginxSource2,
		},
		collectorConf: &metrics.NginxCollectorConfig{
			NginxId: "123",
		},
		env: env,
	}

	nginxCollector.UpdateConfig(&config.Config{})

	mockNginxSource1.AssertExpectations(t)
	mockNginxSource2.AssertExpectations(t)
}

func TestNginxCollector_UpdateCollectorConfig(t *testing.T) {
	mockNginxSource1 := new(NginxSourceMock)
	mockNginxSource1.On("Stop").Once()

	mockNginxSource2 := new(NginxSourceMock)
	mockNginxSource2.On("Stop").Once()

	env := tutils.GetMockEnv()

	binary := tutils.NewMockNginxBinary()
	binary.On("GetAccessLogs").Return(map[string]string{})
	binary.On("GetErrorLogs").Return(map[string]string{})
	binary.On("GetNginxDetailsFromProcess", &core.Process{Name: nginxPid, IsMaster: true}).Return(detailsMap[nginxPid])

	host := env.NewHostInfo("agentVersion", &tutils.InitialConfTags, "/etc/nginx/", false)

	nginxCollector := &NginxCollector{
		sources: []metrics.NginxSource{
			mockNginxSource1,
			mockNginxSource2,
		},
		collectorConf: &metrics.NginxCollectorConfig{
			NginxId: "123",
		},
		env:        env,
		binary:     binary,
		dimensions: metrics.NewCommonDim(host, &config.Config{}, "123"),
	}

	nginxCollector.UpdateCollectorConfig(&metrics.NginxCollectorConfig{StubStatus: "http://localhost:80/api"}, configuration, env)

	// Verify that sources are stopped
	mockNginxSource1.AssertExpectations(t)
	mockNginxSource2.AssertExpectations(t)

	sourceTypes := []string{}
	for _, nginxSource := range nginxCollector.sources {
		sourceTypes = append(sourceTypes, reflect.TypeOf(nginxSource).String())
	}

	// Verify that new sources are created
	assert.Equal(t, 5, len(nginxCollector.sources))
	assert.Equal(t, []string{"*sources.NginxProcess", "*sources.NginxWorker", "*sources.NginxOSS", "*sources.NginxAccessLog", "*sources.NginxErrorLog"}, sourceTypes)
}
