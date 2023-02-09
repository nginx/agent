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
	"math"
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

var httpRequestMetrics = []string{
	"request.time",
	"request.time.count",
	"request.time.max",
	"request.time.median",
	"request.time.pctl95",
}

func (c *NginxAccessLog) logStats(ctx context.Context, logFile, logFormat string) {
	logPattern := convertLogFormat(logFormat)
	log.Debugf("Collecting from: %s using format: %s", logFile, logFormat)
	log.Debugf("Pattern used for tailing logs: %s", logPattern)

	httpCounters, connCounters, headerCounters := getDefaultCounters()
	gzipRatios, requestLengths, requestTimes, connectTimes, headerTimes := []float64{}, []float64{}, []float64{}, []float64{}, []float64{}

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
				log.Error(err)
				continue
			}

			mu.Lock()
			if v, err := strconv.Atoi(access.BodyBytesSent); err == nil {
				n := "request.body_bytes_sent"
				httpCounters[n] = float64(v) + httpCounters[n]
			} else {
				log.Debugf("Error getting body_bytes_sent value from access logs: %v", err)
			}

			if v, err := strconv.Atoi(access.BytesSent); err == nil {
				n := "request.bytes_sent"
				httpCounters[n] = float64(v) + httpCounters[n]
			} else {
				log.Debugf("Error getting bytes_sent value from access logs: %v", err)
			}

			if v, err := strconv.Atoi(access.GzipRatio); err == nil {
				gzipRatios = append(gzipRatios, float64(v))
			} else {
				log.Debugf("Error getting gzip_ratio value from access logs: %v", err)
			}

			if v, err := strconv.Atoi(access.RequestLength); err == nil {
				requestLengths = append(requestLengths, float64(v))
			} else {
				log.Debugf("Error getting request_length value from access logs: %v", err)
			}

			if v, err := strconv.ParseFloat(access.RequestTime, 64); err == nil {
				requestTimes = append(requestTimes, v)
			} else {
				log.Debugf("Error getting request_time value from access logs: %v", err)
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

			for _, cTime := range strings.Split(access.UpstreamConnectTime, ", ") {
				// nginx uses '-' to represent TCP connection failures
				cTime = strings.ReplaceAll(cTime, "-", "0")

				if v, err := strconv.ParseFloat(cTime, 64); err == nil {
					connectTimes = append(connectTimes, v)
				} else {
					log.Debugf("Error getting upstream_connect_time value from access logs, %v", err)
				}
			}

			for _, hTime := range strings.Split(access.UpstreamHeaderTime, ", ") {
				// nginx uses '-' to represent TCP connection failures
				hTime = strings.ReplaceAll(hTime, "-", "0")

				if v, err := strconv.ParseFloat(hTime, 64); err == nil {
					headerTimes = append(headerTimes, v)
				} else {
					log.Debugf("Error getting upstream_header_time value from access logs: %v", err)
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
					log.Debugf("Error getting status value from access logs, %v", err)
				}
			}
			mu.Unlock()

		case <-tick.C:
			c.baseDimensions.NginxType = c.nginxType
			c.baseDimensions.PublishedAPI = logFile

			mu.Lock()

			if len(requestLengths) > 0 {
				httpCounters["request.length"] = getRequestLengthMetricValue(requestLengths)
			}

			if len(gzipRatios) > 0 {
				httpCounters["gzip.ratio"] = getGzipRatioMetricValue(gzipRatios)
			}

			for _, metricName := range httpRequestMetrics {
				httpCounters[metricName] = getTimeMetrics(metricName, requestTimes)
			}

			for metricName := range connCounters {
				connCounters[metricName] = getTimeMetrics(metricName, connectTimes)
			}

			for metricName := range headerCounters {
				headerCounters[metricName] = getTimeMetrics(metricName, headerTimes)
			}

			c.group = "http"
			simpleMetrics := c.convertSamplesToSimpleMetrics(httpCounters)

			c.group = ""
			simpleMetrics = append(simpleMetrics, c.convertSamplesToSimpleMetrics(connCounters)...)
			simpleMetrics = append(simpleMetrics, c.convertSamplesToSimpleMetrics(headerCounters)...)

			log.Tracef("Access log metrics collected: %v", simpleMetrics)

			// reset the counters
			httpCounters, connCounters, headerCounters = getDefaultCounters()
			gzipRatios, requestLengths, requestTimes, connectTimes, headerTimes = []float64{}, []float64{}, []float64{}, []float64{}, []float64{}

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

	re := regexp.MustCompile(`[0-9A-Za-z]+\s.+\s[0-9A-Za-z]+/[0-9A-Za-z]+`)

	if re.FindString(request) == "" {
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

func getRequestLengthMetricValue(requestLengths []float64) float64 {
	value := 0.0

	if len(requestLengths) > 0 {
		sort.Float64s(requestLengths)
		requestLengthSum := 0.0
		for _, requestLength := range requestLengths {
			requestLengthSum += requestLength
		}
		value = requestLengthSum / float64(len(requestLengths))
	}

	return value
}

func getGzipRatioMetricValue(gzipRatios []float64) float64 {
	value := 0.0

	if len(gzipRatios) > 0 {
		sort.Float64s(gzipRatios)
		gzipRatioSum := 0.0
		for _, gzipRatio := range gzipRatios {
			gzipRatioSum += gzipRatio
		}
		value = gzipRatioSum / float64(len(gzipRatios))
	}

	return value
}

func getTimeMetrics(metricName string, times []float64) float64 {
	if len(times) == 0 {
		return 0
	}

	metricType := metricName[strings.LastIndex(metricName, ".")+1:]

	switch metricType {
	case "time":
		// Calculate average
		sum := 0.0
		for _, t := range times {
			sum += t
		}
		return sum / float64(len(times))

	case "count":
		return float64(len(times))

	case "max":
		sort.Float64s(times)
		return times[len(times)-1]

	case "median":
		sort.Float64s(times)

		mNumber := len(times) / 2
		if len(times)%2 != 0 {
			return times[mNumber]
		} else {
			return (times[mNumber-1] + times[mNumber]) / 2
		}

	case "pctl95":
		sort.Float64s(times)

		index := int(math.RoundToEven(float64(0.95)*float64(len(times)))) - 1
		return times[index]
	}

	log.Debugf("Could not get time metrics for %s: invalid metric type", metricName)

	return 0
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

	upstreamConnectCounters := map[string]float64{
		"upstream.connect.time":        0,
		"upstream.connect.time.count":  0,
		"upstream.connect.time.max":    0,
		"upstream.connect.time.median": 0,
		"upstream.connect.time.pctl95": 0,
	}

	upstreamHeaderCounters := map[string]float64{
		"upstream.header.time":        0,
		"upstream.header.time.count":  0,
		"upstream.header.time.max":    0,
		"upstream.header.time.median": 0,
		"upstream.header.time.pctl95": 0,
	}

	return httpCounters, upstreamConnectCounters, upstreamHeaderCounters
}
