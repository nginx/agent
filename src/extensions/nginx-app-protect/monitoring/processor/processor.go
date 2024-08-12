/**
 * Copyright (c) F5, Inc.
 *
 * This source code is licensed under the Apache License, Version 2.0 license found in the
 * LICENSE file in the root directory of this source tree.
 */

package processor

import (
	"context"
	"fmt"
	"regexp"
	"sync"

	"github.com/sirupsen/logrus"

	pb "github.com/nginx/agent/sdk/v2/proto/events"
	"github.com/nginx/agent/v2/src/core/metrics"
	"github.com/nginx/agent/v2/src/extensions/nginx-app-protect/monitoring"
)

const (
	componentName  = "processor"
	hostnameFormat = `(?m)\d+\-(.*)\:\d+\-.*`
)

// logging fields for the component
var componentLogFields = logrus.Fields{
	"component": componentName,
}

// Eventer is the interface implemented to generate an Event from a log entry.
type Eventer interface {
	// GetEvent will generate a protobuf Security Event
	GetEvent(hostPattern *regexp.Regexp, logger *logrus.Entry) (*pb.Event, error)
}

// Client for Processor with capability of logging.
type Client struct {
	logger      *logrus.Entry
	workers     int
	hostPattern *regexp.Regexp
	commonDims  *metrics.CommonDim
}

// GetClient gives you a Client for processing.
func GetClient(cfg *Config) (*Client, error) {
	var c Client

	c.logger = logrus.StandardLogger().WithFields(componentLogFields)
	if cfg.Logger != nil {
		c.logger = cfg.Logger.WithFields(componentLogFields)
	}

	c.workers = 1
	if cfg.Workers > 1 {
		c.workers = cfg.Workers
	}

	hostPattern, err := regexp.Compile(hostnameFormat)
	if err != nil {
		c.logger.Errorf("could not compile the hostname regex: %v", err)
		return &c, err
	}
	c.hostPattern = hostPattern

	if cfg.CommonDims == nil {
		c.logger.Warnf("common dimensions are not passed to NAP Monitoring processor")
		cfg.CommonDims = &metrics.CommonDim{}
	}
	c.commonDims = cfg.CommonDims

	return &c, nil
}

// processorWorker is a worker process to process events.
func (c *Client) processorWorker(ctx context.Context, wg *sync.WaitGroup, id int, collected <-chan *monitoring.RawLog, processed chan<- *pb.Event) {
	// defer wg.Done()

	c.logger.Debugf("Setting up Processor Worker: %d", id)

	for {
		select {
		case logline := <-collected:
			e, err := c.parse(logline.Origin, logline.Logline)
			if err != nil {
				c.logger.Errorf("%d: Error while parsing %s's log: %s, Error: %v", id, logline.Origin, logline.Logline, err)
				break
			}

			var event *pb.Event
			event, err = e.GetEvent(c.hostPattern, c.logger)
			if err != nil {
				c.logger.Errorf("%d: Error while generating event %s: %s", id, logline.Logline, err)
				break
			}

			if event.GetSecurityViolationEvent() == nil {
				c.logger.Errorf("expected SecurityViolationEvent, got %v, skipping sending", event)
				break
			}

			event.GetSecurityViolationEvent().SystemID = c.commonDims.SystemId
			event.GetSecurityViolationEvent().ParentHostname = c.commonDims.Hostname
			// Note: Currently using the Hostname of the machine as the Server Address as well, we may
			// change this to the Host present in Request Header
			event.GetSecurityViolationEvent().ServerAddr = c.commonDims.Hostname
			event.GetSecurityViolationEvent().InstanceTags = c.commonDims.InstanceTags
			event.GetSecurityViolationEvent().InstanceGroup = c.commonDims.InstanceGroup
			event.GetSecurityViolationEvent().DisplayName = c.commonDims.DisplayName

			c.logger.Debugf("worker %d: generated SecurityViolationEvent: %v", id, event)
			processed <- event

		case <-ctx.Done():
			c.logger.Debugf("worker %d: Context cancellation, processor is wrapping up...", id)
			return
		}
	}
}

// Process processes the raw log entries from collected chan into Security Events on processed chan.
func (c *Client) Process(ctx context.Context, _ *sync.WaitGroup, collected <-chan *monitoring.RawLog, processed chan<- *pb.Event) {
	// defer wg.Done()

	c.logger.Info("Setting up Processor")

	for id := 1; id <= c.workers; id++ {
		// wg.Add(1)
		go c.processorWorker(ctx, nil, id, collected, processed)
	}

	c.logger.Infof("Done setting up %v Processor Workers", c.workers)
}

func (c *Client) parse(waf monitoring.WAFType, logentry string) (Eventer, error) {
	switch waf {
	case monitoring.NAP:
		return parseNAP(logentry, c.logger)
	default:
		err := fmt.Errorf("could not parse log entry, invalid WAF type: %s", waf)
		c.logger.Error(err)
		return nil, err
	}
}
