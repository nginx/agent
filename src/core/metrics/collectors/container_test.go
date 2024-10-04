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

func TestNewContainerCollector(t *testing.T) {
	expectedSourceTypes := []string{"*sources.ContainerCPU", "*sources.ContainerMemory"}
	expectedDimensions := &metrics.CommonDim{
		Hostname:     "test-host",
		InstanceTags: "locally-tagged,tagged-locally",
	}

	_, _, cleanupFunc, err := tutils.CreateTestAgentConfigEnv()
	if err != nil {
		t.Fatal(err.Error())
	}
	defer cleanupFunc()

	env := tutils.GetMockEnv()

	containerCollector := NewContainerCollector(env, &config.Config{Tags: tutils.InitialConfTags})

	sourceTypes := []string{}
	for _, containerSource := range containerCollector.sources {
		sourceTypes = append(sourceTypes, reflect.TypeOf(containerSource).String())
	}

	assert.Equal(t, len(expectedSourceTypes), len(containerCollector.sources))
	assert.Equal(t, expectedSourceTypes, sourceTypes)
	assert.Equal(t, expectedDimensions, containerCollector.dim)
}

func TestContainerCollector_Collect(t *testing.T) {
	mockSource1 := GetNginxSourceMock()
	mockSource2 := GetNginxSourceMock()

	containerCollector := &ContainerCollector{
		sources: []metrics.Source{
			mockSource1,
			mockSource2,
		},
		buf: make(chan *metrics.StatsEntityWrapper),
		dim: &metrics.CommonDim{},
	}

	ctx := context.TODO()
	channel := make(chan *metrics.StatsEntityWrapper)
	go containerCollector.Collect(ctx, channel)

	containerCollector.buf <- &metrics.StatsEntityWrapper{Type: proto.MetricsReport_SYSTEM, Data: &proto.StatsEntity{Dimensions: []*proto.Dimension{{Name: "new_dim", Value: "123"}}}}
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

func TestContainerCollector_UpdateConfig(t *testing.T) {
	env := tutils.GetMockEnv()

	containerCollector := &ContainerCollector{
		env: env,
		dim: &metrics.CommonDim{},
	}

	assert.Equal(t, "", containerCollector.dim.InstanceTags)

	containerCollector.UpdateConfig(&config.Config{Tags: []string{"new-tag1", "new-tag-2"}})

	assert.Equal(t, "new-tag1,new-tag-2", containerCollector.dim.InstanceTags)
}
