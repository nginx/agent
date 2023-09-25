package grpc

import (
	"encoding/json"
	"io"
	"os"

	"github.com/nginx/agent/sdk/v2/checksum"
	sdkGRPC "github.com/nginx/agent/sdk/v2/grpc"
	sdkProto "github.com/nginx/agent/sdk/v2/proto"
	"github.com/nginx/agent/sdk/v2/zip"
	log "github.com/sirupsen/logrus"
)

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

func (grpcService *CommandGrpcService) SendConfigApply(nginxId string, nginxConfigFileName string, messageId string) error {
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

	grpcService.downloadChannel <- &sdkProto.DataChunk{
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
		grpcService.downloadChannel <- &sdkProto.DataChunk{
			Chunk: &sdkProto.DataChunk_Data{
				Data: &sdkProto.ChunkedResourceChunk{
					ChunkId: int32(id),
					Data:    chunk,
					Meta:    metadata,
				},
			},
		}
	}

	grpcService.downloadChannel <- nil
	grpcService.toClient <- command

	return nil
}
