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

// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.31.0
// 	protoc        (unknown)
// source: buf/alpha/registry/v1alpha1/studio.proto

package registryv1alpha1

import (
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
	reflect "reflect"
	sync "sync"
)

const (
	// Verify that this generated code is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(20 - protoimpl.MinVersion)
	// Verify that runtime/protoimpl is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(protoimpl.MaxVersion - 20)
)

// The protocols supported by Studio agent.
type StudioAgentProtocol int32

const (
	StudioAgentProtocol_STUDIO_AGENT_PROTOCOL_UNSPECIFIED StudioAgentProtocol = 0
	StudioAgentProtocol_STUDIO_AGENT_PROTOCOL_GRPC        StudioAgentProtocol = 1
	StudioAgentProtocol_STUDIO_AGENT_PROTOCOL_CONNECT     StudioAgentProtocol = 2
)

// Enum value maps for StudioAgentProtocol.
var (
	StudioAgentProtocol_name = map[int32]string{
		0: "STUDIO_AGENT_PROTOCOL_UNSPECIFIED",
		1: "STUDIO_AGENT_PROTOCOL_GRPC",
		2: "STUDIO_AGENT_PROTOCOL_CONNECT",
	}
	StudioAgentProtocol_value = map[string]int32{
		"STUDIO_AGENT_PROTOCOL_UNSPECIFIED": 0,
		"STUDIO_AGENT_PROTOCOL_GRPC":        1,
		"STUDIO_AGENT_PROTOCOL_CONNECT":     2,
	}
)

func (x StudioAgentProtocol) Enum() *StudioAgentProtocol {
	p := new(StudioAgentProtocol)
	*p = x
	return p
}

func (x StudioAgentProtocol) String() string {
	return protoimpl.X.EnumStringOf(x.Descriptor(), protoreflect.EnumNumber(x))
}

func (StudioAgentProtocol) Descriptor() protoreflect.EnumDescriptor {
	return file_buf_alpha_registry_v1alpha1_studio_proto_enumTypes[0].Descriptor()
}

func (StudioAgentProtocol) Type() protoreflect.EnumType {
	return &file_buf_alpha_registry_v1alpha1_studio_proto_enumTypes[0]
}

func (x StudioAgentProtocol) Number() protoreflect.EnumNumber {
	return protoreflect.EnumNumber(x)
}

// Deprecated: Use StudioAgentProtocol.Descriptor instead.
func (StudioAgentProtocol) EnumDescriptor() ([]byte, []int) {
	return file_buf_alpha_registry_v1alpha1_studio_proto_rawDescGZIP(), []int{0}
}

// StudioAgentPreset is the information about an agent preset in the Studio.
type StudioAgentPreset struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// The target agent URL in the Studio.
	Url string `protobuf:"bytes,1,opt,name=url,proto3" json:"url,omitempty"`
	// The optional alias of the agent URL.
	Alias string `protobuf:"bytes,2,opt,name=alias,proto3" json:"alias,omitempty"`
	// The protocol the agent should use to forward requests.
	Protocol StudioAgentProtocol `protobuf:"varint,3,opt,name=protocol,proto3,enum=buf.alpha.registry.v1alpha1.StudioAgentProtocol" json:"protocol,omitempty"`
}

func (x *StudioAgentPreset) Reset() {
	*x = StudioAgentPreset{}
	if protoimpl.UnsafeEnabled {
		mi := &file_buf_alpha_registry_v1alpha1_studio_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *StudioAgentPreset) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*StudioAgentPreset) ProtoMessage() {}

func (x *StudioAgentPreset) ProtoReflect() protoreflect.Message {
	mi := &file_buf_alpha_registry_v1alpha1_studio_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use StudioAgentPreset.ProtoReflect.Descriptor instead.
func (*StudioAgentPreset) Descriptor() ([]byte, []int) {
	return file_buf_alpha_registry_v1alpha1_studio_proto_rawDescGZIP(), []int{0}
}

func (x *StudioAgentPreset) GetUrl() string {
	if x != nil {
		return x.Url
	}
	return ""
}

func (x *StudioAgentPreset) GetAlias() string {
	if x != nil {
		return x.Alias
	}
	return ""
}

func (x *StudioAgentPreset) GetProtocol() StudioAgentProtocol {
	if x != nil {
		return x.Protocol
	}
	return StudioAgentProtocol_STUDIO_AGENT_PROTOCOL_UNSPECIFIED
}

type ListStudioAgentPresetsRequest struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields
}

func (x *ListStudioAgentPresetsRequest) Reset() {
	*x = ListStudioAgentPresetsRequest{}
	if protoimpl.UnsafeEnabled {
		mi := &file_buf_alpha_registry_v1alpha1_studio_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *ListStudioAgentPresetsRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ListStudioAgentPresetsRequest) ProtoMessage() {}

func (x *ListStudioAgentPresetsRequest) ProtoReflect() protoreflect.Message {
	mi := &file_buf_alpha_registry_v1alpha1_studio_proto_msgTypes[1]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ListStudioAgentPresetsRequest.ProtoReflect.Descriptor instead.
func (*ListStudioAgentPresetsRequest) Descriptor() ([]byte, []int) {
	return file_buf_alpha_registry_v1alpha1_studio_proto_rawDescGZIP(), []int{1}
}

type ListStudioAgentPresetsResponse struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Agents []*StudioAgentPreset `protobuf:"bytes,1,rep,name=agents,proto3" json:"agents,omitempty"`
}

func (x *ListStudioAgentPresetsResponse) Reset() {
	*x = ListStudioAgentPresetsResponse{}
	if protoimpl.UnsafeEnabled {
		mi := &file_buf_alpha_registry_v1alpha1_studio_proto_msgTypes[2]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *ListStudioAgentPresetsResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ListStudioAgentPresetsResponse) ProtoMessage() {}

func (x *ListStudioAgentPresetsResponse) ProtoReflect() protoreflect.Message {
	mi := &file_buf_alpha_registry_v1alpha1_studio_proto_msgTypes[2]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ListStudioAgentPresetsResponse.ProtoReflect.Descriptor instead.
func (*ListStudioAgentPresetsResponse) Descriptor() ([]byte, []int) {
	return file_buf_alpha_registry_v1alpha1_studio_proto_rawDescGZIP(), []int{2}
}

func (x *ListStudioAgentPresetsResponse) GetAgents() []*StudioAgentPreset {
	if x != nil {
		return x.Agents
	}
	return nil
}

type SetStudioAgentPresetsRequest struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Agents []*StudioAgentPreset `protobuf:"bytes,1,rep,name=agents,proto3" json:"agents,omitempty"`
}

func (x *SetStudioAgentPresetsRequest) Reset() {
	*x = SetStudioAgentPresetsRequest{}
	if protoimpl.UnsafeEnabled {
		mi := &file_buf_alpha_registry_v1alpha1_studio_proto_msgTypes[3]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *SetStudioAgentPresetsRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*SetStudioAgentPresetsRequest) ProtoMessage() {}

func (x *SetStudioAgentPresetsRequest) ProtoReflect() protoreflect.Message {
	mi := &file_buf_alpha_registry_v1alpha1_studio_proto_msgTypes[3]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use SetStudioAgentPresetsRequest.ProtoReflect.Descriptor instead.
func (*SetStudioAgentPresetsRequest) Descriptor() ([]byte, []int) {
	return file_buf_alpha_registry_v1alpha1_studio_proto_rawDescGZIP(), []int{3}
}

func (x *SetStudioAgentPresetsRequest) GetAgents() []*StudioAgentPreset {
	if x != nil {
		return x.Agents
	}
	return nil
}

type SetStudioAgentPresetsResponse struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields
}

func (x *SetStudioAgentPresetsResponse) Reset() {
	*x = SetStudioAgentPresetsResponse{}
	if protoimpl.UnsafeEnabled {
		mi := &file_buf_alpha_registry_v1alpha1_studio_proto_msgTypes[4]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *SetStudioAgentPresetsResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*SetStudioAgentPresetsResponse) ProtoMessage() {}

func (x *SetStudioAgentPresetsResponse) ProtoReflect() protoreflect.Message {
	mi := &file_buf_alpha_registry_v1alpha1_studio_proto_msgTypes[4]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use SetStudioAgentPresetsResponse.ProtoReflect.Descriptor instead.
func (*SetStudioAgentPresetsResponse) Descriptor() ([]byte, []int) {
	return file_buf_alpha_registry_v1alpha1_studio_proto_rawDescGZIP(), []int{4}
}

var File_buf_alpha_registry_v1alpha1_studio_proto protoreflect.FileDescriptor

var file_buf_alpha_registry_v1alpha1_studio_proto_rawDesc = []byte{
	0x0a, 0x28, 0x62, 0x75, 0x66, 0x2f, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x2f, 0x72, 0x65, 0x67, 0x69,
	0x73, 0x74, 0x72, 0x79, 0x2f, 0x76, 0x31, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x31, 0x2f, 0x73, 0x74,
	0x75, 0x64, 0x69, 0x6f, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x12, 0x1b, 0x62, 0x75, 0x66, 0x2e,
	0x61, 0x6c, 0x70, 0x68, 0x61, 0x2e, 0x72, 0x65, 0x67, 0x69, 0x73, 0x74, 0x72, 0x79, 0x2e, 0x76,
	0x31, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x31, 0x22, 0x89, 0x01, 0x0a, 0x11, 0x53, 0x74, 0x75, 0x64,
	0x69, 0x6f, 0x41, 0x67, 0x65, 0x6e, 0x74, 0x50, 0x72, 0x65, 0x73, 0x65, 0x74, 0x12, 0x10, 0x0a,
	0x03, 0x75, 0x72, 0x6c, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x03, 0x75, 0x72, 0x6c, 0x12,
	0x14, 0x0a, 0x05, 0x61, 0x6c, 0x69, 0x61, 0x73, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x05,
	0x61, 0x6c, 0x69, 0x61, 0x73, 0x12, 0x4c, 0x0a, 0x08, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x63, 0x6f,
	0x6c, 0x18, 0x03, 0x20, 0x01, 0x28, 0x0e, 0x32, 0x30, 0x2e, 0x62, 0x75, 0x66, 0x2e, 0x61, 0x6c,
	0x70, 0x68, 0x61, 0x2e, 0x72, 0x65, 0x67, 0x69, 0x73, 0x74, 0x72, 0x79, 0x2e, 0x76, 0x31, 0x61,
	0x6c, 0x70, 0x68, 0x61, 0x31, 0x2e, 0x53, 0x74, 0x75, 0x64, 0x69, 0x6f, 0x41, 0x67, 0x65, 0x6e,
	0x74, 0x50, 0x72, 0x6f, 0x74, 0x6f, 0x63, 0x6f, 0x6c, 0x52, 0x08, 0x70, 0x72, 0x6f, 0x74, 0x6f,
	0x63, 0x6f, 0x6c, 0x22, 0x1f, 0x0a, 0x1d, 0x4c, 0x69, 0x73, 0x74, 0x53, 0x74, 0x75, 0x64, 0x69,
	0x6f, 0x41, 0x67, 0x65, 0x6e, 0x74, 0x50, 0x72, 0x65, 0x73, 0x65, 0x74, 0x73, 0x52, 0x65, 0x71,
	0x75, 0x65, 0x73, 0x74, 0x22, 0x68, 0x0a, 0x1e, 0x4c, 0x69, 0x73, 0x74, 0x53, 0x74, 0x75, 0x64,
	0x69, 0x6f, 0x41, 0x67, 0x65, 0x6e, 0x74, 0x50, 0x72, 0x65, 0x73, 0x65, 0x74, 0x73, 0x52, 0x65,
	0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x12, 0x46, 0x0a, 0x06, 0x61, 0x67, 0x65, 0x6e, 0x74, 0x73,
	0x18, 0x01, 0x20, 0x03, 0x28, 0x0b, 0x32, 0x2e, 0x2e, 0x62, 0x75, 0x66, 0x2e, 0x61, 0x6c, 0x70,
	0x68, 0x61, 0x2e, 0x72, 0x65, 0x67, 0x69, 0x73, 0x74, 0x72, 0x79, 0x2e, 0x76, 0x31, 0x61, 0x6c,
	0x70, 0x68, 0x61, 0x31, 0x2e, 0x53, 0x74, 0x75, 0x64, 0x69, 0x6f, 0x41, 0x67, 0x65, 0x6e, 0x74,
	0x50, 0x72, 0x65, 0x73, 0x65, 0x74, 0x52, 0x06, 0x61, 0x67, 0x65, 0x6e, 0x74, 0x73, 0x22, 0x66,
	0x0a, 0x1c, 0x53, 0x65, 0x74, 0x53, 0x74, 0x75, 0x64, 0x69, 0x6f, 0x41, 0x67, 0x65, 0x6e, 0x74,
	0x50, 0x72, 0x65, 0x73, 0x65, 0x74, 0x73, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x12, 0x46,
	0x0a, 0x06, 0x61, 0x67, 0x65, 0x6e, 0x74, 0x73, 0x18, 0x01, 0x20, 0x03, 0x28, 0x0b, 0x32, 0x2e,
	0x2e, 0x62, 0x75, 0x66, 0x2e, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x2e, 0x72, 0x65, 0x67, 0x69, 0x73,
	0x74, 0x72, 0x79, 0x2e, 0x76, 0x31, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x31, 0x2e, 0x53, 0x74, 0x75,
	0x64, 0x69, 0x6f, 0x41, 0x67, 0x65, 0x6e, 0x74, 0x50, 0x72, 0x65, 0x73, 0x65, 0x74, 0x52, 0x06,
	0x61, 0x67, 0x65, 0x6e, 0x74, 0x73, 0x22, 0x1f, 0x0a, 0x1d, 0x53, 0x65, 0x74, 0x53, 0x74, 0x75,
	0x64, 0x69, 0x6f, 0x41, 0x67, 0x65, 0x6e, 0x74, 0x50, 0x72, 0x65, 0x73, 0x65, 0x74, 0x73, 0x52,
	0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x2a, 0x7f, 0x0a, 0x13, 0x53, 0x74, 0x75, 0x64, 0x69,
	0x6f, 0x41, 0x67, 0x65, 0x6e, 0x74, 0x50, 0x72, 0x6f, 0x74, 0x6f, 0x63, 0x6f, 0x6c, 0x12, 0x25,
	0x0a, 0x21, 0x53, 0x54, 0x55, 0x44, 0x49, 0x4f, 0x5f, 0x41, 0x47, 0x45, 0x4e, 0x54, 0x5f, 0x50,
	0x52, 0x4f, 0x54, 0x4f, 0x43, 0x4f, 0x4c, 0x5f, 0x55, 0x4e, 0x53, 0x50, 0x45, 0x43, 0x49, 0x46,
	0x49, 0x45, 0x44, 0x10, 0x00, 0x12, 0x1e, 0x0a, 0x1a, 0x53, 0x54, 0x55, 0x44, 0x49, 0x4f, 0x5f,
	0x41, 0x47, 0x45, 0x4e, 0x54, 0x5f, 0x50, 0x52, 0x4f, 0x54, 0x4f, 0x43, 0x4f, 0x4c, 0x5f, 0x47,
	0x52, 0x50, 0x43, 0x10, 0x01, 0x12, 0x21, 0x0a, 0x1d, 0x53, 0x54, 0x55, 0x44, 0x49, 0x4f, 0x5f,
	0x41, 0x47, 0x45, 0x4e, 0x54, 0x5f, 0x50, 0x52, 0x4f, 0x54, 0x4f, 0x43, 0x4f, 0x4c, 0x5f, 0x43,
	0x4f, 0x4e, 0x4e, 0x45, 0x43, 0x54, 0x10, 0x02, 0x32, 0xb9, 0x02, 0x0a, 0x0d, 0x53, 0x74, 0x75,
	0x64, 0x69, 0x6f, 0x53, 0x65, 0x72, 0x76, 0x69, 0x63, 0x65, 0x12, 0x96, 0x01, 0x0a, 0x16, 0x4c,
	0x69, 0x73, 0x74, 0x53, 0x74, 0x75, 0x64, 0x69, 0x6f, 0x41, 0x67, 0x65, 0x6e, 0x74, 0x50, 0x72,
	0x65, 0x73, 0x65, 0x74, 0x73, 0x12, 0x3a, 0x2e, 0x62, 0x75, 0x66, 0x2e, 0x61, 0x6c, 0x70, 0x68,
	0x61, 0x2e, 0x72, 0x65, 0x67, 0x69, 0x73, 0x74, 0x72, 0x79, 0x2e, 0x76, 0x31, 0x61, 0x6c, 0x70,
	0x68, 0x61, 0x31, 0x2e, 0x4c, 0x69, 0x73, 0x74, 0x53, 0x74, 0x75, 0x64, 0x69, 0x6f, 0x41, 0x67,
	0x65, 0x6e, 0x74, 0x50, 0x72, 0x65, 0x73, 0x65, 0x74, 0x73, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73,
	0x74, 0x1a, 0x3b, 0x2e, 0x62, 0x75, 0x66, 0x2e, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x2e, 0x72, 0x65,
	0x67, 0x69, 0x73, 0x74, 0x72, 0x79, 0x2e, 0x76, 0x31, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x31, 0x2e,
	0x4c, 0x69, 0x73, 0x74, 0x53, 0x74, 0x75, 0x64, 0x69, 0x6f, 0x41, 0x67, 0x65, 0x6e, 0x74, 0x50,
	0x72, 0x65, 0x73, 0x65, 0x74, 0x73, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x22, 0x03,
	0x90, 0x02, 0x01, 0x12, 0x8e, 0x01, 0x0a, 0x15, 0x53, 0x65, 0x74, 0x53, 0x74, 0x75, 0x64, 0x69,
	0x6f, 0x41, 0x67, 0x65, 0x6e, 0x74, 0x50, 0x72, 0x65, 0x73, 0x65, 0x74, 0x73, 0x12, 0x39, 0x2e,
	0x62, 0x75, 0x66, 0x2e, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x2e, 0x72, 0x65, 0x67, 0x69, 0x73, 0x74,
	0x72, 0x79, 0x2e, 0x76, 0x31, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x31, 0x2e, 0x53, 0x65, 0x74, 0x53,
	0x74, 0x75, 0x64, 0x69, 0x6f, 0x41, 0x67, 0x65, 0x6e, 0x74, 0x50, 0x72, 0x65, 0x73, 0x65, 0x74,
	0x73, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x1a, 0x3a, 0x2e, 0x62, 0x75, 0x66, 0x2e, 0x61,
	0x6c, 0x70, 0x68, 0x61, 0x2e, 0x72, 0x65, 0x67, 0x69, 0x73, 0x74, 0x72, 0x79, 0x2e, 0x76, 0x31,
	0x61, 0x6c, 0x70, 0x68, 0x61, 0x31, 0x2e, 0x53, 0x65, 0x74, 0x53, 0x74, 0x75, 0x64, 0x69, 0x6f,
	0x41, 0x67, 0x65, 0x6e, 0x74, 0x50, 0x72, 0x65, 0x73, 0x65, 0x74, 0x73, 0x52, 0x65, 0x73, 0x70,
	0x6f, 0x6e, 0x73, 0x65, 0x42, 0x98, 0x02, 0x0a, 0x1f, 0x63, 0x6f, 0x6d, 0x2e, 0x62, 0x75, 0x66,
	0x2e, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x2e, 0x72, 0x65, 0x67, 0x69, 0x73, 0x74, 0x72, 0x79, 0x2e,
	0x76, 0x31, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x31, 0x42, 0x0b, 0x53, 0x74, 0x75, 0x64, 0x69, 0x6f,
	0x50, 0x72, 0x6f, 0x74, 0x6f, 0x50, 0x01, 0x5a, 0x59, 0x67, 0x69, 0x74, 0x68, 0x75, 0x62, 0x2e,
	0x63, 0x6f, 0x6d, 0x2f, 0x62, 0x75, 0x66, 0x62, 0x75, 0x69, 0x6c, 0x64, 0x2f, 0x62, 0x75, 0x66,
	0x2f, 0x70, 0x72, 0x69, 0x76, 0x61, 0x74, 0x65, 0x2f, 0x67, 0x65, 0x6e, 0x2f, 0x70, 0x72, 0x6f,
	0x74, 0x6f, 0x2f, 0x67, 0x6f, 0x2f, 0x62, 0x75, 0x66, 0x2f, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x2f,
	0x72, 0x65, 0x67, 0x69, 0x73, 0x74, 0x72, 0x79, 0x2f, 0x76, 0x31, 0x61, 0x6c, 0x70, 0x68, 0x61,
	0x31, 0x3b, 0x72, 0x65, 0x67, 0x69, 0x73, 0x74, 0x72, 0x79, 0x76, 0x31, 0x61, 0x6c, 0x70, 0x68,
	0x61, 0x31, 0xa2, 0x02, 0x03, 0x42, 0x41, 0x52, 0xaa, 0x02, 0x1b, 0x42, 0x75, 0x66, 0x2e, 0x41,
	0x6c, 0x70, 0x68, 0x61, 0x2e, 0x52, 0x65, 0x67, 0x69, 0x73, 0x74, 0x72, 0x79, 0x2e, 0x56, 0x31,
	0x61, 0x6c, 0x70, 0x68, 0x61, 0x31, 0xca, 0x02, 0x1b, 0x42, 0x75, 0x66, 0x5c, 0x41, 0x6c, 0x70,
	0x68, 0x61, 0x5c, 0x52, 0x65, 0x67, 0x69, 0x73, 0x74, 0x72, 0x79, 0x5c, 0x56, 0x31, 0x61, 0x6c,
	0x70, 0x68, 0x61, 0x31, 0xe2, 0x02, 0x27, 0x42, 0x75, 0x66, 0x5c, 0x41, 0x6c, 0x70, 0x68, 0x61,
	0x5c, 0x52, 0x65, 0x67, 0x69, 0x73, 0x74, 0x72, 0x79, 0x5c, 0x56, 0x31, 0x61, 0x6c, 0x70, 0x68,
	0x61, 0x31, 0x5c, 0x47, 0x50, 0x42, 0x4d, 0x65, 0x74, 0x61, 0x64, 0x61, 0x74, 0x61, 0xea, 0x02,
	0x1e, 0x42, 0x75, 0x66, 0x3a, 0x3a, 0x41, 0x6c, 0x70, 0x68, 0x61, 0x3a, 0x3a, 0x52, 0x65, 0x67,
	0x69, 0x73, 0x74, 0x72, 0x79, 0x3a, 0x3a, 0x56, 0x31, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x31, 0x62,
	0x06, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_buf_alpha_registry_v1alpha1_studio_proto_rawDescOnce sync.Once
	file_buf_alpha_registry_v1alpha1_studio_proto_rawDescData = file_buf_alpha_registry_v1alpha1_studio_proto_rawDesc
)

func file_buf_alpha_registry_v1alpha1_studio_proto_rawDescGZIP() []byte {
	file_buf_alpha_registry_v1alpha1_studio_proto_rawDescOnce.Do(func() {
		file_buf_alpha_registry_v1alpha1_studio_proto_rawDescData = protoimpl.X.CompressGZIP(file_buf_alpha_registry_v1alpha1_studio_proto_rawDescData)
	})
	return file_buf_alpha_registry_v1alpha1_studio_proto_rawDescData
}

var file_buf_alpha_registry_v1alpha1_studio_proto_enumTypes = make([]protoimpl.EnumInfo, 1)
var file_buf_alpha_registry_v1alpha1_studio_proto_msgTypes = make([]protoimpl.MessageInfo, 5)
var file_buf_alpha_registry_v1alpha1_studio_proto_goTypes = []interface{}{
	(StudioAgentProtocol)(0),               // 0: buf.alpha.registry.v1alpha1.StudioAgentProtocol
	(*StudioAgentPreset)(nil),              // 1: buf.alpha.registry.v1alpha1.StudioAgentPreset
	(*ListStudioAgentPresetsRequest)(nil),  // 2: buf.alpha.registry.v1alpha1.ListStudioAgentPresetsRequest
	(*ListStudioAgentPresetsResponse)(nil), // 3: buf.alpha.registry.v1alpha1.ListStudioAgentPresetsResponse
	(*SetStudioAgentPresetsRequest)(nil),   // 4: buf.alpha.registry.v1alpha1.SetStudioAgentPresetsRequest
	(*SetStudioAgentPresetsResponse)(nil),  // 5: buf.alpha.registry.v1alpha1.SetStudioAgentPresetsResponse
}
var file_buf_alpha_registry_v1alpha1_studio_proto_depIdxs = []int32{
	0, // 0: buf.alpha.registry.v1alpha1.StudioAgentPreset.protocol:type_name -> buf.alpha.registry.v1alpha1.StudioAgentProtocol
	1, // 1: buf.alpha.registry.v1alpha1.ListStudioAgentPresetsResponse.agents:type_name -> buf.alpha.registry.v1alpha1.StudioAgentPreset
	1, // 2: buf.alpha.registry.v1alpha1.SetStudioAgentPresetsRequest.agents:type_name -> buf.alpha.registry.v1alpha1.StudioAgentPreset
	2, // 3: buf.alpha.registry.v1alpha1.StudioService.ListStudioAgentPresets:input_type -> buf.alpha.registry.v1alpha1.ListStudioAgentPresetsRequest
	4, // 4: buf.alpha.registry.v1alpha1.StudioService.SetStudioAgentPresets:input_type -> buf.alpha.registry.v1alpha1.SetStudioAgentPresetsRequest
	3, // 5: buf.alpha.registry.v1alpha1.StudioService.ListStudioAgentPresets:output_type -> buf.alpha.registry.v1alpha1.ListStudioAgentPresetsResponse
	5, // 6: buf.alpha.registry.v1alpha1.StudioService.SetStudioAgentPresets:output_type -> buf.alpha.registry.v1alpha1.SetStudioAgentPresetsResponse
	5, // [5:7] is the sub-list for method output_type
	3, // [3:5] is the sub-list for method input_type
	3, // [3:3] is the sub-list for extension type_name
	3, // [3:3] is the sub-list for extension extendee
	0, // [0:3] is the sub-list for field type_name
}

func init() { file_buf_alpha_registry_v1alpha1_studio_proto_init() }
func file_buf_alpha_registry_v1alpha1_studio_proto_init() {
	if File_buf_alpha_registry_v1alpha1_studio_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_buf_alpha_registry_v1alpha1_studio_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*StudioAgentPreset); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_buf_alpha_registry_v1alpha1_studio_proto_msgTypes[1].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*ListStudioAgentPresetsRequest); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_buf_alpha_registry_v1alpha1_studio_proto_msgTypes[2].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*ListStudioAgentPresetsResponse); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_buf_alpha_registry_v1alpha1_studio_proto_msgTypes[3].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*SetStudioAgentPresetsRequest); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_buf_alpha_registry_v1alpha1_studio_proto_msgTypes[4].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*SetStudioAgentPresetsResponse); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: file_buf_alpha_registry_v1alpha1_studio_proto_rawDesc,
			NumEnums:      1,
			NumMessages:   5,
			NumExtensions: 0,
			NumServices:   1,
		},
		GoTypes:           file_buf_alpha_registry_v1alpha1_studio_proto_goTypes,
		DependencyIndexes: file_buf_alpha_registry_v1alpha1_studio_proto_depIdxs,
		EnumInfos:         file_buf_alpha_registry_v1alpha1_studio_proto_enumTypes,
		MessageInfos:      file_buf_alpha_registry_v1alpha1_studio_proto_msgTypes,
	}.Build()
	File_buf_alpha_registry_v1alpha1_studio_proto = out.File
	file_buf_alpha_registry_v1alpha1_studio_proto_rawDesc = nil
	file_buf_alpha_registry_v1alpha1_studio_proto_goTypes = nil
	file_buf_alpha_registry_v1alpha1_studio_proto_depIdxs = nil
}
