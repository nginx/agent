/**
 * Copyright (c) F5, Inc.
 *
 * This source code is licensed under the Apache License, Version 2.0 license found in the
 * LICENSE file in the root directory of this source tree.
 */

package client

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/nginx/agent/sdk/v2"
	"github.com/nginx/agent/sdk/v2/checksum"
	"github.com/nginx/agent/sdk/v2/proto"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"

	sdkGRPC "github.com/nginx/agent/sdk/v2/grpc"
)

var (
	expectedNginxConfig = &proto.NginxConfig{
		Action: proto.NginxConfigAction_TEST,
		ConfigData: &proto.ConfigDescriptor{
			SystemId: "12345",
			NginxId:  "99999",
			Checksum: "",
		},
		Zconfig:      &proto.ZippedFile{},
		Zaux:         &proto.ZippedFile{},
		AccessLogs:   &proto.AccessLogs{},
		ErrorLogs:    &proto.ErrorLogs{},
		Ssl:          &proto.SslCertificates{},
		DirectoryMap: &proto.DirectoryMap{},
	}
	grpcServerMutex = &sync.Mutex{}
)

// Positive Test Cases

func TestCommander_ChuckSize(t *testing.T) {
	commanderClient := NewCommanderClient()
	commanderClient.WithChunkSize(1000)

	assert.Equal(t, 1000, commanderClient.ChunksSize())
}

func TestCommander_Server(t *testing.T) {
	commanderClient := NewCommanderClient()
	commanderClient.WithServer("test")

	assert.Equal(t, "test", commanderClient.Server())
}

func TestCommander_Recv(t *testing.T) {
	grpcServer, commandService, dialer := startCommanderMockServer()

	go func() {
		commandService.handler.toClient <- &proto.Command{Meta: &proto.Metadata{MessageId: "1234"}}
	}()

	ctx := context.TODO()

	commanderClient := createTestCommanderClient(dialer)
	err := commanderClient.Connect(ctx)
	assert.Nil(t, err)

	defer func() {
		commanderClient.Close()
		if err := stopMockServer(grpcServer, dialer); err != nil {
			t.Fatalf("Unable to stop grpc server")
		}
	}()

	select {
	case actual := <-commanderClient.Recv():
		if actual != nil {
			assert.Equal(t, "1234", actual.Meta().MessageId)
		}
	case <-time.After(1 * time.Second):
		t.Fatalf("No message received from commander")
	}
}

func TestCommander_Send(t *testing.T) {
	grpcServer, commandService, dialer := startCommanderMockServer()

	ctx := context.TODO()

	commanderClient := createTestCommanderClient(dialer)
	err := commanderClient.Connect(ctx)
	assert.Nil(t, err)

	defer func() {
		commanderClient.Close()
		if err := stopMockServer(grpcServer, dialer); err != nil {
			t.Fatalf("Unable to stop grpc server")
		}
	}()

	err = commanderClient.Send(ctx, MessageFromCommand(&proto.Command{Meta: &proto.Metadata{MessageId: "1234"}}))
	assert.Nil(t, err)

	select {
	case actual := <-commandService.handler.fromClient:
		if actual != nil {
			assert.Equal(t, "1234", actual.GetMeta().MessageId)
		}
	case <-time.After(1 * time.Second):
		t.Fatalf("No message received from commander")
	}
}

func TestCommander_Download(t *testing.T) {
	grpcServer, commandService, dialer := startCommanderMockServer()

	go func() {
		err := sendNginxConfigInChunks(commandService, expectedNginxConfig)
		if err != nil {
			t.Logf("Error converting nginx config to byte array: %v\n", err)
		}
	}()

	ctx := context.TODO()

	commanderClient := createTestCommanderClient(dialer)
	err := commanderClient.Connect(ctx)
	assert.Nil(t, err)

	defer func() {
		commanderClient.Close()
		if err := stopMockServer(grpcServer, dialer); err != nil {
			t.Fatalf("Unable to stop grpc server")
		}
	}()

	actual, err := commanderClient.Download(ctx, &proto.Metadata{MessageId: "1234"})

	assert.Nil(t, err)
	assert.Equal(t, expectedNginxConfig, actual)
}

func TestCommander_Upload(t *testing.T) {
	grpcServer, commandService, dialer := startCommanderMockServer()

	ctx := context.TODO()

	commanderClient := createTestCommanderClient(dialer)
	err := commanderClient.Connect(ctx)
	assert.Nil(t, err)

	defer func() {
		commanderClient.Close()
		if err := stopMockServer(grpcServer, dialer); err != nil {
			t.Fatalf("Unable to stop grpc server")
		}
	}()

	err = commanderClient.Upload(ctx, expectedNginxConfig, "1234")
	assert.Nil(t, err)

	chunks := []*proto.DataChunk{}
LOOP:
	for {
		select {
		case data := <-commandService.uploadChannel:
			if data == nil {
				break LOOP
			}
			chunks = append(chunks, data)
		default:
			break LOOP
		}
	}

	expectedNginxConfigByteArray, err := json.Marshal(expectedNginxConfig)
	assert.Nil(t, err)

	assert.Equal(t, 2, len(chunks))
	assert.Equal(t, "1234", chunks[0].Chunk.(*proto.DataChunk_Header).Header.Meta.MessageId)
	assert.Equal(t, "1234", chunks[1].Chunk.(*proto.DataChunk_Data).Data.Meta.MessageId)
	assert.Equal(t, int32(0), chunks[1].Chunk.(*proto.DataChunk_Data).Data.ChunkId)
	assert.Equal(t, expectedNginxConfigByteArray, chunks[1].Chunk.(*proto.DataChunk_Data).Data.Data)
}

// Negative Test Cases

func TestCommander_Connect_NoServer(t *testing.T) {
	ctx := context.TODO()

	var grpcDialOptions []grpc.DialOption
	grpcDialOptions = append(grpcDialOptions, sdkGRPC.DefaultClientDialOptions...)
	grpcDialOptions = append(grpcDialOptions, grpc.WithTransportCredentials(insecure.NewCredentials()))

	commanderClient := NewCommanderClient()
	commanderClient.WithServer("unknown")
	commanderClient.WithDialOptions(grpcDialOptions...)
	commanderClient.WithBackoffSettings(BackoffSettings{
		initialInterval: 100 * time.Millisecond,
		maxInterval:     100 * time.Millisecond,
		maxTimeout:      300 * time.Millisecond,
		sendMaxTimeout:  300 * time.Millisecond,
	})

	err := commanderClient.Connect(ctx)
	assert.NotNil(t, err)
}

func TestCommander_Recv_Reconnect(t *testing.T) {
	grpcServer, commandService, dialer := startCommanderMockServer()

	ctx := context.TODO()

	commanderClient := createTestCommanderClient(dialer)
	commanderClient.WithBackoffSettings(BackoffSettings{
		initialInterval: 100 * time.Millisecond,
		maxInterval:     100 * time.Millisecond,
		maxTimeout:      30 * time.Second,
		sendMaxTimeout:  30 * time.Second,
	})
	err := commanderClient.Connect(ctx)
	assert.Nil(t, err)

	// Restart server
	if err := stopMockServer(grpcServer, dialer); err != nil {
		t.Fatalf("Unable to stop grpc server")
	}
	grpcServer, commandService, dialer = startCommanderMockServer()

	go func() {
		commandService.handler.toClient <- &proto.Command{Meta: &proto.Metadata{MessageId: "1234"}}
	}()

	commanderClient.WithDialOptions(getDialOptions(dialer)...)

	defer func() {
		commanderClient.Close()
		if err := stopMockServer(grpcServer, dialer); err != nil {
			t.Fatalf("Unable to stop grpc server")
		}
	}()

	select {
	case actual := <-commanderClient.Recv():
		if actual != nil {
			assert.Equal(t, "1234", actual.Meta().MessageId)
		}
	case <-time.After(1 * time.Second):
		t.Fatalf("No message received from commander")
	}
}

func TestCommander_Send_ServerDies(t *testing.T) {
	grpcServer, _, dialer := startCommanderMockServer()

	ctx := context.TODO()

	commanderClient := createTestCommanderClient(dialer)
	err := commanderClient.Connect(ctx)
	assert.Nil(t, err)

	defer func() {
		commanderClient.Close()
	}()

	if err := stopMockServer(grpcServer, dialer); err != nil {
		t.Fatalf("Unable to stop grpc server")
	}

	err = commanderClient.Send(ctx, MessageFromCommand(&proto.Command{Meta: &proto.Metadata{MessageId: "1234"}}))
	assert.NotNil(t, err)
}

func TestCommander_Send_Reconnect(t *testing.T) {
	grpcServer, _, dialer := startCommanderMockServer()

	ctx := context.TODO()

	commanderClient := createTestCommanderClient(dialer)
	commanderClient.WithBackoffSettings(BackoffSettings{
		initialInterval: 100 * time.Millisecond,
		maxInterval:     100 * time.Millisecond,
		maxTimeout:      30 * time.Second,
		sendMaxTimeout:  30 * time.Second,
	})
	err := commanderClient.Connect(ctx)
	assert.Nil(t, err)

	// Restart server
	if err := stopMockServer(grpcServer, dialer); err != nil {
		t.Fatalf("Unable to stop grpc server")
	}
	grpcServer, _, dialer = startCommanderMockServer()
	commanderClient.WithDialOptions(getDialOptions(dialer)...)

	defer func() {
		commanderClient.Close()
		if err := stopMockServer(grpcServer, dialer); err != nil {
			t.Fatalf("Unable to stop grpc server")
		}
	}()

	err = commanderClient.Send(ctx, MessageFromCommand(&proto.Command{Meta: &proto.Metadata{MessageId: "1234"}}))
	assert.Nil(t, err)
}

func TestCommander_Download_ServerDies(t *testing.T) {
	grpcServer, _, dialer := startCommanderMockServer()

	ctx := context.TODO()

	commanderClient := createTestCommanderClient(dialer)
	err := commanderClient.Connect(ctx)
	assert.Nil(t, err)

	defer func() {
		commanderClient.Close()
	}()

	if err := stopMockServer(grpcServer, dialer); err != nil {
		t.Fatalf("Unable to stop grpc server")
	}

	_, err = commanderClient.Download(ctx, &proto.Metadata{MessageId: "1234"})
	assert.NotNil(t, err)
}

func TestCommander_Download_Reconnect(t *testing.T) {
	grpcServer, commandService, dialer := startCommanderMockServer()

	ctx := context.TODO()

	commanderClient := createTestCommanderClient(dialer)
	commanderClient.WithBackoffSettings(BackoffSettings{
		initialInterval: 100 * time.Millisecond,
		maxInterval:     100 * time.Millisecond,
		maxTimeout:      30 * time.Second,
		sendMaxTimeout:  30 * time.Second,
	})
	err := commanderClient.Connect(ctx)
	assert.Nil(t, err)

	if err := stopMockServer(grpcServer, dialer); err != nil {
		t.Fatalf("Unable to stop grpc server")
	}

	grpcServer, commandService, dialer = startCommanderMockServer()

	go func() {
		err := sendNginxConfigInChunks(commandService, expectedNginxConfig)
		if err != nil {
			t.Logf("Error converting nginx config to byte array: %v\n", err)
		}
	}()

	commanderClient.WithDialOptions(getDialOptions(dialer)...)

	defer func() {
		commanderClient.Close()
		if err := stopMockServer(grpcServer, dialer); err != nil {
			t.Fatalf("Unable to stop grpc server")
		}
	}()

	actual, err := commanderClient.Download(ctx, &proto.Metadata{MessageId: "1234"})

	assert.Nil(t, err)
	assert.Equal(t, expectedNginxConfig, actual)
}

func TestCommander_Download_MissingHeaderChunk(t *testing.T) {
	grpcServer, commandService, dialer := startCommanderMockServer()

	go func() {
		commandService.downloadChannel <- &proto.DataChunk{}
	}()

	ctx := context.TODO()

	commanderClient := createTestCommanderClient(dialer)
	err := commanderClient.Connect(ctx)
	assert.Nil(t, err)

	defer func() {
		commanderClient.Close()
		if err := stopMockServer(grpcServer, dialer); err != nil {
			t.Fatalf("Unable to stop grpc server")
		}
	}()

	_, err = commanderClient.Download(ctx, &proto.Metadata{MessageId: "1234"})
	assert.NotNil(t, err)
	assert.ErrorContains(t, err, "unexpected number of headers")
}

func TestCommander_Download_MultipleHeaderChunksSent(t *testing.T) {
	grpcServer, commandService, dialer := startCommanderMockServer()

	go func() {
		commandService.downloadChannel <- &proto.DataChunk{
			Chunk: &proto.DataChunk_Header{
				Header: &proto.ChunkedResourceHeader{},
			},
		}
		commandService.downloadChannel <- &proto.DataChunk{
			Chunk: &proto.DataChunk_Header{
				Header: &proto.ChunkedResourceHeader{},
			},
		}
	}()

	ctx := context.TODO()

	commanderClient := createTestCommanderClient(dialer)
	err := commanderClient.Connect(ctx)
	assert.Nil(t, err)

	defer func() {
		commanderClient.Close()
		if err := stopMockServer(grpcServer, dialer); err != nil {
			t.Fatalf("Unable to stop grpc server")
		}
	}()

	_, err = commanderClient.Download(ctx, &proto.Metadata{MessageId: "1234"})
	assert.NotNil(t, err)
	assert.ErrorContains(t, err, "unexpected number of headers")
}

func TestCommander_Download_ChecksumMismatch(t *testing.T) {
	grpcServer, commandService, dialer := startCommanderMockServer()

	go func() {
		commandService.downloadChannel <- &proto.DataChunk{
			Chunk: &proto.DataChunk_Header{
				Header: &proto.ChunkedResourceHeader{},
			},
		}
		commandService.downloadChannel <- &proto.DataChunk{
			Chunk: &proto.DataChunk_Data{
				Data: &proto.ChunkedResourceChunk{},
			},
		}
		commandService.downloadChannel <- &proto.DataChunk{}
	}()

	ctx := context.TODO()

	commanderClient := createTestCommanderClient(dialer)
	err := commanderClient.Connect(ctx)
	assert.Nil(t, err)

	defer func() {
		commanderClient.Close()
		if err := stopMockServer(grpcServer, dialer); err != nil {
			t.Fatalf("Unable to stop grpc server")
		}
	}()

	_, err = commanderClient.Download(ctx, &proto.Metadata{MessageId: "1234"})
	assert.NotNil(t, err)
	assert.ErrorContains(t, err, "download checksum mismatch")
}

func TestCommander_Download_InvalidObjectTypeDownloaded(t *testing.T) {
	grpcServer, commandService, dialer := startCommanderMockServer()

	go func() {
		sendInvalidObjectInChunks(commandService)
	}()

	ctx := context.TODO()

	commanderClient := createTestCommanderClient(dialer)
	err := commanderClient.Connect(ctx)
	assert.Nil(t, err)

	defer func() {
		commanderClient.Close()
		if err := stopMockServer(grpcServer, dialer); err != nil {
			t.Fatalf("Unable to stop grpc server")
		}
	}()

	_, err = commanderClient.Download(ctx, &proto.Metadata{MessageId: "1234"})
	assert.NotNil(t, err)
	assert.ErrorContains(t, err, "unable to unmarshal data")
}

func TestCommander_Upload_ServerDies(t *testing.T) {
	grpcServer, _, dialer := startCommanderMockServer()

	ctx := context.TODO()

	commanderClient := createTestCommanderClient(dialer)
	err := commanderClient.Connect(ctx)
	assert.Nil(t, err)

	if err := stopMockServer(grpcServer, dialer); err != nil {
		t.Fatalf("Unable to stop grpc server")
	}

	defer func() {
		commanderClient.Close()
	}()

	err = commanderClient.Upload(ctx, expectedNginxConfig, "1234")
	assert.NotNil(t, err)
}

func TestCommander_Upload_Reconnect(t *testing.T) {
	grpcServer, _, dialer := startCommanderMockServer()

	ctx := context.TODO()

	commanderClient := createTestCommanderClient(dialer)
	commanderClient.WithBackoffSettings(BackoffSettings{
		initialInterval: 100 * time.Millisecond,
		maxInterval:     100 * time.Millisecond,
		maxTimeout:      30 * time.Second,
		sendMaxTimeout:  30 * time.Second,
	})
	err := commanderClient.Connect(ctx)
	assert.Nil(t, err)

	// Restart server
	if err := stopMockServer(grpcServer, dialer); err != nil {
		t.Fatalf("Unable to stop grpc server")
	}
	grpcServer, commandService, dialer := startCommanderMockServer()
	commanderClient.WithDialOptions(getDialOptions(dialer)...)

	defer func() {
		commanderClient.Close()
		if err := stopMockServer(grpcServer, dialer); err != nil {
			t.Fatalf("Unable to stop grpc server")
		}
	}()

	err = commanderClient.Upload(ctx, expectedNginxConfig, "1234")
	assert.Nil(t, err)

	chunks := []*proto.DataChunk{}
LOOP:
	for {
		select {
		case data := <-commandService.uploadChannel:
			if data == nil {
				break LOOP
			}
			chunks = append(chunks, data)
		default:
			break LOOP
		}
	}

	expectedNginxConfigByteArray, err := json.Marshal(expectedNginxConfig)
	assert.Nil(t, err)

	assert.Equal(t, 2, len(chunks))
	assert.Equal(t, "1234", chunks[0].Chunk.(*proto.DataChunk_Header).Header.Meta.MessageId)
	assert.Equal(t, "1234", chunks[1].Chunk.(*proto.DataChunk_Data).Data.Meta.MessageId)
	assert.Equal(t, int32(0), chunks[1].Chunk.(*proto.DataChunk_Data).Data.ChunkId)
	assert.Equal(t, expectedNginxConfigByteArray, chunks[1].Chunk.(*proto.DataChunk_Data).Data.Data)
}

// Helper Functions

type handlerFunc func(proto.Commander_CommandChannelServer, *sync.WaitGroup)

type handler struct {
	recvHandleFunc handlerFunc
	sendHandleFunc handlerFunc
	toClient       chan *proto.Command
	fromClient     chan *proto.Command
}

type mockCommanderService struct {
	sync.RWMutex
	handler         *handler
	downloadChannel chan *proto.DataChunk
	uploadChannel   chan *proto.DataChunk
}

func (c *mockCommanderService) CommandChannel(server proto.Commander_CommandChannelServer) error {
	wg := &sync.WaitGroup{}
	h := c.ensureHandler()
	wg.Add(2)

	recvHandleFunc := h.recvHandleFunc
	if recvHandleFunc == nil {
		recvHandleFunc = h.recvHandle
	}
	sendHandleFunc := h.sendHandleFunc
	if sendHandleFunc == nil {
		sendHandleFunc = h.sendHandle
	}

	go recvHandleFunc(server, wg)
	go sendHandleFunc(server, wg)

	wg.Wait()

	return nil
}

func (c *mockCommanderService) Download(request *proto.DownloadRequest, server proto.Commander_DownloadServer) error {
	for {
		data := <-c.downloadChannel
		fmt.Printf("Download Send: %v\n", data)
		if data != nil {
			err := server.Send(data)
			if err != nil {
				fmt.Printf("Download Send Error: %v\n", err)
				return err
			}
		}
	}
}

func (c *mockCommanderService) Upload(server proto.Commander_UploadServer) error {
	for {
		chunk, err := server.Recv()
		fmt.Printf("Upload Recv: %v\n", chunk)

		if err != nil && err != io.EOF {
			fmt.Printf("Upload Recv Error: %v\n", err)
			return err
		}

		select {
		case c.uploadChannel <- chunk:
		default:
		}

		if err == io.EOF {
			server.SendAndClose(&proto.UploadStatus{Status: proto.UploadStatus_OK})
			return nil
		}
	}
}

func (c *mockCommanderService) ensureHandler() *handler {
	c.RLock()
	if c.handler == nil {
		c.RUnlock()
		c.Lock()
		defer c.Unlock()
		c.handler = &handler{}
		c.handler.toClient = make(chan *proto.Command)
		c.handler.fromClient = make(chan *proto.Command)
		return c.handler
	}
	defer c.RUnlock()
	return c.handler
}

func (h *handler) recvHandle(server proto.Commander_CommandChannelServer, wg *sync.WaitGroup) {
	for {
		cmd, err := server.Recv()
		if cmd != nil {
			fmt.Printf("Recv Command: %v\n", cmd)
			if err != nil {
				fmt.Printf("Recv Command Error: %v\n", err)
				wg.Done()
				return
			}
			h.fromClient <- cmd
		}
	}
}

func (h *handler) sendHandle(server proto.Commander_CommandChannelServer, wg *sync.WaitGroup) {
	for {
		cmd := <-h.toClient
		if cmd != nil {
			err := server.Send(cmd)
			fmt.Printf("Send Command: %v\n", cmd)
			if err != nil {
				fmt.Printf("Send Command Error: %v\n", err)
				wg.Done()
				return
			}
		}

	}
}

func startCommanderMockServer() (*grpc.Server, *mockCommanderService, func(context.Context, string) (net.Conn, error)) {
	listener := bufconn.Listen(1024 * 1024)
	grpcServer := grpc.NewServer(sdkGRPC.DefaultServerDialOptions...)
	commandService := &mockCommanderService{}
	commandService.handler = commandService.ensureHandler()
	commandService.downloadChannel = make(chan *proto.DataChunk)
	commandService.uploadChannel = make(chan *proto.DataChunk, 3)
	proto.RegisterCommanderServer(grpcServer, commandService)

	go func() {
		if err := grpcServer.Serve(listener); err != nil {
			fmt.Printf("Error starting mock GRPC server: %v\n", err)
		}
	}()

	return grpcServer, commandService, func(context.Context, string) (net.Conn, error) {
		return listener.Dial()
	}
}

func stopMockServer(server *grpc.Server, dialer func(context.Context, string) (net.Conn, error)) error {
	ctx := context.TODO()
	conn, err := grpc.DialContext(ctx, "", grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithContextDialer(dialer))
	grpcServerMutex.Lock()
	defer grpcServerMutex.Unlock()
	server.Stop()

	err = sdk.WaitUntil(ctx, 100*time.Millisecond, 100*time.Millisecond, 1*time.Second, func() error {
		state := conn.GetState()
		if state.String() != "TRANSIENT_FAILURE" {
			return errors.New("Still waiting for server to stop")
		}
		return err
	})

	return err
}

func createTestCommanderClient(dialer func(context.Context, string) (net.Conn, error)) Commander {
	commanderClient := NewCommanderClient()
	commanderClient.WithServer("bufnet")
	commanderClient.WithDialOptions(getDialOptions(dialer)...)
	commanderClient.WithBackoffSettings(BackoffSettings{
		initialInterval: 100 * time.Millisecond,
		maxInterval:     100 * time.Millisecond,
		maxTimeout:      300 * time.Millisecond,
		sendMaxTimeout:  300 * time.Millisecond,
	})

	return commanderClient
}

func getDialOptions(dialer func(context.Context, string) (net.Conn, error)) []grpc.DialOption {
	var grpcDialOptions []grpc.DialOption
	grpcDialOptions = append(grpcDialOptions, sdkGRPC.DefaultClientDialOptions...)
	grpcDialOptions = append(grpcDialOptions, grpc.WithContextDialer(dialer))
	grpcDialOptions = append(grpcDialOptions, grpc.WithTransportCredentials(insecure.NewCredentials()))
	return grpcDialOptions
}

func sendNginxConfigInChunks(commandService *mockCommanderService, nginxConfig *proto.NginxConfig) error {
	payload, err := json.Marshal(nginxConfig)
	if err != nil {
		return err
	}

	metadata := sdkGRPC.NewMessageMeta("1234")
	payloadChecksum := checksum.Checksum(payload)
	chunks := checksum.Chunk(payload, DefaultChunkSize)

	commandService.downloadChannel <- &proto.DataChunk{
		Chunk: &proto.DataChunk_Header{
			Header: &proto.ChunkedResourceHeader{
				Chunks:    int32(len(chunks)),
				Checksum:  payloadChecksum,
				Meta:      metadata,
				ChunkSize: int32(DefaultChunkSize),
			},
		},
	}
	for id, chunk := range chunks {
		commandService.downloadChannel <- &proto.DataChunk{
			Chunk: &proto.DataChunk_Data{
				Data: &proto.ChunkedResourceChunk{
					ChunkId: int32(id),
					Data:    chunk,
					Meta:    metadata,
				},
			},
		}
	}

	commandService.downloadChannel <- &proto.DataChunk{}

	return nil
}

func sendInvalidObjectInChunks(commandService *mockCommanderService) {
	payload := []byte{1, 2, 3}

	metadata := sdkGRPC.NewMessageMeta("1234")
	payloadChecksum := checksum.Checksum(payload)
	chunks := checksum.Chunk(payload, DefaultChunkSize)

	commandService.downloadChannel <- &proto.DataChunk{
		Chunk: &proto.DataChunk_Header{
			Header: &proto.ChunkedResourceHeader{
				Chunks:    int32(len(chunks)),
				Checksum:  payloadChecksum,
				Meta:      metadata,
				ChunkSize: int32(DefaultChunkSize),
			},
		},
	}
	for id, chunk := range chunks {
		commandService.downloadChannel <- &proto.DataChunk{
			Chunk: &proto.DataChunk_Data{
				Data: &proto.ChunkedResourceChunk{
					ChunkId: int32(id),
					Data:    chunk,
					Meta:    metadata,
				},
			},
		}
	}

	commandService.downloadChannel <- &proto.DataChunk{}
}
