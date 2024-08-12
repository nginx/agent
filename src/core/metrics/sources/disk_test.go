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

	"github.com/stretchr/testify/assert"
)

func TestNewDiskSource(t *testing.T) {
	namespace := "test"
	env := tutils.GetMockEnv()
	actual := NewDiskSource(namespace, env)

	assert.Equal(t, "disk", actual.group)
	assert.Equal(t, namespace, actual.namespace)
	assert.Equal(t, len(actual.disks), 2)
}

func TestDiskCollect(t *testing.T) {
	namespace := "test"
	env := tutils.GetMockEnv()
	disk := NewDiskSource(namespace, env)

	ctx := context.TODO()
	// wg := &sync.WaitGroup{}
	// wg.Add(1)
	channel := make(chan *metrics.StatsEntityWrapper, 100)
	disk.Collect(ctx, nil, channel)
	// wg.Wait()

	actual := <-channel

	actualMetricNames := []string{}
	for _, simpleMetric := range actual.Data.Simplemetrics {
		actualMetricNames = append(actualMetricNames, simpleMetric.Name)
	}
	sort.Strings(actualMetricNames)
	expected := []string{"test.disk.free", "test.disk.in_use", "test.disk.total", "test.disk.used"}

	assert.Equal(t, expected, actualMetricNames)
}
