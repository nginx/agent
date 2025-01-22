// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.36.3
// 	protoc        (unknown)
// source: mpi/v1/common.proto

package v1

import (
	_ "buf.build/gen/go/bufbuild/protovalidate/protocolbuffers/go/buf/validate"
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
	timestamppb "google.golang.org/protobuf/types/known/timestamppb"
	reflect "reflect"
	sync "sync"
)

const (
	// Verify that this generated code is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(20 - protoimpl.MinVersion)
	// Verify that runtime/protoimpl is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(protoimpl.MaxVersion - 20)
)

// Command status enum
type CommandResponse_CommandStatus int32

const (
	// Unspecified status of command
	CommandResponse_COMMAND_STATUS_UNSPECIFIED CommandResponse_CommandStatus = 0
	// Command was successful
	CommandResponse_COMMAND_STATUS_OK CommandResponse_CommandStatus = 1
	// Command error
	CommandResponse_COMMAND_STATUS_ERROR CommandResponse_CommandStatus = 2
	// Command in-progress
	CommandResponse_COMMAND_STATUS_IN_PROGRESS CommandResponse_CommandStatus = 3
	// Command failure
	CommandResponse_COMMAND_STATUS_FAILURE CommandResponse_CommandStatus = 4
)

// Enum value maps for CommandResponse_CommandStatus.
var (
	CommandResponse_CommandStatus_name = map[int32]string{
		0: "COMMAND_STATUS_UNSPECIFIED",
		1: "COMMAND_STATUS_OK",
		2: "COMMAND_STATUS_ERROR",
		3: "COMMAND_STATUS_IN_PROGRESS",
		4: "COMMAND_STATUS_FAILURE",
	}
	CommandResponse_CommandStatus_value = map[string]int32{
		"COMMAND_STATUS_UNSPECIFIED": 0,
		"COMMAND_STATUS_OK":          1,
		"COMMAND_STATUS_ERROR":       2,
		"COMMAND_STATUS_IN_PROGRESS": 3,
		"COMMAND_STATUS_FAILURE":     4,
	}
)

func (x CommandResponse_CommandStatus) Enum() *CommandResponse_CommandStatus {
	p := new(CommandResponse_CommandStatus)
	*p = x
	return p
}

func (x CommandResponse_CommandStatus) String() string {
	return protoimpl.X.EnumStringOf(x.Descriptor(), protoreflect.EnumNumber(x))
}

func (CommandResponse_CommandStatus) Descriptor() protoreflect.EnumDescriptor {
	return file_mpi_v1_common_proto_enumTypes[0].Descriptor()
}

func (CommandResponse_CommandStatus) Type() protoreflect.EnumType {
	return &file_mpi_v1_common_proto_enumTypes[0]
}

func (x CommandResponse_CommandStatus) Number() protoreflect.EnumNumber {
	return protoreflect.EnumNumber(x)
}

// Deprecated: Use CommandResponse_CommandStatus.Descriptor instead.
func (CommandResponse_CommandStatus) EnumDescriptor() ([]byte, []int) {
	return file_mpi_v1_common_proto_rawDescGZIP(), []int{1, 0}
}

type ServerSettings_ServerType int32

const (
	// Undefined server type
	ServerSettings_SERVER_SETTINGS_TYPE_UNDEFINED ServerSettings_ServerType = 0
	// gRPC server type
	ServerSettings_SERVER_SETTINGS_TYPE_GRPC ServerSettings_ServerType = 1
	// HTTP server type
	ServerSettings_SERVER_SETTINGS_TYPE_HTTP ServerSettings_ServerType = 2
)

// Enum value maps for ServerSettings_ServerType.
var (
	ServerSettings_ServerType_name = map[int32]string{
		0: "SERVER_SETTINGS_TYPE_UNDEFINED",
		1: "SERVER_SETTINGS_TYPE_GRPC",
		2: "SERVER_SETTINGS_TYPE_HTTP",
	}
	ServerSettings_ServerType_value = map[string]int32{
		"SERVER_SETTINGS_TYPE_UNDEFINED": 0,
		"SERVER_SETTINGS_TYPE_GRPC":      1,
		"SERVER_SETTINGS_TYPE_HTTP":      2,
	}
)

func (x ServerSettings_ServerType) Enum() *ServerSettings_ServerType {
	p := new(ServerSettings_ServerType)
	*p = x
	return p
}

func (x ServerSettings_ServerType) String() string {
	return protoimpl.X.EnumStringOf(x.Descriptor(), protoreflect.EnumNumber(x))
}

func (ServerSettings_ServerType) Descriptor() protoreflect.EnumDescriptor {
	return file_mpi_v1_common_proto_enumTypes[1].Descriptor()
}

func (ServerSettings_ServerType) Type() protoreflect.EnumType {
	return &file_mpi_v1_common_proto_enumTypes[1]
}

func (x ServerSettings_ServerType) Number() protoreflect.EnumNumber {
	return protoreflect.EnumNumber(x)
}

// Deprecated: Use ServerSettings_ServerType.Descriptor instead.
func (ServerSettings_ServerType) EnumDescriptor() ([]byte, []int) {
	return file_mpi_v1_common_proto_rawDescGZIP(), []int{2, 0}
}

// Meta-information associated with a message
type MessageMeta struct {
	state protoimpl.MessageState `protogen:"open.v1"`
	// uuid v7 monotonically increasing string
	MessageId string `protobuf:"bytes,1,opt,name=message_id,json=messageId,proto3" json:"message_id,omitempty"`
	// if 2 or more messages associated with the same workflow, use this field as an association
	CorrelationId string `protobuf:"bytes,2,opt,name=correlation_id,json=correlationId,proto3" json:"correlation_id,omitempty"`
	// timestamp for human readable timestamp in UTC format
	Timestamp     *timestamppb.Timestamp `protobuf:"bytes,3,opt,name=timestamp,proto3" json:"timestamp,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *MessageMeta) Reset() {
	*x = MessageMeta{}
	mi := &file_mpi_v1_common_proto_msgTypes[0]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *MessageMeta) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*MessageMeta) ProtoMessage() {}

func (x *MessageMeta) ProtoReflect() protoreflect.Message {
	mi := &file_mpi_v1_common_proto_msgTypes[0]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use MessageMeta.ProtoReflect.Descriptor instead.
func (*MessageMeta) Descriptor() ([]byte, []int) {
	return file_mpi_v1_common_proto_rawDescGZIP(), []int{0}
}

func (x *MessageMeta) GetMessageId() string {
	if x != nil {
		return x.MessageId
	}
	return ""
}

func (x *MessageMeta) GetCorrelationId() string {
	if x != nil {
		return x.CorrelationId
	}
	return ""
}

func (x *MessageMeta) GetTimestamp() *timestamppb.Timestamp {
	if x != nil {
		return x.Timestamp
	}
	return nil
}

// Represents a the status response of an command
type CommandResponse struct {
	state protoimpl.MessageState `protogen:"open.v1"`
	// Command status
	Status CommandResponse_CommandStatus `protobuf:"varint,1,opt,name=status,proto3,enum=mpi.v1.CommandResponse_CommandStatus" json:"status,omitempty"`
	// Provides a user friendly message to describe the response
	Message string `protobuf:"bytes,2,opt,name=message,proto3" json:"message,omitempty"`
	// Provides an error message of why the command failed, only populated when CommandStatus is COMMAND_STATUS_ERROR
	Error         string `protobuf:"bytes,3,opt,name=error,proto3" json:"error,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *CommandResponse) Reset() {
	*x = CommandResponse{}
	mi := &file_mpi_v1_common_proto_msgTypes[1]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *CommandResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*CommandResponse) ProtoMessage() {}

func (x *CommandResponse) ProtoReflect() protoreflect.Message {
	mi := &file_mpi_v1_common_proto_msgTypes[1]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use CommandResponse.ProtoReflect.Descriptor instead.
func (*CommandResponse) Descriptor() ([]byte, []int) {
	return file_mpi_v1_common_proto_rawDescGZIP(), []int{1}
}

func (x *CommandResponse) GetStatus() CommandResponse_CommandStatus {
	if x != nil {
		return x.Status
	}
	return CommandResponse_COMMAND_STATUS_UNSPECIFIED
}

func (x *CommandResponse) GetMessage() string {
	if x != nil {
		return x.Message
	}
	return ""
}

func (x *CommandResponse) GetError() string {
	if x != nil {
		return x.Error
	}
	return ""
}

// The top-level configuration for the command server
type ServerSettings struct {
	state protoimpl.MessageState `protogen:"open.v1"`
	// Command server host
	Host string `protobuf:"bytes,1,opt,name=host,proto3" json:"host,omitempty"`
	// Command server port
	Port int32 `protobuf:"varint,2,opt,name=port,proto3" json:"port,omitempty"`
	// Server type (enum for gRPC, HTTP, etc.)
	Type          ServerSettings_ServerType `protobuf:"varint,3,opt,name=type,proto3,enum=mpi.v1.ServerSettings_ServerType" json:"type,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *ServerSettings) Reset() {
	*x = ServerSettings{}
	mi := &file_mpi_v1_common_proto_msgTypes[2]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *ServerSettings) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ServerSettings) ProtoMessage() {}

func (x *ServerSettings) ProtoReflect() protoreflect.Message {
	mi := &file_mpi_v1_common_proto_msgTypes[2]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ServerSettings.ProtoReflect.Descriptor instead.
func (*ServerSettings) Descriptor() ([]byte, []int) {
	return file_mpi_v1_common_proto_rawDescGZIP(), []int{2}
}

func (x *ServerSettings) GetHost() string {
	if x != nil {
		return x.Host
	}
	return ""
}

func (x *ServerSettings) GetPort() int32 {
	if x != nil {
		return x.Port
	}
	return 0
}

func (x *ServerSettings) GetType() ServerSettings_ServerType {
	if x != nil {
		return x.Type
	}
	return ServerSettings_SERVER_SETTINGS_TYPE_UNDEFINED
}

// Defines the authentication configuration
type AuthSettings struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *AuthSettings) Reset() {
	*x = AuthSettings{}
	mi := &file_mpi_v1_common_proto_msgTypes[3]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *AuthSettings) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*AuthSettings) ProtoMessage() {}

func (x *AuthSettings) ProtoReflect() protoreflect.Message {
	mi := &file_mpi_v1_common_proto_msgTypes[3]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use AuthSettings.ProtoReflect.Descriptor instead.
func (*AuthSettings) Descriptor() ([]byte, []int) {
	return file_mpi_v1_common_proto_rawDescGZIP(), []int{3}
}

type TLSSettings struct {
	state protoimpl.MessageState `protogen:"open.v1"`
	// TLS certificate for the command server (e.g., "/path/to/cert.pem")
	Cert string `protobuf:"bytes,1,opt,name=cert,proto3" json:"cert,omitempty"`
	// TLS key for the command server (e.g., "/path/to/key.pem")
	Key string `protobuf:"bytes,2,opt,name=key,proto3" json:"key,omitempty"`
	// CA certificate for the command server (e.g., "/path/to/ca.pem")
	Ca string `protobuf:"bytes,3,opt,name=ca,proto3" json:"ca,omitempty"`
	// Controls whether a client verifies the server's certificate chain and host name.
	// If skip_verify is true, accepts any certificate presented by the server and any host name in that certificate.
	SkipVerify bool `protobuf:"varint,4,opt,name=skip_verify,json=skipVerify,proto3" json:"skip_verify,omitempty"`
	// Server name for TLS
	ServerName    string `protobuf:"bytes,5,opt,name=server_name,json=serverName,proto3" json:"server_name,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *TLSSettings) Reset() {
	*x = TLSSettings{}
	mi := &file_mpi_v1_common_proto_msgTypes[4]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *TLSSettings) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*TLSSettings) ProtoMessage() {}

func (x *TLSSettings) ProtoReflect() protoreflect.Message {
	mi := &file_mpi_v1_common_proto_msgTypes[4]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use TLSSettings.ProtoReflect.Descriptor instead.
func (*TLSSettings) Descriptor() ([]byte, []int) {
	return file_mpi_v1_common_proto_rawDescGZIP(), []int{4}
}

func (x *TLSSettings) GetCert() string {
	if x != nil {
		return x.Cert
	}
	return ""
}

func (x *TLSSettings) GetKey() string {
	if x != nil {
		return x.Key
	}
	return ""
}

func (x *TLSSettings) GetCa() string {
	if x != nil {
		return x.Ca
	}
	return ""
}

func (x *TLSSettings) GetSkipVerify() bool {
	if x != nil {
		return x.SkipVerify
	}
	return false
}

func (x *TLSSettings) GetServerName() string {
	if x != nil {
		return x.ServerName
	}
	return ""
}

var File_mpi_v1_common_proto protoreflect.FileDescriptor

var file_mpi_v1_common_proto_rawDesc = []byte{
	0x0a, 0x13, 0x6d, 0x70, 0x69, 0x2f, 0x76, 0x31, 0x2f, 0x63, 0x6f, 0x6d, 0x6d, 0x6f, 0x6e, 0x2e,
	0x70, 0x72, 0x6f, 0x74, 0x6f, 0x12, 0x06, 0x6d, 0x70, 0x69, 0x2e, 0x76, 0x31, 0x1a, 0x1f, 0x67,
	0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2f, 0x74,
	0x69, 0x6d, 0x65, 0x73, 0x74, 0x61, 0x6d, 0x70, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x1a, 0x1b,
	0x62, 0x75, 0x66, 0x2f, 0x76, 0x61, 0x6c, 0x69, 0x64, 0x61, 0x74, 0x65, 0x2f, 0x76, 0x61, 0x6c,
	0x69, 0x64, 0x61, 0x74, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x22, 0x8d, 0x01, 0x0a, 0x0b,
	0x4d, 0x65, 0x73, 0x73, 0x61, 0x67, 0x65, 0x4d, 0x65, 0x74, 0x61, 0x12, 0x1d, 0x0a, 0x0a, 0x6d,
	0x65, 0x73, 0x73, 0x61, 0x67, 0x65, 0x5f, 0x69, 0x64, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52,
	0x09, 0x6d, 0x65, 0x73, 0x73, 0x61, 0x67, 0x65, 0x49, 0x64, 0x12, 0x25, 0x0a, 0x0e, 0x63, 0x6f,
	0x72, 0x72, 0x65, 0x6c, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x5f, 0x69, 0x64, 0x18, 0x02, 0x20, 0x01,
	0x28, 0x09, 0x52, 0x0d, 0x63, 0x6f, 0x72, 0x72, 0x65, 0x6c, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x49,
	0x64, 0x12, 0x38, 0x0a, 0x09, 0x74, 0x69, 0x6d, 0x65, 0x73, 0x74, 0x61, 0x6d, 0x70, 0x18, 0x03,
	0x20, 0x01, 0x28, 0x0b, 0x32, 0x1a, 0x2e, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2e, 0x70, 0x72,
	0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2e, 0x54, 0x69, 0x6d, 0x65, 0x73, 0x74, 0x61, 0x6d, 0x70,
	0x52, 0x09, 0x74, 0x69, 0x6d, 0x65, 0x73, 0x74, 0x61, 0x6d, 0x70, 0x22, 0x9f, 0x02, 0x0a, 0x0f,
	0x43, 0x6f, 0x6d, 0x6d, 0x61, 0x6e, 0x64, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x12,
	0x3d, 0x0a, 0x06, 0x73, 0x74, 0x61, 0x74, 0x75, 0x73, 0x18, 0x01, 0x20, 0x01, 0x28, 0x0e, 0x32,
	0x25, 0x2e, 0x6d, 0x70, 0x69, 0x2e, 0x76, 0x31, 0x2e, 0x43, 0x6f, 0x6d, 0x6d, 0x61, 0x6e, 0x64,
	0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x2e, 0x43, 0x6f, 0x6d, 0x6d, 0x61, 0x6e, 0x64,
	0x53, 0x74, 0x61, 0x74, 0x75, 0x73, 0x52, 0x06, 0x73, 0x74, 0x61, 0x74, 0x75, 0x73, 0x12, 0x18,
	0x0a, 0x07, 0x6d, 0x65, 0x73, 0x73, 0x61, 0x67, 0x65, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52,
	0x07, 0x6d, 0x65, 0x73, 0x73, 0x61, 0x67, 0x65, 0x12, 0x14, 0x0a, 0x05, 0x65, 0x72, 0x72, 0x6f,
	0x72, 0x18, 0x03, 0x20, 0x01, 0x28, 0x09, 0x52, 0x05, 0x65, 0x72, 0x72, 0x6f, 0x72, 0x22, 0x9c,
	0x01, 0x0a, 0x0d, 0x43, 0x6f, 0x6d, 0x6d, 0x61, 0x6e, 0x64, 0x53, 0x74, 0x61, 0x74, 0x75, 0x73,
	0x12, 0x1e, 0x0a, 0x1a, 0x43, 0x4f, 0x4d, 0x4d, 0x41, 0x4e, 0x44, 0x5f, 0x53, 0x54, 0x41, 0x54,
	0x55, 0x53, 0x5f, 0x55, 0x4e, 0x53, 0x50, 0x45, 0x43, 0x49, 0x46, 0x49, 0x45, 0x44, 0x10, 0x00,
	0x12, 0x15, 0x0a, 0x11, 0x43, 0x4f, 0x4d, 0x4d, 0x41, 0x4e, 0x44, 0x5f, 0x53, 0x54, 0x41, 0x54,
	0x55, 0x53, 0x5f, 0x4f, 0x4b, 0x10, 0x01, 0x12, 0x18, 0x0a, 0x14, 0x43, 0x4f, 0x4d, 0x4d, 0x41,
	0x4e, 0x44, 0x5f, 0x53, 0x54, 0x41, 0x54, 0x55, 0x53, 0x5f, 0x45, 0x52, 0x52, 0x4f, 0x52, 0x10,
	0x02, 0x12, 0x1e, 0x0a, 0x1a, 0x43, 0x4f, 0x4d, 0x4d, 0x41, 0x4e, 0x44, 0x5f, 0x53, 0x54, 0x41,
	0x54, 0x55, 0x53, 0x5f, 0x49, 0x4e, 0x5f, 0x50, 0x52, 0x4f, 0x47, 0x52, 0x45, 0x53, 0x53, 0x10,
	0x03, 0x12, 0x1a, 0x0a, 0x16, 0x43, 0x4f, 0x4d, 0x4d, 0x41, 0x4e, 0x44, 0x5f, 0x53, 0x54, 0x41,
	0x54, 0x55, 0x53, 0x5f, 0x46, 0x41, 0x49, 0x4c, 0x55, 0x52, 0x45, 0x10, 0x04, 0x22, 0xec, 0x01,
	0x0a, 0x0e, 0x53, 0x65, 0x72, 0x76, 0x65, 0x72, 0x53, 0x65, 0x74, 0x74, 0x69, 0x6e, 0x67, 0x73,
	0x12, 0x12, 0x0a, 0x04, 0x68, 0x6f, 0x73, 0x74, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x04,
	0x68, 0x6f, 0x73, 0x74, 0x12, 0x1f, 0x0a, 0x04, 0x70, 0x6f, 0x72, 0x74, 0x18, 0x02, 0x20, 0x01,
	0x28, 0x05, 0x42, 0x0b, 0xba, 0x48, 0x08, 0x1a, 0x06, 0x18, 0xff, 0xff, 0x03, 0x28, 0x01, 0x52,
	0x04, 0x70, 0x6f, 0x72, 0x74, 0x12, 0x35, 0x0a, 0x04, 0x74, 0x79, 0x70, 0x65, 0x18, 0x03, 0x20,
	0x01, 0x28, 0x0e, 0x32, 0x21, 0x2e, 0x6d, 0x70, 0x69, 0x2e, 0x76, 0x31, 0x2e, 0x53, 0x65, 0x72,
	0x76, 0x65, 0x72, 0x53, 0x65, 0x74, 0x74, 0x69, 0x6e, 0x67, 0x73, 0x2e, 0x53, 0x65, 0x72, 0x76,
	0x65, 0x72, 0x54, 0x79, 0x70, 0x65, 0x52, 0x04, 0x74, 0x79, 0x70, 0x65, 0x22, 0x6e, 0x0a, 0x0a,
	0x53, 0x65, 0x72, 0x76, 0x65, 0x72, 0x54, 0x79, 0x70, 0x65, 0x12, 0x22, 0x0a, 0x1e, 0x53, 0x45,
	0x52, 0x56, 0x45, 0x52, 0x5f, 0x53, 0x45, 0x54, 0x54, 0x49, 0x4e, 0x47, 0x53, 0x5f, 0x54, 0x59,
	0x50, 0x45, 0x5f, 0x55, 0x4e, 0x44, 0x45, 0x46, 0x49, 0x4e, 0x45, 0x44, 0x10, 0x00, 0x12, 0x1d,
	0x0a, 0x19, 0x53, 0x45, 0x52, 0x56, 0x45, 0x52, 0x5f, 0x53, 0x45, 0x54, 0x54, 0x49, 0x4e, 0x47,
	0x53, 0x5f, 0x54, 0x59, 0x50, 0x45, 0x5f, 0x47, 0x52, 0x50, 0x43, 0x10, 0x01, 0x12, 0x1d, 0x0a,
	0x19, 0x53, 0x45, 0x52, 0x56, 0x45, 0x52, 0x5f, 0x53, 0x45, 0x54, 0x54, 0x49, 0x4e, 0x47, 0x53,
	0x5f, 0x54, 0x59, 0x50, 0x45, 0x5f, 0x48, 0x54, 0x54, 0x50, 0x10, 0x02, 0x22, 0x0e, 0x0a, 0x0c,
	0x41, 0x75, 0x74, 0x68, 0x53, 0x65, 0x74, 0x74, 0x69, 0x6e, 0x67, 0x73, 0x22, 0x85, 0x01, 0x0a,
	0x0b, 0x54, 0x4c, 0x53, 0x53, 0x65, 0x74, 0x74, 0x69, 0x6e, 0x67, 0x73, 0x12, 0x12, 0x0a, 0x04,
	0x63, 0x65, 0x72, 0x74, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x04, 0x63, 0x65, 0x72, 0x74,
	0x12, 0x10, 0x0a, 0x03, 0x6b, 0x65, 0x79, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x03, 0x6b,
	0x65, 0x79, 0x12, 0x0e, 0x0a, 0x02, 0x63, 0x61, 0x18, 0x03, 0x20, 0x01, 0x28, 0x09, 0x52, 0x02,
	0x63, 0x61, 0x12, 0x1f, 0x0a, 0x0b, 0x73, 0x6b, 0x69, 0x70, 0x5f, 0x76, 0x65, 0x72, 0x69, 0x66,
	0x79, 0x18, 0x04, 0x20, 0x01, 0x28, 0x08, 0x52, 0x0a, 0x73, 0x6b, 0x69, 0x70, 0x56, 0x65, 0x72,
	0x69, 0x66, 0x79, 0x12, 0x1f, 0x0a, 0x0b, 0x73, 0x65, 0x72, 0x76, 0x65, 0x72, 0x5f, 0x6e, 0x61,
	0x6d, 0x65, 0x18, 0x05, 0x20, 0x01, 0x28, 0x09, 0x52, 0x0a, 0x73, 0x65, 0x72, 0x76, 0x65, 0x72,
	0x4e, 0x61, 0x6d, 0x65, 0x42, 0x08, 0x5a, 0x06, 0x6d, 0x70, 0x69, 0x2f, 0x76, 0x31, 0x62, 0x06,
	0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_mpi_v1_common_proto_rawDescOnce sync.Once
	file_mpi_v1_common_proto_rawDescData = file_mpi_v1_common_proto_rawDesc
)

func file_mpi_v1_common_proto_rawDescGZIP() []byte {
	file_mpi_v1_common_proto_rawDescOnce.Do(func() {
		file_mpi_v1_common_proto_rawDescData = protoimpl.X.CompressGZIP(file_mpi_v1_common_proto_rawDescData)
	})
	return file_mpi_v1_common_proto_rawDescData
}

var file_mpi_v1_common_proto_enumTypes = make([]protoimpl.EnumInfo, 2)
var file_mpi_v1_common_proto_msgTypes = make([]protoimpl.MessageInfo, 5)
var file_mpi_v1_common_proto_goTypes = []any{
	(CommandResponse_CommandStatus)(0), // 0: mpi.v1.CommandResponse.CommandStatus
	(ServerSettings_ServerType)(0),     // 1: mpi.v1.ServerSettings.ServerType
	(*MessageMeta)(nil),                // 2: mpi.v1.MessageMeta
	(*CommandResponse)(nil),            // 3: mpi.v1.CommandResponse
	(*ServerSettings)(nil),             // 4: mpi.v1.ServerSettings
	(*AuthSettings)(nil),               // 5: mpi.v1.AuthSettings
	(*TLSSettings)(nil),                // 6: mpi.v1.TLSSettings
	(*timestamppb.Timestamp)(nil),      // 7: google.protobuf.Timestamp
}
var file_mpi_v1_common_proto_depIdxs = []int32{
	7, // 0: mpi.v1.MessageMeta.timestamp:type_name -> google.protobuf.Timestamp
	0, // 1: mpi.v1.CommandResponse.status:type_name -> mpi.v1.CommandResponse.CommandStatus
	1, // 2: mpi.v1.ServerSettings.type:type_name -> mpi.v1.ServerSettings.ServerType
	3, // [3:3] is the sub-list for method output_type
	3, // [3:3] is the sub-list for method input_type
	3, // [3:3] is the sub-list for extension type_name
	3, // [3:3] is the sub-list for extension extendee
	0, // [0:3] is the sub-list for field type_name
}

func init() { file_mpi_v1_common_proto_init() }
func file_mpi_v1_common_proto_init() {
	if File_mpi_v1_common_proto != nil {
		return
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: file_mpi_v1_common_proto_rawDesc,
			NumEnums:      2,
			NumMessages:   5,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_mpi_v1_common_proto_goTypes,
		DependencyIndexes: file_mpi_v1_common_proto_depIdxs,
		EnumInfos:         file_mpi_v1_common_proto_enumTypes,
		MessageInfos:      file_mpi_v1_common_proto_msgTypes,
	}.Build()
	File_mpi_v1_common_proto = out.File
	file_mpi_v1_common_proto_rawDesc = nil
	file_mpi_v1_common_proto_goTypes = nil
	file_mpi_v1_common_proto_depIdxs = nil
}
