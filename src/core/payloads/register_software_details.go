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

// DataplaneSoftwareDetailsUpdate is an internal payload meant to be used to send software detail updates
type DataplaneSoftwareDetailsUpdate struct {
	dataplaneSoftwareDetails *proto.DataplaneSoftwareDetails
	pluginName               string
}

// NewDataplaneSoftwareDetailsUpdate returns a pointer to an instance of a
// DataplaneSoftwareDetailsUpdate object.
func NewDataplaneSoftwareDetailsUpdate(pluginName string, details *proto.DataplaneSoftwareDetails) *DataplaneSoftwareDetailsUpdate {
	return &DataplaneSoftwareDetailsUpdate{
		dataplaneSoftwareDetails: details,
		pluginName:               pluginName,
	}
}

func (r *DataplaneSoftwareDetailsUpdate) GetPluginName() string {
	return r.pluginName
}

func (r *DataplaneSoftwareDetailsUpdate) GetDataplaneSoftwareDetails() *proto.DataplaneSoftwareDetails {
	return r.dataplaneSoftwareDetails
}
