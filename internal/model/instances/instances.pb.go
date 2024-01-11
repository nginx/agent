//*
// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.31.0
// 	protoc        v4.24.4
// source: internal/model/instances/instances.proto

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
	Type_UNKNOWN   Type = 0
	Type_NGINX     Type = 1
	Type_NGINXPLUS Type = 2
)

// Enum value maps for Type.
var (
	Type_name = map[int32]string{
		0: "UNKNOWN",
		1: "NGINX",
		2: "NGINXPLUS",
	}
	Type_value = map[string]int32{
		"UNKNOWN":   0,
		"NGINX":     1,
		"NGINXPLUS": 2,
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
	return file_internal_model_instances_instances_proto_enumTypes[0].Descriptor()
}

func (Type) Type() protoreflect.EnumType {
	return &file_internal_model_instances_instances_proto_enumTypes[0]
}

func (x Type) Number() protoreflect.EnumNumber {
	return protoreflect.EnumNumber(x)
}

// Deprecated: Use Type.Descriptor instead.
func (Type) EnumDescriptor() ([]byte, []int) {
	return file_internal_model_instances_instances_proto_rawDescGZIP(), []int{0}
}

type Instance struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	InstanceId string `protobuf:"bytes,1,opt,name=instance_id,json=instanceId,proto3" json:"instance_id,omitempty"`
	Meta       *Meta  `protobuf:"bytes,2,opt,name=meta,proto3" json:"meta,omitempty"`
	Type       Type   `protobuf:"varint,3,opt,name=type,proto3,enum=f5.nginx.agent.internal.model.instances.Type" json:"type,omitempty"`
	Version    string `protobuf:"bytes,4,opt,name=version,proto3" json:"version,omitempty"`
}

func (x *Instance) Reset() {
	*x = Instance{}
	if protoimpl.UnsafeEnabled {
		mi := &file_internal_model_instances_instances_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *Instance) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Instance) ProtoMessage() {}

func (x *Instance) ProtoReflect() protoreflect.Message {
	mi := &file_internal_model_instances_instances_proto_msgTypes[0]
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
	return file_internal_model_instances_instances_proto_rawDescGZIP(), []int{0}
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
		mi := &file_internal_model_instances_instances_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *Meta) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Meta) ProtoMessage() {}

func (x *Meta) ProtoReflect() protoreflect.Message {
	mi := &file_internal_model_instances_instances_proto_msgTypes[1]
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
	return file_internal_model_instances_instances_proto_rawDescGZIP(), []int{1}
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

	LoadableModules string `protobuf:"bytes,1,opt,name=loadable_modules,json=loadableModules,proto3" json:"loadable_modules,omitempty"`
	RunnableModules string `protobuf:"bytes,2,opt,name=runnable_modules,json=runnableModules,proto3" json:"runnable_modules,omitempty"`
}

func (x *NginxMeta) Reset() {
	*x = NginxMeta{}
	if protoimpl.UnsafeEnabled {
		mi := &file_internal_model_instances_instances_proto_msgTypes[2]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *NginxMeta) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*NginxMeta) ProtoMessage() {}

func (x *NginxMeta) ProtoReflect() protoreflect.Message {
	mi := &file_internal_model_instances_instances_proto_msgTypes[2]
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
	return file_internal_model_instances_instances_proto_rawDescGZIP(), []int{2}
}

func (x *NginxMeta) GetLoadableModules() string {
	if x != nil {
		return x.LoadableModules
	}
	return ""
}

func (x *NginxMeta) GetRunnableModules() string {
	if x != nil {
		return x.RunnableModules
	}
	return ""
}

var File_internal_model_instances_instances_proto protoreflect.FileDescriptor

var file_internal_model_instances_instances_proto_rawDesc = []byte{
	0x0a, 0x28, 0x69, 0x6e, 0x74, 0x65, 0x72, 0x6e, 0x61, 0x6c, 0x2f, 0x6d, 0x6f, 0x64, 0x65, 0x6c,
	0x2f, 0x69, 0x6e, 0x73, 0x74, 0x61, 0x6e, 0x63, 0x65, 0x73, 0x2f, 0x69, 0x6e, 0x73, 0x74, 0x61,
	0x6e, 0x63, 0x65, 0x73, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x12, 0x27, 0x66, 0x35, 0x2e, 0x6e,
	0x67, 0x69, 0x6e, 0x78, 0x2e, 0x61, 0x67, 0x65, 0x6e, 0x74, 0x2e, 0x69, 0x6e, 0x74, 0x65, 0x72,
	0x6e, 0x61, 0x6c, 0x2e, 0x6d, 0x6f, 0x64, 0x65, 0x6c, 0x2e, 0x69, 0x6e, 0x73, 0x74, 0x61, 0x6e,
	0x63, 0x65, 0x73, 0x22, 0xcb, 0x01, 0x0a, 0x08, 0x49, 0x6e, 0x73, 0x74, 0x61, 0x6e, 0x63, 0x65,
	0x12, 0x1f, 0x0a, 0x0b, 0x69, 0x6e, 0x73, 0x74, 0x61, 0x6e, 0x63, 0x65, 0x5f, 0x69, 0x64, 0x18,
	0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x0a, 0x69, 0x6e, 0x73, 0x74, 0x61, 0x6e, 0x63, 0x65, 0x49,
	0x64, 0x12, 0x41, 0x0a, 0x04, 0x6d, 0x65, 0x74, 0x61, 0x18, 0x02, 0x20, 0x01, 0x28, 0x0b, 0x32,
	0x2d, 0x2e, 0x66, 0x35, 0x2e, 0x6e, 0x67, 0x69, 0x6e, 0x78, 0x2e, 0x61, 0x67, 0x65, 0x6e, 0x74,
	0x2e, 0x69, 0x6e, 0x74, 0x65, 0x72, 0x6e, 0x61, 0x6c, 0x2e, 0x6d, 0x6f, 0x64, 0x65, 0x6c, 0x2e,
	0x69, 0x6e, 0x73, 0x74, 0x61, 0x6e, 0x63, 0x65, 0x73, 0x2e, 0x4d, 0x65, 0x74, 0x61, 0x52, 0x04,
	0x6d, 0x65, 0x74, 0x61, 0x12, 0x41, 0x0a, 0x04, 0x74, 0x79, 0x70, 0x65, 0x18, 0x03, 0x20, 0x01,
	0x28, 0x0e, 0x32, 0x2d, 0x2e, 0x66, 0x35, 0x2e, 0x6e, 0x67, 0x69, 0x6e, 0x78, 0x2e, 0x61, 0x67,
	0x65, 0x6e, 0x74, 0x2e, 0x69, 0x6e, 0x74, 0x65, 0x72, 0x6e, 0x61, 0x6c, 0x2e, 0x6d, 0x6f, 0x64,
	0x65, 0x6c, 0x2e, 0x69, 0x6e, 0x73, 0x74, 0x61, 0x6e, 0x63, 0x65, 0x73, 0x2e, 0x54, 0x79, 0x70,
	0x65, 0x52, 0x04, 0x74, 0x79, 0x70, 0x65, 0x12, 0x18, 0x0a, 0x07, 0x76, 0x65, 0x72, 0x73, 0x69,
	0x6f, 0x6e, 0x18, 0x04, 0x20, 0x01, 0x28, 0x09, 0x52, 0x07, 0x76, 0x65, 0x72, 0x73, 0x69, 0x6f,
	0x6e, 0x22, 0x63, 0x0a, 0x04, 0x4d, 0x65, 0x74, 0x61, 0x12, 0x53, 0x0a, 0x0a, 0x6e, 0x67, 0x69,
	0x6e, 0x78, 0x5f, 0x6d, 0x65, 0x74, 0x61, 0x18, 0x01, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x32, 0x2e,
	0x66, 0x35, 0x2e, 0x6e, 0x67, 0x69, 0x6e, 0x78, 0x2e, 0x61, 0x67, 0x65, 0x6e, 0x74, 0x2e, 0x69,
	0x6e, 0x74, 0x65, 0x72, 0x6e, 0x61, 0x6c, 0x2e, 0x6d, 0x6f, 0x64, 0x65, 0x6c, 0x2e, 0x69, 0x6e,
	0x73, 0x74, 0x61, 0x6e, 0x63, 0x65, 0x73, 0x2e, 0x4e, 0x67, 0x69, 0x6e, 0x78, 0x4d, 0x65, 0x74,
	0x61, 0x48, 0x00, 0x52, 0x09, 0x6e, 0x67, 0x69, 0x6e, 0x78, 0x4d, 0x65, 0x74, 0x61, 0x42, 0x06,
	0x0a, 0x04, 0x6d, 0x65, 0x74, 0x61, 0x22, 0x61, 0x0a, 0x09, 0x4e, 0x67, 0x69, 0x6e, 0x78, 0x4d,
	0x65, 0x74, 0x61, 0x12, 0x29, 0x0a, 0x10, 0x6c, 0x6f, 0x61, 0x64, 0x61, 0x62, 0x6c, 0x65, 0x5f,
	0x6d, 0x6f, 0x64, 0x75, 0x6c, 0x65, 0x73, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x0f, 0x6c,
	0x6f, 0x61, 0x64, 0x61, 0x62, 0x6c, 0x65, 0x4d, 0x6f, 0x64, 0x75, 0x6c, 0x65, 0x73, 0x12, 0x29,
	0x0a, 0x10, 0x72, 0x75, 0x6e, 0x6e, 0x61, 0x62, 0x6c, 0x65, 0x5f, 0x6d, 0x6f, 0x64, 0x75, 0x6c,
	0x65, 0x73, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x0f, 0x72, 0x75, 0x6e, 0x6e, 0x61, 0x62,
	0x6c, 0x65, 0x4d, 0x6f, 0x64, 0x75, 0x6c, 0x65, 0x73, 0x2a, 0x2d, 0x0a, 0x04, 0x54, 0x79, 0x70,
	0x65, 0x12, 0x0b, 0x0a, 0x07, 0x55, 0x4e, 0x4b, 0x4e, 0x4f, 0x57, 0x4e, 0x10, 0x00, 0x12, 0x09,
	0x0a, 0x05, 0x4e, 0x47, 0x49, 0x4e, 0x58, 0x10, 0x01, 0x12, 0x0d, 0x0a, 0x09, 0x4e, 0x47, 0x49,
	0x4e, 0x58, 0x50, 0x4c, 0x55, 0x53, 0x10, 0x02, 0x42, 0x34, 0x5a, 0x32, 0x67, 0x69, 0x74, 0x68,
	0x75, 0x62, 0x2e, 0x63, 0x6f, 0x6d, 0x2f, 0x6e, 0x67, 0x69, 0x6e, 0x78, 0x2f, 0x61, 0x67, 0x65,
	0x6e, 0x74, 0x2f, 0x76, 0x33, 0x2f, 0x69, 0x6e, 0x74, 0x65, 0x72, 0x6e, 0x61, 0x6c, 0x2f, 0x6d,
	0x6f, 0x64, 0x65, 0x6c, 0x2f, 0x69, 0x6e, 0x73, 0x74, 0x61, 0x6e, 0x63, 0x65, 0x73, 0x62, 0x06,
	0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_internal_model_instances_instances_proto_rawDescOnce sync.Once
	file_internal_model_instances_instances_proto_rawDescData = file_internal_model_instances_instances_proto_rawDesc
)

func file_internal_model_instances_instances_proto_rawDescGZIP() []byte {
	file_internal_model_instances_instances_proto_rawDescOnce.Do(func() {
		file_internal_model_instances_instances_proto_rawDescData = protoimpl.X.CompressGZIP(file_internal_model_instances_instances_proto_rawDescData)
	})
	return file_internal_model_instances_instances_proto_rawDescData
}

var file_internal_model_instances_instances_proto_enumTypes = make([]protoimpl.EnumInfo, 1)
var file_internal_model_instances_instances_proto_msgTypes = make([]protoimpl.MessageInfo, 3)
var file_internal_model_instances_instances_proto_goTypes = []interface{}{
	(Type)(0),         // 0: f5.nginx.agent.internal.model.instances.Type
	(*Instance)(nil),  // 1: f5.nginx.agent.internal.model.instances.Instance
	(*Meta)(nil),      // 2: f5.nginx.agent.internal.model.instances.Meta
	(*NginxMeta)(nil), // 3: f5.nginx.agent.internal.model.instances.NginxMeta
}
var file_internal_model_instances_instances_proto_depIdxs = []int32{
	2, // 0: f5.nginx.agent.internal.model.instances.Instance.meta:type_name -> f5.nginx.agent.internal.model.instances.Meta
	0, // 1: f5.nginx.agent.internal.model.instances.Instance.type:type_name -> f5.nginx.agent.internal.model.instances.Type
	3, // 2: f5.nginx.agent.internal.model.instances.Meta.nginx_meta:type_name -> f5.nginx.agent.internal.model.instances.NginxMeta
	3, // [3:3] is the sub-list for method output_type
	3, // [3:3] is the sub-list for method input_type
	3, // [3:3] is the sub-list for extension type_name
	3, // [3:3] is the sub-list for extension extendee
	0, // [0:3] is the sub-list for field type_name
}

func init() { file_internal_model_instances_instances_proto_init() }
func file_internal_model_instances_instances_proto_init() {
	if File_internal_model_instances_instances_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_internal_model_instances_instances_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
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
		file_internal_model_instances_instances_proto_msgTypes[1].Exporter = func(v interface{}, i int) interface{} {
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
		file_internal_model_instances_instances_proto_msgTypes[2].Exporter = func(v interface{}, i int) interface{} {
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
	file_internal_model_instances_instances_proto_msgTypes[1].OneofWrappers = []interface{}{
		(*Meta_NginxMeta)(nil),
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: file_internal_model_instances_instances_proto_rawDesc,
			NumEnums:      1,
			NumMessages:   3,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_internal_model_instances_instances_proto_goTypes,
		DependencyIndexes: file_internal_model_instances_instances_proto_depIdxs,
		EnumInfos:         file_internal_model_instances_instances_proto_enumTypes,
		MessageInfos:      file_internal_model_instances_instances_proto_msgTypes,
	}.Build()
	File_internal_model_instances_instances_proto = out.File
	file_internal_model_instances_instances_proto_rawDesc = nil
	file_internal_model_instances_instances_proto_goTypes = nil
	file_internal_model_instances_instances_proto_depIdxs = nil
}
