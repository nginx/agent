package grpc

import (
	"fmt"
	"net"
	"os"
	"testing"

	"google.golang.org/grpc"

	sdkGRPC "github.com/nginx/agent/sdk/v2/grpc"
	sdkProto "github.com/nginx/agent/sdk/v2/proto"
	"github.com/nginx/agent/test/integration/utils"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"

	agent_config "github.com/nginx/agent/sdk/v2/agent/config"
)

const (
	GRPC_PROTOCOL = "tcp"
	NGINX_CONF    = `
#
# /etc/nginx-agent/nginx-agent.conf
#
# Configuration file for NGINX Agent.
#
# This file is to track agent configuration values that are meant to be statically set. There  
# are additional agent configuration values that are set via the API and agent install script
# which can be found in /var/lib/nginx-agent/agent-dynamic.conf. 

server:
    host: host.docker.internal
    grpcPort: %d

api:
    host: 0.0.0.0
    port: 8038

tls:
    enable: false
    skip_verify: true

log:
    level: info
    path: /var/log/nginx-agent/

nginx:
    exclude_logs: ""
    socket: "unix:/var/run/nginx-agent/nginx.sock"
    config_reload_monitoring_period: 30s

dataplane:
    status:
        poll_interval: 30s
        report_interval: 24h

metrics:
    bulk_size: 20
    report_interval: 1m
    collection_interval: 15s
    mode: aggregated

config_dirs: "/etc/nginx:/usr/local/etc/nginx:/usr/share/nginx/modules:/etc/nms:/usr/local/nginx"
`
)

func TestRegistrationAndConfigApply(t *testing.T) {
	grpcListener, grpcClose := createListener()
	defer grpcClose()

	srvOptions := sdkGRPC.DefaultServerDialOptions
	grpcServer := grpc.NewServer(srvOptions...)
	defer grpcServer.Stop()

	commandService := NewCommandService()
	sdkProto.RegisterCommanderServer(grpcServer, commandService)

	go func() {
		if err := grpcServer.Serve(grpcListener); err != nil {
			log.Fatal("error starting server")
		}
	}()

	nginxConfig := fmt.Sprintf(NGINX_CONF, grpcListener.Addr().(*net.TCPAddr).Port)
	err := os.WriteFile("nginx-agent.conf", []byte(nginxConfig), os.ModePerm)
	defer func() {
		err := os.Remove("./nginx-agent.conf")
		assert.NoError(t, err)
	}()
	assert.NoError(t, err)

	testContainer := utils.SetupTestContainerWithAgent(
		t,
		"grpc-registration-and-config-apply",
		"./nginx-agent.conf:/etc/nginx-agent/nginx-agent.conf",
		"The following core plugins have being registered",
	)

	// Validate that registration is complete

	expectedMessageCount := 0
	receivedAgentConnectRequest := false
	var nginxId string

messageLoop:
	for message := range commandService.fromClient {
		t.Logf("Message Received: %v", message)
		switch messageData := message.Data.(type) {
		case *sdkProto.Command_AgentConnectRequest:
			receivedAgentConnectRequest = true
			expectedMessageCount++
			assert.Equal(t, agent_config.GetDefaultFeatures(), messageData.AgentConnectRequest.GetMeta().GetAgentDetails().GetFeatures())
			nginxId = messageData.AgentConnectRequest.GetDetails()[0].GetNginxId()
		}

		if expectedMessageCount == 1 {
			break messageLoop
		}
	}

	assert.True(t, receivedAgentConnectRequest)

	tests := []struct {
		name                        string
		nginxConfigFileName         string
		messageId                   string
		expectedConfigApplyStatus   sdkProto.CommandStatusResponse_CommandStatus
		expectedMessage             string
		expectedNginxConfigFileName string
	}{
		{
			name:                        "successful config apply",
			nginxConfigFileName:         "nginx-config-apply-test.conf",
			messageId:                   "123",
			expectedConfigApplyStatus:   sdkProto.CommandStatusResponse_CMD_OK,
			expectedMessage:             "config applied successfully",
			expectedNginxConfigFileName: "nginx-config-apply-test.conf",
		},
		{
			name:                        "failed config apply - port already being used",
			nginxConfigFileName:         "nginx-config-apply-port-already-in-use-test.conf",
			messageId:                   "456",
			expectedConfigApplyStatus:   sdkProto.CommandStatusResponse_CMD_ERROR,
			expectedMessage:             "Config apply failed. Errors found during NGINX reload after applying a new configuration:",
			expectedNginxConfigFileName: "nginx-config-apply-test.conf",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Send config apply message

			err := commandService.SendConfigApply(nginxId, tt.nginxConfigFileName, tt.messageId)
			assert.NoError(t, err)

			// Validate that the config apply is successful

			receivedNginxConfigResponse := false
			expectedMessageCount = 0

		configApplyMessageLoop:
			for message := range commandService.fromClient {
				t.Logf("Message Received: %v", message)
				if message.GetMeta().GetMessageId() == tt.messageId {
					switch messageData := message.Data.(type) {
					case *sdkProto.Command_NginxConfigResponse:
						receivedNginxConfigResponse = true
						expectedMessageCount++
						assert.Equal(t, tt.expectedConfigApplyStatus, messageData.NginxConfigResponse.GetStatus().GetStatus())
						assert.Contains(t, messageData.NginxConfigResponse.GetStatus().GetMessage(), tt.expectedMessage)
					}
				}

				if expectedMessageCount == 1 {
					break configApplyMessageLoop
				}
			}

			assert.True(t, receivedNginxConfigResponse)

			// Validate that the nginx.conf file is updated correctly

			expectedNginxConfContent, err := os.ReadFile(tt.expectedNginxConfigFileName)
			assert.NoError(t, err)
			nginxConfContent, err := utils.ExecuteCommand(testContainer, []string{"cat", "/etc/nginx/nginx.conf"})
			assert.NoError(t, err)

			assert.Equal(t, string(expectedNginxConfContent), nginxConfContent)

			// Validate that the new configured server is working as expected

			responseContent, err := utils.ExecuteCommand(testContainer, []string{"curl", "http://127.0.0.1:8089/frontend1"})
			assert.NoError(t, err)
			assert.Contains(t, string(responseContent), "hello from http workload 1")

			// If config apply is successful, check that there is no errors in the NGINX Agent logs
			if tt.expectedConfigApplyStatus == sdkProto.CommandStatusResponse_CMD_OK {
				utils.TestAgentHasNoErrorLogs(t, testContainer)
			}
		})
	}
}

func createListener() (listener net.Listener, close func() error) {
	listen, err := net.Listen(GRPC_PROTOCOL, ":0")
	if err != nil {
		panic(err)
	}
	return listen, listen.Close
}
