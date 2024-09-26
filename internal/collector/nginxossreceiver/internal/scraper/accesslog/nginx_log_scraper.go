// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

package accesslog

import (
	"context"
	"fmt"
	"strconv"
	"sync"
	"time"

	"go.opentelemetry.io/collector/pdata/pcommon"

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/stanza/entry"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/stanza/operator"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/stanza/operator/helper"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/stanza/pipeline"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/configtelemetry"
	"go.opentelemetry.io/collector/extension/experimental/storage"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/receiver"
	"go.opentelemetry.io/collector/receiver/scraperhelper"
	"go.opentelemetry.io/otel"
	metricSdk "go.opentelemetry.io/otel/sdk/metric"
	"go.uber.org/zap"

	"github.com/nginx/agent/v3/internal/collector/nginxossreceiver/internal/config"
	"github.com/nginx/agent/v3/internal/collector/nginxossreceiver/internal/metadata"
	"github.com/nginx/agent/v3/internal/collector/nginxossreceiver/internal/model"
	"github.com/nginx/agent/v3/internal/collector/nginxossreceiver/internal/scraper/accesslog/operator/input/file"
)

const Percentage = 100

type (
	NginxLogScraper struct {
		outChan <-chan []*entry.Entry
		cfg     *config.Config
		logger  *zap.Logger
		mb      *metadata.MetricsBuilder
		pipe    *pipeline.DirectedPipeline
		wg      *sync.WaitGroup
		cancel  context.CancelFunc
		entries []*entry.Entry
		mut     sync.Mutex
	}

	NginxMetrics struct {
		responseStatuses ResponseStatuses
	}

	ResponseStatuses struct {
		oneHundredStatusRange   int64
		twoHundredStatusRange   int64
		threeHundredStatusRange int64
		fourHundredStatusRange  int64
		fiveHundredStatusRange  int64
	}
)

var _ scraperhelper.Scraper = (*NginxLogScraper)(nil)

func NewScraper(
	settings receiver.Settings,
	cfg *config.Config,
) (*NginxLogScraper, error) {
	logger := settings.Logger
	logger.Info("Creating NGINX access log scraper")
	mb := metadata.NewMetricsBuilder(cfg.MetricsBuilderConfig, settings)

	operators := make([]operator.Config, 0)

	for _, accessLog := range cfg.AccessLogs {
		logger.Info("Adding access log file operator", zap.String("file_path", accessLog.FilePath))
		fileInputConfig := file.NewConfig()
		fileInputConfig.AccessLogFormat = accessLog.LogFormat
		fileInputConfig.Include = append(fileInputConfig.Include, accessLog.FilePath)

		inputCfg := operator.NewConfig(fileInputConfig)
		operators = append(operators, inputCfg)
	}

	stanzaPipeline, outChan, err := initStanzaPipeline(operators, settings.Logger)
	if err != nil {
		return nil, fmt.Errorf("init stanza pipeline: %w", err)
	}

	return &NginxLogScraper{
		cfg:     cfg,
		logger:  logger,
		mb:      mb,
		mut:     sync.Mutex{},
		outChan: outChan,
		pipe:    stanzaPipeline,
		wg:      &sync.WaitGroup{},
	}, nil
}

func (nls *NginxLogScraper) ID() component.ID {
	return component.NewID(metadata.Type)
}

func (nls *NginxLogScraper) Start(parentCtx context.Context, _ component.Host) error {
	nls.logger.Info("NGINX access log scraper started")
	ctx, cancel := context.WithCancel(parentCtx)
	nls.cancel = cancel

	err := nls.pipe.Start(storage.NewNopClient())
	if err != nil {
		return fmt.Errorf("stanza pipeline start: %w", err)
	}

	nls.wg.Add(1)
	go nls.runConsumer(ctx)

	return nil
}

func (nls *NginxLogScraper) Scrape(_ context.Context) (pmetric.Metrics, error) {
	nls.mut.Lock()
	defer nls.mut.Unlock()

	nginxMetrics := NginxMetrics{}

	for _, ent := range nls.entries {
		nls.logger.Debug("Scraping NGINX access log", zap.Any("entity", ent))
		item, ok := ent.Body.(*model.NginxAccessItem)
		if !ok {
			nls.logger.Warn("Failed to cast log entry to *model.NginxAccessItem", zap.Any("entry", ent.Body))
			continue
		}

		if v, err := strconv.Atoi(item.Status); err == nil {
			codeRange := fmt.Sprintf("%dxx", v/Percentage)

			switch codeRange {
			case "1xx":
				nginxMetrics.responseStatuses.oneHundredStatusRange++
			case "2xx":
				nginxMetrics.responseStatuses.twoHundredStatusRange++
			case "3xx":
				nginxMetrics.responseStatuses.threeHundredStatusRange++
			case "4xx":
				nginxMetrics.responseStatuses.fourHundredStatusRange++
			case "5xx":
				nginxMetrics.responseStatuses.fiveHundredStatusRange++
			default:
				nls.logger.Error("Unknown status range", zap.String("codeRange", codeRange))
				continue
			}
		}
	}

	nls.entries = make([]*entry.Entry, 0)
	timeNow := pcommon.NewTimestampFromTime(time.Now())

	nls.mb.RecordNginxHTTPResponseStatusDataPoint(
		timeNow,
		nginxMetrics.responseStatuses.oneHundredStatusRange,
		metadata.AttributeNginxStatusRange1xx,
	)
	nls.mb.RecordNginxHTTPResponseStatusDataPoint(
		timeNow,
		nginxMetrics.responseStatuses.twoHundredStatusRange,
		metadata.AttributeNginxStatusRange2xx,
	)
	nls.mb.RecordNginxHTTPResponseStatusDataPoint(
		timeNow,
		nginxMetrics.responseStatuses.threeHundredStatusRange,
		metadata.AttributeNginxStatusRange3xx,
	)
	nls.mb.RecordNginxHTTPResponseStatusDataPoint(
		timeNow,
		nginxMetrics.responseStatuses.fourHundredStatusRange,
		metadata.AttributeNginxStatusRange4xx,
	)
	nls.mb.RecordNginxHTTPResponseStatusDataPoint(
		timeNow,
		nginxMetrics.responseStatuses.fiveHundredStatusRange,
		metadata.AttributeNginxStatusRange5xx,
	)

	return nls.mb.Emit(), nil
}

func (nls *NginxLogScraper) Shutdown(_ context.Context) error {
	nls.logger.Info("Shutting down NGINX access log scraper")
	nls.cancel()
	nls.wg.Wait()

	return nls.pipe.Stop()
}

func initStanzaPipeline(
	operators []operator.Config,
	logger *zap.Logger,
) (*pipeline.DirectedPipeline, <-chan []*entry.Entry, error) {
	mp := otel.GetMeterProvider()
	if mp == nil {
		mp = metricSdk.NewMeterProvider()
		otel.SetMeterProvider(mp)
	}

	settings := component.TelemetrySettings{
		Logger:        logger,
		MeterProvider: mp,
		MetricsLevel:  configtelemetry.LevelNone,
	}

	emitter := helper.NewLogEmitter(settings)
	pipe, err := pipeline.Config{
		Operators:     operators,
		DefaultOutput: emitter,
	}.Build(settings)

	return pipe, emitter.OutChannel(), err
}

func (nls *NginxLogScraper) runConsumer(ctx context.Context) {
	nls.logger.Info("Starting NGINX access log receiver's consumer")
	defer nls.wg.Done()

	entryChan := nls.outChan
	for {
		select {
		case <-ctx.Done():
			nls.logger.Info("Closing NGINX access log receiver consumer")
			return
		case entries, ok := <-entryChan:
			if !ok {
				nls.logger.Info("Emitter channel closed, shutting down NGINX access log consumer")
				return
			}

			nls.mut.Lock()
			nls.entries = append(nls.entries, entries...)
			nls.mut.Unlock()
		}
	}
}
