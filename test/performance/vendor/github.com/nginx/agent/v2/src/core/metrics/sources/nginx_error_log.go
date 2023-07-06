/**
 * Copyright (c) F5, Inc.
 *
 * This source code is licensed under the Apache License, Version 2.0 license found in the
 * LICENSE file in the root directory of this source tree.
 */

package sources

import (
	"context"
	re "regexp"
	"sync"
	"time"

	"github.com/nginx/agent/sdk/v2/proto"

	"github.com/nginx/agent/v2/src/core"
	"github.com/nginx/agent/v2/src/core/metrics"
	"github.com/nginx/agent/v2/src/core/tailer"

	log "github.com/sirupsen/logrus"
)

const (
	HttpRequestBufferedMetricName      = "http.request.buffered"
	UpstreamResponseBufferedMetricName = "upstream.response.buffered"
	UpstreamRequestFailedMetricName    = "upstream.request.failed"
	UpstreamResponseFailedMetricName   = "upstream.response.failed"
)

var regularExpressionErrorMap = map[string][]*re.Regexp{
	HttpRequestBufferedMetricName: {
		re.MustCompile(`.*client request body is buffered.*`),
	},
	UpstreamResponseBufferedMetricName: {
		re.MustCompile(`.*upstream response is buffered.*`),
	},
	UpstreamRequestFailedMetricName: {
		re.MustCompile(`.*failed.*while connecting to upstream, client.*`),
		re.MustCompile(`.*upstream timed out.*while connecting to upstream, client.*`),
		re.MustCompile(`.*upstream queue is full while connecting to upstream.*`),
		re.MustCompile(`.*no live upstreams while connecting to upstream, client.*`),
		re.MustCompile(`.*upstream connection is closed too while sending request to upstream, client.*`),
	},
	UpstreamResponseFailedMetricName: {
		re.MustCompile(`.*failed.*while reading upstream.*`),
		re.MustCompile(`.*failed.*while reading response header from upstream, client.*`),
		re.MustCompile(`.*upstream timed out.*while reading response header from upstream, client.*`),
		re.MustCompile(`.*upstream buffer is too small to read response.*`),
		re.MustCompile(`.*upstream prematurely closed connection while reading response header from upstream, client.*`),
		re.MustCompile(`.*upstream sent no valid.*header while reading response.*`),
		re.MustCompile(`.*upstream sent invalid header.*`),
		re.MustCompile(`.*upstream sent invalid chunked response.*`),
		re.MustCompile(`.*upstream sent too big header while reading response header from upstream.*`),
	},
}

type NginxErrorLog struct {
	baseDimensions *metrics.CommonDim
	*namedMetric
	mu                 *sync.Mutex
	logFormats         map[string]string
	logs               map[string]context.CancelFunc
	binary             core.NginxBinary
	nginxType          string
	collectionInterval time.Duration
	buf                []*metrics.StatsEntityWrapper
}

func NewNginxErrorLog(
	baseDimensions *metrics.CommonDim,
	namespace string,
	binary core.NginxBinary,
	nginxType string,
	collectionInterval time.Duration,
) *NginxErrorLog {
	log.Trace("Creating NewNginxErrorLog")
	nginxErrorLog := &NginxErrorLog{
		baseDimensions,
		&namedMetric{namespace: namespace},
		&sync.Mutex{},
		make(map[string]string),
		make(map[string]context.CancelFunc),
		binary,
		nginxType,
		collectionInterval,
		[]*metrics.StatsEntityWrapper{},
	}

	logs := binary.GetErrorLogs()

	for logFile, logFormat := range logs {
		log.Infof("Adding error log tailer: %s", logFile)
		logCTX, fn := context.WithCancel(context.Background())
		nginxErrorLog.logs[logFile] = fn
		nginxErrorLog.logFormats[logFile] = logFormat
		go nginxErrorLog.logStats(logCTX, logFile)
	}

	return nginxErrorLog
}

func (c *NginxErrorLog) Collect(ctx context.Context, wg *sync.WaitGroup, m chan<- *metrics.StatsEntityWrapper) {
	defer wg.Done()

	c.collectLogStats(ctx, m)
}

func (c *NginxErrorLog) Stop() {
	for f, fn := range c.logs {
		log.Infof("Removing error log tailer: %s", f)
		fn()
		delete(c.logs, f)
	}
	log.Debugf("Stopping NginxErrorLog source for nginx id: %v", c.baseDimensions.NginxId)
}

func (c *NginxErrorLog) Update(dimensions *metrics.CommonDim, collectorConf *metrics.NginxCollectorConfig) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.baseDimensions = dimensions

	if c.collectionInterval != collectorConf.CollectionInterval {
		c.collectionInterval = collectorConf.CollectionInterval
		// remove old error logs
		// add new error logs
		c.recreateLogs()
	} else {
		// add, remove or update existing log trailers
		c.syncLogs()
	}
}

func (c *NginxErrorLog) recreateLogs() {
	for f, fn := range c.logs {
		c.stopTailer(f, fn)
	}

	logs := c.binary.GetErrorLogs()

	for logFile, logFormat := range logs {
		c.startTailer(logFile, logFormat)
	}
}

func (c *NginxErrorLog) syncLogs() {
	logs := c.binary.GetErrorLogs()

	for f, fn := range c.logs {
		if _, ok := logs[f]; !ok {
			c.stopTailer(f, fn)
		}
	}

	for logFile, logFormat := range logs {
		if _, ok := c.logs[logFile]; !ok {
			c.startTailer(logFile, logFormat)
		} else if c.logFormats[logFile] != logFormat {
			// cancel tailer with old log format
			c.logs[logFile]()
			c.startTailer(logFile, logFormat)
		}
	}
}

func (c *NginxErrorLog) startTailer(logFile string, logFormat string) {
	log.Infof("Adding error log tailer: %s", logFile)
	logCTX, fn := context.WithCancel(context.Background())
	c.logs[logFile] = fn
	c.logFormats[logFile] = logFormat
	go c.logStats(logCTX, logFile)
}

func (c *NginxErrorLog) stopTailer(logFile string, cancelFunction context.CancelFunc) {
	log.Infof("Removing error log tailer: %s", logFile)
	cancelFunction()
	delete(c.logs, logFile)
	delete(c.logFormats, logFile)
}

func (c *NginxErrorLog) collectLogStats(ctx context.Context, m chan<- *metrics.StatsEntityWrapper) {
	c.mu.Lock()
	defer c.mu.Unlock()

	for _, stat := range c.buf {
		m <- stat
	}
	c.buf = []*metrics.StatsEntityWrapper{}
}

func (c *NginxErrorLog) logStats(ctx context.Context, logFile string) {
	log.Debugf("Collecting from error log: %s", logFile)

	counters := map[string]float64{
		HttpRequestBufferedMetricName:      0,
		UpstreamResponseBufferedMetricName: 0,
		UpstreamRequestFailedMetricName:    0,
		UpstreamResponseFailedMetricName:   0,
	}
	mu := sync.Mutex{}

	t, err := tailer.NewTailer(logFile)
	if err != nil {
		log.Errorf("Unable to tail %q: %v", logFile, err)
		return
	}
	data := make(chan string, 1024)
	go t.Tail(ctx, data)

	tick := time.NewTicker(c.collectionInterval)
	defer tick.Stop()

	for {
		select {
		case d := <-data:
			mu.Lock()

			for metricName, regularExpressionList := range regularExpressionErrorMap {
				for _, re := range regularExpressionList {
					if re.MatchString(d) {
						counters[metricName] = counters[metricName] + 1
						break
					}
				}
			}

			mu.Unlock()

		case <-tick.C:
			c.baseDimensions.NginxType = c.nginxType
			c.baseDimensions.PublishedAPI = logFile

			mu.Lock()
			simpleMetrics := c.convertSamplesToSimpleMetrics(counters)
			log.Tracef("Error log metrics collected: %v", simpleMetrics)

			// reset the counters
			counters = map[string]float64{
				HttpRequestBufferedMetricName:      0,
				UpstreamResponseBufferedMetricName: 0,
				UpstreamRequestFailedMetricName:    0,
				UpstreamResponseFailedMetricName:   0,
			}

			c.buf = append(c.buf, metrics.NewStatsEntityWrapper(c.baseDimensions.ToDimensions(), simpleMetrics, proto.MetricsReport_INSTANCE))

			mu.Unlock()

		case <-ctx.Done():
			err := ctx.Err()
			if err != nil {
				log.Errorf("NginxErrorLog: error in done context logStats %v", err)
			}
			log.Info("NginxErrorLog: logStats are done")
			return
		}
	}
}
