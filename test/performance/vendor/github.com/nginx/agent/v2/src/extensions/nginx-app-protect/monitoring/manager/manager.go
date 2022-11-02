package manager

import (
	"context"
	"runtime"
	"sync"

	log "github.com/sirupsen/logrus"

	models "github.com/nginx/agent/sdk/v2/proto/events"
	"github.com/nginx/agent/v2/src/core/config"
	"github.com/nginx/agent/v2/src/core/metrics"
	"github.com/nginx/agent/v2/src/extensions/nginx-app-protect/monitoring"
	"github.com/nginx/agent/v2/src/extensions/nginx-app-protect/monitoring/collector"
	"github.com/nginx/agent/v2/src/extensions/nginx-app-protect/monitoring/processor"
)

const (
	componentName              = "security-events-manager"
	defaultCollectorBufferSize = 50000
	defaultProcessorBufferSize = 50000
)

type Manager struct {
	config     *config.Config
	syslogIP   string
	syslogPort int
	logger     *log.Entry

	collector   collector.Collector
	collectChan chan *monitoring.RawLog

	processor     *processor.Client
	processorChan chan *models.Event
}

func NewManager(config *config.Config, commonDims *metrics.CommonDim) (*Manager, error) {
	m := &Manager{
		config:     config,
		syslogIP:   config.NAPMonitoring.SyslogIP,
		syslogPort: config.NAPMonitoring.SyslogPort,
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

	if s.config.NAPMonitoring.CollectorBufferSize > 0 {
		s.collectChan = make(chan *monitoring.RawLog, s.config.NAPMonitoring.CollectorBufferSize)
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

	if s.config.NAPMonitoring.ProcessorBufferSize > 0 {
		s.processorChan = make(chan *models.Event, s.config.NAPMonitoring.ProcessorBufferSize)
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
