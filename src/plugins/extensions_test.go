package plugins

import (
	"context"
	"testing"
	"time"

	"github.com/nginx/agent/sdk/v2/proto"
	"github.com/nginx/agent/v2/src/core"
	"github.com/nginx/agent/v2/src/core/config"
	tutils "github.com/nginx/agent/v2/test/utils"
	"github.com/stretchr/testify/assert"
)

func TestExtensions_Process(t *testing.T) {
	testCases := []struct {
		testName      string
		extensionKey  string
		extensionName string
	}{
		{
			testName:      "Advanced Metrics",
			extensionKey:  config.AdvancedMetricsKey,
			extensionName: "Advanced Metrics Plugin",
		},
		{
			testName:      "Nginx App Protect",
			extensionKey:  config.NginxAppProtectKey,
			extensionName: "Nginx App Protect",
		},
	}

	// Create an agent config and initialize Viper config properties
	// based off of it, clean up when done.
	// TODO: The test agent config is going to be geting modified.
	// Need to either not run parallel or properly lock the code.
	_, _, cleanupFunc, err := tutils.CreateTestAgentConfigEnv()
	if err != nil {
		t.Fatalf(err.Error())
	}
	defer cleanupFunc()

	ctx, cancelCTX := context.WithCancel(context.Background())
	env := tutils.NewMockEnvironment()

	configuration, _ := config.GetConfig("1234")

	env.On("NewHostInfo", "agentVersion", &[]string{"locally-tagged", "tagged-locally"}).Return(&proto.HostInfo{})

	pluginUnderTest := NewExtensions(configuration, env)

	for _, tc := range testCases {
		t.Run(tc.testName, func(t *testing.T) {
			messagePipe := core.SetupMockMessagePipe(t, ctx, pluginUnderTest)

			// Assert that only the extensions plugin is registered
			assert.Equal(t, 1, len(messagePipe.GetPlugins()))
			assert.Equal(t, "Extensions Plugin", messagePipe.GetPlugins()[0].Info().Name())

			messagePipe.Process(core.NewMessage(core.EnableExtension, tc.extensionKey))
			messagePipe.Run()
			time.Sleep(250 * time.Millisecond)

			processedMessages := messagePipe.GetProcessedMessages()
			assert.Equal(t, 1, len(processedMessages))

			assert.Equal(t, 2, len(messagePipe.GetPlugins()))
			assert.Equal(t, "Extensions Plugin", messagePipe.GetPlugins()[0].Info().Name())
			assert.Equal(t, tc.extensionName, messagePipe.GetPlugins()[1].Info().Name())
		})
	}

	cancelCTX()
	pluginUnderTest.Close()
}
