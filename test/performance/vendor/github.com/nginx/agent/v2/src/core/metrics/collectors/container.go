/**
 * Copyright (c) F5, Inc.
 *
 * This source code is licensed under the Apache License, Version 2.0 license found in the
 * LICENSE file in the root directory of this source tree.
 */

package collectors

import (
	"context"

	log "github.com/sirupsen/logrus"

	"github.com/nginx/agent/v2/src/core"
	"github.com/nginx/agent/v2/src/core/config"
	"github.com/nginx/agent/v2/src/core/metrics"
	"github.com/nginx/agent/v2/src/core/metrics/sources"
	cgroup "github.com/nginx/agent/v2/src/core/metrics/sources/cgroup"
)

var _ metrics.Collector = (*ContainerCollector)(nil)

type ContainerCollector struct {
	sources []metrics.Source
	buf     chan *metrics.StatsEntityWrapper
	dim     *metrics.CommonDim
	env     core.Environment
}

func NewContainerCollector(env core.Environment, conf *config.Config) *ContainerCollector {
	log.Trace("Creating new container collector")

	containerSources := []metrics.Source{
		sources.NewContainerCPUSource(sources.ContainerNamespace, cgroup.CgroupBasePath),
		sources.NewContainerMemorySource(sources.ContainerNamespace, cgroup.CgroupBasePath),
	}

	return &ContainerCollector{
		sources: containerSources,
		buf:     make(chan *metrics.StatsEntityWrapper, 65535),
		dim:     metrics.NewCommonDim(env.NewHostInfo("agentVersion", &conf.Tags, conf.ConfigDirs, false), conf, ""),
		env:     env,
	}
}

func (c *ContainerCollector) collectMetrics(ctx context.Context) {
	for _, containerSource := range c.sources {
		go containerSource.Collect(ctx, c.buf)
	}
}

func (c *ContainerCollector) Collect(ctx context.Context, m chan<- *metrics.StatsEntityWrapper) {
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

func (c *ContainerCollector) UpdateConfig(config *config.Config) {
	c.dim = metrics.NewCommonDim(c.env.NewHostInfo("agentVersion", &config.Tags, config.ConfigDirs, false), config, "")
}
