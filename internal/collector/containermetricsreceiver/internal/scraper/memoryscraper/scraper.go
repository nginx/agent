package memoryscraper

import (
	"context"
	"time"

	"github.com/nginx/agent/v3/internal/collector/containermetricsreceiver/internal/scraper/memoryscraper/internal/cgroup"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.uber.org/zap"

	"github.com/nginx/agent/v3/internal/collector/containermetricsreceiver/internal/scraper/memoryscraper/internal/metadata"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/receiver"
)

const BasePath = "/sys/fs/cgroup/"

type CPUScraper struct {
	cfg          *Config
	mb           *metadata.MetricsBuilder
	rb           *metadata.ResourceBuilder
	settings     receiver.Settings
	memorySource *cgroup.MemorySource
}

func NewScraper(
	settings receiver.Settings,
	cfg *Config,
) *CPUScraper {
	logger := settings.Logger
	logger.Info("Creating container CPU scraper")

	mb := metadata.NewMetricsBuilder(cfg.MetricsBuilderConfig, settings)
	rb := mb.NewResourceBuilder()

	return &CPUScraper{
		settings: settings,
		cfg:      cfg,
		mb:       mb,
		rb:       rb,
	}
}

func (s *CPUScraper) ID() component.ID {
	return component.NewID(metadata.Type)
}

func (s *CPUScraper) Start(_ context.Context, _ component.Host) error {
	s.settings.Logger.Info("Starting container memory scraper")
	s.memorySource = cgroup.NewMemorySource(BasePath)
	return nil
}

func (s *CPUScraper) Shutdown(_ context.Context) error {
	return nil
}

func (s *CPUScraper) Scrape(context.Context) (pmetric.Metrics, error) {
	s.settings.Logger.Debug("Scraping container memory metrics")
	if s.memorySource == nil {
		s.memorySource = cgroup.NewMemorySource(BasePath)
	}

	now := pcommon.NewTimestampFromTime(time.Now())

	stats, err := s.memorySource.VirtualMemoryStatWithContext(context.Background())
	if err != nil {
		return pmetric.NewMetrics(), err
	}

	s.settings.Logger.Debug("Collected container memory metrics", zap.Any("metrics", stats))

	s.mb.RecordSystemMemoryUsageDataPoint(now, int64(stats.Used), metadata.AttributeStateUsed)
	s.mb.RecordSystemMemoryUsageDataPoint(now, int64(stats.Free), metadata.AttributeStateFree)

	return s.mb.Emit(metadata.WithResource(s.rb.Emit())), nil
}
