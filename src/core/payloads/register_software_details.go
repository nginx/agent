/**
 * Copyright (c) F5, Inc.
 *
 * This source code is licensed under the Apache License, Version 2.0 license found in the
 * LICENSE file in the root directory of this source tree.
 */

package payloads

import (
	"github.com/nginx/agent/sdk/v2/proto"
)

// RegisterWithDataplaneSoftwareDetailsPayload is an internal payload meant to be used as
// part of registration when there are plugins reporting software details.
type RegisterWithDataplaneSoftwareDetailsPayload struct {
	dataplaneSoftwareDetails *proto.DataplaneSoftwareDetails
	pluginName               string
}

// NewRegisterWithDataplaneSoftwareDetailsPayload returns a pointer to an instance of a
// RegisterWithDataplaneSoftwareDetailsPayload object.
func NewRegisterWithDataplaneSoftwareDetailsPayload(pluginName string, details *proto.DataplaneSoftwareDetails) *RegisterWithDataplaneSoftwareDetailsPayload {
	return &RegisterWithDataplaneSoftwareDetailsPayload{
		dataplaneSoftwareDetails: details,
		pluginName:               pluginName,
	}
}

func (r *RegisterWithDataplaneSoftwareDetailsPayload) GetPluginName() string {
	return r.pluginName
}

func (r *RegisterWithDataplaneSoftwareDetailsPayload) GetDataplaneSoftwareDetails() *proto.DataplaneSoftwareDetails {
	return r.dataplaneSoftwareDetails
}
