package forwarder

import (
	"context"
	"sync"

	"github.com/sirupsen/logrus"

	pb "github.com/nginx/agent/sdk/v2/proto"
	models "github.com/nginx/agent/sdk/v2/proto/events"
)

const (
	componentName = "forwarder"
)

var (
	// logging fields for the component
	componentLogFields = logrus.Fields{
		"component": componentName,
	}
)

// Forwarder is the interface implemented by forwarders who want to
// forward Security Event to a broker on a given subject.
type Forwarder interface {
	// Forward messages received on processed chan to the broker on the specified subject
	Forward(ctx context.Context, wg *sync.WaitGroup, processed <-chan *models.Event)
}

// Broker is the interface implemented by a Message Queue to publish
// messages on a specified subject.
type Broker interface {
	// Publish will publish to the broker and wait for an ACK
	Publish(subject string, data []byte) error
}

// Client for Forwarder with capability of logging.
type Client struct {
	logger  *logrus.Entry
	channel pb.MetricsService_StreamEventsClient
}

// NewClient gives you a Client for forwarding.
func NewClient(channel pb.MetricsService_StreamEventsClient) (*Client, error) {
	var c Client

	c.logger = logrus.WithFields(componentLogFields)
	c.channel = channel
	return &c, nil
}

// Forward messages received on processed chan to broker on specified subject.
func (c *Client) Forward(ctx context.Context, wg *sync.WaitGroup, processed <-chan *models.Event) {
	defer wg.Done()

	c.logger.Debugf("Forwarding security events to events service")

	for {
		select {
		case event := <-processed:
			c.logger.Debugf("Forwarding: %v", event)

			err := c.channel.Send(&models.EventReport{Events: []*models.Event{event}})
			if err != nil {
				c.logger.Errorf("Error while publishing event to the gRPC events service: %v, event: %v", err, event)
			}

		case <-ctx.Done():
			_, err := c.channel.CloseAndRecv()
			if err != nil {
				c.logger.Errorf("Error while closing events service gRPC stream: %v", err)
			}
			c.logger.Info("Context cancellation, forwarder is wrapping up...")
			return
		}
	}
}
