/**
 * Copyright (c) F5, Inc.
 *
 * This source code is licensed under the Apache License, Version 2.0 license found in the
 * LICENSE file in the root directory of this source tree.
 */

package sources

import (
	"context"
	"sort"
	"testing"

	"github.com/nginx/agent/v2/src/core/metrics"

	cgroup "github.com/nginx/agent/v2/src/core/metrics/sources/cgroup"
	tutils "github.com/nginx/agent/v2/test/utils"
	"github.com/shirou/gopsutil/v3/mem"
	"github.com/stretchr/testify/assert"
)

func TestNewVirtualMemorySource(t *testing.T) {
	namespace := "test"
	tests := []struct {
		name        string
		isContainer bool
		expected    *VirtualMemory
	}{
		{
			"VM",
			false,
			&VirtualMemory{NewMetricSourceLogger(), &namedMetric{namespace, MemoryGroup}, mem.VirtualMemoryWithContext},
		},
		{
			"container",
			true,
			&VirtualMemory{NewMetricSourceLogger(), &namedMetric{namespace, MemoryGroup}, cgroup.NewCgroupMemSource(cgroup.CgroupBasePath).VirtualMemoryStatWithContext},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(tt *testing.T) {
			env := tutils.GetMockEnv()
			env.Mock.On("IsContainer").Return(test.isContainer)

			actual := NewVirtualMemorySource(namespace, env)
			assert.Equal(tt, test.expected.group, actual.group)
			assert.Equal(tt, test.expected.namespace, actual.namespace)
		})
	}
}

func TestVirtualMemoryCollect(t *testing.T) {
	env := tutils.NewMockEnvironment()
	env.On("IsContainer").Return(false)
	virtualMemorySource := NewVirtualMemorySource("test", env)

	ctx := context.TODO()
	// wg := &sync.WaitGroup{}
	// wg.Add(1)
	channel := make(chan *metrics.StatsEntityWrapper, 1)
	go virtualMemorySource.Collect(ctx, nil, channel)
	// wg.Wait()

	actual := <-channel

	actualMetricNames := []string{}
	for _, simpleMetric := range actual.Data.Simplemetrics {
		actualMetricNames = append(actualMetricNames, simpleMetric.Name)
	}
	sort.Strings(actualMetricNames)
	expected := []string{
		"test.mem.available",
		"test.mem.buffered",
		"test.mem.cached",
		"test.mem.free",
		"test.mem.pct_used",
		"test.mem.shared",
		"test.mem.total",
		"test.mem.used",
	}

	assert.Equal(t, expected, actualMetricNames)
}
