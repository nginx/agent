/**
 * Copyright (c) F5, Inc.
 *
 * This source code is licensed under the Apache License, Version 2.0 license found in the
 * LICENSE file in the root directory of this source tree.
 */

package plugins

import (
	"github.com/gogo/protobuf/types"
	"github.com/nginx/agent/sdk/v2/proto"
	"github.com/nginx/agent/v2/src/core"
	"strings"
)

func generateMetricsReports(stats []*proto.StatsEntity, singleReport bool) []core.Payload {

	reports := make([]core.Payload, 0)
	cache_zones, upstreams, other := groupStatsEntities(stats, singleReport)

	for cz := range cache_zones {
		reports = append(reports, &proto.MetricsReport{
			Meta: &proto.Metadata{
				Timestamp: types.TimestampNow(),
			},
			Type: proto.MetricsReport_CACHE_ZONE,
			Data: cache_zones[cz],
		})
	}

	for ups := range upstreams {
		reports = append(reports, &proto.MetricsReport{
			Meta: &proto.Metadata{
				Timestamp: types.TimestampNow(),
			},
			Type: proto.MetricsReport_UPSTREAMS,
			Data: upstreams[ups],
		})
	}

	if len(other) > 0 {
		reports = append(reports, &proto.MetricsReport{
			Meta: &proto.Metadata{
				Timestamp: types.TimestampNow(),
			},
			Type: proto.MetricsReport_SYSTEM,
			Data: other,
		})
	}

	return reports
}

func groupStatsEntities(stats []*proto.StatsEntity, singleReport bool) (map[string][]*proto.StatsEntity, map[string][]*proto.StatsEntity, []*proto.StatsEntity) {
	cache_zones := make(map[string][]*proto.StatsEntity, 0)
	upstreams := make(map[string][]*proto.StatsEntity, 0)
	other := make([]*proto.StatsEntity, 0)

	for _, s := range stats {
		if !singleReport {
			if cz, exists := hasNonEmptyDimension(s, "cache_zone"); exists {
				if _, ok := cache_zones[cz]; !ok {
					cache_zones[cz] = make([]*proto.StatsEntity, 0)
				}
				cache_zones[cz] = append(cache_zones[cz], s)
			} else if ups, exists := hasNonEmptyDimension(s, "upstream"); exists {
				if _, ok := upstreams[ups]; !ok {
					upstreams[ups] = make([]*proto.StatsEntity, 0)
				}
				upstreams[ups] = append(upstreams[ups], s)
			} else {
				other = append(other, s)
			}
		} else {
			other = append(other, s)
		}
	}

	return cache_zones, upstreams, other
}

func getAllStatsEntities(payload core.Payload) []*proto.StatsEntity {
	ses := make([]*proto.StatsEntity, 0)
	switch p := payload.(type) {
	case *proto.MetricsReport:
		ses = append(ses, p.GetData()...)
	}
	return ses
}

func hasNonEmptyDimension(s *proto.StatsEntity, name string) (string, bool) {
	if s == nil {
		return "", false
	}
	dims := s.GetDimensions()
	lcName := strings.ToLower(name)
	for _, dim := range dims {
		if strings.ToLower(dim.Name) == lcName && strings.TrimSpace(dim.Value) != "" {
			return dim.Value, true
		}
	}
	return "", false
}
