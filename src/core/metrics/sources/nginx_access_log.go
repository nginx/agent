/**
 * Copyright (c) F5, Inc.
 *
 * This source code is licensed under the Apache License, Version 2.0 license found in the
 * LICENSE file in the root directory of this source tree.
 */

package sources

import (
	"context"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/nginx/agent/sdk/v2/proto"
	"github.com/nginx/agent/v2/src/core"
	"github.com/nginx/agent/v2/src/core/metrics"
	"github.com/nginx/agent/v2/src/core/tailer"

	log "github.com/sirupsen/logrus"
)

const (
	spaceDelim = " "
	pattern    = `[A-Z]+\s.+\s[A-Z]+/.+`
)

var logVarMap = map[string]string{
	"$remote_addr":              "%{IPORHOST:remote_addr}",
	"$remote_user":              "%{USERNAME:remote_user}",
	"$time_local":               `%{HTTPDATE:time_local}`,
	"$status":                   "%{INT:status}",
	"$body_bytes_sent":          "%{NUMBER:body_bytes_sent}",
	"$http_referer":             "%{DATA:http_referer}",
	"$http_user_agent":          "%{DATA:http_user_agent}",
	"$http_x_forwarded_for":     "%{DATA:http_x_forwarded_for}",
	"$bytes_sent":               "%{NUMBER:bytes_sent}",
	"$gzip_ratio":               "%{DATA:gzip_ratio}",
	"$server_protocol":          "%{DATA:server_protocol}",
	"$request_length":           "%{INT:request_length}",
	"$request_time":             "%{DATA:request_time}",
	"\"$request\"":              "\"%{DATA:request}\"",
	"$request ":                 "%{DATA:request} ",
	"$upstream_connect_time":    "%{DATA:upstream_connect_time}",
	"$upstream_header_time":     "%{DATA:upstream_header_time}",
	"$upstream_response_time":   "%{DATA:upstream_response_time}",
	"$upstream_response_length": "%{DATA:upstream_response_length}",
	"$upstream_status":          "%{DATA:upstream_status}",
	"$upstream_cache_status":    "%{DATA:upstream_cache_status}",
	"[":                         "\\[",
	"]":                         "\\]",
}

// This metrics source is used to tail the NGINX access logs to retrieve metrics.

type NginxAccessLog struct {
	baseDimensions *metrics.CommonDim
	*namedMetric
	mu                 *sync.Mutex
	logFormats         map[string]string
	logs               map[string]context.CancelFunc
	binary             core.NginxBinary
	nginxType          string
	collectionInterval time.Duration
	buf                []*metrics.StatsEntityWrapper
	logger             *MetricSourceLogger
}

func NewNginxAccessLog(
	baseDimensions *metrics.CommonDim,
	namespace string,
	binary core.NginxBinary,
	nginxType string,
	collectionInterval time.Duration) *NginxAccessLog {
	log.Trace("Creating NginxAccessLog")

	nginxAccessLog := &NginxAccessLog{
		baseDimensions,
		&namedMetric{namespace: namespace},
		&sync.Mutex{},
		make(map[string]string),
		make(map[string]context.CancelFunc),
		binary,
		nginxType,
		collectionInterval,
		[]*metrics.StatsEntityWrapper{},
		NewMetricSourceLogger(),
	}

	logs := binary.GetAccessLogs()

	for logFile, logFormat := range logs {
		log.Infof("Adding access log tailer: %s", logFile)
		logCTX, fn := context.WithCancel(context.Background())
		nginxAccessLog.logs[logFile] = fn
		nginxAccessLog.logFormats[logFile] = logFormat
		go nginxAccessLog.logStats(logCTX, logFile, logFormat)
	}
	return nginxAccessLog
}

func (c *NginxAccessLog) Collect(ctx context.Context, wg *sync.WaitGroup, m chan<- *metrics.StatsEntityWrapper) {
	defer wg.Done()
	c.collectLogStats(ctx, m)
}

func (c *NginxAccessLog) Update(dimensions *metrics.CommonDim, collectorConf *metrics.NginxCollectorConfig) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.baseDimensions = dimensions

	if c.collectionInterval != collectorConf.CollectionInterval {
		c.collectionInterval = collectorConf.CollectionInterval
		// remove old access logs
		// add new access logs
		c.recreateLogs()
	} else {
		// add, remove or update existing log trailers
		c.syncLogs()
	}

}

func (c *NginxAccessLog) recreateLogs() {
	for f, fn := range c.logs {
		c.stopTailer(f, fn)
	}

	logs := c.binary.GetAccessLogs()

	for logFile, logFormat := range logs {
		c.startTailer(logFile, logFormat)
	}
}

func (c *NginxAccessLog) syncLogs() {
	logs := c.binary.GetAccessLogs()

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

func (c *NginxAccessLog) startTailer(logFile string, logFormat string) {
	log.Infof("Adding access log tailer: %s", logFile)
	logCTX, fn := context.WithCancel(context.Background())
	c.logs[logFile] = fn
	c.logFormats[logFile] = logFormat
	go c.logStats(logCTX, logFile, logFormat)
}

func (c *NginxAccessLog) stopTailer(logFile string, cancelFunction context.CancelFunc) {
	log.Infof("Removing access log tailer: %s", logFile)
	cancelFunction()
	delete(c.logs, logFile)
	delete(c.logFormats, logFile)
}

func (c *NginxAccessLog) Stop() {
	for f, fn := range c.logs {
		log.Infof("Removing access log tailer: %s", f)
		fn()
		delete(c.logs, f)
	}
	log.Debugf("Stopping NginxAccessLog source for nginx id: %v", c.baseDimensions.NginxId)
}

func (c *NginxAccessLog) collectLogStats(ctx context.Context, m chan<- *metrics.StatsEntityWrapper) {
	c.mu.Lock()
	defer c.mu.Unlock()

	for _, stat := range c.buf {
		m <- stat
	}
	c.buf = []*metrics.StatsEntityWrapper{}
}

func (c *NginxAccessLog) logStats(ctx context.Context, logFile, logFormat string) {
	logPattern := logFormat

	for key, value := range logVarMap {
		logPattern = strings.ReplaceAll(logPattern, key, value)
	}
	log.Debugf("LogPattern = %s", logPattern)
	r, err := regexp.Compile(`[\$]([a-z_]+)`)
	if err != nil {
		log.Warnf("unable to compile access log regex: %v", err)
	}

	variables := r.FindAllString(logPattern, -1)

	log.Debugf("LogPattern = %s Matched variables = %v", logPattern, variables)
	for _, variable := range variables {
		replacement := "%" + fmt.Sprintf("{DATA:%s}", strings.Trim(variable, "$"))
		logPattern = strings.Replace(logPattern, variable, replacement, -1)
	}

	log.Debugf("Collecting from: %s using format: %s", logFile, logFormat)
	log.Debugf("Pattern used for tailing logs: %s", logPattern)

	httpCounters, upstreamCounters, upstreamCacheCounters := getDefaultCounters()
	gzipRatios, requestLengths, requestTimes, upstreamResponseLength, upstreamResponseTimes, upstreamConnectTimes, upstreamHeaderTimes := []float64{}, []float64{}, []float64{}, []float64{}, []float64{}, []float64{}, []float64{}

	mu := sync.Mutex{}

	t, err := tailer.NewPatternTailer(logFile, map[string]string{"DEFAULT": logPattern})
	if err != nil {
		log.Errorf("unable to tail %q: %v", logFile, err)
		return
	}
	data := make(chan map[string]string, 1024)
	go t.Tail(ctx, data)

	tick := time.NewTicker(c.collectionInterval)
	defer tick.Stop()
	for {
		select {
		case d := <-data:
			access, err := tailer.NewNginxAccessItem(d)
			upstreamRequest := false
			if err != nil {
				c.logger.Log(fmt.Sprintf("Error decoding access log entry, %v", err))
				continue
			}

			mu.Lock()

			httpCounters = c.parseAccessLogFloatCounters("request.body_bytes_sent", access.BodyBytesSent, httpCounters)

			httpCounters = c.parseAccessLogFloatCounters("request.bytes_sent", access.BytesSent, httpCounters)

			gzipRatios = c.parseAccessLogFloatTimes("gzip_ratio", access.GzipRatio, gzipRatios)

			requestLengths = c.parseAccessLogFloatTimes("request_length", access.RequestLength, requestLengths)

			requestTimes = c.parseAccessLogFloatTimes("request_time", access.RequestTime, requestTimes)

			upstreamConnectTimes = c.parseAccessLogUpstream("upstream_connect_time", access.UpstreamConnectTime, upstreamConnectTimes)

			upstreamHeaderTimes = c.parseAccessLogUpstream("upstream_header_time", access.UpstreamHeaderTime, upstreamHeaderTimes)

			upstreamResponseLength = c.parseAccessLogUpstream("upstream_response_length", access.UpstreamResponseLength, upstreamResponseLength)

			upstreamResponseTimes = c.parseAccessLogUpstream("upstream_response_time", access.UpstreamResponseTime, upstreamResponseTimes)

			if access.Request != "" {
				method, _, protocol := getParsedRequest(access.Request)
				n := fmt.Sprintf("method.%s", strings.ToLower(method))
				if isOtherMethod(n) {
					n = "method.others"
				}
				httpCounters[n] = httpCounters[n] + 1

				if access.ServerProtocol == "" {
					calculateServerProtocol(protocol, httpCounters)
				}
			}

			if access.ServerProtocol != "" {
				calculateServerProtocol(access.ServerProtocol, httpCounters)
			}

			if access.UpstreamStatus != "" && access.UpstreamStatus != "-" {
				upstreamRequest = true
				statusValues := strings.Split(access.UpstreamStatus, ",")
				for _, value := range statusValues {
					if v, err := strconv.Atoi(value); err == nil {
						n := fmt.Sprintf("upstream.status.%dxx", v/100)
						upstreamCounters[n] = upstreamCounters[n] + 1
					} else {
						log.Debugf("Error getting upstream status value from access logs, %v", err)
					}
				}

			}

			if access.UpstreamCacheStatus != "" && access.UpstreamCacheStatus != "-" {
				upstreamRequest = true
				calculateUpstreamCacheStatus(access.UpstreamCacheStatus, upstreamCacheCounters)
			}

			// don't need the http status for NGINX Plus
			if c.nginxType == OSSNginxType {
				c.calculateHttpStatus(access.Status, httpCounters)
			}

			if access.UpstreamConnectTime != "" || access.UpstreamHeaderTime != "" || access.UpstreamResponseTime != "" {
				upstreamTimes := []string{access.UpstreamConnectTime, access.UpstreamHeaderTime, access.UpstreamResponseTime}
				upstreamRequest, upstreamCounters = calculateUpstreamNextCount(upstreamTimes, upstreamCounters)
			}

			if upstreamRequest == true {
				upstreamCounters["upstream.request.count"] = upstreamCounters["upstream.request.count"] + 1
			}

			mu.Unlock()

		case <-tick.C:
			c.baseDimensions.NginxType = c.nginxType
			c.baseDimensions.PublishedAPI = logFile

			mu.Lock()

			if len(requestLengths) > 0 {
				httpCounters["request.length"] = getAverageMetricValue(requestLengths)
			}

			if len(gzipRatios) > 0 {
				httpCounters["gzip.ratio"] = getAverageMetricValue(gzipRatios)
			}

			if len(requestTimes) > 0 {
				calculateTimeMetricsMap("request.time", requestTimes, httpCounters)
			}

			if len(upstreamConnectTimes) > 0 {
				calculateTimeMetricsMap("upstream.connect.time", upstreamConnectTimes, upstreamCounters)
			}

			if len(upstreamHeaderTimes) > 0 {
				calculateTimeMetricsMap("upstream.header.time", upstreamHeaderTimes, upstreamCounters)
			}

			if len(upstreamResponseTimes) > 0 {
				calculateTimeMetricsMap("upstream.response.time", upstreamResponseTimes, upstreamCounters)
			}

			if len(upstreamResponseLength) > 0 {
				upstreamCounters["upstream.response.length"] = getAverageMetricValue(upstreamResponseLength)
			}
			c.group = "http"
			simpleMetrics := c.convertSamplesToSimpleMetrics(httpCounters)

			c.group = ""
			simpleMetrics = append(simpleMetrics, c.convertSamplesToSimpleMetrics(upstreamCounters)...)

			c.group = ""
			simpleMetrics = append(simpleMetrics, c.convertSamplesToSimpleMetrics(upstreamCacheCounters)...)

			log.Tracef("Access log metrics collected: %v", simpleMetrics)

			// reset the counters
			httpCounters, upstreamCounters, upstreamCacheCounters = getDefaultCounters()
			gzipRatios, requestLengths, requestTimes, upstreamResponseLength, upstreamResponseTimes, upstreamConnectTimes, upstreamHeaderTimes = []float64{}, []float64{}, []float64{}, []float64{}, []float64{}, []float64{}, []float64{}

			c.buf = append(c.buf, metrics.NewStatsEntityWrapper(c.baseDimensions.ToDimensions(), simpleMetrics, proto.MetricsReport_INSTANCE))

			mu.Unlock()

		case <-ctx.Done():
			err := ctx.Err()
			if err != nil {
				log.Tracef("NginxAccessLog: error in done context logStats %v", err)
			}
			log.Info("NginxAccessLog: logStats are done")
			return
		}
	}
}

func calculateUpstreamNextCount(metricValues []string, upstreamCounters map[string]float64) (bool, map[string]float64) {
	upstreamRequest := false
	for _, upstreamTimes := range metricValues {
		if upstreamTimes != "" && upstreamTimes != "-" {
			upstreamRequest = true
			times := strings.Split(upstreamTimes, ", ")
			if len(times) > 1 {
				upstreamCounters["upstream.next.count"] = upstreamCounters["upstream.next.count"] + (float64(len(times)) - 1)
				return upstreamRequest, upstreamCounters
			}
		}
	}
	return upstreamRequest, upstreamCounters
}

func (c *NginxAccessLog) parseAccessLogFloatTimes(metricName string, metric string, counter []float64) []float64 {
	if metric != "" && metric != "-" {
		if v, err := strconv.ParseFloat(metric, 64); err == nil {
			counter = append(counter, v)
			return counter
		} else {
			c.logger.Log(fmt.Sprintf("Error getting %s value from access logs, %v", metricName, err))
		}
	}
	return counter
}

func (c *NginxAccessLog) parseAccessLogUpstream(metricName string, metric string, counter []float64) []float64 {
	if metric != "" && metric != "-" {
		metricValues := strings.Split(metric, ", ")
		for _, value := range metricValues {
			if value != "" && value != "-" {
				if v, err := strconv.ParseFloat(value, 64); err == nil {
					counter = append(counter, v)
				} else {
					c.logger.Log(fmt.Sprintf("Error getting %s value from access logs, %v", metricName, err))
				}
			}
		}
		return counter

	}
	return counter
}

func (c *NginxAccessLog) parseAccessLogFloatCounters(metricName string, metric string, counters map[string]float64) map[string]float64 {
	if metric != "" {
		if v, err := strconv.ParseFloat(metric, 64); err == nil {
			counters[metricName] = v + counters[metricName]
			return counters
		} else {
			c.logger.Log(fmt.Sprintf("Error getting %s value from access logs, %v", metricName, err))
		}
	}
	return counters
}

func calculateServerProtocol(protocol string, counters map[string]float64) {
	if strings.Count(protocol, "/") == 1 {
		httpProtocolVersion := strings.Split(protocol, "/")[1]
		httpProtocolVersion = strings.ReplaceAll(httpProtocolVersion, ".", "_")
		n := fmt.Sprintf("v%s", httpProtocolVersion)
		counters[n] = counters[n] + 1
	}
}

func getParsedRequest(request string) (method string, uri string, protocol string) {

	// Looking for capital letters, a space, anything, a space, capital letters, forward slash then anything.
	// Example: DELETE nginx_status HTTP/1.1
	regex, err := regexp.Compile(pattern)

	if err != nil {
		return
	}

	if regex.FindString(request) == "" {
		return
	}

	if len(request) == 0 {
		return
	}

	startURIIdx := strings.Index(request, spaceDelim)
	if startURIIdx == -1 {
		return
	}

	endURIIdx := strings.LastIndex(request, spaceDelim)
	// Ideally, endURIIdx should never be -1 here, as startURIIdx should have handled it already
	if endURIIdx == -1 {
		return
	}

	// For Example: GET /user/register?ahref<random>p' or '</random> HTTP/1.1

	// method -> GET
	method = request[:startURIIdx]

	// uri -> /user/register?ahref<random>p' or '</random>
	uri = request[startURIIdx+1 : endURIIdx]

	// protocol -> HTTP/1.1
	protocol = request[endURIIdx+1:]
	return
}

func getAverageMetricValue(metricValues []float64) float64 {
	value := 0.0

	if len(metricValues) > 0 {
		sort.Float64s(metricValues)
		metricValueSum := 0.0
		for _, metricValue := range metricValues {
			metricValueSum += metricValue
		}
		value = metricValueSum / float64(len(metricValues))
	}

	return value
}

func (c *NginxAccessLog) calculateHttpStatus(status string, counter map[string]float64) {

	if v, err := strconv.Atoi(status); err == nil {
		n := fmt.Sprintf("status.%dxx", v/100)
		counter[n] = counter[n] + 1
		switch v {
		case 403, 404, 500, 502, 503, 504:
			n := fmt.Sprintf("status.%d", v)
			counter[n] = counter[n] + 1
		case 499:
			n := "status.discarded"
			counter[n] = counter[n] + 1
		case 400:
			n := "request.malformed"
			counter[n] = counter[n] + 1
		}
	} else {
		c.logger.Log(fmt.Sprintf("Error getting status value from access logs, %v", err))
	}
}

func calculateUpstreamCacheStatus(status string, counter map[string]float64) {

	n := fmt.Sprintf("cache.%s", strings.ToLower(status))

	switch status {
	case "BYPASS", "EXPIRED", "HIT", "MISS", "REVALIDATED", "STALE", "UPDATING":
		counter[n] = counter[n] + 1
		return
	}
}

func calculateTimeMetricsMap(metricName string, times []float64, counter map[string]float64) {

	timeMetrics := map[string]float64{
		metricName:             0,
		metricName + ".count":  0,
		metricName + ".median": 0,
		metricName + ".max":    0,
		metricName + ".pctl95": 0,
	}

	for metric := range timeMetrics {

		metricType := metric[strings.LastIndex(metric, ".")+1:]

		counter[metric] = metrics.GetTimeMetrics(times, metricType)

	}

}

func isOtherMethod(method string) bool {
	return method != "method.post" &&
		method != "method.get" &&
		method != "method.delete" &&
		method != "method.put" &&
		method != "method.head" &&
		method != "method.options"
}

func getDefaultCounters() (map[string]float64, map[string]float64, map[string]float64) {
	httpCounters := map[string]float64{
		"gzip.ratio":              0,
		"method.delete":           0,
		"method.get":              0,
		"method.head":             0,
		"method.options":          0,
		"method.post":             0,
		"method.put":              0,
		"method.others":           0,
		"request.body_bytes_sent": 0,
		"request.bytes_sent":      0,
		"request.length":          0,
		"request.malformed":       0,
		"request.time":            0,
		"request.time.count":      0,
		"request.time.max":        0,
		"request.time.median":     0,
		"request.time.pctl95":     0,
		"status.403":              0,
		"status.404":              0,
		"status.500":              0,
		"status.502":              0,
		"status.503":              0,
		"status.504":              0,
		"status.discarded":        0,
		"status.1xx":              0,
		"status.2xx":              0,
		"status.3xx":              0,
		"status.4xx":              0,
		"status.5xx":              0,
		"v0_9":                    0,
		"v1_0":                    0,
		"v1_1":                    0,
		"v2":                      0,
	}

	upstreamCounters := map[string]float64{
		"upstream.connect.time":         0,
		"upstream.connect.time.count":   0,
		"upstream.connect.time.max":     0,
		"upstream.connect.time.median":  0,
		"upstream.connect.time.pctl95":  0,
		"upstream.header.time":          0,
		"upstream.header.time.count":    0,
		"upstream.header.time.max":      0,
		"upstream.header.time.median":   0,
		"upstream.header.time.pctl95":   0,
		"upstream.request.count":        0,
		"upstream.next.count":           0,
		"upstream.response.time":        0,
		"upstream.response.time.count":  0,
		"upstream.response.time.max":    0,
		"upstream.response.time.median": 0,
		"upstream.response.time.pctl95": 0,
		"upstream.response.length":      0,
		"upstream.status.1xx":           0,
		"upstream.status.2xx":           0,
		"upstream.status.3xx":           0,
		"upstream.status.4xx":           0,
		"upstream.status.5xx":           0,
	}

	upstreamCacheCounters := map[string]float64{
		"cache.bypass":      0,
		"cache.expired":     0,
		"cache.hit":         0,
		"cache.miss":        0,
		"cache.revalidated": 0,
		"cache.stale":       0,
		"cache.updating":    0,
	}

	return httpCounters, upstreamCounters, upstreamCacheCounters
}
