package plugins

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/nginx/agent/sdk/v2/proto"
	"github.com/nginx/agent/v2/src/core"
	"github.com/nginx/agent/v2/src/core/config"

	tutils "github.com/nginx/agent/v2/test/utils"
)

func TestRegistration_Process(t *testing.T) {
	tests := []struct {
		name                 string
		expectedMessageCount int
	}{
		{
			name:                 "test registration",
			expectedMessageCount: 2,
		},
	}
	t.Parallel()
	for _, test := range tests {
		test := test
		t.Run(test.name, func(tt *testing.T) {
			tt.Parallel()

			binary := tutils.GetMockNginxBinary()
			binary.On("ReadConfig", mock.Anything, mock.Anything, mock.Anything).Return(&proto.NginxConfig{}, nil)
			env := tutils.GetMockEnvWithHostAndProcess()

			cfg := &config.Config{
				NginxAppProtect: config.NginxAppProtect{
					ReportInterval: time.Duration(1) * time.Second,
				},
			}

			pluginUnderTest := NewOneTimeRegistration(cfg, binary, env, &proto.Metadata{}, "0.0.0")
			pluginUnderTest.dataplaneSoftwareDetails[napPluginName] = &proto.DataplaneSoftwareDetails{
				Data: testNAPDetailsActive,
			}
			defer pluginUnderTest.Close()

			messagePipe := core.SetupMockMessagePipe(t, context.TODO(), pluginUnderTest)

			messagePipe.Run()
			messages := messagePipe.GetProcessedMessages()
			assert.Len(tt, messages, test.expectedMessageCount)

			assert.Equal(tt, messages[0].Topic(), core.CommRegister)
			// host info checked elsewhere
			assert.NotNil(tt, messages[0].Data())

			assert.Equal(tt, messages[1].Topic(), core.RegistrationCompletedTopic)
			assert.Nil(tt, messages[1].Data())
		})
	}
}

func TestRegistration_DataplaneReady(t *testing.T) {
	conf := tutils.GetMockAgentConfig()
	conf.NginxAppProtect = config.NginxAppProtect{ReportInterval: time.Duration(15) * time.Second}

	pluginUnderTest := NewOneTimeRegistration(conf, nil, tutils.GetMockEnv(), nil, "")

	assert.NoError(t, pluginUnderTest.dataplaneSoftwareDetailsReady())
}

func TestRegistration_Subscriptions(t *testing.T) {
	pluginUnderTest := NewOneTimeRegistration(tutils.GetMockAgentConfig(), nil, tutils.GetMockEnv(), nil, "")

	assert.Equal(t, []string{core.RegistrationCompletedTopic, core.RegisterWithDataplaneSoftwareDetails}, pluginUnderTest.Subscriptions())
}

func TestRegistration_Info(t *testing.T) {
	pluginUnderTest := NewOneTimeRegistration(tutils.GetMockAgentConfig(), nil, tutils.GetMockEnv(), nil, "")

	assert.Equal(t, "OneTimeRegistration", pluginUnderTest.Info().Name())
}
