package plugins

import (
	"testing"

	"github.com/nginx/agent/v2/src/core/config"

	"github.com/stretchr/testify/assert"

	tutils "github.com/nginx/agent/v2/test/utils"
)

func TestNAPMonitoring(t *testing.T) {
	type test struct {
		name string
		*config.Config
		error         bool
		errorContains string
	}
	tests := []test{
		{
			name: "valid config",
			Config: &config.Config{
				NAPMonitoring: config.NAPMonitoring{
					CollectorBufferSize: 1,
					ProcessorBufferSize: 1,
					SyslogIP:            "127.0.0.1",
					SyslogPort:          1234,
				},
			},
			error: false,
		},
		{
			name: "invalid Syslog IP address",
			Config: &config.Config{
				NAPMonitoring: config.NAPMonitoring{
					CollectorBufferSize: 1,
					ProcessorBufferSize: 1,
					SyslogIP:            "no_such_host",
					SyslogPort:          1234,
				},
			},
			error:         true,
			errorContains: "lookup",
		},
		{
			// Current behaviour is logging a warning and then
			// defaulting to the default buffer size = 50000 if the passed parameter is invalid
			name: "invalid buffer sizes",
			Config: &config.Config{
				NAPMonitoring: config.NAPMonitoring{
					CollectorBufferSize: -1,
					ProcessorBufferSize: -1,
					SyslogIP:            "127.0.0.1",
					SyslogPort:          4321,
				},
			},
			error: false,
		},
		{
			name: "invalid Syslog port",
			Config: &config.Config{
				NAPMonitoring: config.NAPMonitoring{
					CollectorBufferSize: 1,
					ProcessorBufferSize: 1,
					SyslogIP:            "127.0.0.1",
					SyslogPort:          -4321,
				},
			},
			error:         true,
			errorContains: "invalid port",
		},
	}

	env := tutils.GetMockEnv()

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := NewNAPMonitoring(env, test.Config)

			if test.error {
				assert.Contains(t, err.Error(), test.errorContains)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestNAPMonitoring_Info(t *testing.T) {
	pluginUnderTest, err := NewNAPMonitoring(tutils.GetMockEnv(), tutils.GetMockAgentConfig())

	assert.NoError(t, err)
	assert.Equal(t, "Nginx App Protect Monitor", pluginUnderTest.Info().Name())
}
