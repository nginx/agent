/**
 * Copyright (c) F5, Inc.
 *
 * This source code is licensed under the Apache License, Version 2.0 license found in the
 * LICENSE file in the root directory of this source tree.
 */

package sources

import (
	"context"
	"github.com/nginx/agent/v2/src/core/metrics"
	"sort"
	"testing"

	"github.com/nginx/agent/sdk/v2/proto"
	"github.com/stretchr/testify/assert"
)

func TestNamedMetricLabel(t *testing.T) {
	tests := []struct {
		name        string
		namedMetric namedMetric
		input       string
		expected    string
	}{
		{
			"empty input",
			namedMetric{namespace: "core", group: "test"},
			"",
			"",
		},
		{
			"join input with namespace and group",
			namedMetric{namespace: "core", group: "test"},
			"metric",
			"core.test.metric",
		},
		{
			"join input with namespace only",
			namedMetric{namespace: "core", group: ""},
			"metric",
			"core.metric",
		},
		{
			"join input with group only",
			namedMetric{namespace: "", group: "test"},
			"metric",
			"test.metric",
		},
		{
			"no namespace or group",
			namedMetric{namespace: "", group: ""},
			"metric",
			"metric",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(tt *testing.T) {
			actual := test.namedMetric.label(test.input)
			assert.Equal(tt, test.expected, actual)
		})
	}
}

func TestNamedMetricConvertSamplesToSimpleMetrics(t *testing.T) {
	underTest := namedMetric{namespace: "core", group: "test"}
	expected := []*proto.SimpleMetric{{Name: "core.test.metric1", Value: 2.3}, {Name: "core.test.metric2", Value: 8.2}}
	actual := underTest.convertSamplesToSimpleMetrics(map[string]float64{
		"metric1": 2.3,
		"metric2": 8.2,
	})

	sort.Slice(actual, func(i, j int) bool {
		return actual[i].Name < actual[j].Name
	})

	assert.Equal(t, 2, len(actual))
	assert.Equal(t, expected[0].Name, actual[0].Name)
	assert.Equal(t, expected[0].Value, actual[0].Value)
	assert.Equal(t, expected[1].Name, actual[1].Name)
	assert.Equal(t, expected[1].Value, actual[1].Value)
}

func TestCommonNewFloatMetric(t *testing.T) {
	expected := &proto.SimpleMetric{Name: "metric", Value: 2.3}
	actual := newFloatMetric("metric", 2.3)
	assert.Equal(t, expected, actual)
}

func TestCommonDelta(t *testing.T) {
	tests := []struct {
		name     string
		current  map[string]map[string]float64
		previous map[string]map[string]float64
		expected map[string]map[string]float64
	}{
		{
			name: "delta",
			current: map[string]map[string]float64{
				"interface1": {
					"bytes_sent": 32.32,
					"bytes_rcvd": 324,
				},
				"interface2": {
					"bytes_sent": 22,
					"bytes_rcvd": 666,
				},
			},
			previous: map[string]map[string]float64{
				"interface1": {
					"bytes_sent": 66,
					"bytes_rcvd": 300,
				},
				"interface3": {
					"bytes_sent": 33,
					"bytes_rcvd": 999,
				},
			},
			expected: map[string]map[string]float64{
				"interface1": {
					"bytes_sent": -33.68,
					"bytes_rcvd": 24,
				},
				"interface2": {
					"bytes_sent": 22,
					"bytes_rcvd": 666,
				},
				"interface3": {
					"bytes_sent": -33,
					"bytes_rcvd": -999,
				},
			},
		},
		{
			name:     "empty current & previous",
			current:  map[string]map[string]float64{},
			previous: map[string]map[string]float64{},
			expected: map[string]map[string]float64{},
		},
		{
			name:    "empty current",
			current: map[string]map[string]float64{},
			previous: map[string]map[string]float64{
				"interface1": {
					"bytes_sent": 32.32,
					"bytes_rcvd": 324.1,
				},
			},
			expected: map[string]map[string]float64{
				"interface1": {
					"bytes_sent": -32.32,
					"bytes_rcvd": -324.1,
				},
			},
		},
		{
			name: "empty previous",
			current: map[string]map[string]float64{
				"interface1": {
					"bytes_sent": 32.32,
					"bytes_rcvd": 324.1,
				},
			},
			previous: map[string]map[string]float64{},
			expected: map[string]map[string]float64{
				"interface1": {
					"bytes_sent": 32.32,
					"bytes_rcvd": 324.1,
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(tt *testing.T) {
			actual := Delta(test.current, test.previous)
			assert.Equal(tt, test.expected, actual)
		})
	}
}

func TestCommonSendNginxDownStatus(t *testing.T) {
	m := make(chan *metrics.StatsEntityWrapper, 1)
	expected := &proto.SimpleMetric{Name: "nginx.status", Value: 0.0}
	SendNginxDownStatus(context.TODO(), []*proto.Dimension{}, m)
	actual := <-m
	assert.Equal(t, 1, len(actual.Data.Simplemetrics))
	assert.Equal(t, expected, actual.Data.Simplemetrics[0])
}
