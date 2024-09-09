/**
 * Copyright (c) F5, Inc.
 *
 * This source code is licensed under the Apache License, Version 2.0 license found in the
 * LICENSE file in the root directory of this source tree.
 */

package payloads

import (
	"testing"

	"github.com/nginx/agent/sdk/v2/proto"
	"github.com/stretchr/testify/assert"
)

func TestNewDataplaneSoftwareDetailsUpdate(t *testing.T) {
	pluginName := "test-plugin"
	details := &proto.DataplaneSoftwareDetails{}

	update := NewDataplaneSoftwareDetailsUpdate(pluginName, details)

	assert.NotNil(t, update, "NewDataplaneSoftwareDetailsUpdate should not return nil")
	assert.Equal(t, pluginName, update.GetPluginName(), "PluginName should match the one passed to the constructor")
	assert.Equal(t, details, update.GetDataplaneSoftwareDetails(), "DataplaneSoftwareDetails should match the one passed to the constructor")
}

func TestDataplaneSoftwareDetailsUpdate_GetPluginName(t *testing.T) {
	pluginName := "test-plugin"
	update := NewDataplaneSoftwareDetailsUpdate(pluginName, nil)

	assert.Equal(t, pluginName, update.GetPluginName(), "GetPluginName should return the correct plugin name")
}

func TestDataplaneSoftwareDetailsUpdate_GetDataplaneSoftwareDetails(t *testing.T) {
	details := &proto.DataplaneSoftwareDetails{}
	update := NewDataplaneSoftwareDetailsUpdate("test-plugin", details)

	assert.Equal(t, details, update.GetDataplaneSoftwareDetails(), "GetDataplaneSoftwareDetails should return the correct details")
}
