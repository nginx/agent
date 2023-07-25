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
	"fmt"
	"io"
	"net"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/nginx/agent/sdk/v2/backoff"
	"github.com/nginx/agent/sdk/v2/checksum"
	"github.com/nginx/agent/sdk/v2/proto"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
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
	grpcServerCommanderMutex = &sync.Mutex{}
	backOffSettings          = backoff.BackoffSettings{
		InitialInterval: 100 * time.Millisecond,
		MaxInterval:     100 * time.Millisecond,
		MaxElapsedTime:  10 * time.Second,
	}
)

const (
	bufSize    = 1024 * 1024
	serverName = "bufnet"
)

// Positive Test Cases

func TestCommander_ChuckSize(t *testing.T) {
	commanderClient := NewCommanderClient()
	commanderClient.WithChunkSize(1000)

	assert.Equal(t, 1000, commanderClient.ChunksSize())
	t.Cleanup(func() {
		commanderClient.Close()
	})
}

func TestCommander_Server(t *testing.T) {
	commanderClient := NewCommanderClient()
	commanderClient.WithServer("test")

	t.Cleanup(func() {
		commanderClient.Close()
	})

	assert.Equal(t, "test", commanderClient.Server())
}

func TestCommander_Recv(t *testing.T) {
	wg := &sync.WaitGroup{}
	wg.Add(1)
	serverName, grpcServer, commandService, dialer := startCommanderMockServer(uuid.New().String())

	go func() {
		commandService.handler.toClient <- &proto.Command{Meta: &proto.Metadata{MessageId: "1234"}}
	}()

	ctx := context.Background()

	commanderClient := createTestCommanderClient(serverName, dialer)
	err := commanderClient.Connect(ctx)
	assert.Nil(t, err)

	t.Cleanup(func() {
		commanderClient.Close()
		if err := stopMockCommandServer(ctx, grpcServer, dialer); err != nil {
			t.Fatalf("Unable to stop grpc server")
		}
	})

	go func() {
		defer wg.Done()
		select {
		case actual := <-commanderClient.Recv():
			if actual != nil {
				assert.Equal(t, "1234", actual.Meta().MessageId)
			}
		case <-time.After(1 * time.Second):
		}
	}()
	wg.Wait()
}

func TestCommander_Send(t *testing.T) {
	serverName, grpcServer, commandService, dialer := startCommanderMockServer(uuid.New().String())

	ctx := context.Background()

	commanderClient := createTestCommanderClient(serverName, dialer)
	err := commanderClient.Connect(ctx)
	assert.Nil(t, err)

	t.Cleanup(func() {
		commanderClient.Close()
		if err := stopMockCommandServer(ctx, grpcServer, dialer); err != nil {
			t.Fatalf("Unable to stop grpc server")
		}
	})

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
	serverName, grpcServer, commandService, dialer := startCommanderMockServer(uuid.New().String())

	go func() {
		err := sendNginxConfigInChunks(commandService, expectedNginxConfig)
		if err != nil {
			t.Logf("Error converting nginx config to byte array: %v\n", err)
		}
	}()

	ctx := context.Background()

	commanderClient := createTestCommanderClient(serverName, dialer)
	err := commanderClient.Connect(ctx)
	assert.Nil(t, err)

	t.Cleanup(func() {
		commanderClient.Close()
		if err := stopMockCommandServer(ctx, grpcServer, dialer); err != nil {
			t.Fatalf("Unable to stop grpc server")
		}
	})

	actual, err := commanderClient.Download(ctx, &proto.Metadata{MessageId: "1234"})

	assert.Nil(t, err)
	assert.Equal(t, expectedNginxConfig, actual)
}

func TestCommander_Upload(t *testing.T) {
	serverName, grpcServer, commandService, dialer := startCommanderMockServer(uuid.New().String())

	ctx := context.Background()

	commanderClient := createTestCommanderClient(serverName, dialer)
	err := commanderClient.Connect(ctx)
	assert.Nil(t, err)

	t.Cleanup(func() {
		commanderClient.Close()
		if err := stopMockCommandServer(ctx, grpcServer, dialer); err != nil {
			t.Fatalf("Unable to stop grpc server")
		}
	})

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
	ctx, cncl := context.WithTimeout(context.Background(), 200*time.Millisecond)

	var grpcDialOptions []grpc.DialOption
	grpcDialOptions = append(grpcDialOptions, sdkGRPC.DefaultClientDialOptions...)
	grpcDialOptions = append(grpcDialOptions, grpc.WithTransportCredentials(insecure.NewCredentials()))

	commanderClient := NewCommanderClient()
	commanderClient.WithServer("unknown")
	commanderClient.WithDialOptions(grpcDialOptions...)
	commanderClient.WithBackoffSettings(backoff.BackoffSettings{
		InitialInterval: 100 * time.Millisecond,
		MaxInterval:     100 * time.Millisecond,
		MaxElapsedTime:  300 * time.Millisecond,
	})

	err := commanderClient.Connect(ctx)
	assert.NotNil(t, err)
	assert.Equal(t, codes.Unknown, status.Code(err))

	t.Cleanup(func() {
		commanderClient.Close()
		cncl()
	})
}

func TestCommander_Recv_Reconnect(t *testing.T) {
	wg := &sync.WaitGroup{}
	wg.Add(1)
	serverPrefix := uuid.New().String()

	serverName, grpcServer, commandService, dialer := startCommanderMockServer(serverPrefix)

	ctx := context.Background()

	commanderClient := createTestCommanderClient(serverName, dialer)
	commanderClient.WithBackoffSettings(backOffSettings)
	err := commanderClient.Connect(ctx)
	assert.Nil(t, err)

	// Restart server
	if err := stopMockCommandServer(ctx, grpcServer, dialer); err != nil {
		t.Fatalf("Unable to stop grpc server TestCommander_Recv_Reconnect 1")
	}
	serverName, grpcServer, commandService, dialer = startCommanderMockServer(serverPrefix)

	go func() {
		commandService.handler.toClient <- &proto.Command{Meta: &proto.Metadata{MessageId: "1234"}}
	}()

	commanderClient.WithDialOptions(getDialOptions(dialer)...)
	commanderClient.WithServer(serverName)

	t.Cleanup(func() {
		commanderClient.Close()
		if err := stopMockCommandServer(ctx, grpcServer, dialer); err != nil {
			t.Fatalf("Unable to stop grpc server TestCommander_Recv_Reconnect 2")
		}
	})

	go func() {
		defer wg.Done()
		select {
		case actual := <-commanderClient.Recv():
			if actual != nil {
				assert.Equal(t, "1234", actual.Meta().MessageId)
			}
		case <-time.After(5 * time.Second):
			assert.Fail(t, "No message received from commander")
		}
	}()

	wg.Wait()
}

func TestCommander_Send_ServerDies(t *testing.T) {
	serverName, grpcServer, _, dialer := startCommanderMockServer(uuid.New().String())

	ctx := context.Background()

	commanderClient := createTestCommanderClient(serverName, dialer)
	err := commanderClient.Connect(ctx)
	assert.Nil(t, err)

	t.Cleanup(func() {
		commanderClient.Close()
	})

	if err := stopMockCommandServer(ctx, grpcServer, dialer); err != nil {
		t.Fatalf("Unable to stop grpc server")
	}

	err = commanderClient.Send(ctx, MessageFromCommand(&proto.Command{Meta: &proto.Metadata{MessageId: "1234"}}))
	assert.NotNil(t, err)
	assert.Equal(t, codes.Unavailable, status.Code(err))
}

func TestCommander_Send_Reconnect(t *testing.T) {
	serverPrefix := uuid.New().String()
	serverName, grpcServer, _, dialer := startCommanderMockServer(serverPrefix)

	ctx := context.Background()

	commanderClient := createTestCommanderClient(serverName, dialer)
	commanderClient.WithBackoffSettings(backOffSettings)
	err := commanderClient.Connect(ctx)
	assert.Nil(t, err)

	// stop server and recreate
	if err := stopMockCommandServer(ctx, grpcServer, dialer); err != nil {
		t.Fatalf("Unable to stop grpc server")
	}
	serverName, grpcServer, _, dialer = startCommanderMockServer(serverPrefix)
	commanderClient.WithDialOptions(getDialOptions(dialer)...)
	commanderClient.WithServer(serverName)

	t.Cleanup(func() {
		commanderClient.Close()
		if err := stopMockCommandServer(ctx, grpcServer, dialer); err != nil {
			t.Fatalf("Unable to stop grpc server")
		}
	})

	err = commanderClient.Send(ctx, MessageFromCommand(&proto.Command{Meta: &proto.Metadata{MessageId: "1234"}}))
	assert.Nil(t, err)
}

func TestCommander_Download_ServerDies(t *testing.T) {
	serverName, grpcServer, _, dialer := startCommanderMockServer(uuid.New().String())

	ctx := context.Background()

	commanderClient := createTestCommanderClient(serverName, dialer)
	err := commanderClient.Connect(ctx)
	assert.Nil(t, err)

	t.Cleanup(func() {
		commanderClient.Close()
	})

	if err := stopMockCommandServer(ctx, grpcServer, dialer); err != nil {
		t.Fatalf("Unable to stop grpc server")
	}

	_, err = commanderClient.Download(ctx, &proto.Metadata{MessageId: "1234"})
	assert.NotNil(t, err)
	assert.Equal(t, codes.Unavailable, status.Code(err))
}

func TestCommander_Download_Reconnect(t *testing.T) {
	wg := &sync.WaitGroup{}
	wg.Add(1)
	serverPrefix := uuid.New().String()
	serverName, grpcServer, commandService, dialer := startCommanderMockServer(serverPrefix)

	ctx := context.Background()

	commanderClient := createTestCommanderClient(serverName, dialer)
	commanderClient.WithBackoffSettings(backOffSettings)
	err := commanderClient.Connect(ctx)
	assert.Nil(t, err)

	if err := stopMockCommandServer(ctx, grpcServer, dialer); err != nil {
		t.Fatalf("Unable to stop grpc server")
	}

	_, grpcServer, commandService, dialer = startCommanderMockServer(serverPrefix)

	go func() {
		err := sendNginxConfigInChunks(commandService, expectedNginxConfig)
		if err != nil {
			t.Logf("Error converting nginx config to byte array: %v\n", err)
		}
	}()

	go func() {
		defer wg.Done()
		actual, err := commanderClient.Download(ctx, &proto.Metadata{MessageId: "1234"})

		assert.Nil(t, err)
		assert.Equal(t, expectedNginxConfig, actual)
	}()
	wg.Wait()

	t.Cleanup(func() {
		commanderClient.Close()
		if err := stopMockCommandServer(ctx, grpcServer, dialer); err != nil {
			t.Fatalf("Unable to stop grpc server")
		}
	})
}

func TestCommander_Download_MissingHeaderChunk(t *testing.T) {
	serverName, grpcServer, commandService, dialer := startCommanderMockServer(uuid.New().String())
	wg := &sync.WaitGroup{}
	wg.Add(1)

	go func() {
		commandService.downloadChannel <- &proto.DataChunk{}
	}()

	ctx := context.Background()

	commanderClient := createTestCommanderClient(serverName, dialer)
	err := commanderClient.Connect(ctx)
	assert.Nil(t, err)

	t.Cleanup(func() {
		commanderClient.Close()
		if err := stopMockCommandServer(ctx, grpcServer, dialer); err != nil {
			t.Fatalf("Unable to stop grpc server")
		}
	})

	go func() {
		defer wg.Done()
		_, err = commanderClient.Download(ctx, &proto.Metadata{MessageId: "1234"})
		assert.NotNil(t, err)
		assert.Equal(t, codes.Unknown, status.Code(err))
		assert.ErrorContains(t, err, "unexpected number of headers")
	}()
	wg.Wait()
}

func TestCommander_Download_MultipleHeaderChunksSent(t *testing.T) {
	serverName, grpcServer, commandService, dialer := startCommanderMockServer(uuid.New().String())

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

	ctx := context.Background()

	commanderClient := createTestCommanderClient(serverName, dialer)
	err := commanderClient.Connect(ctx)
	assert.Nil(t, err)

	t.Cleanup(func() {
		commanderClient.Close()
		if err := stopMockCommandServer(ctx, grpcServer, dialer); err != nil {
			t.Fatalf("Unable to stop grpc server")
		}
	})

	_, err = commanderClient.Download(ctx, &proto.Metadata{MessageId: "1234"})
	assert.NotNil(t, err)
	assert.Equal(t, codes.Unknown, status.Code(err))
	assert.ErrorContains(t, err, "unexpected number of headers")
}

func TestCommander_Download_ChecksumMismatch(t *testing.T) {
	serverName, grpcServer, commandService, dialer := startCommanderMockServer(uuid.New().String())

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

	ctx := context.Background()

	commanderClient := createTestCommanderClient(serverName, dialer)
	err := commanderClient.Connect(ctx)
	assert.Nil(t, err)

	t.Cleanup(func() {
		commanderClient.Close()
		if err := stopMockCommandServer(ctx, grpcServer, dialer); err != nil {
			t.Fatalf("Unable to stop grpc server")
		}
	})

	_, err = commanderClient.Download(ctx, &proto.Metadata{MessageId: "1234"})
	assert.NotNil(t, err)
	assert.Equal(t, codes.Unknown, status.Code(err))
	assert.ErrorContains(t, err, "download checksum mismatch")
}

func TestCommander_Download_InvalidObjectTypeDownloaded(t *testing.T) {
	serverName, grpcServer, commandService, dialer := startCommanderMockServer(uuid.New().String())

	go func() {
		sendInvalidObjectInChunks(commandService)
	}()

	ctx := context.Background()

	commanderClient := createTestCommanderClient(serverName, dialer)
	err := commanderClient.Connect(ctx)
	assert.Nil(t, err)

	t.Cleanup(func() {
		commanderClient.Close()
		if err := stopMockCommandServer(ctx, grpcServer, dialer); err != nil {
			t.Fatalf("Unable to stop grpc server")
		}
	})

	_, err = commanderClient.Download(ctx, &proto.Metadata{MessageId: "1234"})
	assert.NotNil(t, err)
	assert.Equal(t, codes.Unknown, status.Code(err))
	assert.ErrorContains(t, err, "unable to unmarshal data")
}

func TestCommander_Upload_ServerDies(t *testing.T) {
	serverName, grpcServer, _, dialer := startCommanderMockServer(uuid.New().String())

	ctx := context.Background()

	commanderClient := createTestCommanderClient(serverName, dialer)
	err := commanderClient.Connect(ctx)
	assert.Nil(t, err)

	if err := stopMockCommandServer(ctx, grpcServer, dialer); err != nil {
		t.Fatalf("Unable to stop grpc server")
	}

	t.Cleanup(func() {
		commanderClient.Close()
	})

	err = commanderClient.Upload(ctx, expectedNginxConfig, "1234")
	assert.NotNil(t, err)
	assert.Equal(t, codes.Unavailable, status.Code(err))
}

func TestCommander_Upload_Reconnect(t *testing.T) {
	serverPrefix := uuid.New().String()
	serverName, grpcServer, _, dialer := startCommanderMockServer(serverPrefix)

	ctx := context.Background()
	wg := &sync.WaitGroup{}
	wg.Add(1)

	commanderClient := createTestCommanderClient(serverName, dialer)
	commanderClient.WithBackoffSettings(backOffSettings)
	err := commanderClient.Connect(ctx)
	assert.Nil(t, err)

	// Restart server
	if err := stopMockCommandServer(ctx, grpcServer, dialer); err != nil {
		t.Fatalf("Unable to stop grpc server")
	}
	serverName, grpcServer, commandService, dialer := startCommanderMockServer(serverPrefix)
	commanderClient.WithDialOptions(getDialOptions(dialer)...)
	commanderClient.WithServer(serverName)

	t.Cleanup(func() {
		commanderClient.Close()
		if err := stopMockCommandServer(ctx, grpcServer, dialer); err != nil {
			t.Fatalf("Unable to stop grpc server")
		}
	})

	time.Sleep(50 * time.Millisecond)

	err = commanderClient.Upload(ctx, expectedNginxConfig, "1234")
	assert.Nil(t, err)

	chunks := []*proto.DataChunk{}
LOOP:
	for {
		select {
		case data := <-commandService.uploadChannel:
			if data == nil {
				wg.Done()
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
	defer wg.Done()
	for {
		cmd, err := server.Recv()
		if cmd != nil {
			fmt.Printf("Recv Command: %v\n", cmd)
			if err != nil {
				fmt.Printf("Recv Command Error: %v\n", err)
				return
			}
			h.fromClient <- cmd
		}
	}
}

func (h *handler) sendHandle(server proto.Commander_CommandChannelServer, wg *sync.WaitGroup) {
	defer wg.Done()
	for {
		cmd := <-h.toClient
		if cmd != nil {
			err := server.Send(cmd)
			fmt.Printf("Send Command: %v\n", cmd)
			if err != nil {
				fmt.Printf("Send Command Error: %v\n", err)
				return
			}
		}
	}
}

func startCommanderMockServer(serverPrefix string) (string, *grpc.Server, *mockCommanderService, func(context.Context, string) (net.Conn, error)) {
	grpcServerCommanderMutex.Lock()
	defer grpcServerCommanderMutex.Unlock()
	serverName := fmt.Sprintf("%s_%s", serverPrefix, serverName)
	listener := bufconn.Listen(bufSize)
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

	time.Sleep(200 * time.Millisecond)

	return serverName, grpcServer, commandService, func(context.Context, string) (net.Conn, error) {
		return listener.Dial()
	}
}

func stopMockCommandServer(ctx context.Context, server *grpc.Server, dialer func(context.Context, string) (net.Conn, error)) error {
	grpcServerCommanderMutex.Lock()
	defer grpcServerCommanderMutex.Unlock()
	return stopMockServer(ctx, server, dialer)
}

func stopMockServer(ctx context.Context, server *grpc.Server, dialer func(context.Context, string) (net.Conn, error)) error {
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	done := make(chan bool, 1)

	go func() {
		signal.Stop(sigs)
		server.Stop()
		fmt.Println()
		time.Sleep(200 * time.Millisecond)
		done <- true
	}()

	fmt.Println("awaiting signal")
	<-done
	server.GracefulStop()
	fmt.Println("exiting")
	return nil
}

func createTestCommanderClient(serverName string, dialer func(context.Context, string) (net.Conn, error)) Commander {
	grpcServerCommanderMutex.Lock()
	defer grpcServerCommanderMutex.Unlock()
	commanderClient := NewCommanderClient()
	commanderClient.WithServer(serverName)
	commanderClient.WithDialOptions(getDialOptions(dialer)...)
	commanderClient.WithBackoffSettings(backoff.BackoffSettings{
		InitialInterval: 100 * time.Millisecond,
		MaxInterval:     100 * time.Millisecond,
		MaxElapsedTime:  300 * time.Millisecond,
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
