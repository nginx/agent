/**
 * Copyright (c) F5, Inc.
 *
 * This source code is licensed under the Apache License, Version 2.0 license found in the
 * LICENSE file in the root directory of this source tree.
 */

package collectors

import (
	"context"
	"sync"

	"github.com/nginx/agent/v2/src/core"
	"github.com/nginx/agent/v2/src/core/config"
	"github.com/nginx/agent/v2/src/core/metrics"
	"github.com/nginx/agent/v2/src/core/metrics/sources"
)

var _ metrics.Collector = (*SystemCollector)(nil)

type SystemCollector struct {
	sources []metrics.Source
	buf     chan *metrics.StatsEntityWrapper
	dim     *metrics.CommonDim
	env     core.Environment
}

func NewSystemCollector(env core.Environment, conf *config.Config) *SystemCollector {
	var systemSources []metrics.Source

	if env.IsContainer() {
		systemSources = []metrics.Source{
			sources.NewVirtualMemorySource(sources.SystemNamespace, env),
			sources.NewCPUTimesSource(sources.SystemNamespace, env),
			sources.NewNetIOSource(sources.SystemNamespace, env),
			sources.NewSwapSource(sources.SystemNamespace, env),
		}
	} else {
		systemSources = []metrics.Source{
			sources.NewVirtualMemorySource(sources.SystemNamespace, env),
			sources.NewCPUTimesSource(sources.SystemNamespace, env),
			sources.NewDiskSource(sources.SystemNamespace, env),
			sources.NewDiskIOSource(sources.SystemNamespace, env),
			sources.NewNetIOSource(sources.SystemNamespace, env),
			sources.NewLoadSource(sources.SystemNamespace),
			sources.NewSwapSource(sources.SystemNamespace, env),
		}
	}

	return &SystemCollector{
		sources: systemSources,
		buf:     make(chan *metrics.StatsEntityWrapper, 65535),
		dim:     metrics.NewCommonDim(env.NewHostInfo("agentVersion", &conf.Tags, conf.ConfigDirs, false), conf, ""),
		env:     env,
	}
}

func (c *SystemCollector) collectMetrics(ctx context.Context) {
	// using a separate WaitGroup, since we need to wait for our own buffer to be filled
	// this ensures the collection is done before our own for/select loop to pull things off the buf
	for _, systemSource := range c.sources {
		go systemSource.Collect(ctx, nil, c.buf)
	}
}

func (c *SystemCollector) Collect(ctx context.Context, _ *sync.WaitGroup, m chan<- *metrics.StatsEntityWrapper) {
	// defer wg.Done()
	c.collectMetrics(ctx)

	commonDims := c.dim.ToDimensions()
	for {
		select {
		case <-ctx.Done():
			return
		case sample := <-c.buf:
			sample.Data.Dimensions = append(commonDims, sample.Data.Dimensions...)

			select {
			case <-ctx.Done():
				return
			case m <- sample:
			}
		default:
			return
		}
	}
}

func (c *SystemCollector) UpdateConfig(config *config.Config) {
	c.dim = metrics.NewCommonDim(c.env.NewHostInfo("agentVersion", &config.Tags, config.ConfigDirs, false), config, "")
}
