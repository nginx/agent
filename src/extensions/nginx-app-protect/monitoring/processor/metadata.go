/*
 * Copyright (C) F5 Inc. 2022
 * All rights reserved.
 *
 * No part of the software may be reproduced or transmitted in any
 * form or by any means, electronic or mechanical, for any purpose,
 * without express written permission of F5 Inc.
 */

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
