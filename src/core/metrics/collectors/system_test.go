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
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/nginx/agent/sdk/v2/proto"
	"github.com/nginx/agent/v2/src/core/config"
	"github.com/nginx/agent/v2/src/core/metrics"
	tutils "github.com/nginx/agent/v2/test/utils"
)

func TestNewSystemCollector(t *testing.T) {
	testCases := []struct {
		testName            string
		isContainer         bool
		expectedSourceTypes []string
		expectedDimensions  *metrics.CommonDim
	}{
		{
			testName:    "VM",
			isContainer: false,
			expectedSourceTypes: []string{
				"*sources.VirtualMemory",
				"*sources.CPUTimes",
				"*sources.Disk",
				"*sources.DiskIO",
				"*sources.NetIO",
				"*sources.Load",
				"*sources.Swap",
			},
			expectedDimensions: &metrics.CommonDim{
				Hostname:     "test-host",
				InstanceTags: "locally-tagged,tagged-locally",
			},
		},
		{
			testName:    "Container",
			isContainer: true,
			expectedSourceTypes: []string{
				"*sources.VirtualMemory",
				"*sources.CPUTimes",
				"*sources.NetIO",
				"*sources.Swap",
			},
			expectedDimensions: &metrics.CommonDim{
				Hostname:     "test-host",
				InstanceTags: "locally-tagged,tagged-locally",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.testName, func(t *testing.T) {
			env := tutils.GetMockEnv()
			env.Mock.On("IsContainer").Return(tc.isContainer)

			systemCollector := NewSystemCollector(env, &config.Config{Tags: tutils.InitialConfTags})

			sourceTypes := []string{}
			for _, containerSource := range systemCollector.sources {
				sourceTypes = append(sourceTypes, reflect.TypeOf(containerSource).String())
			}

			assert.Equal(t, len(tc.expectedSourceTypes), len(systemCollector.sources))
			assert.Equal(t, tc.expectedSourceTypes, sourceTypes)
			assert.Equal(t, tc.expectedDimensions, systemCollector.dim)
		})
	}
}

func TestSystemCollector_Collect(t *testing.T) {
	mockSource1 := GetNginxSourceMock()
	mockSource2 := GetNginxSourceMock()

	systemCollector := &SystemCollector{
		sources: []metrics.Source{
			mockSource1,
			mockSource2,
		},
		buf: make(chan *metrics.StatsEntityWrapper),
		dim: &metrics.CommonDim{},
	}

	ctx := context.TODO()

	channel := make(chan *metrics.StatsEntityWrapper)
	go systemCollector.Collect(ctx, channel)

	systemCollector.buf <- &metrics.StatsEntityWrapper{Type: proto.MetricsReport_SYSTEM, Data: &proto.StatsEntity{Dimensions: []*proto.Dimension{{Name: "new_dim", Value: "123"}}}}
	actual := <-channel

	time.Sleep(100 * time.Millisecond)

	mockSource1.AssertExpectations(t)
	mockSource2.AssertExpectations(t)

	expectedDimensions := []*proto.Dimension{
		{Name: "system_id", Value: ""},
		{Name: "hostname", Value: ""},
		{Name: "system.tags", Value: ""},
		{Name: "instance_group", Value: ""},
		{Name: "display_name", Value: ""},
		{Name: "nginx_id", Value: ""},
		{Name: "new_dim", Value: "123"},
	}
	assert.Equal(t, expectedDimensions, actual.Data.Dimensions)
}

func TestSystemCollector_UpdateConfig(t *testing.T) {
	env := tutils.GetMockEnv()

	systemCollector := &SystemCollector{
		env: env,
		dim: &metrics.CommonDim{},
	}

	assert.Equal(t, "", systemCollector.dim.InstanceTags)

	systemCollector.UpdateConfig(&config.Config{Tags: []string{"new-tag1", "new-tag-2"}})

	assert.Equal(t, "new-tag1,new-tag-2", systemCollector.dim.InstanceTags)
}
