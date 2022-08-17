package services

import (
	"encoding/json"
	"io"

	log "github.com/sirupsen/logrus"

	"github.com/nginx/agent/sdk/v2"
	"github.com/nginx/agent/sdk/v2/grpc"
	"github.com/nginx/agent/sdk/v2/proto"
)

type CommandGrpcService struct {
	toClient         chan *proto.Command
	fromClient       chan *proto.Command
	downloadChannel  chan *proto.DataChunk
	uploadChannel    chan *proto.DataChunk
	registrationData *proto.AgentConnectRequest
	configData       *proto.NginxConfig
	nginxes          []*proto.NginxDetails
	configChunks     []*proto.DataChunk
}

func NewCommandService() *CommandGrpcService {
	return &CommandGrpcService{
		downloadChannel: make(chan *proto.DataChunk, 100),
		uploadChannel:   make(chan *proto.DataChunk, 100),
		toClient:        make(chan *proto.Command, 100),
		fromClient:      make(chan *proto.Command, 100),
	}
}

func (grpcService *CommandGrpcService) CommandChannel(stream proto.Commander_CommandChannelServer) error {
	log.Trace("CommandChannel")

	go grpcService.recvHandle(stream)

	for {
		select {
		case out := <-grpcService.toClient:
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
			log.Trace("command channel complete")
			return nil
		}
	}
}

func (grpcService *CommandGrpcService) recvHandle(server proto.Commander_CommandChannelServer) {
	for {
		cmd, err := server.Recv()
		if err != nil {
			// recommend handling error
			log.Debugf("Error in recvHandle %v", err)
			return
		}
		grpcService.handleCommand(cmd)
		grpcService.fromClient <- cmd
	}
}

// Download from the server to the client
func (grpcService *CommandGrpcService) Download(req *proto.DownloadRequest, download proto.Commander_DownloadServer) error {
	for {
		data := <-grpcService.downloadChannel
		log.Trace("Download")
		err := download.Send(data)
		if err != nil {
			return err
		}
	}
}

// Upload to the server from the client.
func (grpcService *CommandGrpcService) Upload(upload proto.Commander_UploadServer) error {
	for {
		chunk, err := upload.Recv()

		if err != nil && err != io.EOF {
			log.Warnf("Upload Recv Error: %v\n", err)
			return err
		}

		select {
		case grpcService.uploadChannel <- chunk:
			log.Infof("Received chunk")
			if chunk != nil {
				if _, header := chunk.Chunk.(*proto.DataChunk_Header); header {
					// if a header, reset chunks
					grpcService.configChunks = make([]*proto.DataChunk, 0)
				}

				grpcService.configChunks = append(grpcService.configChunks, chunk)
			}
		default:
		}

		if err == io.EOF {
			upload.SendAndClose(&proto.UploadStatus{Status: proto.UploadStatus_OK})
			return nil
		}
	}
}

func (grpcService *CommandGrpcService) handleCommand(cmd *proto.Command) {
	if cmd != nil {
		switch commandData := cmd.Data.(type) {
		// Step 1: Receive AgentConnectRequest from Agent
		case *proto.Command_AgentConnectRequest:
			log.Infof("Got agentConnectRequest from Agent %v", commandData)
			grpcService.registrationData = commandData.AgentConnectRequest
			grpcService.nginxes = commandData.AgentConnectRequest.Details
			// Step 2: Send AgentConnectResponse to Agent
			grpcService.sendAgentConnectResponse(cmd)
		case *proto.Command_NginxConfig:
			grpcService.configData = commandData.NginxConfig
		default:
			log.Tracef("unhandled command: %T", cmd.Data)
		}
	}
}

func (grpcService *CommandGrpcService) GetRegistration() *proto.AgentConnectRequest {
	return grpcService.registrationData
}

func (grpcService *CommandGrpcService) GetNginxes() []*proto.NginxDetails {
	return grpcService.nginxes
}

func (grpcService *CommandGrpcService) GetChunks() []*proto.DataChunk {
	return grpcService.configChunks
}

func (grpcService *CommandGrpcService) GetContents() (confFiles, auxFiles []*proto.File) {
	confFiles, auxFiles, err := sdk.GetNginxConfigFiles(grpcService.configData)
	if err != nil {
		return nil, nil
	}
	return confFiles, auxFiles
}

func (grpcService *CommandGrpcService) GetConfigs() *proto.NginxConfig {
	headers := []*proto.DataChunk_Header{}
	datas := []*proto.DataChunk_Data{}

	if len(grpcService.configChunks) == 0 {
		return nil
	}

	for _, in := range grpcService.configChunks {
		switch v := in.Chunk.(type) {
		case *proto.DataChunk_Header:
			headers = append(headers, v)
		case *proto.DataChunk_Data:
			datas = append(datas, v)
		default:
			log.Errorf("unexpected chunk type: %v", v)
		}
	}
	header := headers[0]
	log.Tracef("Showing header checksum %s", header.Header.GetChecksum())

	contents := make([]byte, 0)
	for _, data := range datas {
		contents = append(contents, data.Data.Data...)
	}

	var nginxConfig *proto.NginxConfig
	if err := json.Unmarshal(contents, &nginxConfig); err != nil {
		log.Warnf("unmarshal error")
		return nil
	}

	grpcService.configData = nginxConfig

	return grpcService.configData
}

func (grpcService *CommandGrpcService) sendAgentConnectResponse(cmd *proto.Command) {
	response := &proto.Command{
		Data: &proto.Command_AgentConnectResponse{
			AgentConnectResponse: &proto.AgentConnectResponse{
				AgentConfig: &proto.AgentConfig{
					Configs: &proto.ConfigReport{
						Meta: grpc.NewMessageMeta(cmd.Meta.MessageId),
						Configs: []*proto.ConfigDescriptor{
							{
								Checksum: "",
								// only one nginx id in this example
								NginxId:  cmd.GetAgentConnectRequest().GetDetails()[0].GetNginxId(),
								SystemId: cmd.GetAgentConnectRequest().GetMeta().GetSystemUid(),
							},
						},
					},
				},
				Status: &proto.AgentConnectStatus{
					// if conditions not met, should consider different status codes and filling in errors
					StatusCode: proto.AgentConnectStatus_CONNECT_OK,
					Message:    "Connected",
				},
			}},
		Meta: grpc.NewMessageMeta(cmd.Meta.MessageId),
		Type: proto.Command_NORMAL,
	}

	grpcService.toClient <- response
}
