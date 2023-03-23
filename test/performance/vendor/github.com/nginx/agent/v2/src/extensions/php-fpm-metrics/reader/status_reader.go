package reader

import (
	"context"
	"fmt"
	"net/http"
	"time"
)

const (
	defaultPhpStatusTimeout   = 3 * time.Second
	statusPageJsonFmt         = "%s?json"
	phpFpmConnAcceptedMetric  = "php.fpm.conn.accepted"
	phpFpmQueueCurrentMetric  = "php.fpm.queue.current"
	phpFpmQueueMaxMetric      = "php.fpm.queue.max"
	phpFpmQueueLenMetric      = "php.fpm.queue.len"
	phpFpmProcIdleMetric      = "php.fpm.proc.idle"
	phpFpmProcActiveMetric    = "php.fpm.proc.active"
	phpFpmProcTotalMetric     = "php.fpm.proc.total"
	phpFpmProcMaxActiveMetric = "php.fpm.proc.max_active"
	phpFpmProcMaxChildMetric  = "php.fpm.proc.max_child"
	phpFpmSlowReqMetric       = "php.fpm.slow_req"
	phpFpmIdDim               = "php_id"
)

type StatusPageReader struct {
	ctx       context.Context
	client    *http.Client
	aggPeriod time.Duration
	endpoint  string
	//metricsChannels chan<- []*publisher.MetricSet
}

type PhpFpmStatusPage struct {
	Pool               string `json:"pool"`
	ProcessManager     string `json:"process manager"`
	StartTime          int64  `json:"start time"`
	StartSince         int64  `json:"start since"`
	AcceptedConn       int64  `json:"accepted conn"`
	ListenQueue        int64  `json:"listen queue"`
	MaxListenQueue     int64  `json:"max listen queue"`
	ListenQueueLen     int64  `json:"listen queue len"`
	IdleProcs          int64  `json:"idle processes"`
	ActiveProcs        int64  `json:"active processes"`
	TotalProcs         int64  `json:"total processes"`
	MaxActiveProcs     int64  `json:"max active processes"`
	MaxChildrenReached int64  `json:"max children reached"`
	SlowReqs           int64  `json:"slow requests"`
}

func NewStatusPageReader(ctx context.Context,
	aggPeriod time.Duration,
	endpoint string,
	//metricsChannels chan []*publisher.MetricSet
) *StatusPageReader {
	client := &http.Client{
		Timeout: defaultPhpStatusTimeout,
	}
	return &StatusPageReader{
		ctx:       ctx,
		aggPeriod: aggPeriod,
		client:    client,
		endpoint:  fmt.Sprintf(statusPageJsonFmt, endpoint),
		//metricsChannels: metricsChannels,
	}
}
