/**
 * Copyright (c) F5, Inc.
 *
 * This source code is licensed under the Apache License, Version 2.0 license found in the
 * LICENSE file in the root directory of this source tree.
 */

package sources

import (
	"context"
	"sync"
	"testing"

	"github.com/nginx/agent/sdk/v2/proto"
	cgroup "github.com/nginx/agent/v2/src/core/metrics/sources/cgroup"
	tutils "github.com/nginx/agent/v2/test/utils"
	"github.com/shirou/gopsutil/v3/mem"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestNewSwapSource(t *testing.T) {
	namespace := "test"
	tests := []struct {
		name        string
		isContainer bool
		expected    *Swap
	}{
		{
			"VM",
			false,
			&Swap{NewMetricSourceLogger(), &namedMetric{namespace, "swap"}, mem.SwapMemory},
		},
		{
			"container",
			true,
			&Swap{NewMetricSourceLogger(), &namedMetric{namespace, "swap"}, cgroup.NewCgroupSwapSource(cgroup.CgroupBasePath).SwapMemoryStat},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(tt *testing.T) {
			env := tutils.NewMockEnvironment()

			env.Mock.On("NewHostInfo", mock.Anything, mock.Anything, mock.Anything).Return(&proto.HostInfo{
				Hostname: "test-host",
			})

			env.Mock.On("IsContainer").Return(test.isContainer)

			actual := NewSwapSource(namespace, env)
			assert.Equal(tt, test.expected.group, actual.group)
			assert.Equal(tt, test.expected.namespace, actual.namespace)
		})
	}
}

func TestSwapSource_Collect(t *testing.T) {
	expectedMetrics := map[string]float64{
		"swap.used":     30,
		"swap.total":    100,
		"swap.free":     70,
		"swap.pct_free": 70,
	}

	tests := []struct {
		name string
		m    chan *proto.StatsEntity
	}{
		{
			"basic swap test",
			make(chan *proto.StatsEntity, 1),
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(tt *testing.T) {
			env := tutils.NewMockEnvironment()
			env.On("IsContainer").Return(false)
			ctx := context.TODO()

			c := NewSwapSource("", env)
			c.statFunc = func() (*mem.SwapMemoryStat, error) {
				return &mem.SwapMemoryStat{
					Total:       100,
					Used:        30,
					Free:        70,
					UsedPercent: 30,
				}, nil
			}
			wg := &sync.WaitGroup{}
			wg.Add(1)
			go c.Collect(ctx, wg, test.m)
			wg.Wait()

			statsEntity := <-test.m
			assert.Len(tt, statsEntity.Simplemetrics, len(expectedMetrics))
			for _, metric := range statsEntity.Simplemetrics {
				assert.Contains(tt, expectedMetrics, metric.Name)
				assert.Equal(t, expectedMetrics[metric.Name], metric.Value)
			}
		})
	}
}
