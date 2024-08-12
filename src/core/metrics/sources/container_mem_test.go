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
	"testing"
	"time"

	"github.com/nginx/agent/v2/src/core/metrics"

	"github.com/nginx/agent/sdk/v2/proto"
	"github.com/stretchr/testify/assert"
)

func TestContainerMemorySource(t *testing.T) {
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
					Name:  "container.mem.oom",
					Value: 1,
				},
				{
					Name:  "container.mem.oom.kill",
					Value: 5,
				},
			},
		}},
		{"v1 bad data", localDirectory + "/testdata/bad_data/v1/", nil},
		{"v2 good data", localDirectory + "/testdata/good_data/v2/", &proto.StatsEntity{
			Simplemetrics: []*proto.SimpleMetric{
				{
					Name:  "container.mem.oom",
					Value: 1,
				},
				{
					Name:  "container.mem.oom.kill",
					Value: 3,
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
			// wg := &sync.WaitGroup{}
			// wg.Add(1)

			containerMemorySource := NewContainerMemorySource("container", test.basePath)
			go containerMemorySource.Collect(ctx, nil, actual)
			// wg.Wait()

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
