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
	"github.com/nginx/agent/v2/src/core/metrics/sources/tailer"

	log "github.com/sirupsen/logrus"
)

const (
	spaceDelim = " "
	pattern    = `[A-Z]+\s.+\s[A-Z]+/.+`
)

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
	buf                []*proto.StatsEntity
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
		[]*proto.StatsEntity{},
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

func (c *NginxAccessLog) Collect(ctx context.Context, wg *sync.WaitGroup, m chan<- *proto.StatsEntity) {
	defer wg.Done()
	c.collectLogStats(ctx, m)
}

func (c *NginxAccessLog) Update(dimensions *metrics.CommonDim, collectorConf *metrics.NginxCollectorConfig) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.baseDimensions = dimensions

	if c.collectionInterval != collectorConf.CollectionInterval {
		c.collectionInterval = collectorConf.CollectionInterval

		for f, fn := range c.logs {
			log.Infof("Removing access log tailer: %s", f)
			fn()
			delete(c.logs, f)
			delete(c.logFormats, f)
		}

		logs := c.binary.GetAccessLogs()

		for logFile, logFormat := range logs {
			if _, ok := c.logs[logFile]; !ok {
				log.Infof("Adding access log tailer: %s", logFile)
				logCTX, fn := context.WithCancel(context.Background())
				c.logs[logFile] = fn
				c.logFormats[logFile] = logFormat
				go c.logStats(logCTX, logFile, logFormat)
			}
		}
	}
}

func (c *NginxAccessLog) Stop() {
	for f, fn := range c.logs {
		log.Infof("Removing access log tailer: %s", f)
		fn()
		delete(c.logs, f)
	}
	log.Debugf("Stopping NginxAccessLog source for nginx id: %v", c.baseDimensions.NginxId)
}

func (c *NginxAccessLog) collectLogStats(ctx context.Context, m chan<- *proto.StatsEntity) {
	c.mu.Lock()
	defer c.mu.Unlock()
	logs := c.binary.GetAccessLogs()

	if c.binary.UpdateLogs(c.logFormats, logs) {
		log.Info("Access logs updated")
		// cancel any removed access logs
		for f, fn := range c.logs {
			if _, ok := logs[f]; !ok {
				log.Infof("Removing access log tailer: %s", f)
				fn()
				delete(c.logs, f)
			}
		}
		// add any new ones
		for logFile, logFormat := range logs {
			if _, ok := c.logs[logFile]; !ok {
				log.Infof("Adding access log tailer: %s", logFile)
				logCTX, fn := context.WithCancel(context.Background())
				c.logs[logFile] = fn
				go c.logStats(logCTX, logFile, logFormat)
			}
		}
	}

	for _, stat := range c.buf {
		m <- stat
	}
	c.buf = []*proto.StatsEntity{}
}

func (c *NginxAccessLog) logStats(ctx context.Context, logFile, logFormat string) {
	logPattern := convertLogFormat(logFormat)
	log.Debugf("Collecting from: %s using format: %s", logFile, logFormat)
	log.Debugf("Pattern used for tailing logs: %s", logPattern)

	httpCounters, upstreamCounters := getDefaultCounters()
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
			if err != nil {
				c.logger.Log(fmt.Sprintf("Error decoding access log entry, %v", err))
				continue
			}

			mu.Lock()
			if access.BodyBytesSent != "" {
				if v, err := strconv.Atoi(access.BodyBytesSent); err == nil {
					n := "request.body_bytes_sent"
					httpCounters[n] = float64(v) + httpCounters[n]
				} else {
					c.logger.Log(fmt.Sprintf("Error getting body_bytes_sent value from access logs, %v", err))
				}
			}

			if access.BytesSent != "" {
				if v, err := strconv.Atoi(access.BytesSent); err == nil {
					n := "request.bytes_sent"
					httpCounters[n] = float64(v) + httpCounters[n]
				} else {
					c.logger.Log(fmt.Sprintf("Error getting bytes_sent value from access logs, %v", err))
				}
			}

			if access.GzipRatio != "-" && access.GzipRatio != "" {
				if v, err := strconv.Atoi(access.GzipRatio); err == nil {
					gzipRatios = append(gzipRatios, float64(v))
				} else {
					c.logger.Log(fmt.Sprintf("Error getting gzip_ratio value from access logs, %v", err))
				}
			}

			if access.RequestLength != "" {
				if v, err := strconv.Atoi(access.RequestLength); err == nil {
					requestLengths = append(requestLengths, float64(v))
				} else {
					c.logger.Log(fmt.Sprintf("Error getting request_length value from access logs, %v", err))
				}
			}

			if access.RequestTime != "" {
				if v, err := strconv.ParseFloat(access.RequestTime, 64); err == nil {
					requestTimes = append(requestTimes, v)
				} else {
					c.logger.Log(fmt.Sprintf("Error getting request_time value from access logs, %v", err))
				}
			}

			if access.Request != "" {
				method, _, protocol := getParsedRequest(access.Request)
				n := fmt.Sprintf("method.%s", strings.ToLower(method))
				if isOtherMethod(n) {
					n = "method.others"
				}
				httpCounters[n] = httpCounters[n] + 1

				if access.ServerProtocol == "" {
					if strings.Count(protocol, "/") == 1 {
						httpProtocolVersion := strings.Split(protocol, "/")[1]
						httpProtocolVersion = strings.ReplaceAll(httpProtocolVersion, ".", "_")
						n = fmt.Sprintf("v%s", httpProtocolVersion)
						httpCounters[n] = httpCounters[n] + 1
					}
				}
			}

			if access.UpstreamConnectTime != "-" && access.UpstreamConnectTime != "" {
				if v, err := strconv.ParseFloat(access.UpstreamConnectTime, 64); err == nil {
					upstreamConnectTimes = append(upstreamConnectTimes, v)
				} else {
					c.logger.Log(fmt.Sprintf("Error getting upstream_connect_time value from access logs, %v", err))
				}
			}

			if access.UpstreamHeaderTime != "-" && access.UpstreamHeaderTime != "" {
				if v, err := strconv.ParseFloat(access.UpstreamHeaderTime, 64); err == nil {
					upstreamHeaderTimes = append(upstreamHeaderTimes, v)
				} else {
					c.logger.Log(fmt.Sprintf("Error getting upstream_header_time value from access logs, %v", err))
				}
			}

			if access.UpstreamResponseLength != "-" && access.UpstreamResponseLength != "" {
				if v, err := strconv.ParseFloat(access.UpstreamResponseLength, 64); err == nil {
					upstreamResponseLength = append(upstreamResponseLength, v)
				} else {
					c.logger.Log(fmt.Sprintf("Error getting upstream_response_length value from access logs, %v", err))
				}

			}

			if access.UpstreamResponseTime != "-" && access.UpstreamResponseTime != "" {
				if v, err := strconv.ParseFloat(access.UpstreamResponseTime, 64); err == nil {
					upstreamResponseTimes = append(upstreamResponseTimes, v)
				} else {
					c.logger.Log(fmt.Sprintf("Error getting upstream_response_time value from access logs, %v", err))
				}
			}

			if access.ServerProtocol != "" {
				if strings.Count(access.ServerProtocol, "/") == 1 {
					httpProtocolVersion := strings.Split(access.ServerProtocol, "/")[1]
					httpProtocolVersion = strings.ReplaceAll(httpProtocolVersion, ".", "_")
					n := fmt.Sprintf("v%s", httpProtocolVersion)
					httpCounters[n] = httpCounters[n] + 1
				}
			}

			if access.UpstreamStatus != "" && access.UpstreamStatus != "-" {
				if v, err := strconv.Atoi(access.UpstreamStatus); err == nil {
					n := fmt.Sprintf("upstream.status.%dxx", v/100)
					upstreamCounters[n] = upstreamCounters[n] + 1
				} else {
					log.Debugf("Error getting upstream status value from access logs, %v", err)
				}
			}
			// don't need the http status for NGINX Plus
			if c.nginxType == OSSNginxType {
				if v, err := strconv.Atoi(access.Status); err == nil {
					n := fmt.Sprintf("status.%dxx", v/100)
					httpCounters[n] = httpCounters[n] + 1
					if v == 403 || v == 404 || v == 500 || v == 502 || v == 503 || v == 504 {
						n := fmt.Sprintf("status.%d", v)
						httpCounters[n] = httpCounters[n] + 1
					}
					if v == 499 {
						n := "status.discarded"
						httpCounters[n] = httpCounters[n] + 1
					}
					if v == 400 {
						n := "request.malformed"
						httpCounters[n] = httpCounters[n] + 1
					}
				} else {
					c.logger.Log(fmt.Sprintf("Error getting status value from access logs, %v", err))
				}
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
				getTimeMetricsMap("request.time", requestTimes, httpCounters)
			}

			if len(upstreamConnectTimes) > 0 {
				getTimeMetricsMap("upstream.connect.time", upstreamConnectTimes, upstreamCounters)
			}

			if len(upstreamHeaderTimes) > 0 {
				getTimeMetricsMap("upstream.header.time", upstreamHeaderTimes, upstreamCounters)
			}

			if len(upstreamResponseTimes) > 0 {
				getTimeMetricsMap("upstream.response.time", upstreamResponseTimes, upstreamCounters)
			}

			if len(upstreamResponseLength) > 0 {
				upstreamCounters["upstream.response.length"] = getAverageMetricValue(upstreamResponseLength)
			}
			c.group = "http"
			simpleMetrics := c.convertSamplesToSimpleMetrics(httpCounters)

			c.group = ""
			simpleMetrics = append(simpleMetrics, c.convertSamplesToSimpleMetrics(upstreamCounters)...)

			log.Tracef("Access log metrics collected: %v", simpleMetrics)

			// reset the counters
			httpCounters, upstreamCounters = getDefaultCounters()
			gzipRatios, requestLengths, requestTimes, upstreamResponseLength, upstreamResponseTimes, upstreamConnectTimes, upstreamHeaderTimes = []float64{}, []float64{}, []float64{}, []float64{}, []float64{}, []float64{}, []float64{}

			c.buf = append(c.buf, metrics.NewStatsEntity(c.baseDimensions.ToDimensions(), simpleMetrics))

			mu.Unlock()

		case <-ctx.Done():
			err := ctx.Err()
			if err != nil {
				log.Errorf("NginxAccessLog: error in done context logStats %v", err)
			}
			log.Info("NginxAccessLog: logStats are done")
			return
		}
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

func getTimeMetricsMap(metricName string, times []float64, counter map[string]float64) {

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

// convertLogFormat converts log format into a pattern that can be parsed by the tailer
func convertLogFormat(logFormat string) string {
	newLogFormat := strings.ReplaceAll(logFormat, "$remote_addr", "%{IPORHOST:remote_addr}")
	newLogFormat = strings.ReplaceAll(newLogFormat, "$remote_user", "%{USERNAME:remote_user}")
	newLogFormat = strings.ReplaceAll(newLogFormat, "$time_local", `%{HTTPDATE:time_local}`)
	newLogFormat = strings.ReplaceAll(newLogFormat, "$status", "%{INT:status}")
	newLogFormat = strings.ReplaceAll(newLogFormat, "$body_bytes_sent", "%{NUMBER:body_bytes_sent}")
	newLogFormat = strings.ReplaceAll(newLogFormat, "$http_referer", "%{DATA:http_referer}")
	newLogFormat = strings.ReplaceAll(newLogFormat, "$http_user_agent", "%{DATA:http_user_agent}")
	newLogFormat = strings.ReplaceAll(newLogFormat, "$http_x_forwarded_for", "%{DATA:http_x_forwarded_for}")
	newLogFormat = strings.ReplaceAll(newLogFormat, "$bytes_sent", "%{NUMBER:bytes_sent}")
	newLogFormat = strings.ReplaceAll(newLogFormat, "$gzip_ratio", "%{DATA:gzip_ratio}")
	newLogFormat = strings.ReplaceAll(newLogFormat, "$server_protocol", "%{DATA:server_protocol}")
	newLogFormat = strings.ReplaceAll(newLogFormat, "$request_length", "%{INT:request_length}")
	newLogFormat = strings.ReplaceAll(newLogFormat, "$request_time", "%{DATA:request_time}")
	newLogFormat = strings.ReplaceAll(newLogFormat, "\"$request\"", "\"%{DATA:request}\"")
	newLogFormat = strings.ReplaceAll(newLogFormat, "$request ", "%{DATA:request} ")
	newLogFormat = strings.ReplaceAll(newLogFormat, "$upstream_connect_time", "%{DATA:upstream_connect_time}")
	newLogFormat = strings.ReplaceAll(newLogFormat, "$upstream_header_time", "%{DATA:upstream_header_time}")
	newLogFormat = strings.ReplaceAll(newLogFormat, "$upstream_response_time", "%{DATA:upstream_response_time}")
	newLogFormat = strings.ReplaceAll(newLogFormat, "$upstream_response_length", "%{DATA:upstream_response_length}")
	newLogFormat = strings.ReplaceAll(newLogFormat, "$upstream_status", "%{DATA:upstream_status}")
	newLogFormat = strings.ReplaceAll(newLogFormat, "[", "\\[")
	newLogFormat = strings.ReplaceAll(newLogFormat, "]", "\\]")
	return newLogFormat
}

func isOtherMethod(method string) bool {
	return method != "method.post" &&
		method != "method.get" &&
		method != "method.delete" &&
		method != "method.put" &&
		method != "method.head" &&
		method != "method.options"
}

func getDefaultCounters() (map[string]float64, map[string]float64) {
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

	return httpCounters, upstreamCounters
}
