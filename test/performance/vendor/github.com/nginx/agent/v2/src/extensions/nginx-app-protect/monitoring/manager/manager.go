/**
 * Copyright (c) F5, Inc.
 *
 * This source code is licensed under the Apache License, Version 2.0 license found in the
 * LICENSE file in the root directory of this source tree.
 */

package manager

import (
	"context"
	"runtime"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"

	models "github.com/nginx/agent/sdk/v2/proto/events"
	"github.com/nginx/agent/v2/src/core/metrics"
	"github.com/nginx/agent/v2/src/extensions/nginx-app-protect/monitoring"
	"github.com/nginx/agent/v2/src/extensions/nginx-app-protect/monitoring/collector"
	"github.com/nginx/agent/v2/src/extensions/nginx-app-protect/monitoring/processor"
)

const (
	componentName              = "nginx-app-protect-monitoring"
	defaultCollectorBufferSize = 50000
	defaultProcessorBufferSize = 50000
)

type NginxAppProtectMonitoringConfig struct {
	CollectorBufferSize int           `mapstructure:"collector_buffer_size" yaml:"-"`
	ProcessorBufferSize int           `mapstructure:"processor_buffer_size" yaml:"-"`
	SyslogIP            string        `mapstructure:"syslog_ip" yaml:"-"`
	SyslogPort          int           `mapstructure:"syslog_port" yaml:"-"`
	ReportInterval      time.Duration `mapstructure:"report_interval" yaml:"-"`
	ReportCount         int           `mapstructure:"report_count" yaml:"-"`
}

type Manager struct {
	nginxAppProtectMonitoringConfig *NginxAppProtectMonitoringConfig
	syslogIP                        string
	syslogPort                      int
	logger                          *log.Entry

	collector   collector.Collector
	collectChan chan *monitoring.RawLog

	processor     *processor.Client
	processorChan chan *models.Event
}

func NewManager(nginxAppProtectMonitoringConfig *NginxAppProtectMonitoringConfig, commonDims *metrics.CommonDim) (*Manager, error) {
	m := &Manager{
		nginxAppProtectMonitoringConfig: nginxAppProtectMonitoringConfig,
		syslogIP:                        nginxAppProtectMonitoringConfig.SyslogIP,
		syslogPort:                      nginxAppProtectMonitoringConfig.SyslogPort,
	}

	err := m.init(commonDims)
	if err != nil {
		return nil, err
	}

	return m, nil
}

func (s *Manager) Name() string {
	return componentName
}

func (s *Manager) init(commonDims *metrics.CommonDim) error {
	var err error

	s.initLogging()
	s.logger.Infof("Initializing %s", componentName)

	err = s.initCollector()
	if err != nil {
		s.logger.Errorf("Could not initialize %s collector: %s", componentName, err)
		return err
	}

	err = s.initProcessor(commonDims)
	if err != nil {
		s.logger.Errorf("Could not initialize %s processor: %s", componentName, err)
		return err
	}

	return nil
}

func (s *Manager) initLogging() {
	s.logger = log.WithFields(log.Fields{
		"extension": componentName,
	})
}

func (s *Manager) initCollector() error {
	var err error

	s.logger.Infof("Initializing %s collector", componentName)

	s.collector, err = collector.NewNAPCollector(&collector.NAPConfig{
		SyslogIP:   s.syslogIP,
		SyslogPort: s.syslogPort,
		Logger:     s.logger,
	})

	if err != nil {
		s.logger.Errorf("Could not setup a %s collector. Got %v.", monitoring.NAP, err)
		return err
	}

	if s.nginxAppProtectMonitoringConfig.CollectorBufferSize > 0 {
		s.collectChan = make(chan *monitoring.RawLog, s.nginxAppProtectMonitoringConfig.CollectorBufferSize)
	} else {
		s.logger.Warnf("CollectorBufferSize cannot be zero or negative. Defaulting to %v", defaultCollectorBufferSize)
		s.collectChan = make(chan *monitoring.RawLog, defaultCollectorBufferSize)
	}

	return nil
}

func (s *Manager) initProcessor(commonDims *metrics.CommonDim) error {
	var err error

	s.logger.Infof("Initializing %s processor", componentName)

	s.processor, err = processor.GetClient(&processor.Config{
		Logger:     s.logger,
		Workers:    runtime.NumCPU(),
		CommonDims: commonDims,
	})

	if err != nil {
		s.logger.Errorf("Could not get a Processor Client: %s", err)
		return err
	}

	if s.nginxAppProtectMonitoringConfig.ProcessorBufferSize > 0 {
		s.processorChan = make(chan *models.Event, s.nginxAppProtectMonitoringConfig.ProcessorBufferSize)
	} else {
		s.logger.Warnf("ProcessorBufferSize cannot be zero or negative. Defaulting to %v", defaultProcessorBufferSize)
		s.processorChan = make(chan *models.Event, defaultProcessorBufferSize)
	}

	return nil
}

func (s *Manager) Run(ctx context.Context) {
	s.logger.Infof("Starting to run %s", componentName)

	chtx, cancel := context.WithCancel(ctx)
	defer cancel()

	waitGroup := &sync.WaitGroup{}
	waitGroup.Add(2)

	go s.collector.Collect(chtx, waitGroup, s.collectChan)
	go s.processor.Process(chtx, waitGroup, s.collectChan, s.processorChan)

	<-ctx.Done()
	s.logger.Infof("Received Context cancellation, %s is wrapping up...", componentName)

	waitGroup.Wait()

	s.logger.Infof("Context cancellation, %s wrapped up...", componentName)
}

// OutChannel returns processorChan channel which will publish events
func (m *Manager) OutChannel() chan *models.Event {
	return m.processorChan
}
