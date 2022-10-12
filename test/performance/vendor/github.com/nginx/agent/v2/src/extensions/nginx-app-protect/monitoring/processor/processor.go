package processor

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"sync"

	"github.com/sirupsen/logrus"

	pb "github.com/nginx/agent/sdk/v2/proto/events"
	"github.com/nginx/agent/v2/src/extensions/nginx-app-protect/monitoring"
)

const (
	componentName = "processor"

	defualtSigDBFile             = "/opt/app_protect/db/PLC"
	defualtSigDBFilePollInterval = 10
	sigTable                     = "NEGSIG_SIGNATURES"
	sigIdCol                     = "sig_id"
	sigNameCol                   = "sig_name"
	errorPopulatingMapMsg        = "Error populating signature map: %s"
	hostnameFormat               = `(?m)\d+\-(.*)\:\d+\-.*`
)

var (
	// logging fields for the component
	componentLogFields = logrus.Fields{
		"component": componentName,
	}
	errFalsePositive = errors.New("false positive event detected, will not generate event")
)

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

	return &c, nil
}

// processorWorker is a worker process to process events.
func (c *Client) processorWorker(ctx context.Context, wg *sync.WaitGroup, id int, collected <-chan *monitoring.RawLog, processed chan<- *pb.Event) {
	defer wg.Done()

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
				if errors.Is(err, errFalsePositive) {
					c.logger.Debugf("%d: Event %s generated: %s", id, logline.Logline, err)
				} else {
					c.logger.Errorf("%d: Error while generating event %s: %s", id, logline.Logline, err)
				}
				break
			}

			c.logger.Debugf("%d: Generated Event: %s", id, event)
			processed <- event

		case <-ctx.Done():
			c.logger.Debugf("%d: Context cancellation, processor is wrapping up...", id)
			return
		}
	}
}

// Process processes the raw log entries from collected chan into Security Events on processed chan.
func (c *Client) Process(ctx context.Context, wg *sync.WaitGroup, collected <-chan *monitoring.RawLog, processed chan<- *pb.Event) {
	defer wg.Done()

	c.logger.Info("Setting up Processor")

	for id := 1; id <= c.workers; id++ {
		wg.Add(1)

		go c.processorWorker(ctx, wg, id, collected, processed)
	}

	c.logger.Infof("Done setting up %v Processor Workers", c.workers)
}

func (c *Client) parse(waf monitoring.WAFType, logentry string) (Eventer, error) {
	switch waf {
	case monitoring.NAPWAF:
		return parseNAPWAF(logentry, c.logger)
	default:
		err := fmt.Errorf("could not parse logentry, invalid WAF type: %s", waf)
		c.logger.Error(err)
		return nil, err
	}
}
