package cpuscraper

import (
	"context"
	"time"

	"github.com/nginx/agent/v3/internal/collector/containermetricsreceiver/internal/scraper/cpuscraper/internal/cgroup"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/receiver"
	"go.uber.org/zap"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/pdata/pmetric"

	"github.com/nginx/agent/v3/internal/collector/containermetricsreceiver/internal/scraper/cpuscraper/internal/metadata"
)

const BasePath = "/sys/fs/cgroup/"

type CPUScraper struct {
	cfg       *Config
	mb        *metadata.MetricsBuilder
	rb        *metadata.ResourceBuilder
	settings  receiver.Settings
	cpuSource *cgroup.CPUSource
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
	s.settings.Logger.Info("Starting container CPU scraper")
	s.cpuSource = cgroup.NewCPUSource(BasePath)
	return nil
}

func (s *CPUScraper) Shutdown(_ context.Context) error {
	return nil
}

func (s *CPUScraper) Scrape(context.Context) (pmetric.Metrics, error) {
	s.settings.Logger.Debug("Scraping container CPU metrics")
	if s.cpuSource == nil {
		s.cpuSource = cgroup.NewCPUSource(BasePath)
	}

	now := pcommon.NewTimestampFromTime(time.Now())

	stats, err := s.cpuSource.Collect()
	if err != nil {
		return pmetric.NewMetrics(), err
	}

	s.settings.Logger.Debug("Collected container CPU metrics", zap.Any("cpu", stats))

	s.mb.RecordSystemCPUUtilizationDataPoint(now, stats.User, metadata.AttributeStateUser)
	s.mb.RecordSystemCPUUtilizationDataPoint(now, stats.System, metadata.AttributeStateSystem)

	return s.mb.Emit(metadata.WithResource(s.rb.Emit())), nil
}
