// Copyright 2020-2022 Buf Technologies, Inc.
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
// Source: buf/alpha/registry/v1alpha1/admin.proto

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
	// AdminServiceName is the fully-qualified name of the AdminService service.
	AdminServiceName = "buf.alpha.registry.v1alpha1.AdminService"
)

// AdminServiceClient is a client for the buf.alpha.registry.v1alpha1.AdminService service.
type AdminServiceClient interface {
	// ForceDeleteUser forces to delete a user. Resources and organizations that are
	// solely owned by the user will also be deleted.
	ForceDeleteUser(context.Context, *connect_go.Request[v1alpha1.ForceDeleteUserRequest]) (*connect_go.Response[v1alpha1.ForceDeleteUserResponse], error)
	// Update a user's verification status.
	UpdateUserVerificationStatus(context.Context, *connect_go.Request[v1alpha1.UpdateUserVerificationStatusRequest]) (*connect_go.Response[v1alpha1.UpdateUserVerificationStatusResponse], error)
	// Update a organization's verification.
	UpdateOrganizationVerificationStatus(context.Context, *connect_go.Request[v1alpha1.UpdateOrganizationVerificationStatusRequest]) (*connect_go.Response[v1alpha1.UpdateOrganizationVerificationStatusResponse], error)
	// Create a new machine user on the server.
	CreateMachineUser(context.Context, *connect_go.Request[v1alpha1.CreateMachineUserRequest]) (*connect_go.Response[v1alpha1.CreateMachineUserResponse], error)
}

// NewAdminServiceClient constructs a client for the buf.alpha.registry.v1alpha1.AdminService
// service. By default, it uses the Connect protocol with the binary Protobuf Codec, asks for
// gzipped responses, and sends uncompressed requests. To use the gRPC or gRPC-Web protocols, supply
// the connect.WithGRPC() or connect.WithGRPCWeb() options.
//
// The URL supplied here should be the base URL for the Connect or gRPC server (for example,
// http://api.acme.com or https://acme.com/grpc).
func NewAdminServiceClient(httpClient connect_go.HTTPClient, baseURL string, opts ...connect_go.ClientOption) AdminServiceClient {
	baseURL = strings.TrimRight(baseURL, "/")
	return &adminServiceClient{
		forceDeleteUser: connect_go.NewClient[v1alpha1.ForceDeleteUserRequest, v1alpha1.ForceDeleteUserResponse](
			httpClient,
			baseURL+"/buf.alpha.registry.v1alpha1.AdminService/ForceDeleteUser",
			opts...,
		),
		updateUserVerificationStatus: connect_go.NewClient[v1alpha1.UpdateUserVerificationStatusRequest, v1alpha1.UpdateUserVerificationStatusResponse](
			httpClient,
			baseURL+"/buf.alpha.registry.v1alpha1.AdminService/UpdateUserVerificationStatus",
			opts...,
		),
		updateOrganizationVerificationStatus: connect_go.NewClient[v1alpha1.UpdateOrganizationVerificationStatusRequest, v1alpha1.UpdateOrganizationVerificationStatusResponse](
			httpClient,
			baseURL+"/buf.alpha.registry.v1alpha1.AdminService/UpdateOrganizationVerificationStatus",
			opts...,
		),
		createMachineUser: connect_go.NewClient[v1alpha1.CreateMachineUserRequest, v1alpha1.CreateMachineUserResponse](
			httpClient,
			baseURL+"/buf.alpha.registry.v1alpha1.AdminService/CreateMachineUser",
			opts...,
		),
	}
}

// adminServiceClient implements AdminServiceClient.
type adminServiceClient struct {
	forceDeleteUser                      *connect_go.Client[v1alpha1.ForceDeleteUserRequest, v1alpha1.ForceDeleteUserResponse]
	updateUserVerificationStatus         *connect_go.Client[v1alpha1.UpdateUserVerificationStatusRequest, v1alpha1.UpdateUserVerificationStatusResponse]
	updateOrganizationVerificationStatus *connect_go.Client[v1alpha1.UpdateOrganizationVerificationStatusRequest, v1alpha1.UpdateOrganizationVerificationStatusResponse]
	createMachineUser                    *connect_go.Client[v1alpha1.CreateMachineUserRequest, v1alpha1.CreateMachineUserResponse]
}

// ForceDeleteUser calls buf.alpha.registry.v1alpha1.AdminService.ForceDeleteUser.
func (c *adminServiceClient) ForceDeleteUser(ctx context.Context, req *connect_go.Request[v1alpha1.ForceDeleteUserRequest]) (*connect_go.Response[v1alpha1.ForceDeleteUserResponse], error) {
	return c.forceDeleteUser.CallUnary(ctx, req)
}

// UpdateUserVerificationStatus calls
// buf.alpha.registry.v1alpha1.AdminService.UpdateUserVerificationStatus.
func (c *adminServiceClient) UpdateUserVerificationStatus(ctx context.Context, req *connect_go.Request[v1alpha1.UpdateUserVerificationStatusRequest]) (*connect_go.Response[v1alpha1.UpdateUserVerificationStatusResponse], error) {
	return c.updateUserVerificationStatus.CallUnary(ctx, req)
}

// UpdateOrganizationVerificationStatus calls
// buf.alpha.registry.v1alpha1.AdminService.UpdateOrganizationVerificationStatus.
func (c *adminServiceClient) UpdateOrganizationVerificationStatus(ctx context.Context, req *connect_go.Request[v1alpha1.UpdateOrganizationVerificationStatusRequest]) (*connect_go.Response[v1alpha1.UpdateOrganizationVerificationStatusResponse], error) {
	return c.updateOrganizationVerificationStatus.CallUnary(ctx, req)
}

// CreateMachineUser calls buf.alpha.registry.v1alpha1.AdminService.CreateMachineUser.
func (c *adminServiceClient) CreateMachineUser(ctx context.Context, req *connect_go.Request[v1alpha1.CreateMachineUserRequest]) (*connect_go.Response[v1alpha1.CreateMachineUserResponse], error) {
	return c.createMachineUser.CallUnary(ctx, req)
}

// AdminServiceHandler is an implementation of the buf.alpha.registry.v1alpha1.AdminService service.
type AdminServiceHandler interface {
	// ForceDeleteUser forces to delete a user. Resources and organizations that are
	// solely owned by the user will also be deleted.
	ForceDeleteUser(context.Context, *connect_go.Request[v1alpha1.ForceDeleteUserRequest]) (*connect_go.Response[v1alpha1.ForceDeleteUserResponse], error)
	// Update a user's verification status.
	UpdateUserVerificationStatus(context.Context, *connect_go.Request[v1alpha1.UpdateUserVerificationStatusRequest]) (*connect_go.Response[v1alpha1.UpdateUserVerificationStatusResponse], error)
	// Update a organization's verification.
	UpdateOrganizationVerificationStatus(context.Context, *connect_go.Request[v1alpha1.UpdateOrganizationVerificationStatusRequest]) (*connect_go.Response[v1alpha1.UpdateOrganizationVerificationStatusResponse], error)
	// Create a new machine user on the server.
	CreateMachineUser(context.Context, *connect_go.Request[v1alpha1.CreateMachineUserRequest]) (*connect_go.Response[v1alpha1.CreateMachineUserResponse], error)
}

// NewAdminServiceHandler builds an HTTP handler from the service implementation. It returns the
// path on which to mount the handler and the handler itself.
//
// By default, handlers support the Connect, gRPC, and gRPC-Web protocols with the binary Protobuf
// and JSON codecs. They also support gzip compression.
func NewAdminServiceHandler(svc AdminServiceHandler, opts ...connect_go.HandlerOption) (string, http.Handler) {
	mux := http.NewServeMux()
	mux.Handle("/buf.alpha.registry.v1alpha1.AdminService/ForceDeleteUser", connect_go.NewUnaryHandler(
		"/buf.alpha.registry.v1alpha1.AdminService/ForceDeleteUser",
		svc.ForceDeleteUser,
		opts...,
	))
	mux.Handle("/buf.alpha.registry.v1alpha1.AdminService/UpdateUserVerificationStatus", connect_go.NewUnaryHandler(
		"/buf.alpha.registry.v1alpha1.AdminService/UpdateUserVerificationStatus",
		svc.UpdateUserVerificationStatus,
		opts...,
	))
	mux.Handle("/buf.alpha.registry.v1alpha1.AdminService/UpdateOrganizationVerificationStatus", connect_go.NewUnaryHandler(
		"/buf.alpha.registry.v1alpha1.AdminService/UpdateOrganizationVerificationStatus",
		svc.UpdateOrganizationVerificationStatus,
		opts...,
	))
	mux.Handle("/buf.alpha.registry.v1alpha1.AdminService/CreateMachineUser", connect_go.NewUnaryHandler(
		"/buf.alpha.registry.v1alpha1.AdminService/CreateMachineUser",
		svc.CreateMachineUser,
		opts...,
	))
	return "/buf.alpha.registry.v1alpha1.AdminService/", mux
}

// UnimplementedAdminServiceHandler returns CodeUnimplemented from all methods.
type UnimplementedAdminServiceHandler struct{}

func (UnimplementedAdminServiceHandler) ForceDeleteUser(context.Context, *connect_go.Request[v1alpha1.ForceDeleteUserRequest]) (*connect_go.Response[v1alpha1.ForceDeleteUserResponse], error) {
	return nil, connect_go.NewError(connect_go.CodeUnimplemented, errors.New("buf.alpha.registry.v1alpha1.AdminService.ForceDeleteUser is not implemented"))
}

func (UnimplementedAdminServiceHandler) UpdateUserVerificationStatus(context.Context, *connect_go.Request[v1alpha1.UpdateUserVerificationStatusRequest]) (*connect_go.Response[v1alpha1.UpdateUserVerificationStatusResponse], error) {
	return nil, connect_go.NewError(connect_go.CodeUnimplemented, errors.New("buf.alpha.registry.v1alpha1.AdminService.UpdateUserVerificationStatus is not implemented"))
}

func (UnimplementedAdminServiceHandler) UpdateOrganizationVerificationStatus(context.Context, *connect_go.Request[v1alpha1.UpdateOrganizationVerificationStatusRequest]) (*connect_go.Response[v1alpha1.UpdateOrganizationVerificationStatusResponse], error) {
	return nil, connect_go.NewError(connect_go.CodeUnimplemented, errors.New("buf.alpha.registry.v1alpha1.AdminService.UpdateOrganizationVerificationStatus is not implemented"))
}

func (UnimplementedAdminServiceHandler) CreateMachineUser(context.Context, *connect_go.Request[v1alpha1.CreateMachineUserRequest]) (*connect_go.Response[v1alpha1.CreateMachineUserResponse], error) {
	return nil, connect_go.NewError(connect_go.CodeUnimplemented, errors.New("buf.alpha.registry.v1alpha1.AdminService.CreateMachineUser is not implemented"))
}
