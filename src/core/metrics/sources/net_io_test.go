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

	tutils "github.com/nginx/agent/v2/test/utils"
	"github.com/shirou/gopsutil/v3/net"
	"github.com/stretchr/testify/assert"
)

func TestNewNetIOSource(t *testing.T) {
	namespace := "test"
	actual := NewNetIOSource(namespace, tutils.NewMockEnvironment())

	assert.Equal(t, "net", actual.group)
	assert.Equal(t, namespace, actual.namespace)
}

func TestNetIOCollect(t *testing.T) {
	namespace := "test"
	env := tutils.NewMockEnvironment()
	env.On("GetNetOverflow").Return(0.0, nil)

	nioSource := NewNetIOSource(namespace, env)
	nioSource.netIOCountersFunc = func(ctx context.Context, pernic bool) ([]net.IOCountersStat, error) {
		return []net.IOCountersStat{
			{Name: "eth0"},
			{Name: "lo"},
		}, nil
	}
	nioSource.netIOInterfacesFunc = func(ctx context.Context) (net.InterfaceStatList, error) {
		return net.InterfaceStatList{
			{Name: "eth0", Flags: []string{"up"}},
			{Name: "eth1", Flags: []string{"down"}},
			{Name: "lo", Flags: []string{"up"}},
		}, nil
	}

	ctx := context.TODO()
	// wg := &sync.WaitGroup{}
	// wg.Add(1)
	channel := make(chan *metrics.StatsEntityWrapper, 100)
	nioSource.Collect(ctx, nil, channel)
	// wg.Wait()

	actual := <-channel

	actualMetricNames := []string{}
	for _, simpleMetric := range actual.Data.Simplemetrics {
		actualMetricNames = append(actualMetricNames, simpleMetric.Name)
	}
	sort.Strings(actualMetricNames)

	expected := []string{
		"test.net.bytes_rcvd",
		"test.net.bytes_sent",
		"test.net.drops_in.count",
		"test.net.drops_out.count",
		"test.net.packets_in.count",
		"test.net.packets_in.error",
		"test.net.packets_out.count",
		"test.net.packets_out.error",
	}

	assert.Contains(t, nioSource.netIOStats, "lo")
	assert.Contains(t, nioSource.netIOStats, "eth0")
	assert.NotContains(t, nioSource.netIOStats, "eth1")
	assert.Equal(t, expected, actualMetricNames)
}
