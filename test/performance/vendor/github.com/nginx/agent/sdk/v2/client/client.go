/**
 * Copyright (c) F5, Inc.
 *
 * This source code is licensed under the Apache License, Version 2.0 license found in the
 * LICENSE file in the root directory of this source tree.
 */

//go:generate enumer -type=MsgClassification -text -yaml -json -transform=snake -trimprefix=MsgClassification

package client

import (
	"context"
	"time"

	"google.golang.org/grpc"

	"github.com/nginx/agent/sdk/v2"
	"github.com/nginx/agent/sdk/v2/interceptors"
	"github.com/nginx/agent/sdk/v2/proto"
)

type BackoffSettings struct {
	initialInterval      time.Duration
	maxInterval          time.Duration
	maxTimeout           time.Duration
	sendMaxTimeout       time.Duration
	multiplier           float64
	randomization_factor float64
}

type MsgClassification int

const (
	MsgClassificationCommand MsgClassification = iota
	MsgClassificationMetric
	MsgClassificationEvent
)

var (
	DefaultBackoffSettings = BackoffSettings{
		initialInterval:      10 * time.Second,
		maxInterval:          60 * time.Second,
		maxTimeout:           0,
		sendMaxTimeout:       2 * time.Minute,
		randomization_factor: sdk.BACKOFF_JITTER,
		multiplier:           sdk.BACKOFF_MULTIPLIER,
	}
)

type (
	MsgType interface {
		String() string
		EnumDescriptor() ([]byte, []int)
	}
	Message interface {
		Meta() *proto.Metadata
		Type() MsgType
		Classification() MsgClassification
		Data() interface{}
		Raw() interface{}
	}
	Client interface {
		Connect(ctx context.Context) error
		Close() error

		Server() string
		WithServer(string) Client

		DialOptions() []grpc.DialOption
		WithDialOptions(options ...grpc.DialOption) Client

		WithInterceptor(interceptor interceptors.Interceptor) Client
		WithClientInterceptor(interceptor interceptors.ClientInterceptor) Client

		WithBackoffSettings(backoffSettings BackoffSettings) Client
	}
	MetricReporter interface {
		Client
		Send(context.Context, Message) error
	}
	Commander interface {
		Client
		ChunksSize() int
		WithChunkSize(int) Client
		Send(context.Context, Message) error
		Download(context.Context, *proto.Metadata) (*proto.NginxConfig, error)
		Upload(context.Context, *proto.NginxConfig, string) error
		Recv() <-chan Message
	}
	Controller interface {
		WithClient(Client) Controller
		Context() context.Context
		WithContext(context.Context) Controller
		Connect() error
		Close() error
	}
)
