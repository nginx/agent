/**
 * Copyright (c) F5, Inc.
 *
 * This source code is licensed under the Apache License, Version 2.0 license found in the
 * LICENSE file in the root directory of this source tree.
 */

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

	agent_config "github.com/nginx/agent/sdk/v2/agent/config"
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

	for _, test := range tests {
		test := test
		t.Run(test.name, func(tt *testing.T) {
			binary := tutils.GetMockNginxBinary()
			binary.On("ReadConfig", mock.Anything, mock.Anything, mock.Anything).Return(&proto.NginxConfig{}, nil)
			env := tutils.GetMockEnvWithHostAndProcess()

			cfg := &config.Config{
				Extensions: []string{agent_config.NginxAppProtectExtensionPlugin},
			}

			pluginUnderTest := NewOneTimeRegistration(cfg, binary, env, &proto.Metadata{}, "0.0.0")
			pluginUnderTest.dataplaneSoftwareDetails[agent_config.NginxAppProtectExtensionPlugin] = &proto.DataplaneSoftwareDetails{
				Data: testNAPDetailsActive,
			}
			defer pluginUnderTest.Close()

			messagePipe := core.SetupMockMessagePipe(t, context.TODO(), []core.Plugin{pluginUnderTest}, []core.ExtensionPlugin{})
			messagePipe.Run()

			assert.Eventually(
				tt,
				func() bool { return len(messagePipe.GetMessages()) == test.expectedMessageCount },
				time.Duration(5*time.Second),
				3*time.Millisecond,
			)

			messages := messagePipe.GetMessages()

			assert.Equal(tt, messages[0].Topic(), core.CommRegister)
			// host info checked elsewhere
			assert.NotNil(tt, messages[0].Data())

			assert.Equal(tt, messages[1].Topic(), core.RegistrationCompletedTopic)
			assert.Nil(tt, messages[1].Data())
		})
	}
}

func TestRegistration_areDataplaneSoftwareDetailsReady(t *testing.T) {
	conf := tutils.GetMockAgentConfig()
	conf.Extensions = []string{agent_config.NginxAppProtectExtensionPlugin}

	pluginUnderTest := NewOneTimeRegistration(conf, nil, tutils.GetMockEnv(), nil, "")
	softwareDetails := make(map[string]*proto.DataplaneSoftwareDetails)
	softwareDetails[agent_config.NginxAppProtectExtensionPlugin] = &proto.DataplaneSoftwareDetails{}
	pluginUnderTest.dataplaneSoftwareDetails = softwareDetails

	assert.NoError(t, pluginUnderTest.areDataplaneSoftwareDetailsReady())
}

func TestRegistration_Subscriptions(t *testing.T) {
	pluginUnderTest := NewOneTimeRegistration(tutils.GetMockAgentConfig(), nil, tutils.GetMockEnv(), nil, "")

	assert.Equal(t, []string{core.RegistrationCompletedTopic, core.DataplaneSoftwareDetailsUpdated}, pluginUnderTest.Subscriptions())
}

func TestRegistration_Info(t *testing.T) {
	pluginUnderTest := NewOneTimeRegistration(tutils.GetMockAgentConfig(), nil, tutils.GetMockEnv(), nil, "")

	assert.Equal(t, "OneTimeRegistration", pluginUnderTest.Info().Name())
}
