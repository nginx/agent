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
	log "github.com/sirupsen/logrus"
)

type NginxCollector struct {
	sources       []metrics.NginxSource
	buf           chan *metrics.StatsEntityWrapper
	dimensions    *metrics.CommonDim
	collectorConf *metrics.NginxCollectorConfig
	env           core.Environment
	binary        core.NginxBinary
}

func NewNginxCollector(conf *config.Config, env core.Environment, collectorConf *metrics.NginxCollectorConfig, binary core.NginxBinary) *NginxCollector {
	host := env.NewHostInfo("agentVersion", &conf.Tags, conf.ConfigDirs, false)
	dimensions := metrics.NewCommonDim(host, conf, collectorConf.NginxId)
	dimensions.NginxConfPath = collectorConf.ConfPath
	dimensions.NginxAccessLogPaths = collectorConf.AccessLogs

	return &NginxCollector{
		sources:       buildSources(dimensions, binary, collectorConf),
		buf:           make(chan *metrics.StatsEntityWrapper, 65535),
		dimensions:    dimensions,
		collectorConf: collectorConf,
		env:           env,
		binary:        binary,
	}
}

func buildSources(dimensions *metrics.CommonDim, binary core.NginxBinary, collectorConf *metrics.NginxCollectorConfig) []metrics.NginxSource {
	var nginxSources []metrics.NginxSource
	// worker metrics
	nginxSources = append(nginxSources, sources.NewNginxProcess(dimensions, sources.OSSNamespace, binary))
	nginxSources = append(nginxSources, sources.NewNginxWorker(dimensions, sources.OSSNamespace, binary, sources.NewNginxWorkerClient()))

	if collectorConf.StubStatus != "" {
		nginxSources = append(nginxSources, sources.NewNginxOSS(dimensions, sources.OSSNamespace, collectorConf.StubStatus))
		nginxSources = append(nginxSources, sources.NewNginxAccessLog(dimensions, sources.OSSNamespace, binary, sources.OSSNginxType, collectorConf.CollectionInterval))
		nginxSources = append(nginxSources, sources.NewNginxErrorLog(dimensions, sources.OSSNamespace, binary, sources.OSSNginxType, collectorConf.CollectionInterval))
	} else if collectorConf.PlusAPI != "" {
		nginxSources = append(nginxSources, sources.NewNginxPlus(dimensions, sources.OSSNamespace, sources.PlusNamespace, collectorConf.PlusAPI, collectorConf.ClientVersion))
		nginxSources = append(nginxSources, sources.NewNginxAccessLog(dimensions, sources.OSSNamespace, binary, sources.PlusNginxType, collectorConf.CollectionInterval))
		nginxSources = append(nginxSources, sources.NewNginxErrorLog(dimensions, sources.OSSNamespace, binary, sources.PlusNginxType, collectorConf.CollectionInterval))
	} else {
		// if Plus API or stub_status are not setup, run the NGINX static collector and return nginx.status = 0
		log.Warnf("The NGINX API is not configured. Please configure it to collect NGINX metrics.")
		nginxSources = append(nginxSources, sources.NewNginxStatic(dimensions, sources.OSSNamespace))
	}
	return nginxSources
}

func (c *NginxCollector) collectMetrics(ctx context.Context) {
	// using a separate WaitGroup, since we need to wait for our own buffer to be filled
	// this ensures the collection is done before our own for/select loop to pull things off the buf
	wg := &sync.WaitGroup{}
	for _, nginxSource := range c.sources {
		wg.Add(1)
		go nginxSource.Collect(ctx, wg, c.buf)
	}
	wg.Wait()
}

func (c *NginxCollector) Collect(ctx context.Context, wg *sync.WaitGroup, m chan<- *metrics.StatsEntityWrapper) {
	defer wg.Done()
	c.collectMetrics(ctx)
	for {
		select {
		case <-ctx.Done():
			return
		case stat := <-c.buf:
			select {
			case <-ctx.Done():
				return
			case m <- stat:
			}
		default:
			return
		}
	}
}

func (c *NginxCollector) UpdateConfig(config *config.Config) {
	host := c.env.NewHostInfo("agentVersion", &config.Tags, config.ConfigDirs, false)
	c.dimensions = metrics.NewCommonDim(host, config, c.collectorConf.NginxId)
	c.dimensions.NginxConfPath = c.collectorConf.ConfPath
	c.dimensions.NginxAccessLogPaths = c.collectorConf.AccessLogs

	for _, nginxSource := range c.sources {
		nginxSource.Update(c.dimensions, c.collectorConf)
	}
}

func (c *NginxCollector) UpdateCollectorConfig(collectorConfig *metrics.NginxCollectorConfig) {
	// If the metrics API has being enabled or disabled then we need to stop all nginx sources and rebuild them again
	if c.collectorConf.StubStatus != collectorConfig.StubStatus || c.collectorConf.PlusAPI != collectorConfig.PlusAPI {
		c.Stop()
		c.sources = buildSources(c.dimensions, c.binary, collectorConfig)
	}

	c.collectorConf = collectorConfig
}

func (c *NginxCollector) Stop() {
	log.Tracef("Stopping Nginx collector sources for nginxId %s", c.GetNginxId())
	for _, nginxSource := range c.sources {
		nginxSource.Stop()
	}
}

func (c *NginxCollector) GetNginxId() string {
	return c.dimensions.NginxId
}

func (c *NginxCollector) UpdateSources() {
	for _, nginxSource := range c.sources {
		nginxSource.Update(c.dimensions, c.collectorConf)
	}
}
