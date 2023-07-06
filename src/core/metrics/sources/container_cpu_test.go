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
	"time"

	"github.com/nginx/agent/v2/src/core/metrics"

	"github.com/nginx/agent/sdk/v2/proto"
	"github.com/stretchr/testify/assert"
)

func TestContainerCPUSource(t *testing.T) {
	_, filename, _, _ := runtime.Caller(0)
	localDirectory := path.Dir(filename)

	tests := []struct {
		name     string
		basePath string
		stats    *proto.StatsEntity
	}{
		{"v1 good data", localDirectory + "/testdata/good_data/v1/", &proto.StatsEntity{
			Simplemetrics: []*proto.SimpleMetric{
				{
					Name:  "container.cpu.cores",
					Value: 2,
				},
				{
					Name:  "container.cpu.period",
					Value: 500,
				},
				{
					Name:  "container.cpu.quota",
					Value: 1000,
				},
				{
					Name:  "container.cpu.shares",
					Value: 1024,
				},
				{
					Name:  "container.cpu.set.cores",
					Value: 3,
				},
				{
					Name:  "container.cpu.throttling.time",
					Value: 300,
				},
				{
					Name:  "container.cpu.throttling.throttled",
					Value: 200,
				},
				{
					Name:  "container.cpu.throttling.periods",
					Value: 500,
				},
				{
					Name:  "container.cpu.throttling.percent",
					Value: 40,
				},
			},
		}},
		{"v1 bad data", localDirectory + "/testdata/bad_data/v1/", nil},
		{"v2 good data", localDirectory + "/testdata/good_data/v2/", &proto.StatsEntity{
			Simplemetrics: []*proto.SimpleMetric{
				{
					Name:  "container.cpu.cores",
					Value: 1.5,
				},
				{
					Name:  "container.cpu.period",
					Value: 100000,
				},
				{
					Name:  "container.cpu.quota",
					Value: 150000,
				},
				{
					Name:  "container.cpu.shares",
					Value: 2046,
				},
				{
					Name:  "container.cpu.set.cores",
					Value: 4,
				},
				{
					Name:  "container.cpu.throttling.time",
					Value: 200,
				},
				{
					Name:  "container.cpu.throttling.throttled",
					Value: 100,
				},
				{
					Name:  "container.cpu.throttling.periods",
					Value: 500,
				},
				{
					Name:  "container.cpu.throttling.percent",
					Value: 20,
				},
			},
		}},
		{"v2 bad data", localDirectory + "/testdata/bad_data/v2/", nil},
		{"no file", localDirectory + "/unknown/", nil},
	}

	for _, test := range tests {
		t.Run(test.name, func(tt *testing.T) {
			actual := make(chan *metrics.StatsEntityWrapper, 1)
			ctx := context.TODO()
			wg := &sync.WaitGroup{}
			wg.Add(1)

			containerCPUSource := NewContainerCPUSource("container", test.basePath)
			go containerCPUSource.Collect(ctx, wg, actual)
			wg.Wait()

			select {
			case result := <-actual:
				sort.SliceStable(test.stats.Simplemetrics, func(i, j int) bool {
					return test.stats.Simplemetrics[i].Name < test.stats.Simplemetrics[j].Name
				})
				sort.SliceStable(result.Data.Simplemetrics, func(i, j int) bool {
					return result.Data.Simplemetrics[i].Name < result.Data.Simplemetrics[j].Name
				})
				assert.Equal(tt, test.stats.Simplemetrics, result.Data.Simplemetrics)
			case <-time.After(10 * time.Millisecond):
				assert.Nil(tt, test.stats)
			}
		})
	}
}
