/**
 * Copyright (c) F5, Inc.
 *
 * This source code is licensed under the Apache License, Version 2.0 license found in the
 * LICENSE file in the root directory of this source tree.
 */

package extensions

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/nginx/agent/v2/src/core/config"
	"github.com/nginx/agent/v2/src/extensions/nginx-app-protect/monitoring/manager"
	tutils "github.com/nginx/agent/v2/test/utils"
)

func TestNAPMonitoring(t *testing.T) {
	type test struct {
		name          string
		conf          manager.NginxAppProtectMonitoringConfig
		error         bool
		errorContains string
	}
	tests := []test{
		{
			name: "valid config",
			conf: manager.NginxAppProtectMonitoringConfig{
				CollectorBufferSize: 1,
				ProcessorBufferSize: 1,
				SyslogIP:            "127.0.0.1",
				SyslogPort:          1234,
			},
			error: false,
		},
		{
			name: "invalid Syslog IP address",
			conf: manager.NginxAppProtectMonitoringConfig{
				CollectorBufferSize: 1,
				ProcessorBufferSize: 1,
				SyslogIP:            "no_such_host",
				SyslogPort:          1236,
			},
			error:         true,
			errorContains: "lookup",
		},
		{
			// Current behaviour is logging a warning and then
			// defaulting to the default buffer size = 50000 if the passed parameter is invalid
			name: "invalid buffer sizes",
			conf: manager.NginxAppProtectMonitoringConfig{
				CollectorBufferSize: -1,
				ProcessorBufferSize: -1,
				SyslogIP:            "127.0.0.1",
				SyslogPort:          4321,
			},
			error: false,
		},
		{
			name: "invalid Syslog port",
			conf: manager.NginxAppProtectMonitoringConfig{
				CollectorBufferSize: 1,
				ProcessorBufferSize: 1,
				SyslogIP:            "127.0.0.1",
				SyslogPort:          -4321,
			},
			error:         true,
			errorContains: "invalid port",
		},
	}

	env := tutils.GetMockEnv()

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := NewNAPMonitoring(env, &config.Config{}, test.conf)

			if test.error {
				assert.Contains(t, err.Error(), test.errorContains)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestNAPMonitoring_Info(t *testing.T) {
	pluginUnderTest, err := NewNAPMonitoring(tutils.GetMockEnv(), tutils.GetMockAgentConfig(), manager.NginxAppProtectMonitoringConfig{})

	assert.NoError(t, err)
	assert.Equal(t, "nap-monitoring", pluginUnderTest.Info().Name())
}
