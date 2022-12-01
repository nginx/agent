/**
 * Copyright (c) F5, Inc.
 *
 * This source code is licensed under the Apache License, Version 2.0 license found in the
 * LICENSE file in the root directory of this source tree.
 */

package sample

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMetricAdd(t *testing.T) {
	metric := NewMetric(1)

	metric.Add(11)
	assert.Equal(t, Metric{
		Count: 2,
		Last:  11,
		Min:   1,
		Max:   11,
		Sum:   12,
	}, metric)

	metric.Add(100)
	assert.Equal(t, Metric{
		Count: 3,
		Last:  100,
		Min:   1,
		Max:   100,
		Sum:   112,
	}, metric)
}

func TestMetricAddMetric(t *testing.T) {
	metric := NewMetric(100)

	metric.AddMetric(Metric{
		Count: 1,
		Last:  11,
		Min:   11,
		Max:   11,
		Sum:   11,
	})
	assert.Equal(t, Metric{
		Count: 2,
		Last:  11,
		Min:   11,
		Max:   100,
		Sum:   111,
	}, metric)

	metric.AddMetric(Metric{
		Count: 1,
		Last:  200,
		Min:   200,
		Max:   200,
		Sum:   200,
	})
	assert.Equal(t, Metric{
		Count: 3,
		Last:  200,
		Min:   11,
		Max:   200,
		Sum:   311,
	}, metric)
}
