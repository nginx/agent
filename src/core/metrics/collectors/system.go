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

	"github.com/nginx/agent/sdk/v2/proto"
	"github.com/nginx/agent/v2/src/core"
	"github.com/nginx/agent/v2/src/core/config"
	"github.com/nginx/agent/v2/src/core/metrics"
	"github.com/nginx/agent/v2/src/core/metrics/sources"
	log "github.com/sirupsen/logrus"
)

var (
	_ metrics.Collector = (*SystemCollector)(nil)
)

type SystemCollector struct {
	sources      []metrics.Source
	buf          chan *proto.StatsEntity
	dim          *metrics.CommonDim
	env          core.Environment
	sourceErrors map[string]error
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
			sources.NewDiskSource(sources.SystemNamespace),
			sources.NewDiskIOSource(sources.SystemNamespace, env),
			sources.NewNetIOSource(sources.SystemNamespace, env),
			sources.NewLoadSource(sources.SystemNamespace),
			sources.NewSwapSource(sources.SystemNamespace, env),
		}
	}

	return &SystemCollector{
		sources:      systemSources,
		buf:          make(chan *proto.StatsEntity, 65535),
		dim:          metrics.NewCommonDim(env.NewHostInfo("agentVersion", &conf.Tags, conf.ConfigDirs, false), conf, ""),
		env:          env,
		sourceErrors: make(map[string]error),
	}
}

func (c *SystemCollector) collectMetrics(ctx context.Context) {
	// using a separate WaitGroup, since we need to wait for our own buffer to be filled
	// this ensures the collection is done before our own for/select loop to pull things off the buf
	wg := &sync.WaitGroup{}
	for _, systemSource := range c.sources {
		wg.Add(1)
		go systemSource.Collect(ctx, wg, c.buf)
	}
	wg.Wait()
}

func (c *SystemCollector) Collect(ctx context.Context, wg *sync.WaitGroup, m chan<- *proto.StatsEntity) {
	defer wg.Done()
	c.collectMetrics(ctx)
	c.checkSourcesForErrors()

	commonDims := c.dim.ToDimensions()
	for {
		select {
		case <-ctx.Done():
			return
		case sample := <-c.buf:
			sample.Dimensions = append(commonDims, sample.Dimensions...)

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

func (c *SystemCollector) checkSourcesForErrors() {
	for _, containerSource := range c.sources {
		if containerSource.ErrorCollectingMetrics() != nil {
			if _, ok := c.sourceErrors[containerSource.Name()]; !ok {
				log.Warnf("Unable to collect system metrics from source %s, %v", containerSource.Name(), containerSource.ErrorCollectingMetrics())
				c.sourceErrors[containerSource.Name()] = containerSource.ErrorCollectingMetrics()
			}
		} else {
			delete(c.sourceErrors, containerSource.Name())
		}
	}
}
