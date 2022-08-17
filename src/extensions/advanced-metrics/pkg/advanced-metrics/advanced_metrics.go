package advanced_metrics

import (
	"context"
	"fmt"
	"time"

	"github.com/nginx/agent/v2/src/extensions/advanced-metrics/aggregator"
	"github.com/nginx/agent/v2/src/extensions/advanced-metrics/ingester"
	"github.com/nginx/agent/v2/src/extensions/advanced-metrics/pkg/publisher"
	"github.com/nginx/agent/v2/src/extensions/advanced-metrics/reader"
	"github.com/nginx/agent/v2/src/extensions/advanced-metrics/tables"
	"github.com/nginx/agent/v2/src/extensions/advanced-metrics/tables/limits"
	"github.com/nginx/agent/v2/src/extensions/advanced-metrics/tables/schema"
	"golang.org/x/sync/errgroup"
)

// AggregatorConfig store aggregator configuration
type AggregatorConfig struct {
	// AggregationPeriod defines period of time after which metrics stored in StagingTable
	// will be aggregated into PriorityTable.
	AggregationPeriod time.Duration
	// PublishingPeriod defines period of time after which aggregated metrics from PriorityTable will be sent
	// over OutChannel in Publisher.
	PublishingPeriod time.Duration
}

// TableSizesLimits specified size limitation for staging and priority tables.
// This config values determines collapsing of dimensions per each internal table.
// Collapsing will change value of dimension to "AGGR" value.
type TableSizesLimits struct {
	// StagingTableMaxSize specify soft limit of samples in the staging table.
	StagingTableMaxSize int `mapstructure:"staging_table_max_size" yaml:"-"`
	// StagingTableThreshold specify threshold level of staging table.
	// This threshold specify number of elements after which collapsing of dimensions will occur.
	// Dimensions collapsing is specified by CollapsingLevel.
	// Collapsing of dimensions of sample which is to be added to staging table will occur when:
	// DimensionCollapsingLevel < (CurrentSizeOfStagingTable - StagingTableThreshold)/(StagingTableMaxSize - StagingTableThreshold) * 100
	// which is just a percentage of use of table above the threshold level.
	StagingTableThreshold int `mapstructure:"staging_table_threshold" yaml:"-"`
	// PriorityTableMaxSize specify soft limit of samples in the priority table.
	PriorityTableMaxSize int `mapstructure:"priority_table_max_size" yaml:"-"`
	// PriorityTableThreshold specify threshold level of priority table.
	// This threshold specify number of elements after which collapsing of dimensions will occur.
	// Dimensions collapsing is specified by CollapsingLevel.
	// Collapsing of dimensions of sample will occur only if sample will not fit into priority queue which size is equal PriorityTableThreshold.
	// Priority queue is prioritized by samples hit count.
	// Dimensions will be collapsed if:
	// DimensionCollapsingLevel < (CurrentSizeOfPriorityTable - PriorityTableThreshold)/(PriorityTableMaxSize - PriorityTableThreshold) * 100
	// which is just a percentage of use of table above the threshold level.
	PriorityTableThreshold int `mapstructure:"priority_table_threshold" yaml:"-"`
}

// Config keeps configuration for app centric metric server
type Config struct {
	// Unix socket address on which AppCentricMetrics should listen for incoming metrics
	Address string

	AggregatorConfig
	TableSizesLimits
}

// AdvancedMetrics is structure responsible for app centric metrics pipeline.
// This structure starts and stops all pipeline parts and exposes public interface for
// receiving aggregated app centric metrics.
type AdvancedMetrics struct {
	config Config

	metricsChannel chan []*publisher.MetricSet
	publisher      *publisher.Publisher
	reader         *reader.Reader
	ingester       *ingester.Ingester
	aggregator     *aggregator.Aggregator
}

func NewAdvancedMetrics(config Config, schema *schema.Schema) (*AdvancedMetrics, error) {
	l, err := limits.NewLimits(config.TableSizesLimits.StagingTableMaxSize, config.TableSizesLimits.StagingTableThreshold)
	if err != nil {
		return nil, fmt.Errorf("failed to create staging table limits: %w", err)
	}
	stagingTable := tables.NewStagingTable(schema, l)
	metricsChannel := make(chan []*publisher.MetricSet)
	publisher := publisher.New(metricsChannel, schema)
	reader := reader.NewReader(config.Address)
	ingester := ingester.NewIngester(reader.OutChannel(), stagingTable)
	l, err = limits.NewLimits(config.PriorityTableMaxSize, config.PriorityTableThreshold)
	if err != nil {
		return nil, fmt.Errorf("failed to create priority table limits: %w", err)
	}
	aggregator := aggregator.New(stagingTable, publisher, schema, l)

	return &AdvancedMetrics{
		metricsChannel: metricsChannel,
		publisher:      publisher,
		reader:         reader,
		ingester:       ingester,
		aggregator:     aggregator,
		config:         config,
	}, nil
}

// OutChannel returns publisher channel which will publish metrics sets
// in configured intervals(config.PublishingPeriod)
func (m *AdvancedMetrics) OutChannel() chan []*publisher.MetricSet {
	return m.metricsChannel
}

func (m *AdvancedMetrics) Run(ctx context.Context) error {
	defer func() {
		close(m.metricsChannel)
	}()

	group, ctx := errgroup.WithContext(ctx)

	group.Go(func() error {
		return m.reader.Run(ctx)
	})

	group.Go(func() error {
		m.ingester.Run(ctx)
		return nil
	})

	group.Go(func() error {
		aggregationTicker := time.NewTicker(m.config.AggregationPeriod)
		publishTicker := time.NewTicker(m.config.PublishingPeriod)
		defer func() {
			aggregationTicker.Stop()
			publishTicker.Stop()
		}()
		m.aggregator.Run(ctx, aggregationTicker.C, publishTicker.C)
		return nil
	})

	return group.Wait()
}
