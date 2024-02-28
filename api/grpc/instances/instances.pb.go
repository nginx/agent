// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.32.0
// 	protoc        v4.24.4
// source: api/grpc/instances/instances.proto

package instances

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

type Type int32

const (
	Type_UNKNOWN              Type = 0
	Type_NGINX                Type = 1
	Type_NGINX_PLUS           Type = 2
	Type_NGINX_GATEWAY_FABRIC Type = 3
)

// Enum value maps for Type.
var (
	Type_name = map[int32]string{
		0: "UNKNOWN",
		1: "NGINX",
		2: "NGINX_PLUS",
		3: "NGINX_GATEWAY_FABRIC",
	}
	Type_value = map[string]int32{
		"UNKNOWN":              0,
		"NGINX":                1,
		"NGINX_PLUS":           2,
		"NGINX_GATEWAY_FABRIC": 3,
	}
)

func (x Type) Enum() *Type {
	p := new(Type)
	*p = x
	return p
}

func (x Type) String() string {
	return protoimpl.X.EnumStringOf(x.Descriptor(), protoreflect.EnumNumber(x))
}

func (Type) Descriptor() protoreflect.EnumDescriptor {
	return file_api_grpc_instances_instances_proto_enumTypes[0].Descriptor()
}

func (Type) Type() protoreflect.EnumType {
	return &file_api_grpc_instances_instances_proto_enumTypes[0]
}

func (x Type) Number() protoreflect.EnumNumber {
	return protoreflect.EnumNumber(x)
}

// Deprecated: Use Type.Descriptor instead.
func (Type) EnumDescriptor() ([]byte, []int) {
	return file_api_grpc_instances_instances_proto_rawDescGZIP(), []int{0}
}

type Instance struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	InstanceId string `protobuf:"bytes,1,opt,name=instance_id,json=instanceId,proto3" json:"instance_id,omitempty"`
	Meta       *Meta  `protobuf:"bytes,2,opt,name=meta,proto3" json:"meta,omitempty"`
	Type       Type   `protobuf:"varint,3,opt,name=type,proto3,enum=f5.nginx.agent.api.grpc.instances.Type" json:"type,omitempty"`
	Version    string `protobuf:"bytes,4,opt,name=version,proto3" json:"version,omitempty"`
}

func (x *Instance) Reset() {
	*x = Instance{}
	if protoimpl.UnsafeEnabled {
		mi := &file_api_grpc_instances_instances_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *Instance) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Instance) ProtoMessage() {}

func (x *Instance) ProtoReflect() protoreflect.Message {
	mi := &file_api_grpc_instances_instances_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Instance.ProtoReflect.Descriptor instead.
func (*Instance) Descriptor() ([]byte, []int) {
	return file_api_grpc_instances_instances_proto_rawDescGZIP(), []int{0}
}

func (x *Instance) GetInstanceId() string {
	if x != nil {
		return x.InstanceId
	}
	return ""
}

func (x *Instance) GetMeta() *Meta {
	if x != nil {
		return x.Meta
	}
	return nil
}

func (x *Instance) GetType() Type {
	if x != nil {
		return x.Type
	}
	return Type_UNKNOWN
}

func (x *Instance) GetVersion() string {
	if x != nil {
		return x.Version
	}
	return ""
}

type Meta struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// Types that are assignable to Meta:
	//
	//	*Meta_NginxMeta
	Meta isMeta_Meta `protobuf_oneof:"meta"`
}

func (x *Meta) Reset() {
	*x = Meta{}
	if protoimpl.UnsafeEnabled {
		mi := &file_api_grpc_instances_instances_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *Meta) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Meta) ProtoMessage() {}

func (x *Meta) ProtoReflect() protoreflect.Message {
	mi := &file_api_grpc_instances_instances_proto_msgTypes[1]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Meta.ProtoReflect.Descriptor instead.
func (*Meta) Descriptor() ([]byte, []int) {
	return file_api_grpc_instances_instances_proto_rawDescGZIP(), []int{1}
}

func (m *Meta) GetMeta() isMeta_Meta {
	if m != nil {
		return m.Meta
	}
	return nil
}

func (x *Meta) GetNginxMeta() *NginxMeta {
	if x, ok := x.GetMeta().(*Meta_NginxMeta); ok {
		return x.NginxMeta
	}
	return nil
}

type isMeta_Meta interface {
	isMeta_Meta()
}

type Meta_NginxMeta struct {
	NginxMeta *NginxMeta `protobuf:"bytes,1,opt,name=nginx_meta,json=nginxMeta,proto3,oneof"`
}

func (*Meta_NginxMeta) isMeta_Meta() {}

type NginxMeta struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	ConfigPath string `protobuf:"bytes,1,opt,name=config_path,json=configPath,proto3" json:"config_path,omitempty"`
	ExePath    string `protobuf:"bytes,2,opt,name=exe_path,json=exePath,proto3" json:"exe_path,omitempty"`
	ProcessId  string `protobuf:"bytes,3,opt,name=process_id,json=processId,proto3" json:"process_id,omitempty"`
}

func (x *NginxMeta) Reset() {
	*x = NginxMeta{}
	if protoimpl.UnsafeEnabled {
		mi := &file_api_grpc_instances_instances_proto_msgTypes[2]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *NginxMeta) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*NginxMeta) ProtoMessage() {}

func (x *NginxMeta) ProtoReflect() protoreflect.Message {
	mi := &file_api_grpc_instances_instances_proto_msgTypes[2]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use NginxMeta.ProtoReflect.Descriptor instead.
func (*NginxMeta) Descriptor() ([]byte, []int) {
	return file_api_grpc_instances_instances_proto_rawDescGZIP(), []int{2}
}

func (x *NginxMeta) GetConfigPath() string {
	if x != nil {
		return x.ConfigPath
	}
	return ""
}

func (x *NginxMeta) GetExePath() string {
	if x != nil {
		return x.ExePath
	}
	return ""
}

func (x *NginxMeta) GetProcessId() string {
	if x != nil {
		return x.ProcessId
	}
	return ""
}

var File_api_grpc_instances_instances_proto protoreflect.FileDescriptor

var file_api_grpc_instances_instances_proto_rawDesc = []byte{
	0x0a, 0x22, 0x61, 0x70, 0x69, 0x2f, 0x67, 0x72, 0x70, 0x63, 0x2f, 0x69, 0x6e, 0x73, 0x74, 0x61,
	0x6e, 0x63, 0x65, 0x73, 0x2f, 0x69, 0x6e, 0x73, 0x74, 0x61, 0x6e, 0x63, 0x65, 0x73, 0x2e, 0x70,
	0x72, 0x6f, 0x74, 0x6f, 0x12, 0x21, 0x66, 0x35, 0x2e, 0x6e, 0x67, 0x69, 0x6e, 0x78, 0x2e, 0x61,
	0x67, 0x65, 0x6e, 0x74, 0x2e, 0x61, 0x70, 0x69, 0x2e, 0x67, 0x72, 0x70, 0x63, 0x2e, 0x69, 0x6e,
	0x73, 0x74, 0x61, 0x6e, 0x63, 0x65, 0x73, 0x22, 0xbf, 0x01, 0x0a, 0x08, 0x49, 0x6e, 0x73, 0x74,
	0x61, 0x6e, 0x63, 0x65, 0x12, 0x1f, 0x0a, 0x0b, 0x69, 0x6e, 0x73, 0x74, 0x61, 0x6e, 0x63, 0x65,
	0x5f, 0x69, 0x64, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x0a, 0x69, 0x6e, 0x73, 0x74, 0x61,
	0x6e, 0x63, 0x65, 0x49, 0x64, 0x12, 0x3b, 0x0a, 0x04, 0x6d, 0x65, 0x74, 0x61, 0x18, 0x02, 0x20,
	0x01, 0x28, 0x0b, 0x32, 0x27, 0x2e, 0x66, 0x35, 0x2e, 0x6e, 0x67, 0x69, 0x6e, 0x78, 0x2e, 0x61,
	0x67, 0x65, 0x6e, 0x74, 0x2e, 0x61, 0x70, 0x69, 0x2e, 0x67, 0x72, 0x70, 0x63, 0x2e, 0x69, 0x6e,
	0x73, 0x74, 0x61, 0x6e, 0x63, 0x65, 0x73, 0x2e, 0x4d, 0x65, 0x74, 0x61, 0x52, 0x04, 0x6d, 0x65,
	0x74, 0x61, 0x12, 0x3b, 0x0a, 0x04, 0x74, 0x79, 0x70, 0x65, 0x18, 0x03, 0x20, 0x01, 0x28, 0x0e,
	0x32, 0x27, 0x2e, 0x66, 0x35, 0x2e, 0x6e, 0x67, 0x69, 0x6e, 0x78, 0x2e, 0x61, 0x67, 0x65, 0x6e,
	0x74, 0x2e, 0x61, 0x70, 0x69, 0x2e, 0x67, 0x72, 0x70, 0x63, 0x2e, 0x69, 0x6e, 0x73, 0x74, 0x61,
	0x6e, 0x63, 0x65, 0x73, 0x2e, 0x54, 0x79, 0x70, 0x65, 0x52, 0x04, 0x74, 0x79, 0x70, 0x65, 0x12,
	0x18, 0x0a, 0x07, 0x76, 0x65, 0x72, 0x73, 0x69, 0x6f, 0x6e, 0x18, 0x04, 0x20, 0x01, 0x28, 0x09,
	0x52, 0x07, 0x76, 0x65, 0x72, 0x73, 0x69, 0x6f, 0x6e, 0x22, 0x5d, 0x0a, 0x04, 0x4d, 0x65, 0x74,
	0x61, 0x12, 0x4d, 0x0a, 0x0a, 0x6e, 0x67, 0x69, 0x6e, 0x78, 0x5f, 0x6d, 0x65, 0x74, 0x61, 0x18,
	0x01, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x2c, 0x2e, 0x66, 0x35, 0x2e, 0x6e, 0x67, 0x69, 0x6e, 0x78,
	0x2e, 0x61, 0x67, 0x65, 0x6e, 0x74, 0x2e, 0x61, 0x70, 0x69, 0x2e, 0x67, 0x72, 0x70, 0x63, 0x2e,
	0x69, 0x6e, 0x73, 0x74, 0x61, 0x6e, 0x63, 0x65, 0x73, 0x2e, 0x4e, 0x67, 0x69, 0x6e, 0x78, 0x4d,
	0x65, 0x74, 0x61, 0x48, 0x00, 0x52, 0x09, 0x6e, 0x67, 0x69, 0x6e, 0x78, 0x4d, 0x65, 0x74, 0x61,
	0x42, 0x06, 0x0a, 0x04, 0x6d, 0x65, 0x74, 0x61, 0x22, 0x66, 0x0a, 0x09, 0x4e, 0x67, 0x69, 0x6e,
	0x78, 0x4d, 0x65, 0x74, 0x61, 0x12, 0x1f, 0x0a, 0x0b, 0x63, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x5f,
	0x70, 0x61, 0x74, 0x68, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x0a, 0x63, 0x6f, 0x6e, 0x66,
	0x69, 0x67, 0x50, 0x61, 0x74, 0x68, 0x12, 0x19, 0x0a, 0x08, 0x65, 0x78, 0x65, 0x5f, 0x70, 0x61,
	0x74, 0x68, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x07, 0x65, 0x78, 0x65, 0x50, 0x61, 0x74,
	0x68, 0x12, 0x1d, 0x0a, 0x0a, 0x70, 0x72, 0x6f, 0x63, 0x65, 0x73, 0x73, 0x5f, 0x69, 0x64, 0x18,
	0x03, 0x20, 0x01, 0x28, 0x09, 0x52, 0x09, 0x70, 0x72, 0x6f, 0x63, 0x65, 0x73, 0x73, 0x49, 0x64,
	0x2a, 0x48, 0x0a, 0x04, 0x54, 0x79, 0x70, 0x65, 0x12, 0x0b, 0x0a, 0x07, 0x55, 0x4e, 0x4b, 0x4e,
	0x4f, 0x57, 0x4e, 0x10, 0x00, 0x12, 0x09, 0x0a, 0x05, 0x4e, 0x47, 0x49, 0x4e, 0x58, 0x10, 0x01,
	0x12, 0x0e, 0x0a, 0x0a, 0x4e, 0x47, 0x49, 0x4e, 0x58, 0x5f, 0x50, 0x4c, 0x55, 0x53, 0x10, 0x02,
	0x12, 0x18, 0x0a, 0x14, 0x4e, 0x47, 0x49, 0x4e, 0x58, 0x5f, 0x47, 0x41, 0x54, 0x45, 0x57, 0x41,
	0x59, 0x5f, 0x46, 0x41, 0x42, 0x52, 0x49, 0x43, 0x10, 0x03, 0x42, 0x2e, 0x5a, 0x2c, 0x67, 0x69,
	0x74, 0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f, 0x6d, 0x2f, 0x6e, 0x67, 0x69, 0x6e, 0x78, 0x2f, 0x61,
	0x67, 0x65, 0x6e, 0x74, 0x2f, 0x76, 0x33, 0x2f, 0x61, 0x70, 0x69, 0x2f, 0x67, 0x72, 0x70, 0x63,
	0x2f, 0x69, 0x6e, 0x73, 0x74, 0x61, 0x6e, 0x63, 0x65, 0x73, 0x62, 0x06, 0x70, 0x72, 0x6f, 0x74,
	0x6f, 0x33,
}

var (
	file_api_grpc_instances_instances_proto_rawDescOnce sync.Once
	file_api_grpc_instances_instances_proto_rawDescData = file_api_grpc_instances_instances_proto_rawDesc
)

func file_api_grpc_instances_instances_proto_rawDescGZIP() []byte {
	file_api_grpc_instances_instances_proto_rawDescOnce.Do(func() {
		file_api_grpc_instances_instances_proto_rawDescData = protoimpl.X.CompressGZIP(file_api_grpc_instances_instances_proto_rawDescData)
	})
	return file_api_grpc_instances_instances_proto_rawDescData
}

var file_api_grpc_instances_instances_proto_enumTypes = make([]protoimpl.EnumInfo, 1)
var file_api_grpc_instances_instances_proto_msgTypes = make([]protoimpl.MessageInfo, 3)
var file_api_grpc_instances_instances_proto_goTypes = []interface{}{
	(Type)(0),         // 0: f5.nginx.agent.api.grpc.instances.Type
	(*Instance)(nil),  // 1: f5.nginx.agent.api.grpc.instances.Instance
	(*Meta)(nil),      // 2: f5.nginx.agent.api.grpc.instances.Meta
	(*NginxMeta)(nil), // 3: f5.nginx.agent.api.grpc.instances.NginxMeta
}
var file_api_grpc_instances_instances_proto_depIdxs = []int32{
	2, // 0: f5.nginx.agent.api.grpc.instances.Instance.meta:type_name -> f5.nginx.agent.api.grpc.instances.Meta
	0, // 1: f5.nginx.agent.api.grpc.instances.Instance.type:type_name -> f5.nginx.agent.api.grpc.instances.Type
	3, // 2: f5.nginx.agent.api.grpc.instances.Meta.nginx_meta:type_name -> f5.nginx.agent.api.grpc.instances.NginxMeta
	3, // [3:3] is the sub-list for method output_type
	3, // [3:3] is the sub-list for method input_type
	3, // [3:3] is the sub-list for extension type_name
	3, // [3:3] is the sub-list for extension extendee
	0, // [0:3] is the sub-list for field type_name
}

func init() { file_api_grpc_instances_instances_proto_init() }
func file_api_grpc_instances_instances_proto_init() {
	if File_api_grpc_instances_instances_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_api_grpc_instances_instances_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*Instance); i {
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
		file_api_grpc_instances_instances_proto_msgTypes[1].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*Meta); i {
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
		file_api_grpc_instances_instances_proto_msgTypes[2].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*NginxMeta); i {
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
	file_api_grpc_instances_instances_proto_msgTypes[1].OneofWrappers = []interface{}{
		(*Meta_NginxMeta)(nil),
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: file_api_grpc_instances_instances_proto_rawDesc,
			NumEnums:      1,
			NumMessages:   3,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_api_grpc_instances_instances_proto_goTypes,
		DependencyIndexes: file_api_grpc_instances_instances_proto_depIdxs,
		EnumInfos:         file_api_grpc_instances_instances_proto_enumTypes,
		MessageInfos:      file_api_grpc_instances_instances_proto_msgTypes,
	}.Build()
	File_api_grpc_instances_instances_proto = out.File
	file_api_grpc_instances_instances_proto_rawDesc = nil
	file_api_grpc_instances_instances_proto_goTypes = nil
	file_api_grpc_instances_instances_proto_depIdxs = nil
}
