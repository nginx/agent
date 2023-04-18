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
// Source: buf/alpha/registry/v1alpha1/doc.proto

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
	// DocServiceName is the fully-qualified name of the DocService service.
	DocServiceName = "buf.alpha.registry.v1alpha1.DocService"
)

// These constants are the fully-qualified names of the RPCs defined in this package. They're
// exposed at runtime as Spec.Procedure and as the final two segments of the HTTP route.
//
// Note that these are different from the fully-qualified method names used by
// google.golang.org/protobuf/reflect/protoreflect. To convert from these constants to
// reflection-formatted method names, remove the leading slash and convert the remaining slash to a
// period.
const (
	// DocServiceGetSourceDirectoryInfoProcedure is the fully-qualified name of the DocService's
	// GetSourceDirectoryInfo RPC.
	DocServiceGetSourceDirectoryInfoProcedure = "/buf.alpha.registry.v1alpha1.DocService/GetSourceDirectoryInfo"
	// DocServiceGetSourceFileProcedure is the fully-qualified name of the DocService's GetSourceFile
	// RPC.
	DocServiceGetSourceFileProcedure = "/buf.alpha.registry.v1alpha1.DocService/GetSourceFile"
	// DocServiceGetModulePackagesProcedure is the fully-qualified name of the DocService's
	// GetModulePackages RPC.
	DocServiceGetModulePackagesProcedure = "/buf.alpha.registry.v1alpha1.DocService/GetModulePackages"
	// DocServiceGetModuleDocumentationProcedure is the fully-qualified name of the DocService's
	// GetModuleDocumentation RPC.
	DocServiceGetModuleDocumentationProcedure = "/buf.alpha.registry.v1alpha1.DocService/GetModuleDocumentation"
	// DocServiceGetPackageDocumentationProcedure is the fully-qualified name of the DocService's
	// GetPackageDocumentation RPC.
	DocServiceGetPackageDocumentationProcedure = "/buf.alpha.registry.v1alpha1.DocService/GetPackageDocumentation"
)

// DocServiceClient is a client for the buf.alpha.registry.v1alpha1.DocService service.
type DocServiceClient interface {
	// GetSourceDirectoryInfo retrieves the directory and file structure for the
	// given owner, repository and reference.
	//
	// The purpose of this is to get a representation of the file tree for a given
	// module to enable exploring the module by navigating through its contents.
	GetSourceDirectoryInfo(context.Context, *connect_go.Request[v1alpha1.GetSourceDirectoryInfoRequest]) (*connect_go.Response[v1alpha1.GetSourceDirectoryInfoResponse], error)
	// GetSourceFile retrieves the source contents for the given owner, repository,
	// reference, and path.
	GetSourceFile(context.Context, *connect_go.Request[v1alpha1.GetSourceFileRequest]) (*connect_go.Response[v1alpha1.GetSourceFileResponse], error)
	// GetModulePackages retrieves the list of packages for the module based on the given
	// owner, repository, and reference.
	GetModulePackages(context.Context, *connect_go.Request[v1alpha1.GetModulePackagesRequest]) (*connect_go.Response[v1alpha1.GetModulePackagesResponse], error)
	// GetModuleDocumentation retrieves the documentations including buf.md and LICENSE files
	// for module based on the given owner, repository, and reference.
	GetModuleDocumentation(context.Context, *connect_go.Request[v1alpha1.GetModuleDocumentationRequest]) (*connect_go.Response[v1alpha1.GetModuleDocumentationResponse], error)
	// GetPackageDocumentation retrieves a a slice of documentation structures
	// for the given owner, repository, reference, and package name.
	GetPackageDocumentation(context.Context, *connect_go.Request[v1alpha1.GetPackageDocumentationRequest]) (*connect_go.Response[v1alpha1.GetPackageDocumentationResponse], error)
}

// NewDocServiceClient constructs a client for the buf.alpha.registry.v1alpha1.DocService service.
// By default, it uses the Connect protocol with the binary Protobuf Codec, asks for gzipped
// responses, and sends uncompressed requests. To use the gRPC or gRPC-Web protocols, supply the
// connect.WithGRPC() or connect.WithGRPCWeb() options.
//
// The URL supplied here should be the base URL for the Connect or gRPC server (for example,
// http://api.acme.com or https://acme.com/grpc).
func NewDocServiceClient(httpClient connect_go.HTTPClient, baseURL string, opts ...connect_go.ClientOption) DocServiceClient {
	baseURL = strings.TrimRight(baseURL, "/")
	return &docServiceClient{
		getSourceDirectoryInfo: connect_go.NewClient[v1alpha1.GetSourceDirectoryInfoRequest, v1alpha1.GetSourceDirectoryInfoResponse](
			httpClient,
			baseURL+DocServiceGetSourceDirectoryInfoProcedure,
			opts...,
		),
		getSourceFile: connect_go.NewClient[v1alpha1.GetSourceFileRequest, v1alpha1.GetSourceFileResponse](
			httpClient,
			baseURL+DocServiceGetSourceFileProcedure,
			opts...,
		),
		getModulePackages: connect_go.NewClient[v1alpha1.GetModulePackagesRequest, v1alpha1.GetModulePackagesResponse](
			httpClient,
			baseURL+DocServiceGetModulePackagesProcedure,
			opts...,
		),
		getModuleDocumentation: connect_go.NewClient[v1alpha1.GetModuleDocumentationRequest, v1alpha1.GetModuleDocumentationResponse](
			httpClient,
			baseURL+DocServiceGetModuleDocumentationProcedure,
			opts...,
		),
		getPackageDocumentation: connect_go.NewClient[v1alpha1.GetPackageDocumentationRequest, v1alpha1.GetPackageDocumentationResponse](
			httpClient,
			baseURL+DocServiceGetPackageDocumentationProcedure,
			opts...,
		),
	}
}

// docServiceClient implements DocServiceClient.
type docServiceClient struct {
	getSourceDirectoryInfo  *connect_go.Client[v1alpha1.GetSourceDirectoryInfoRequest, v1alpha1.GetSourceDirectoryInfoResponse]
	getSourceFile           *connect_go.Client[v1alpha1.GetSourceFileRequest, v1alpha1.GetSourceFileResponse]
	getModulePackages       *connect_go.Client[v1alpha1.GetModulePackagesRequest, v1alpha1.GetModulePackagesResponse]
	getModuleDocumentation  *connect_go.Client[v1alpha1.GetModuleDocumentationRequest, v1alpha1.GetModuleDocumentationResponse]
	getPackageDocumentation *connect_go.Client[v1alpha1.GetPackageDocumentationRequest, v1alpha1.GetPackageDocumentationResponse]
}

// GetSourceDirectoryInfo calls buf.alpha.registry.v1alpha1.DocService.GetSourceDirectoryInfo.
func (c *docServiceClient) GetSourceDirectoryInfo(ctx context.Context, req *connect_go.Request[v1alpha1.GetSourceDirectoryInfoRequest]) (*connect_go.Response[v1alpha1.GetSourceDirectoryInfoResponse], error) {
	return c.getSourceDirectoryInfo.CallUnary(ctx, req)
}

// GetSourceFile calls buf.alpha.registry.v1alpha1.DocService.GetSourceFile.
func (c *docServiceClient) GetSourceFile(ctx context.Context, req *connect_go.Request[v1alpha1.GetSourceFileRequest]) (*connect_go.Response[v1alpha1.GetSourceFileResponse], error) {
	return c.getSourceFile.CallUnary(ctx, req)
}

// GetModulePackages calls buf.alpha.registry.v1alpha1.DocService.GetModulePackages.
func (c *docServiceClient) GetModulePackages(ctx context.Context, req *connect_go.Request[v1alpha1.GetModulePackagesRequest]) (*connect_go.Response[v1alpha1.GetModulePackagesResponse], error) {
	return c.getModulePackages.CallUnary(ctx, req)
}

// GetModuleDocumentation calls buf.alpha.registry.v1alpha1.DocService.GetModuleDocumentation.
func (c *docServiceClient) GetModuleDocumentation(ctx context.Context, req *connect_go.Request[v1alpha1.GetModuleDocumentationRequest]) (*connect_go.Response[v1alpha1.GetModuleDocumentationResponse], error) {
	return c.getModuleDocumentation.CallUnary(ctx, req)
}

// GetPackageDocumentation calls buf.alpha.registry.v1alpha1.DocService.GetPackageDocumentation.
func (c *docServiceClient) GetPackageDocumentation(ctx context.Context, req *connect_go.Request[v1alpha1.GetPackageDocumentationRequest]) (*connect_go.Response[v1alpha1.GetPackageDocumentationResponse], error) {
	return c.getPackageDocumentation.CallUnary(ctx, req)
}

// DocServiceHandler is an implementation of the buf.alpha.registry.v1alpha1.DocService service.
type DocServiceHandler interface {
	// GetSourceDirectoryInfo retrieves the directory and file structure for the
	// given owner, repository and reference.
	//
	// The purpose of this is to get a representation of the file tree for a given
	// module to enable exploring the module by navigating through its contents.
	GetSourceDirectoryInfo(context.Context, *connect_go.Request[v1alpha1.GetSourceDirectoryInfoRequest]) (*connect_go.Response[v1alpha1.GetSourceDirectoryInfoResponse], error)
	// GetSourceFile retrieves the source contents for the given owner, repository,
	// reference, and path.
	GetSourceFile(context.Context, *connect_go.Request[v1alpha1.GetSourceFileRequest]) (*connect_go.Response[v1alpha1.GetSourceFileResponse], error)
	// GetModulePackages retrieves the list of packages for the module based on the given
	// owner, repository, and reference.
	GetModulePackages(context.Context, *connect_go.Request[v1alpha1.GetModulePackagesRequest]) (*connect_go.Response[v1alpha1.GetModulePackagesResponse], error)
	// GetModuleDocumentation retrieves the documentations including buf.md and LICENSE files
	// for module based on the given owner, repository, and reference.
	GetModuleDocumentation(context.Context, *connect_go.Request[v1alpha1.GetModuleDocumentationRequest]) (*connect_go.Response[v1alpha1.GetModuleDocumentationResponse], error)
	// GetPackageDocumentation retrieves a a slice of documentation structures
	// for the given owner, repository, reference, and package name.
	GetPackageDocumentation(context.Context, *connect_go.Request[v1alpha1.GetPackageDocumentationRequest]) (*connect_go.Response[v1alpha1.GetPackageDocumentationResponse], error)
}

// NewDocServiceHandler builds an HTTP handler from the service implementation. It returns the path
// on which to mount the handler and the handler itself.
//
// By default, handlers support the Connect, gRPC, and gRPC-Web protocols with the binary Protobuf
// and JSON codecs. They also support gzip compression.
func NewDocServiceHandler(svc DocServiceHandler, opts ...connect_go.HandlerOption) (string, http.Handler) {
	mux := http.NewServeMux()
	mux.Handle(DocServiceGetSourceDirectoryInfoProcedure, connect_go.NewUnaryHandler(
		DocServiceGetSourceDirectoryInfoProcedure,
		svc.GetSourceDirectoryInfo,
		opts...,
	))
	mux.Handle(DocServiceGetSourceFileProcedure, connect_go.NewUnaryHandler(
		DocServiceGetSourceFileProcedure,
		svc.GetSourceFile,
		opts...,
	))
	mux.Handle(DocServiceGetModulePackagesProcedure, connect_go.NewUnaryHandler(
		DocServiceGetModulePackagesProcedure,
		svc.GetModulePackages,
		opts...,
	))
	mux.Handle(DocServiceGetModuleDocumentationProcedure, connect_go.NewUnaryHandler(
		DocServiceGetModuleDocumentationProcedure,
		svc.GetModuleDocumentation,
		opts...,
	))
	mux.Handle(DocServiceGetPackageDocumentationProcedure, connect_go.NewUnaryHandler(
		DocServiceGetPackageDocumentationProcedure,
		svc.GetPackageDocumentation,
		opts...,
	))
	return "/buf.alpha.registry.v1alpha1.DocService/", mux
}

// UnimplementedDocServiceHandler returns CodeUnimplemented from all methods.
type UnimplementedDocServiceHandler struct{}

func (UnimplementedDocServiceHandler) GetSourceDirectoryInfo(context.Context, *connect_go.Request[v1alpha1.GetSourceDirectoryInfoRequest]) (*connect_go.Response[v1alpha1.GetSourceDirectoryInfoResponse], error) {
	return nil, connect_go.NewError(connect_go.CodeUnimplemented, errors.New("buf.alpha.registry.v1alpha1.DocService.GetSourceDirectoryInfo is not implemented"))
}

func (UnimplementedDocServiceHandler) GetSourceFile(context.Context, *connect_go.Request[v1alpha1.GetSourceFileRequest]) (*connect_go.Response[v1alpha1.GetSourceFileResponse], error) {
	return nil, connect_go.NewError(connect_go.CodeUnimplemented, errors.New("buf.alpha.registry.v1alpha1.DocService.GetSourceFile is not implemented"))
}

func (UnimplementedDocServiceHandler) GetModulePackages(context.Context, *connect_go.Request[v1alpha1.GetModulePackagesRequest]) (*connect_go.Response[v1alpha1.GetModulePackagesResponse], error) {
	return nil, connect_go.NewError(connect_go.CodeUnimplemented, errors.New("buf.alpha.registry.v1alpha1.DocService.GetModulePackages is not implemented"))
}

func (UnimplementedDocServiceHandler) GetModuleDocumentation(context.Context, *connect_go.Request[v1alpha1.GetModuleDocumentationRequest]) (*connect_go.Response[v1alpha1.GetModuleDocumentationResponse], error) {
	return nil, connect_go.NewError(connect_go.CodeUnimplemented, errors.New("buf.alpha.registry.v1alpha1.DocService.GetModuleDocumentation is not implemented"))
}

func (UnimplementedDocServiceHandler) GetPackageDocumentation(context.Context, *connect_go.Request[v1alpha1.GetPackageDocumentationRequest]) (*connect_go.Response[v1alpha1.GetPackageDocumentationResponse], error) {
	return nil, connect_go.NewError(connect_go.CodeUnimplemented, errors.New("buf.alpha.registry.v1alpha1.DocService.GetPackageDocumentation is not implemented"))
}
