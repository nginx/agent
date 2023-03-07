/**
 * Copyright (c) F5, Inc.
 *
 * This source code is licensed under the Apache License, Version 2.0 license found in the
 * LICENSE file in the root directory of this source tree.
 */

package sources

import (
	"context"
	"path"
	"runtime"
	"sort"
	"sync"
	"testing"

	"github.com/nginx/agent/sdk/v2/proto"
	cgroup "github.com/nginx/agent/v2/src/core/metrics/sources/cgroup"
	tutils "github.com/nginx/agent/v2/test/utils"

	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/stretchr/testify/assert"
)

func TestNewCPUTimesSource(t *testing.T) {
	namespace := "cpu"
	tests := []struct {
		name        string
		isContainer bool
		expected    *CPUTimes
	}{
		{
			"VM",
			false,
			&CPUTimes{&namedMetric{namespace, CpuGroup}, false, nil, NewMetricSourceLogger(), cpu.Times},
		},
		{
			"container",
			true,
			&CPUTimes{&namedMetric{namespace, CpuGroup}, true, cgroup.NewCgroupCPUSource(cgroup.CgroupBasePath), NewMetricSourceLogger(), nil},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(tt *testing.T) {
			env := tutils.GetMockEnv()
			env.Mock.On("IsContainer").Return(test.isContainer)

			actual := NewCPUTimesSource(namespace, env)
			assert.Equal(tt, test.expected.group, actual.group)
			assert.Equal(tt, test.expected.namespace, actual.namespace)
			assert.Equal(tt, test.expected.isDocker, actual.isDocker)
		})
	}
}

func TestCPUTimeDiff(t *testing.T) {

	tests := []struct {
		name             string
		lastTime         cpu.TimesStat
		currentTime      cpu.TimesStat
		expectedTimeDiff cpu.TimesStat
	}{
		{
			"good data",
			cpu.TimesStat{User: 4, System: 7, Idle: 10, Nice: 17, Iowait: 27, Irq: 12, Softirq: 14, Steal: 11, Guest: 13, GuestNice: 15},
			cpu.TimesStat{User: 14, System: 10, Idle: 10, Nice: 19, Iowait: 29, Irq: 13, Softirq: 16, Steal: 18, Guest: 20, GuestNice: 25},
			cpu.TimesStat{User: 10, System: 3, Idle: 0, Nice: 2, Iowait: 2, Irq: 1, Softirq: 2, Steal: 7, Guest: 7, GuestNice: 10},
		},
		{
			"bad data",
			cpu.TimesStat{},
			cpu.TimesStat{},
			cpu.TimesStat{User: 0, System: 0, Idle: 0, Nice: 0, Iowait: 0, Irq: 0, Softirq: 0, Steal: 0, Guest: 0, GuestNice: 0},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(tt *testing.T) {
			timediff := diffTimeStat(test.lastTime, test.currentTime)

			assert.Equal(tt, test.expectedTimeDiff, timediff)

		})
	}

}

func TestCPUTimesCollect_VM(t *testing.T) {
	env := tutils.GetMockEnv()
	env.Mock.On("IsContainer").Return(false)

	cpuTimes := NewCPUTimesSource("test", env)
	cpuTimes.timesFunc = func(b bool) ([]cpu.TimesStat, error) {
		return []cpu.TimesStat{
			{
				User:   4,
				System: 7,
				Idle:   10,
				Nice:   17,
				Iowait: 27,
				Steal:  11,
			},
		}, nil
	}

	ctx := context.TODO()
	wg := &sync.WaitGroup{}
	wg.Add(1)
	channel := make(chan *proto.StatsEntity, 1)
	cpuTimes.Collect(ctx, wg, channel)
	wg.Wait()

	actual := <-channel

	actualMetricNames := []string{}
	for _, simpleMetric := range actual.Simplemetrics {
		actualMetricNames = append(actualMetricNames, simpleMetric.Name)
		switch simpleMetric.Name {
		case "test.cpu.idle":
			assert.Equal(t, 13.157894736842104, simpleMetric.Value)
		case "test.cpu.iowait":
			assert.Equal(t, 35.526315789473685, simpleMetric.Value)
		case "test.cpu.system":
			assert.Equal(t, 9.210526315789473, simpleMetric.Value)
		case "test.cpu.user":
			assert.Equal(t, 27.631578947368425, simpleMetric.Value)
		case "test.cpu.stolen":
			assert.Equal(t, 14.473684210526317, simpleMetric.Value)
		}
	}
	sort.Strings(actualMetricNames)
	expected := []string{"test.cpu.idle", "test.cpu.iowait", "test.cpu.stolen", "test.cpu.system", "test.cpu.user"}

	assert.Equal(t, expected, actualMetricNames)

	cpuTimes.timesFunc = func(b bool) ([]cpu.TimesStat, error) {
		return []cpu.TimesStat{
			{
				User:   21,
				System: 67,
				Idle:   90,
				Nice:   67,
				Iowait: 97,
				Steal:  55,
			},
		}, nil
	}

	ctx = context.TODO()
	wg = &sync.WaitGroup{}
	wg.Add(1)
	channel = make(chan *proto.StatsEntity, 1)
	cpuTimes.Collect(ctx, wg, channel)
	wg.Wait()

	actual = <-channel

	actualMetricNames = []string{}
	for _, simpleMetric := range actual.Simplemetrics {
		actualMetricNames = append(actualMetricNames, simpleMetric.Name)
		switch simpleMetric.Name {
		case "test.cpu.idle":
			assert.Equal(t, 24.922118380062305, simpleMetric.Value)
		case "test.cpu.iowait":
			assert.Equal(t, 21.806853582554517, simpleMetric.Value)
		case "test.cpu.system":
			assert.Equal(t, 18.69158878504673, simpleMetric.Value)
		case "test.cpu.user":
			assert.Equal(t, 20.87227414330218, simpleMetric.Value)
		case "test.cpu.stolen":
			assert.Equal(t, 13.707165109034266, simpleMetric.Value)

		}
	}
	sort.Strings(actualMetricNames)
	expected = []string{"test.cpu.idle", "test.cpu.iowait", "test.cpu.stolen", "test.cpu.system", "test.cpu.user"}

	assert.Equal(t, expected, actualMetricNames)
}

func TestCPUTimesCollect_Container(t *testing.T) {
	_, filename, _, _ := runtime.Caller(0)
	localDirectory := path.Dir(filename)

	env := tutils.GetMockEnv()
	env.Mock.On("IsContainer").Return(true)

	cpuTimes := NewCPUTimesSource("test", env)
	cpuTimes.cgroupCPUSource = cgroup.NewCgroupCPUSource(path.Join(localDirectory, "/testdata/good_data/v1/"))
	cgroup.CpuStatsPath = localDirectory + "/testdata/proc/stat"

	ctx := context.TODO()
	wg := &sync.WaitGroup{}
	wg.Add(1)
	channel := make(chan *proto.StatsEntity, 1)
	cpuTimes.Collect(ctx, wg, channel)
	wg.Wait()
	actual := <-channel

	actualMetricNames := []string{}
	for _, simpleMetric := range actual.Simplemetrics {
		actualMetricNames = append(actualMetricNames, simpleMetric.Name)
	}
	sort.Strings(actualMetricNames)
	expected := []string{"test.cpu.system", "test.cpu.user"}

	assert.Equal(t, expected, actualMetricNames)
}
