package grpc

import (
	"encoding/json"
	"io"
	"net"
	"os"
	"testing"

	"google.golang.org/grpc"

	"github.com/nginx/agent/sdk/v2/checksum"
	sdkGRPC "github.com/nginx/agent/sdk/v2/grpc"
	sdkProto "github.com/nginx/agent/sdk/v2/proto"
	"github.com/nginx/agent/sdk/v2/zip"
	"github.com/nginx/agent/test/integration/utils"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"

	agent_config "github.com/nginx/agent/sdk/v2/agent/config"
)

const (
	GRPC_ADDRESS  = ":54789"
	GRPC_PROTOCOL = "tcp"
)

func TestRegistrationAndConfigApply(t *testing.T) {
	grpcListener, grpcClose := createListener(GRPC_ADDRESS)
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

	testContainer := utils.SetupTestContainerWithAgent(t)

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

			err := sendConfigApply(nginxId, tt.nginxConfigFileName, tt.messageId, commandService)
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

type CommandGrpcService struct {
	toClient        chan *sdkProto.Command
	fromClient      chan *sdkProto.Command
	downloadChannel chan *sdkProto.DataChunk
	configChunks    []*sdkProto.DataChunk
}

func NewCommandService() *CommandGrpcService {
	return &CommandGrpcService{
		downloadChannel: make(chan *sdkProto.DataChunk, 100),
		toClient:        make(chan *sdkProto.Command, 100),
		fromClient:      make(chan *sdkProto.Command, 100),
	}
}

func (grpcService *CommandGrpcService) CommandChannel(stream sdkProto.Commander_CommandChannelServer) error {
	go grpcService.recvHandle(stream)

	for {
		select {
		case out := <-grpcService.toClient:
			log.Infof("CommandChannel: sending data %v", out)
			err := stream.Send(out)
			if err == io.EOF {
				log.Info("command channel EOF")
				return nil
			}
			if err != nil {
				log.Error("exception sending outgoing command: ", err)
				continue
			}
		case <-stream.Context().Done():
			log.Info("command channel complete")
			return nil
		}
	}
}

func (grpcService *CommandGrpcService) recvHandle(server sdkProto.Commander_CommandChannelServer) {
	for {
		cmd, err := server.Recv()
		if err != nil {
			continue
		}
		grpcService.handleCommand(cmd)
		grpcService.fromClient <- cmd
	}
}

// Download from the server to the client
func (grpcService *CommandGrpcService) Download(req *sdkProto.DownloadRequest, download sdkProto.Commander_DownloadServer) error {
	for {
		data := <-grpcService.downloadChannel
		log.Infof("Download: sending data %v", data)
		if data == nil {
			return nil
		}
		err := download.Send(data)
		if err != nil {
			log.Warnf("Download Send Error: %v\n", err)
		}
	}
}

// Upload to the server from the client.
func (grpcService *CommandGrpcService) Upload(upload sdkProto.Commander_UploadServer) error {
	var expectedNumberOfChunks int32 = 0
	var actualNumberOfChunks int32 = 0
	fileDone := false

LOOP:
	for {
		chunk, err := upload.Recv()
		log.Infof("Received chunk: %v", chunk)

		if err != nil {
			log.Warnf("Upload Recv Error: %v\n", err)
			if fileDone {
				break LOOP
			}
			return err

		}

		if chunk == nil {
			break LOOP
		} else {
			if chunk, header := chunk.Chunk.(*sdkProto.DataChunk_Header); header {
				expectedNumberOfChunks = chunk.Header.Chunks
				grpcService.configChunks = make([]*sdkProto.DataChunk, 0)
			} else {
				actualNumberOfChunks++
			}

			grpcService.configChunks = append(grpcService.configChunks, chunk)
		}

		if actualNumberOfChunks == expectedNumberOfChunks {
			fileDone = true
			actualNumberOfChunks = 0
			upload.SendAndClose(&sdkProto.UploadStatus{Status: sdkProto.UploadStatus_OK})
		}
	}

	return nil
}

func (grpcService *CommandGrpcService) handleCommand(cmd *sdkProto.Command) {
	if cmd != nil {
		switch cmd.Data.(type) {
		case *sdkProto.Command_AgentConnectRequest:
			grpcService.sendAgentConnectResponse(cmd)
		}
	}
}

func (grpcService *CommandGrpcService) sendAgentConnectResponse(cmd *sdkProto.Command) {
	log.Info("sendAgentConnectResponse")
	response := &sdkProto.Command{
		Data: &sdkProto.Command_AgentConnectResponse{
			AgentConnectResponse: &sdkProto.AgentConnectResponse{
				AgentConfig: &sdkProto.AgentConfig{
					Configs: &sdkProto.ConfigReport{
						Meta:    sdkGRPC.NewMessageMeta(cmd.Meta.MessageId),
						Configs: cmd.GetAgentConfig().GetConfigs().GetConfigs(),
					},
					Details: cmd.GetAgentConfig().GetDetails(),
				},
				Status: &sdkProto.AgentConnectStatus{
					StatusCode: sdkProto.AgentConnectStatus_CONNECT_OK,
					Message:    "Connected",
				},
			},
		},
		Meta: sdkGRPC.NewMessageMeta(cmd.Meta.MessageId),
		Type: sdkProto.Command_NORMAL,
	}

	grpcService.toClient <- response
}

func createListener(address string) (listener net.Listener, close func() error) {
	listen, err := net.Listen(GRPC_PROTOCOL, address)
	if err != nil {
		panic(err)
	}
	return listen, listen.Close
}

func sendConfigApply(nginxId string, nginxConfigFileName string, messageId string, commandService *CommandGrpcService) error {
	nginxConfigContents, err := os.ReadFile(nginxConfigFileName)
	if err != nil {
		return err
	}

	zipWriter, err := zip.NewWriter("/etc/nginx/")
	if err != nil {
		return err
	}

	reader, err := os.Open(nginxConfigFileName)
	if err != nil {
		return err
	}
	defer reader.Close()

	err = zipWriter.Add("nginx.conf", zip.DefaultFileMode, reader)
	if err != nil {
		return err
	}

	zippedConfig, err := zipWriter.Proto()
	if err != nil {
		return err
	}

	nginxConfig := &sdkProto.NginxConfig{
		Action: sdkProto.NginxConfigAction_APPLY,
		ConfigData: &sdkProto.ConfigDescriptor{
			NginxId:  nginxId,
			Checksum: checksum.Checksum(nginxConfigContents),
		},
		Zconfig:    zippedConfig,
		Zaux:       &sdkProto.ZippedFile{},
		AccessLogs: &sdkProto.AccessLogs{},
		ErrorLogs:  &sdkProto.ErrorLogs{},
		Ssl:        &sdkProto.SslCertificates{},
		DirectoryMap: &sdkProto.DirectoryMap{
			Directories: []*sdkProto.Directory{
				{
					Name: "/etc/nginx",
					Files: []*sdkProto.File{
						{
							Name:   "nginx.conf",
							Action: sdkProto.File_add,
						},
					},
				},
			},
		},
	}

	command := &sdkProto.Command{
		Meta: &sdkProto.Metadata{
			MessageId: messageId,
		},
		Type: sdkProto.Command_DOWNLOAD,
		Data: &sdkProto.Command_NginxConfig{
			NginxConfig: nginxConfig,
		},
	}

	payload, err := json.Marshal(nginxConfig)
	if err != nil {
		return err
	}

	metadata := sdkGRPC.NewMessageMeta(messageId)
	payloadChecksum := checksum.Checksum(payload)
	chunks := checksum.Chunk(payload, 4*1024)

	commandService.downloadChannel <- &sdkProto.DataChunk{
		Chunk: &sdkProto.DataChunk_Header{
			Header: &sdkProto.ChunkedResourceHeader{
				Chunks:    int32(len(chunks)),
				Checksum:  payloadChecksum,
				Meta:      metadata,
				ChunkSize: 4 * 1024,
			},
		},
	}

	for id, chunk := range chunks {
		commandService.downloadChannel <- &sdkProto.DataChunk{
			Chunk: &sdkProto.DataChunk_Data{
				Data: &sdkProto.ChunkedResourceChunk{
					ChunkId: int32(id),
					Data:    chunk,
					Meta:    metadata,
				},
			},
		}
	}

	commandService.downloadChannel <- nil
	commandService.toClient <- command

	return nil
}
