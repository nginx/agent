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
	"strconv"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/status"

	"github.com/nginx/agent/sdk/v2"
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
	backoffSettings BackoffSettings
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
	err := sdk.WaitUntil(
		c.ctx,
		c.backoffSettings.initialInterval,
		c.backoffSettings.maxInterval,
		c.backoffSettings.maxTimeout,
		c.createClient,
	)
	if err != nil {
		return err
	}

	go c.recvLoop()

	return nil
}

func (c *commander) Close() error {
	err := c.channel.CloseSend()
	if err != nil {
		return err
	}
	return c.grpc.Close()
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

func (c *commander) WithBackoffSettings(backoffSettings BackoffSettings) Client {
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

	err := sdk.WaitUntil(c.ctx, c.backoffSettings.initialInterval, c.backoffSettings.maxInterval, c.backoffSettings.sendMaxTimeout, func() error {
		if err := c.channel.Send(cmd); err != nil {
			return c.handleGrpcError("Commander Channel Send", err, nil)
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

	err := sdk.WaitUntil(c.ctx, c.backoffSettings.initialInterval, c.backoffSettings.maxInterval, c.backoffSettings.sendMaxTimeout, func() error {
		var (
			header *proto.DataChunk_Header
			body   []byte
		)

		downloader, err := c.client.Download(c.ctx, &proto.DownloadRequest{Meta: metadata})
		if err != nil {
			return c.handleGrpcError("Commander Downloader", err, nil)
		}

	LOOP:
		for {
			chunk, err := downloader.Recv()
			if err != nil && err != io.EOF {
				return c.handleGrpcError("Commander Downloader", err, nil)
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

	return sdk.WaitUntil(c.ctx, c.backoffSettings.initialInterval, c.backoffSettings.maxInterval, c.backoffSettings.sendMaxTimeout, func() error {
		sender, err := c.client.Upload(c.ctx)
		if err != nil {
			return c.handleGrpcError("Commander Upload", err, nil)
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
			return c.handleGrpcError("Commander Upload Header", err, nil)
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
				return c.handleGrpcError("Commander Upload"+strconv.Itoa(id), err, nil)
			}
		}

		log.Infof("Upload sending done %s (chunks=%d)", metadata.MessageId, len(chunks))
		status, err := sender.CloseAndRecv()
		if err != nil {
			return c.handleGrpcError("Commander Upload CloseAndRecv", err, nil)
		}

		if status.Status != proto.UploadStatus_OK {
			return fmt.Errorf(status.Reason)
		}

		return nil
	})
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

	return nil
}

func (c *commander) recvLoop() {
	log.Debug("Commander receive loop starting")
	for {
		err := sdk.WaitUntilWithJitterAndMultiplier(c.ctx, c.backoffSettings.initialInterval, c.backoffSettings.maxInterval,
			c.backoffSettings.maxTimeout, c.backoffSettings.randomization_factor, c.backoffSettings.multiplier, func() error {
				cmd, err := c.channel.Recv()
				log.Infof("Commander received %v, %v", cmd, err)
				if err != nil {
					return c.handleGrpcError("Commander Channel Recv", err, cmd)
				}

				select {
				case <-c.ctx.Done():
				case c.recvChan <- MessageFromCommand(cmd):
				}

				return nil
			})
		if err != nil {
			log.Errorf("Error retrying to receive messages from the commander channel: %v", err)
		}
	}
}

func (c *commander) handleGrpcError(messagePrefix string, err error, cmd *proto.Command) error {
	if st, ok := status.FromError(err); ok {
		log.Errorf("%s: error communicating with %s, code=%s, message=%v", messagePrefix, c.grpc.Target(), st.Code().String(), st.Message())
	} else if err == io.EOF {
		log.Errorf("%s: server %s is not processing requests, code=%s, message=%v", messagePrefix, c.grpc.Target(), st.Code().String(), st.Message())
	} else {
		log.Errorf("%s: unknown grpc error while communicating with %s, %v", messagePrefix, c.grpc.Target(), err)
	}

	log.Infof("%s: retrying to connect to %s", messagePrefix, c.grpc.Target())
	_ = c.createClient()

	c.resetBackoffSettings(cmd)
	return err
}

func (c *commander) resetBackoffSettings(cmd *proto.Command) {
	if cmd == nil {
		return
	}
	if cmd.GetAgentConfig() == nil {
		return
	}
	if cmd.GetAgentConfig().GetDetails() == nil {
		return
	}
	if cmd.GetAgentConfig().GetDetails().GetServer() == nil {
		return
	}
	sBackoff := cmd.GetAgentConfig().GetDetails().GetServer().Backoff
	if sBackoff == nil {
		return
	}

	smultiplier := sdk.BACKOFF_MULTIPLIER
	if sBackoff.GetMultiplier() != 0 {
		smultiplier = sBackoff.GetMultiplier()
	}

	srandomization_factor := sdk.BACKOFF_JITTER
	if sBackoff.GetRandomizationFactor() != 0 {
		srandomization_factor = sBackoff.GetRandomizationFactor()
	}

	cBackoff := BackoffSettings{
		initialInterval:      time.Duration(sBackoff.InitialInterval),
		maxInterval:          time.Duration(sBackoff.MaxInterval),
		sendMaxTimeout:       time.Duration(sBackoff.MaxElapsedTime),
		multiplier:           smultiplier,
		randomization_factor: srandomization_factor,
	}
	log.Infof("reset client backoff settings to %+v, for a pause command %+v", cBackoff, cmd)
	c.WithBackoffSettings(cBackoff)
}
