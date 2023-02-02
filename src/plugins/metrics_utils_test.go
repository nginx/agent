/**
 * Copyright (c) F5, Inc.
 *
 * This source code is licensed under the Apache License, Version 2.0 license found in the
 * LICENSE file in the root directory of this source tree.
 */

package plugins

import (
	"fmt"
	"github.com/nginx/agent/sdk/v2/proto"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

const (
	dimName     = "test-dim"
	dimValue    = "test-dim-value"
	emptyString = ""
)

func TestMetricsThrottle_GenerateMetricsReports_Single(t *testing.T) {
	allEntitiesTimes2 := []*proto.StatsEntity{
		{Dimensions: []*proto.Dimension{{Name: "cache_zone", Value: "cz"}}},
		{Dimensions: []*proto.Dimension{{Name: "upstream", Value: "up"}}},
		{Dimensions: []*proto.Dimension{{Name: dimName, Value: dimValue}}},
		{Dimensions: []*proto.Dimension{{Name: "cache_zone", Value: "cz2"}}},
		{Dimensions: []*proto.Dimension{{Name: "upstream", Value: "up2"}}},
		{Dimensions: []*proto.Dimension{{Name: dimName, Value: dimValue}}},
	}

	payloads := generateMetricsReports(allEntitiesTimes2, true)
	assert.Equal(t, 1, len(payloads))
	switch payloads[0].(type) {
	case *proto.MetricsReport:
	default:
		assert.Failf(t, fmt.Sprintf("expected payload of type *proto.MetricsReport, got %T", payloads[0]), "")
	}
	report := payloads[0].(*proto.MetricsReport)
	assert.Equal(t, len(allEntitiesTimes2), len(report.GetData()))

	count := 0
	for _, se := range report.GetData() {
		if v, c := hasNonEmptyDimension(se, "cache_zone"); c && v == "cz" {
			count++
		} else if v, c := hasNonEmptyDimension(se, "cache_zone"); c && v == "cz2" {
			count++
		} else if v, c := hasNonEmptyDimension(se, "upstream"); c && v == "up" {
			count++
		} else if v, c := hasNonEmptyDimension(se, "upstream"); c && v == "up2" {
			count++
		} else if v, c := hasNonEmptyDimension(se, dimName); c && v == dimValue {
			count++
		}
	}
	assert.Equal(t, len(allEntitiesTimes2), count)
}

func TestMetricsThrottle_GenerateMetricsReports_ManySameValues(t *testing.T) {
	allEntitiesTimes2 := []*proto.StatsEntity{
		{Dimensions: []*proto.Dimension{{Name: "cache_zone", Value: "cz"}}},
		{Dimensions: []*proto.Dimension{{Name: "upstream", Value: "up"}}},
		{Dimensions: []*proto.Dimension{{Name: dimName, Value: dimValue}}},
		{Dimensions: []*proto.Dimension{{Name: "cache_zone", Value: "cz"}}},
		{Dimensions: []*proto.Dimension{{Name: "upstream", Value: "up"}}},
		{Dimensions: []*proto.Dimension{{Name: dimName, Value: dimValue}}},
	}

	payloads := generateMetricsReports(allEntitiesTimes2, false)
	assert.Equal(t, 3, len(payloads))
	switch payloads[0].(type) {
	case *proto.MetricsReport:
	default:
		assert.Failf(t, fmt.Sprintf("expected payload of type *proto.MetricsReport, got %T", payloads[0]), "")
	}

	cache_report := &proto.MetricsReport{}
	upstream_report := &proto.MetricsReport{}
	other_report := &proto.MetricsReport{}

	for _, payload := range payloads {
		report := payload.(*proto.MetricsReport)
		if report.Type == proto.MetricsReport_CACHE_ZONE {
			cache_report = report
		} else if report.Type == proto.MetricsReport_UPSTREAMS {
			upstream_report = report
		} else if report.Type == proto.MetricsReport_SYSTEM {
			other_report = report
		}
	}

	assert.Equal(t, 2, len(cache_report.GetData()))
	count := 0
	for _, se := range cache_report.GetData() {
		if v, c := hasNonEmptyDimension(se, "cache_zone"); c && v == "cz" {
			count++
		} else if v, c := hasNonEmptyDimension(se, "cache_zone"); c && v == "cz2" {
			count++
		}
	}
	assert.Equal(t, 2, count)

	assert.Equal(t, 2, len(upstream_report.GetData()))
	count = 0
	for _, se := range upstream_report.GetData() {
		if v, c := hasNonEmptyDimension(se, "upstream"); c && v == "up" {
			count++
		} else if v, c := hasNonEmptyDimension(se, "upstream"); c && v == "up2" {
			count++
		}
	}
	assert.Equal(t, 2, count)

	assert.Equal(t, 2, len(cache_report.GetData()))
	count = 0
	for _, se := range other_report.GetData() {
		if v, c := hasNonEmptyDimension(se, dimName); c && v == dimValue {
			count++
		}
	}
	assert.Equal(t, 2, count)
}

func TestMetricsThrottle_GenerateMetricsReports_ManyDifferentValues(t *testing.T) {
	allEntitiesTimes2 := []*proto.StatsEntity{
		{Dimensions: []*proto.Dimension{{Name: "cache_zone", Value: "cz"}}},
		{Dimensions: []*proto.Dimension{{Name: "upstream", Value: "up"}}},
		{Dimensions: []*proto.Dimension{{Name: dimName, Value: dimValue}}},
		{Dimensions: []*proto.Dimension{{Name: "cache_zone", Value: "cz2"}}},
		{Dimensions: []*proto.Dimension{{Name: "upstream", Value: "up2"}}},
		{Dimensions: []*proto.Dimension{{Name: dimName, Value: dimValue}}},
	}

	payloads := generateMetricsReports(allEntitiesTimes2, false)
	assert.Equal(t, 5, len(payloads))
	switch payloads[0].(type) {
	case *proto.MetricsReport:
	default:
		assert.Failf(t, fmt.Sprintf("expected payload of type *proto.MetricsReport, got %T", payloads[0]), "")
	}

	cache_reports := []*proto.MetricsReport{}
	upstream_reports := []*proto.MetricsReport{}
	other_reports := []*proto.MetricsReport{}

	for _, payload := range payloads {
		report := payload.(*proto.MetricsReport)
		if report.Type == proto.MetricsReport_CACHE_ZONE {
			cache_reports = append(cache_reports, report)
		} else if report.Type == proto.MetricsReport_UPSTREAMS {
			upstream_reports = append(upstream_reports, report)
		} else if report.Type == proto.MetricsReport_SYSTEM {
			other_reports = append(other_reports, report)
		}
	}

	// Cache_zone reports
	assert.Equal(t, 2, len(cache_reports))
	values := make(map[string]bool, 0)
	count := 0
	for _, se := range cache_reports[0].GetData() {
		v, c := hasNonEmptyDimension(se, "cache_zone")
		values[v] = true
		if c && v == "cz" {
			count++
		} else if c && v == "cz2" {
			count++
		}
	}
	assert.Equal(t, 1, count)

	count = 0
	for _, se := range cache_reports[1].GetData() {
		v, c := hasNonEmptyDimension(se, "cache_zone")
		values[v] = true
		if c && v == "cz" {
			count++
		} else if c && v == "cz2" {
			count++
		}
	}
	assert.Equal(t, 1, count)
	assert.Equal(t, 2, len(values))

	// Upstream reports
	assert.Equal(t, 2, len(upstream_reports))
	values = make(map[string]bool, 0)
	count = 0
	for _, se := range upstream_reports[0].GetData() {
		v, c := hasNonEmptyDimension(se, "upstream")
		values[v] = true
		if c && v == "up" {
			count++
		} else if c && v == "up2" {
			count++
		}
	}
	assert.Equal(t, 1, count)

	count = 0
	for _, se := range upstream_reports[1].GetData() {
		v, c := hasNonEmptyDimension(se, "upstream")
		values[v] = true
		if c && v == "up" {
			count++
		} else if c && v == "up2" {
			count++
		}
	}
	assert.Equal(t, 1, count)
	assert.Equal(t, 2, len(values))

	// Other reports
	assert.Equal(t, 1, len(other_reports))
	values = make(map[string]bool, 0)
	count = 0
	for _, se := range other_reports[0].GetData() {
		v, c := hasNonEmptyDimension(se, dimName)
		values[v] = true
		if c && v == dimValue {
			count++
		}
	}
	assert.Equal(t, 2, count)
	assert.Equal(t, 1, len(values))

}

func TestMetricsThrottle_GroupStatsEntities(t *testing.T) {
	allEntities := []*proto.StatsEntity{
		{Dimensions: []*proto.Dimension{{Name: "cache_zone", Value: "cz"}}},
		{Dimensions: []*proto.Dimension{{Name: "upstream", Value: "up"}}},
		{Dimensions: []*proto.Dimension{{Name: dimName, Value: dimValue}}},
	}
	allEntitiesTimes2 := []*proto.StatsEntity{
		{Dimensions: []*proto.Dimension{{Name: "cache_zone", Value: "cz"}}},
		{Dimensions: []*proto.Dimension{{Name: "upstream", Value: "up"}}},
		{Dimensions: []*proto.Dimension{{Name: dimName, Value: dimValue}}},
		{Dimensions: []*proto.Dimension{{Name: "cache_zone", Value: "cz2"}}},
		{Dimensions: []*proto.Dimension{{Name: "upstream", Value: "up2"}}},
		{Dimensions: []*proto.Dimension{{Name: dimName, Value: dimValue}}},
	}

	cache_zones, upstreams, other := groupStatsEntities(allEntities, true)
	assert.Equal(t, 0, len(cache_zones))
	assert.Equal(t, 0, len(cache_zones["cz"]))
	assert.Equal(t, 0, len(upstreams))
	assert.Equal(t, 0, len(upstreams["up"]))
	assert.Equal(t, 3, len(other))

	cache_zones, upstreams, other = groupStatsEntities(allEntities, false)
	assert.Equal(t, 1, len(cache_zones))
	assert.Equal(t, 1, len(cache_zones["cz"]))
	assert.Equal(t, 1, len(upstreams))
	assert.Equal(t, 1, len(upstreams["up"]))
	assert.Equal(t, 1, len(other))
	assert.Equal(t, dimName, other[0].GetDimensions()[0].Name)
	assert.Equal(t, dimValue, other[0].GetDimensions()[0].Value)

	cache_zones, upstreams, other = groupStatsEntities(allEntitiesTimes2, false)
	assert.Equal(t, 2, len(cache_zones))
	assert.Equal(t, 1, len(cache_zones["cz"]))
	assert.Equal(t, 2, len(upstreams))
	assert.Equal(t, 1, len(upstreams["up"]))
	assert.Equal(t, 2, len(other))
	assert.Equal(t, dimName, other[0].GetDimensions()[0].Name)
	assert.Equal(t, dimValue, other[0].GetDimensions()[0].Value)
}

func TestMetricsThrottle_GetAllStatsEntities(t *testing.T) {
	validPayload := &proto.MetricsReport{Data: []*proto.StatsEntity{{Dimensions: []*proto.Dimension{{Name: dimName, Value: dimValue}}}}}
	emptyPayload := &proto.MetricsReport{Data: []*proto.StatsEntity{}}
	invalidPayload := dimName

	result := getAllStatsEntities(validPayload)
	assert.Equal(t, len(validPayload.GetData()), len(result))

	result = getAllStatsEntities(emptyPayload)
	assert.Equal(t, 0, len(result))

	result = getAllStatsEntities(invalidPayload)
	assert.Equal(t, 0, len(result))
}

func TestMetricsThrottle_HasNonEmptyDimension(t *testing.T) {

	validSE := &proto.StatsEntity{Dimensions: []*proto.Dimension{{Name: dimName, Value: dimValue}}}
	validSE2 := &proto.StatsEntity{Dimensions: []*proto.Dimension{{Name: strings.ToUpper(dimName), Value: dimValue}}}
	invalidSE := &proto.StatsEntity{Dimensions: []*proto.Dimension{}}
	invalidSE2 := &proto.StatsEntity{Dimensions: []*proto.Dimension{{Name: dimName, Value: emptyString}}}

	value, outcome := hasNonEmptyDimension(validSE, dimName)
	assert.Equal(t, dimValue, value)
	assert.True(t, outcome)

	value, outcome = hasNonEmptyDimension(validSE2, dimName)
	assert.Equal(t, dimValue, value)
	assert.True(t, outcome)

	value, outcome = hasNonEmptyDimension(nil, emptyString)
	assert.Equal(t, emptyString, value)
	assert.False(t, outcome)

	value, outcome = hasNonEmptyDimension(invalidSE, emptyString)
	assert.Equal(t, emptyString, value)
	assert.False(t, outcome)

	value, outcome = hasNonEmptyDimension(invalidSE2, emptyString)
	assert.Equal(t, emptyString, value)
	assert.False(t, outcome)
}
