package metrics

import (
	"context"
	"sync"
	"time"

	"github.com/gogo/protobuf/types"

	"github.com/nginx/agent/sdk/v2/proto"
	"github.com/nginx/agent/v2/src/core/config"
)

type Collector interface {
	Collect(ctx context.Context, wg *sync.WaitGroup, m chan<- *proto.StatsEntity)
	UpdateConfig(config *config.Config)
}

type Source interface {
	Collect(ctx context.Context, wg *sync.WaitGroup, m chan<- *proto.StatsEntity)
}

type NginxSource interface {
	Collect(ctx context.Context, wg *sync.WaitGroup, m chan<- *proto.StatsEntity)
	Update(dimensions *CommonDim, collectorConf *NginxCollectorConfig)
	Stop()
}

type NginxCollectorConfig struct {
	NginxId            string
	StubStatus         string
	PlusAPI            string
	BinPath            string
	ConfPath           string
	CollectionInterval time.Duration
	AccessLogs         []string
	ErrorLogs          []string
	ClientVersion      int
}

func NewStatsEntity(dims []*proto.Dimension, samples []*proto.SimpleMetric) *proto.StatsEntity {
	return &proto.StatsEntity{
		Timestamp:     types.TimestampNow(),
		Dimensions:    dims,
		Simplemetrics: samples,
	}
}
