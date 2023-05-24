// Copyright 2020-2023 Buf Technologies, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Code generated by protoc-gen-connect-go. DO NOT EDIT.
//
// Source: buf/alpha/registry/v1alpha1/owner.proto

package registryv1alpha1connect

import (
	context "context"
	errors "errors"
	v1alpha1 "github.com/bufbuild/buf/private/gen/proto/go/buf/alpha/registry/v1alpha1"
	connect_go "github.com/bufbuild/connect-go"
	http "net/http"
	strings "strings"
)

// This is a compile-time assertion to ensure that this generated file and the connect package are
// compatible. If you get a compiler error that this constant is not defined, this code was
// generated with a version of connect newer than the one compiled into your binary. You can fix the
// problem by either regenerating this code with an older version of connect or updating the connect
// version compiled into your binary.
const _ = connect_go.IsAtLeastVersion1_7_0

const (
	// OwnerServiceName is the fully-qualified name of the OwnerService service.
	OwnerServiceName = "buf.alpha.registry.v1alpha1.OwnerService"
)

// These constants are the fully-qualified names of the RPCs defined in this package. They're
// exposed at runtime as Spec.Procedure and as the final two segments of the HTTP route.
//
// Note that these are different from the fully-qualified method names used by
// google.golang.org/protobuf/reflect/protoreflect. To convert from these constants to
// reflection-formatted method names, remove the leading slash and convert the remaining slash to a
// period.
const (
	// OwnerServiceGetOwnerByNameProcedure is the fully-qualified name of the OwnerService's
	// GetOwnerByName RPC.
	OwnerServiceGetOwnerByNameProcedure = "/buf.alpha.registry.v1alpha1.OwnerService/GetOwnerByName"
)

// OwnerServiceClient is a client for the buf.alpha.registry.v1alpha1.OwnerService service.
type OwnerServiceClient interface {
	// GetOwnerByName takes an owner name and returns the owner as
	// either a user or organization.
	GetOwnerByName(context.Context, *connect_go.Request[v1alpha1.GetOwnerByNameRequest]) (*connect_go.Response[v1alpha1.GetOwnerByNameResponse], error)
}

// NewOwnerServiceClient constructs a client for the buf.alpha.registry.v1alpha1.OwnerService
// service. By default, it uses the Connect protocol with the binary Protobuf Codec, asks for
// gzipped responses, and sends uncompressed requests. To use the gRPC or gRPC-Web protocols, supply
// the connect.WithGRPC() or connect.WithGRPCWeb() options.
//
// The URL supplied here should be the base URL for the Connect or gRPC server (for example,
// http://api.acme.com or https://acme.com/grpc).
func NewOwnerServiceClient(httpClient connect_go.HTTPClient, baseURL string, opts ...connect_go.ClientOption) OwnerServiceClient {
	baseURL = strings.TrimRight(baseURL, "/")
	return &ownerServiceClient{
		getOwnerByName: connect_go.NewClient[v1alpha1.GetOwnerByNameRequest, v1alpha1.GetOwnerByNameResponse](
			httpClient,
			baseURL+OwnerServiceGetOwnerByNameProcedure,
			connect_go.WithIdempotency(connect_go.IdempotencyNoSideEffects),
			connect_go.WithClientOptions(opts...),
		),
	}
}

// ownerServiceClient implements OwnerServiceClient.
type ownerServiceClient struct {
	getOwnerByName *connect_go.Client[v1alpha1.GetOwnerByNameRequest, v1alpha1.GetOwnerByNameResponse]
}

// GetOwnerByName calls buf.alpha.registry.v1alpha1.OwnerService.GetOwnerByName.
func (c *ownerServiceClient) GetOwnerByName(ctx context.Context, req *connect_go.Request[v1alpha1.GetOwnerByNameRequest]) (*connect_go.Response[v1alpha1.GetOwnerByNameResponse], error) {
	return c.getOwnerByName.CallUnary(ctx, req)
}

// OwnerServiceHandler is an implementation of the buf.alpha.registry.v1alpha1.OwnerService service.
type OwnerServiceHandler interface {
	// GetOwnerByName takes an owner name and returns the owner as
	// either a user or organization.
	GetOwnerByName(context.Context, *connect_go.Request[v1alpha1.GetOwnerByNameRequest]) (*connect_go.Response[v1alpha1.GetOwnerByNameResponse], error)
}

// NewOwnerServiceHandler builds an HTTP handler from the service implementation. It returns the
// path on which to mount the handler and the handler itself.
//
// By default, handlers support the Connect, gRPC, and gRPC-Web protocols with the binary Protobuf
// and JSON codecs. They also support gzip compression.
func NewOwnerServiceHandler(svc OwnerServiceHandler, opts ...connect_go.HandlerOption) (string, http.Handler) {
	mux := http.NewServeMux()
	mux.Handle(OwnerServiceGetOwnerByNameProcedure, connect_go.NewUnaryHandler(
		OwnerServiceGetOwnerByNameProcedure,
		svc.GetOwnerByName,
		connect_go.WithIdempotency(connect_go.IdempotencyNoSideEffects),
		connect_go.WithHandlerOptions(opts...),
	))
	return "/buf.alpha.registry.v1alpha1.OwnerService/", mux
}

// UnimplementedOwnerServiceHandler returns CodeUnimplemented from all methods.
type UnimplementedOwnerServiceHandler struct{}

func (UnimplementedOwnerServiceHandler) GetOwnerByName(context.Context, *connect_go.Request[v1alpha1.GetOwnerByNameRequest]) (*connect_go.Response[v1alpha1.GetOwnerByNameResponse], error) {
	return nil, connect_go.NewError(connect_go.CodeUnimplemented, errors.New("buf.alpha.registry.v1alpha1.OwnerService.GetOwnerByName is not implemented"))
}
