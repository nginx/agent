/*
 * Copyright (C) F5 Inc. 2022
 * All rights reserved.
 *
 * No part of the software may be reproduced or transmitted in any
 * form or by any means, electronic or mechanical, for any purpose,
 * without express written permission of F5 Inc.
 */

package manager

import (
	"context"
	"fmt"
	"runtime"
	"sync"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/status"

	sdkGRPC "github.com/nginx/agent/sdk/v2/grpc"
	pb "github.com/nginx/agent/sdk/v2/proto"
	models "github.com/nginx/agent/sdk/v2/proto/events"
	"github.com/nginx/agent/v2/src/core/config"
	"github.com/nginx/agent/v2/src/extensions/nginx-app-protect/monitoring"
	"github.com/nginx/agent/v2/src/extensions/nginx-app-protect/monitoring/collector"
	"github.com/nginx/agent/v2/src/extensions/nginx-app-protect/monitoring/forwarder"
	"github.com/nginx/agent/v2/src/extensions/nginx-app-protect/monitoring/processor"
)

const (
	componentName = "security-events-manager"

	// Get subject name from config
	eventsSubjectName = "external.events.security"

	defaultCollectorBufferSize = 50000
	defaultProcessorBufferSize = 50000
)

type SecurityEventManager struct {
	config     *config.Config
	syslogIP   string
	syslogPort int
	logger     *log.Entry

	collector   collector.Collector
	collectChan chan *monitoring.RawLog

	processor     *processor.Client
	processorChan chan *models.Event

	forwarder  forwarder.Forwarder
	fwdrctx    context.Context
	fwdrcancel context.CancelFunc
}

func NewSecurityEventManager(config *config.Config) (*SecurityEventManager, error) {
	sem := &SecurityEventManager{
		config:     config,
		syslogIP:   config.NAPMonitoring.SyslogIP,
		syslogPort: config.NAPMonitoring.SyslogPort,
	}

	err := sem.init()
	if err != nil {
		return nil, err
	}

	return sem, nil
}

func (s *SecurityEventManager) Name() string {
	return componentName
}

func (s *SecurityEventManager) init() error {
	var err error

	s.initLogging()

	s.logger.Infof("Initializing %s", componentName)

	err = s.initCollector()
	if err != nil {
		s.logger.Errorf("Could not initialize %s collector: %s", componentName, err)
		return err
	}

	err = s.initProcessor()
	if err != nil {
		s.logger.Errorf("Could not initialize %s processor: %s", componentName, err)
		return err
	}

	return nil
}

func (s *SecurityEventManager) initLogging() {
	s.logger = log.WithFields(log.Fields{
		"extension": componentName,
	})
}

func (s *SecurityEventManager) initCollector() error {
	var err error

	s.logger.Infof("Initializing %s collector", componentName)

	s.collector, err = collector.NewNAPWAFCollector(&collector.NAPWAFConfig{
		SyslogIP:   s.syslogIP,
		SyslogPort: s.syslogPort,
		Logger:     s.logger,
	})

	if err != nil {
		s.logger.Errorf("Could not setup a %s collector. Got %v.", monitoring.NAPWAF, err)
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

func (s *SecurityEventManager) initProcessor() error {
	var err error

	s.logger.Infof("Initializing %s processor", componentName)

	s.processor, err = processor.GetClient(&processor.Config{
		Logger:  s.logger,
		Workers: runtime.NumCPU(),
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

func (s *SecurityEventManager) initForwarder(ctx context.Context) error {
	var err error

	s.logger.Infof("Initializing %s forwarder", componentName)

	server := fmt.Sprintf("%v:%v", s.config.Server.Host, s.config.Server.GrpcPort)

	var grpcDialOptions []grpc.DialOption
	grpcDialOptions = append(grpcDialOptions, sdkGRPC.DefaultClientDialOptions...)
	grpcDialOptions = append(grpcDialOptions, sdkGRPC.DataplaneConnectionDialOptions(s.config.Server.Token, sdkGRPC.NewMessageMeta(uuid.NewString()))...)

	secureCmdDialOpts, err := sdkGRPC.SecureDialOptions(
		s.config.TLS.Enable,
		s.config.TLS.Cert,
		s.config.TLS.Key,
		s.config.TLS.Ca,
		s.config.Server.Command,
		s.config.TLS.SkipVerify)
	if err != nil {
		s.logger.Fatalf("Failed to load secure command gRPC dial options: %v", err)
	}

	grpcDialOptions = append(grpcDialOptions, secureCmdDialOpts)

	conn, err := sdkGRPC.NewGrpcConnectionWithContext(ctx, server, grpcDialOptions)
	if err != nil {
		s.logger.Errorf("error while connecting to the grpcServer %v with dialOps %v: %v", server, grpcDialOptions, err)
		return err
	}
	s.logger.Infof("connecting to %s", server)

	client := pb.NewIngesterClient(conn)
	eventsChan, err := client.StreamEventReport(ctx, sdkGRPC.GetCallOptions()...)
	if err != nil {
		s.logger.Warnf("channel error: %s", err)
		statusRepresentation, ok := status.FromError(err)
		if ok {
			s.logger.Warnf("error creating events channel - GRPC Code: %s Message: %s",
				statusRepresentation.Code(), statusRepresentation.Message())
		} else {
			s.logger.Errorf("error creating events channel: %s", err)
		}
		return err
	}

	s.logger.Infof("Initializing %s forwarder", componentName)
	s.forwarder, err = forwarder.NewClient(eventsChan)
	if err != nil {
		s.logger.Errorf("error initializing the forwarder: %v", err)
		return err
	}
	return nil
}

func (s *SecurityEventManager) Run(ctx context.Context) {
	s.logger.Infof("Starting to run %s", componentName)

	chtx, cancel := context.WithCancel(ctx)
	defer cancel()

	waitGroup := &sync.WaitGroup{}
	waitGroup.Add(2)

	go s.collector.Collect(chtx, waitGroup, s.collectChan)
	go s.processor.Process(chtx, waitGroup, s.collectChan, s.processorChan)

	s.fwdrctx, s.fwdrcancel = context.WithCancel(ctx)
	defer s.fwdrcancel()

	fwdrWaitGroup := &sync.WaitGroup{}

	if err := s.initForwarder(s.fwdrctx); err != nil {
		s.logger.Errorf("Could not initialize %s forwarder: %s", componentName, err)
		s.logger.Infof("%s is exiting gracefully...", componentName)

		s.logger.Debugf("Cancelling %s's %s context", componentName, "default")
		cancel()

		s.logger.Debugf("Waiting for %s's %s context to exit gracefully...", componentName, "default")
		waitGroup.Wait()

		return
	}

	s.logger.Infof("Starting forwarding events to %s", eventsSubjectName)

	fwdrWaitGroup.Add(1)
	go s.forwarder.Forward(
		s.fwdrctx, fwdrWaitGroup,
		s.processorChan,
	)

	s.logger.Info("Successfully initialized a forwarder")

	<-ctx.Done()
	s.logger.Infof("Received Context cancellation, %s is wrapping up...", componentName)

	waitGroup.Wait()
	fwdrWaitGroup.Wait()

	s.logger.Infof("Context cancellation, %s wrapped up...", componentName)
}
