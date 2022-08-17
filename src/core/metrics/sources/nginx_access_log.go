package sources

import (
	"context"
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/nginx/agent/sdk/v2/proto"
	log "github.com/sirupsen/logrus"

	"github.com/nginx/agent/v2/src/core"
	"github.com/nginx/agent/v2/src/core/metrics"
	"github.com/nginx/agent/v2/src/core/metrics/sources/tailer"
)

// This metrics source is used to tail the NGINX access logs to retrieve http metrics.

type NginxAccessLog struct {
	baseDimensions *metrics.CommonDim
	*namedMetric
	mu                 *sync.Mutex
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
	log.Trace("Creating NewNginxAccessLog")
	nginxAccessLog := &NginxAccessLog{
		baseDimensions,
		&namedMetric{namespace: namespace},
		&sync.Mutex{},
		make(map[string]context.CancelFunc),
		binary,
		nginxType,
		collectionInterval,
		[]*proto.StatsEntity{},
	}

	_, logs := binary.UpdatedAccessLogs()

	for logFile, logFormat := range logs {
		log.Infof("Adding access log tailer: %s", logFile)
		logCTX, fn := context.WithCancel(context.Background())
		nginxAccessLog.logs[logFile] = fn
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
		}

		_, logs := c.binary.UpdatedAccessLogs()

		for logFile, logFormat := range logs {
			if _, ok := c.logs[logFile]; !ok {
				log.Infof("Adding access log tailer: %s", logFile)
				logCTX, fn := context.WithCancel(context.Background())
				c.logs[logFile] = fn
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
	updated, logs := c.binary.UpdatedAccessLogs()

	if updated {
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
		c.binary.SetAccessLogUpdated(false)
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

	counters := getDefaultCounters()
	gzipRatios := []float64{}
	requestLengths := []float64{}
	requestTimes := []float64{}
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
				counters[n] = float64(v) + counters[n]
			} else {
				log.Warnf("Error getting body_bytes_sent value from access logs, %v", err)
			}

			if v, err := strconv.Atoi(access.BytesSent); err == nil {
				n := "request.bytes_sent"
				counters[n] = float64(v) + counters[n]
			} else {
				log.Warnf("Error getting bytes_sent value from access logs, %v", err)
			}

			if v, err := strconv.Atoi(access.GzipRatio); err == nil {
				gzipRatios = append(gzipRatios, float64(v))
			} else {
				log.Warnf("Error getting gzip_ratio value from access logs, %v", err)
			}

			if v, err := strconv.Atoi(access.RequestLength); err == nil {
				requestLengths = append(requestLengths, float64(v))
			} else {
				log.Warnf("Error getting request_length value from access logs, %v", err)
			}

			if v, err := strconv.ParseFloat(access.RequestTime, 64); err == nil {
				requestTimes = append(requestTimes, v)
			} else {
				log.Warnf("Error getting request_time value from access logs, %v", err)
			}

			if access.Request != "" {
				splitRequest := strings.Split(access.Request, " ")
				n := fmt.Sprintf("method.%s", strings.ToLower(splitRequest[0]))
				if isOtherMethod(n) {
					n = "method.others"
				}
				counters[n] = counters[n] + 1

				if access.ServerProtocol == "" {
					httpProtocolVersion := strings.Split(splitRequest[2], "/")[1]
					httpProtocolVersion = strings.ReplaceAll(httpProtocolVersion, ".", "_")
					n = fmt.Sprintf("v%s", httpProtocolVersion)
					counters[n] = counters[n] + 1
				}
			}

			if access.ServerProtocol != "" {
				httpProtocolVersion := strings.Split(access.ServerProtocol, "/")[1]
				httpProtocolVersion = strings.ReplaceAll(httpProtocolVersion, ".", "_")
				n := fmt.Sprintf("v%s", httpProtocolVersion)
				counters[n] = counters[n] + 1
			}

			// don't need the http status for NGINX Plus
			if c.nginxType == OSSNginxType {
				if v, err := strconv.Atoi(access.Status); err == nil {
					n := fmt.Sprintf("status.%dxx", v/100)
					counters[n] = counters[n] + 1
					if v == 403 || v == 404 || v == 500 || v == 502 || v == 503 || v == 504 {
						n := fmt.Sprintf("status.%d", v)
						counters[n] = counters[n] + 1
					}
					if v == 499 {
						n := "status.discarded"
						counters[n] = counters[n] + 1
					}
					if v == 400 {
						n := "request.malformed"
						counters[n] = counters[n] + 1
					}
				} else {
					log.Warnf("Error getting status value from access logs, %v", err)
				}
			}
			mu.Unlock()

		case <-tick.C:
			c.baseDimensions.NginxType = c.nginxType
			c.baseDimensions.PublishedAPI = logFile
			c.group = "http"

			mu.Lock()

			if len(requestLengths) > 0 {
				counters["request.length"] = getRequestLengthMetricValue(requestLengths)
			}
			if len(gzipRatios) > 0 {
				counters["gzip.ratio"] = getGzipRatioMetricValue(gzipRatios)
			}

			for key, value := range getRequestTimeMetrics(requestTimes) {
				counters[key] = value
			}

			simpleMetrics := c.convertSamplesToSimpleMetrics(counters)
			log.Tracef("Access log metrics collected: %v", simpleMetrics)

			// reset the counters
			counters = getDefaultCounters()
			gzipRatios = []float64{}
			requestLengths = []float64{}
			requestTimes = []float64{}

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

func getRequestTimeMetrics(requestTimes []float64) map[string]float64 {
	counters := make(map[string]float64)

	if len(requestTimes) > 0 {
		// Calculate request time average
		sort.Float64s(requestTimes)
		requestTimesSum := 0.0
		for _, requestTime := range requestTimes {
			requestTimesSum += requestTime
		}

		counters["request.time"] = requestTimesSum / float64(len(requestTimes))

		// Calculate request time count
		sort.Float64s(requestTimes)
		counters["request.time.count"] = float64(len(requestTimes))

		// Calculate request time max
		sort.Float64s(requestTimes)
		counters["request.time.max"] = requestTimes[len(requestTimes)-1]

		// Calculate request time median
		mNumber := len(requestTimes) / 2
		if len(requestTimes)%2 != 0 {
			counters["request.time.median"] = requestTimes[mNumber]
		} else {
			counters["request.time.median"] = (requestTimes[mNumber-1] + requestTimes[mNumber]) / 2
		}

		// Calculate request time 95 percentile
		index := int(math.RoundToEven(float64(0.95)*float64(len(requestTimes)))) - 1
		counters["request.time.pctl95"] = requestTimes[index]
	}

	return counters
}

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

func getDefaultCounters() map[string]float64 {
	return map[string]float64{
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
}
