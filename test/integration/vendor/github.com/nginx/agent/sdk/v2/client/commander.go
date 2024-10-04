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
	"sync"

	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/status"

	"github.com/nginx/agent/sdk/v2/backoff"
	"github.com/nginx/agent/sdk/v2/checksum"
	sdkGRPC "github.com/nginx/agent/sdk/v2/grpc"
	"github.com/nginx/agent/sdk/v2/interceptors"
	"github.com/nginx/agent/sdk/v2/proto"
)

const (
	DefaultChunkSize = 4 * 1024
)

func NewCommanderClient() Commander {
	return &commander{
		recvChan:        make(chan Message, 1),
		downloadChan:    make(chan *proto.DataChunk, 1),
		connector:       newConnector(),
		chunkSize:       DefaultChunkSize,
		backoffSettings: DefaultBackoffSettings,
		isRetrying:      false,
	}
}

type commander struct {
	*connector
	chunkSize       int
	client          proto.CommanderClient
	channel         proto.Commander_CommandChannelClient
	recvChan        chan Message
	downloadChan    chan *proto.DataChunk
	ctx             context.Context
	mu              sync.Mutex
	backoffSettings backoff.BackoffSettings
	cancel          context.CancelFunc
	isRetrying      bool
	retryLock       sync.Mutex
}

func (c *commander) WithInterceptor(interceptor interceptors.Interceptor) Client {
	c.connector.interceptors = append(c.connector.interceptors, interceptor)
	return c
}

func (c *commander) WithClientInterceptor(interceptor interceptors.ClientInterceptor) Client {
	c.connector.clientInterceptors = append(c.connector.clientInterceptors, interceptor)
	return c
}

func (c *commander) WithGrpcConnection(clientConnection *grpc.ClientConn) Client {
	c.connector.grpc = clientConnection
	return c
}

func (c *commander) Connect(ctx context.Context) error {
	log.Debugf("Commander connecting to %s", c.server)

	c.ctx = ctx

	c.retryLock.Lock()
	err := backoff.WaitUntil(
		c.ctx,
		c.backoffSettings,
		c.createClient,
	)
	c.retryLock.Unlock()

	if err != nil {
		return err
	}

	recvLoopCtx, cancel := context.WithCancel(ctx)
	c.cancel = cancel
	go c.recvLoop(recvLoopCtx)

	return nil
}

func (c *commander) Close() error {
	var err error
	if c.channel != nil {
		err = c.channel.CloseSend()
		if err != nil {
			return err
		}
	}

	if c.grpc != nil {
		err = c.grpc.Close()
	}

	if c.cancel != nil {
		c.cancel()
	}

	return err
}

func (c *commander) Server() string {
	return c.server
}

func (c *commander) WithServer(s string) Client {
	c.server = s
	return c
}

func (c *commander) DialOptions() []grpc.DialOption {
	return c.dialOptions
}

func (c *commander) WithDialOptions(options ...grpc.DialOption) Client {
	c.dialOptions = append(c.dialOptions, options...)
	return c
}

func (c *commander) WithChunkSize(i int) Client {
	c.chunkSize = i
	return c
}

func (c *commander) ChunksSize() int {
	return c.chunkSize
}

func (c *commander) WithBackoffSettings(backoffSettings backoff.BackoffSettings) Client {
	c.backoffSettings = backoffSettings
	return c
}

func (c *commander) Send(ctx context.Context, message Message) error {
	var (
		cmd *proto.Command
		ok  bool
	)

	switch message.Classification() {
	case MsgClassificationCommand:
		if cmd, ok = message.Raw().(*proto.Command); !ok {
			return fmt.Errorf("expected a command message, but received %T", message.Data())
		}
	default:
		return fmt.Errorf("expected a command message, but received %T", message.Data())
	}

	err := backoff.WaitUntil(c.ctx, c.backoffSettings, func() error {
		err := c.checkClientConnection()
		if err != nil {
			return err
		}

		if c.channel == nil {
			c.setIsRetrying(true)
			return c.handleGrpcError("Commander Channel Send", errors.New("command channel client not created yet"))
		}

		if err := c.channel.Send(cmd); err != nil {
			c.setIsRetrying(true)
			return c.handleGrpcError("Commander Channel Send", err)
		}

		log.Tracef("Commander sent command %v", cmd)

		return nil
	})

	return err
}

func (c *commander) Recv() <-chan Message {
	return c.recvChan
}

func (c *commander) Download(ctx context.Context, metadata *proto.Metadata) (*proto.NginxConfig, error) {
	log.Debugf("Downloading config (messageId=%s)", metadata.GetMessageId())
	cfg := &proto.NginxConfig{}

	err := backoff.WaitUntil(c.ctx, c.backoffSettings, func() error {
		err := c.checkClientConnection()
		if err != nil {
			return err
		}

		var (
			header *proto.DataChunk_Header
			body   []byte
		)

		downloader, err := c.client.Download(c.ctx, &proto.DownloadRequest{Meta: metadata})
		if err != nil {
			c.setIsRetrying(true)
			return c.handleGrpcError("Commander Downloader", err)
		}

	LOOP:
		for {
			chunk, err := downloader.Recv()
			if err != nil && err != io.EOF {
				c.setIsRetrying(true)
				return c.handleGrpcError("Commander Downloader", err)
			}

			if chunk == nil {
				break LOOP
			}

			switch dataChunk := chunk.Chunk.(type) {
			case *proto.DataChunk_Header:
				if header != nil {
					return ErrDownloadHeaderUnexpectedNumber
				}
				header = dataChunk
			case *proto.DataChunk_Data:
				body = append(body, dataChunk.Data.Data...)
			case nil:
				break LOOP
			}
		}

		if header == nil {
			return ErrDownloadHeaderUnexpectedNumber
		}

		if checksum.Checksum(body) != header.Header.Checksum {
			return ErrDownloadChecksumMismatch
		}

		err = json.Unmarshal(body, cfg)
		if err != nil {
			log.Warnf("Download failed to unmarshal: %s", err)
			return ErrUnmarshallingData
		}

		return nil
	})

	return cfg, err
}

func (c *commander) Upload(ctx context.Context, cfg *proto.NginxConfig, messageId string) error {
	payload, err := json.Marshal(cfg)
	if err != nil {
		return err
	}

	metadata := sdkGRPC.NewMessageMeta(messageId)
	payloadChecksum := checksum.Checksum(payload)
	chunks := checksum.Chunk(payload, c.chunkSize)

	return backoff.WaitUntil(c.ctx, c.backoffSettings, func() error {
		err := c.checkClientConnection()
		if err != nil {
			return err
		}

		sender, err := c.client.Upload(c.ctx)
		if err != nil {
			c.setIsRetrying(true)
			return c.handleGrpcError("Commander Upload", err)
		}

		err = sender.Send(&proto.DataChunk{
			Chunk: &proto.DataChunk_Header{
				Header: &proto.ChunkedResourceHeader{
					Chunks:    int32(len(chunks)),
					Checksum:  payloadChecksum,
					Meta:      metadata,
					ChunkSize: int32(c.ChunksSize()),
				},
			},
		})
		if err != nil {
			c.setIsRetrying(true)
			return c.handleGrpcError("Commander Upload Header", err)
		}

		for id, chunk := range chunks {
			log.Infof("Upload: Sending data chunk data %d (messageId=%s)", int32(id), metadata.GetMessageId())
			if err = sender.Send(&proto.DataChunk{
				Chunk: &proto.DataChunk_Data{
					Data: &proto.ChunkedResourceChunk{
						ChunkId: int32(id),
						Data:    chunk,
						Meta:    metadata,
					},
				},
			}); err != nil {
				c.setIsRetrying(true)
				return c.handleGrpcError(fmt.Sprintf("Commander Upload (chunks=%d)", id), err)
			}
		}

		log.Infof("Upload sending done %s (chunks=%d)", metadata.MessageId, len(chunks))
		status, err := sender.CloseAndRecv()
		if err != nil {
			c.setIsRetrying(true)
			return c.handleGrpcError("Commander Upload CloseAndRecv", err)
		}

		if status.Status != proto.UploadStatus_OK {
			return fmt.Errorf("%s", status.Reason)
		}

		return nil
	})
}

func (c *commander) checkClientConnection() error {
	c.retryLock.Lock()
	defer c.retryLock.Unlock()

	if c.isRetrying {
		log.Infof("Retrying to connect to %s", c.grpc.Target())
		err := c.createClient()
		if err != nil {
			return err
		}
	}

	return nil
}

func (c *commander) createClient() error {
	log.Debug("Creating commander client")
	c.mu.Lock()
	defer c.mu.Unlock()

	// Making sure that the previous client connection is closed before creating a new one
	if c.grpc != nil {
		err := c.grpc.Close()
		if err != nil {
			log.Warnf("Error closing old grpc connection: %v", err)
		}
	}

	grpc, err := sdkGRPC.NewGrpcConnectionWithContext(c.ctx, c.server, c.DialOptions())
	if err != nil {
		log.Errorf("Unable to create client connection to %s: %s", c.server, err)
		log.Infof("Commander retrying to connect to %s", c.grpc.Target())
		return err
	}
	c.grpc = grpc

	c.client = proto.NewCommanderClient(c.grpc)

	channel, err := c.client.CommandChannel(c.ctx)
	if err != nil {
		log.Errorf("Unable to create command channel: %s", err)
		log.Infof("Commander retrying to connect to %s", c.grpc.Target())
		return err
	}
	c.channel = channel

	c.isRetrying = false

	return nil
}

func (c *commander) recvLoop(ctx context.Context) {
	log.Debug("Commander receive loop starting")
loop:
	for {
		select {
		case <-ctx.Done():
			break loop
		default:
			err := backoff.WaitUntil(ctx, c.backoffSettings, func() error {
				select {
				case <-ctx.Done():
					return nil
				default:
					err := c.checkClientConnection()
					if err != nil {
						return err
					}

					cmd, err := c.channel.Recv()
					log.Infof("Commander received %v, %v", cmd, err)
					if err != nil {
						c.setIsRetrying(true)
						return c.handleGrpcError("Commander Channel Recv", err)
					}

					c.recvChan <- MessageFromCommand(cmd)

					return nil
				}
			})
			if err != nil {
				log.Errorf("Error retrying to receive messages from the commander channel: %v", err)
			}
		}
	}
}

func (c *commander) handleGrpcError(messagePrefix string, err error) error {
	if st, ok := status.FromError(err); ok {
		log.Errorf("%s: error communicating with %s, code=%s, message=%v", messagePrefix, c.grpc.Target(), st.Code().String(), st.Message())
	} else if err == io.EOF {
		_, err = c.channel.Recv()
		if st, ok = status.FromError(err); ok {
			log.Errorf("%s: server %s is not processing requests, code=%s, message=%v", messagePrefix, c.grpc.Target(), st.Code().String(), st.Message())
		} else {
			log.Errorf("%s: unable to receive error message for EOF from %s, %v", messagePrefix, c.grpc.Target(), err)
		}
	} else {
		log.Errorf("%s: unknown grpc error while communicating with %s, %v", messagePrefix, c.grpc.Target(), err)
	}

	return err
}

func (c *commander) setIsRetrying(value bool) {
	c.retryLock.Lock()
	defer c.retryLock.Unlock()
	c.isRetrying = value
}
