package processor

import (
	"github.com/gogo/protobuf/types"
	"github.com/google/uuid"

	pb "github.com/nginx/agent/sdk/v2/proto/events"
)

// NewMetadata provides the event metadata for a given timestamp and correlationID.
func NewMetadata(timestamp *types.Timestamp, correlationID string) (*pb.Metadata, error) {
	var (
		metadata pb.Metadata
		err      error
	)

	id, err := uuid.NewUUID()
	if err != nil {
		return nil, err
	}

	metadata.Module = "Agent"
	metadata.UUID = id.String()
	metadata.CorrelationID = correlationID
	metadata.Timestamp = timestamp

	metadata.Type = "Nginx"
	metadata.Category = "AppProtect"

	metadata.EventLevel = "ERROR"

	return &metadata, err
}
