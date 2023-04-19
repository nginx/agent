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
// Source: buf/alpha/registry/v1alpha1/search.proto

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
const _ = connect_go.IsAtLeastVersion0_1_0

const (
	// SearchServiceName is the fully-qualified name of the SearchService service.
	SearchServiceName = "buf.alpha.registry.v1alpha1.SearchService"
)

// These constants are the fully-qualified names of the RPCs defined in this package. They're
// exposed at runtime as Spec.Procedure and as the final two segments of the HTTP route.
//
// Note that these are different from the fully-qualified method names used by
// google.golang.org/protobuf/reflect/protoreflect. To convert from these constants to
// reflection-formatted method names, remove the leading slash and convert the remaining slash to a
// period.
const (
	// SearchServiceSearchProcedure is the fully-qualified name of the SearchService's Search RPC.
	SearchServiceSearchProcedure = "/buf.alpha.registry.v1alpha1.SearchService/Search"
	// SearchServiceSearchTagProcedure is the fully-qualified name of the SearchService's SearchTag RPC.
	SearchServiceSearchTagProcedure = "/buf.alpha.registry.v1alpha1.SearchService/SearchTag"
	// SearchServiceSearchDraftProcedure is the fully-qualified name of the SearchService's SearchDraft
	// RPC.
	SearchServiceSearchDraftProcedure = "/buf.alpha.registry.v1alpha1.SearchService/SearchDraft"
)

// SearchServiceClient is a client for the buf.alpha.registry.v1alpha1.SearchService service.
type SearchServiceClient interface {
	// Search searches the BSR.
	Search(context.Context, *connect_go.Request[v1alpha1.SearchRequest]) (*connect_go.Response[v1alpha1.SearchResponse], error)
	// SearchTag searches for tags in a repository
	SearchTag(context.Context, *connect_go.Request[v1alpha1.SearchTagRequest]) (*connect_go.Response[v1alpha1.SearchTagResponse], error)
	// SearchDraft searches for drafts in a repository
	SearchDraft(context.Context, *connect_go.Request[v1alpha1.SearchDraftRequest]) (*connect_go.Response[v1alpha1.SearchDraftResponse], error)
}

// NewSearchServiceClient constructs a client for the buf.alpha.registry.v1alpha1.SearchService
// service. By default, it uses the Connect protocol with the binary Protobuf Codec, asks for
// gzipped responses, and sends uncompressed requests. To use the gRPC or gRPC-Web protocols, supply
// the connect.WithGRPC() or connect.WithGRPCWeb() options.
//
// The URL supplied here should be the base URL for the Connect or gRPC server (for example,
// http://api.acme.com or https://acme.com/grpc).
func NewSearchServiceClient(httpClient connect_go.HTTPClient, baseURL string, opts ...connect_go.ClientOption) SearchServiceClient {
	baseURL = strings.TrimRight(baseURL, "/")
	return &searchServiceClient{
		search: connect_go.NewClient[v1alpha1.SearchRequest, v1alpha1.SearchResponse](
			httpClient,
			baseURL+SearchServiceSearchProcedure,
			opts...,
		),
		searchTag: connect_go.NewClient[v1alpha1.SearchTagRequest, v1alpha1.SearchTagResponse](
			httpClient,
			baseURL+SearchServiceSearchTagProcedure,
			opts...,
		),
		searchDraft: connect_go.NewClient[v1alpha1.SearchDraftRequest, v1alpha1.SearchDraftResponse](
			httpClient,
			baseURL+SearchServiceSearchDraftProcedure,
			opts...,
		),
	}
}

// searchServiceClient implements SearchServiceClient.
type searchServiceClient struct {
	search      *connect_go.Client[v1alpha1.SearchRequest, v1alpha1.SearchResponse]
	searchTag   *connect_go.Client[v1alpha1.SearchTagRequest, v1alpha1.SearchTagResponse]
	searchDraft *connect_go.Client[v1alpha1.SearchDraftRequest, v1alpha1.SearchDraftResponse]
}

// Search calls buf.alpha.registry.v1alpha1.SearchService.Search.
func (c *searchServiceClient) Search(ctx context.Context, req *connect_go.Request[v1alpha1.SearchRequest]) (*connect_go.Response[v1alpha1.SearchResponse], error) {
	return c.search.CallUnary(ctx, req)
}

// SearchTag calls buf.alpha.registry.v1alpha1.SearchService.SearchTag.
func (c *searchServiceClient) SearchTag(ctx context.Context, req *connect_go.Request[v1alpha1.SearchTagRequest]) (*connect_go.Response[v1alpha1.SearchTagResponse], error) {
	return c.searchTag.CallUnary(ctx, req)
}

// SearchDraft calls buf.alpha.registry.v1alpha1.SearchService.SearchDraft.
func (c *searchServiceClient) SearchDraft(ctx context.Context, req *connect_go.Request[v1alpha1.SearchDraftRequest]) (*connect_go.Response[v1alpha1.SearchDraftResponse], error) {
	return c.searchDraft.CallUnary(ctx, req)
}

// SearchServiceHandler is an implementation of the buf.alpha.registry.v1alpha1.SearchService
// service.
type SearchServiceHandler interface {
	// Search searches the BSR.
	Search(context.Context, *connect_go.Request[v1alpha1.SearchRequest]) (*connect_go.Response[v1alpha1.SearchResponse], error)
	// SearchTag searches for tags in a repository
	SearchTag(context.Context, *connect_go.Request[v1alpha1.SearchTagRequest]) (*connect_go.Response[v1alpha1.SearchTagResponse], error)
	// SearchDraft searches for drafts in a repository
	SearchDraft(context.Context, *connect_go.Request[v1alpha1.SearchDraftRequest]) (*connect_go.Response[v1alpha1.SearchDraftResponse], error)
}

// NewSearchServiceHandler builds an HTTP handler from the service implementation. It returns the
// path on which to mount the handler and the handler itself.
//
// By default, handlers support the Connect, gRPC, and gRPC-Web protocols with the binary Protobuf
// and JSON codecs. They also support gzip compression.
func NewSearchServiceHandler(svc SearchServiceHandler, opts ...connect_go.HandlerOption) (string, http.Handler) {
	mux := http.NewServeMux()
	mux.Handle(SearchServiceSearchProcedure, connect_go.NewUnaryHandler(
		SearchServiceSearchProcedure,
		svc.Search,
		opts...,
	))
	mux.Handle(SearchServiceSearchTagProcedure, connect_go.NewUnaryHandler(
		SearchServiceSearchTagProcedure,
		svc.SearchTag,
		opts...,
	))
	mux.Handle(SearchServiceSearchDraftProcedure, connect_go.NewUnaryHandler(
		SearchServiceSearchDraftProcedure,
		svc.SearchDraft,
		opts...,
	))
	return "/buf.alpha.registry.v1alpha1.SearchService/", mux
}

// UnimplementedSearchServiceHandler returns CodeUnimplemented from all methods.
type UnimplementedSearchServiceHandler struct{}

func (UnimplementedSearchServiceHandler) Search(context.Context, *connect_go.Request[v1alpha1.SearchRequest]) (*connect_go.Response[v1alpha1.SearchResponse], error) {
	return nil, connect_go.NewError(connect_go.CodeUnimplemented, errors.New("buf.alpha.registry.v1alpha1.SearchService.Search is not implemented"))
}

func (UnimplementedSearchServiceHandler) SearchTag(context.Context, *connect_go.Request[v1alpha1.SearchTagRequest]) (*connect_go.Response[v1alpha1.SearchTagResponse], error) {
	return nil, connect_go.NewError(connect_go.CodeUnimplemented, errors.New("buf.alpha.registry.v1alpha1.SearchService.SearchTag is not implemented"))
}

func (UnimplementedSearchServiceHandler) SearchDraft(context.Context, *connect_go.Request[v1alpha1.SearchDraftRequest]) (*connect_go.Response[v1alpha1.SearchDraftResponse], error) {
	return nil, connect_go.NewError(connect_go.CodeUnimplemented, errors.New("buf.alpha.registry.v1alpha1.SearchService.SearchDraft is not implemented"))
}
