/**
 * Copyright (c) F5, Inc.
 *
 * This source code is licensed under the Apache License, Version 2.0 license found in the
 * LICENSE file in the root directory of this source tree.
 */

package grpc

import (
	"github.com/gogo/protobuf/types"

	sdk "github.com/nginx/agent/sdk/v2/proto"
)

var meta = &sdk.Metadata{}

func InitMeta(clientID, cloudAccountID string) {
	meta.ClientId = clientID
	meta.CloudAccountId = cloudAccountID
}

func NewMessageMeta(messageID string) *sdk.Metadata {
	return &sdk.Metadata{
		Timestamp:      types.TimestampNow(),
		ClientId:       meta.ClientId,
		CloudAccountId: meta.CloudAccountId,
		MessageId:      messageID,
	}
}

// NewMeta returns a new Metadata struct defined in the sdk/proto folder
func NewMeta(clientID, messageID, cloudID string) *sdk.Metadata {
	return &sdk.Metadata{
		Timestamp:      types.TimestampNow(),
		ClientId:       clientID,
		MessageId:      messageID,
		CloudAccountId: cloudID,
	}
}
