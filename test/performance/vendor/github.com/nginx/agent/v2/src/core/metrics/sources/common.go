/**
 * Copyright (c) F5, Inc.
 *
 * This source code is licensed under the Apache License, Version 2.0 license found in the
 * LICENSE file in the root directory of this source tree.
 */

package sources

import (
	"context"
	"strings"

	"github.com/nginx/agent/sdk/v2/proto"
	"github.com/nginx/agent/v2/src/core/metrics"
)

const (
	OSSNamespace       = "nginx"
	PlusNamespace      = "plus"
	SystemNamespace    = "system"
	ContainerNamespace = "container"

	OSSNginxType  = "oss"
	PlusNginxType = "plus"

	CpuGroup    = "cpu"
	MemoryGroup = "mem"
)

type namedMetric struct {
	namespace, group string
}

func (n *namedMetric) label(name string) string {
	if name == "" {
		return ""
	}
	switch {
	case n.namespace != "" && n.group != "":
		return strings.Join([]string{n.namespace, n.group, name}, ".")
	case n.namespace != "":
		return strings.Join([]string{n.namespace, name}, ".")
	case n.group != "":
		return strings.Join([]string{n.group, name}, ".")
	}
	return name
}

func (n *namedMetric) convertSamplesToSimpleMetrics(samples map[string]float64) (simpleMetrics []*proto.SimpleMetric) {
	for key, val := range samples {
		simpleMetrics = append(simpleMetrics, newFloatMetric(n.label(key), val))
	}
	return simpleMetrics
}

func newFloatMetric(name string, value float64) *proto.SimpleMetric {
	return &proto.SimpleMetric{
		Name:  name,
		Value: value,
	}
}

func Delta(current, previous map[string]map[string]float64) map[string]map[string]float64 {
	diff := make(map[string]map[string]float64)
	for currentKey, currentValue := range current {
		diff[currentKey] = make(map[string]float64)
		for key, value := range currentValue {
			previousValue, ok := previous[currentKey][key]
			if !ok {
				diff[currentKey][key] = value
				continue
			}
			diff[currentKey][key] = value - previousValue
		}
	}

	for previousKey, previousValue := range previous {
		_, ok := current[previousKey]
		if !ok {
			diff[previousKey] = make(map[string]float64)
			for key, value := range previousValue {
				diff[previousKey][key] = -value
			}
		}
	}

	return diff
}

func SendNginxDownStatus(ctx context.Context, dims []*proto.Dimension, m chan<- *metrics.StatsEntityWrapper) {
	simpleMetrics := []*proto.SimpleMetric{newFloatMetric("nginx.status", float64(0))}

	select {
	case <-ctx.Done():
	case m <- metrics.NewStatsEntityWrapper(dims, simpleMetrics, proto.MetricsReport_INSTANCE):
	}
}
