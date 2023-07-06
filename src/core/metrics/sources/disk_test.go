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
	"sync"
	"testing"

	"github.com/nginx/agent/v2/src/core/metrics"

	"github.com/stretchr/testify/assert"
)

func TestNewDiskSource(t *testing.T) {
	namespace := "test"
	actual := NewDiskSource(namespace)

	assert.Equal(t, "disk", actual.group)
	assert.Equal(t, namespace, actual.namespace)
	assert.Greater(t, len(actual.disks), 1)
}

func TestDiskCollect(t *testing.T) {
	namespace := "test"
	disk := NewDiskSource(namespace)

	ctx := context.TODO()
	wg := &sync.WaitGroup{}
	wg.Add(1)
	channel := make(chan *metrics.StatsEntityWrapper, 100)
	disk.Collect(ctx, wg, channel)
	wg.Wait()

	actual := <-channel

	actualMetricNames := []string{}
	for _, simpleMetric := range actual.Data.Simplemetrics {
		actualMetricNames = append(actualMetricNames, simpleMetric.Name)
	}
	sort.Strings(actualMetricNames)
	expected := []string{"test.disk.free", "test.disk.in_use", "test.disk.total", "test.disk.used"}

	assert.Equal(t, expected, actualMetricNames)
}
