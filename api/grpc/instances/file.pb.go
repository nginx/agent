// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.32.0
// 	protoc        v4.25.3
// source: api/grpc/instances/file.proto

package instances

import (
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

type Files struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Files []*File `protobuf:"bytes,1,rep,name=files,proto3" json:"files,omitempty"`
}

func (x *Files) Reset() {
	*x = Files{}
	if protoimpl.UnsafeEnabled {
		mi := &file_api_grpc_instances_file_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *Files) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Files) ProtoMessage() {}

func (x *Files) ProtoReflect() protoreflect.Message {
	mi := &file_api_grpc_instances_file_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Files.ProtoReflect.Descriptor instead.
func (*Files) Descriptor() ([]byte, []int) {
	return file_api_grpc_instances_file_proto_rawDescGZIP(), []int{0}
}

func (x *Files) GetFiles() []*File {
	if x != nil {
		return x.Files
	}
	return nil
}

type File struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	LastModified *timestamppb.Timestamp `protobuf:"bytes,1,opt,name=last_modified,json=lastModified,proto3" json:"last_modified,omitempty"`
	Path         string                 `protobuf:"bytes,2,opt,name=path,proto3" json:"path,omitempty"`
	Version      string                 `protobuf:"bytes,3,opt,name=version,proto3" json:"version,omitempty"`
}

func (x *File) Reset() {
	*x = File{}
	if protoimpl.UnsafeEnabled {
		mi := &file_api_grpc_instances_file_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *File) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*File) ProtoMessage() {}

func (x *File) ProtoReflect() protoreflect.Message {
	mi := &file_api_grpc_instances_file_proto_msgTypes[1]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use File.ProtoReflect.Descriptor instead.
func (*File) Descriptor() ([]byte, []int) {
	return file_api_grpc_instances_file_proto_rawDescGZIP(), []int{1}
}

func (x *File) GetLastModified() *timestamppb.Timestamp {
	if x != nil {
		return x.LastModified
	}
	return nil
}

func (x *File) GetPath() string {
	if x != nil {
		return x.Path
	}
	return ""
}

func (x *File) GetVersion() string {
	if x != nil {
		return x.Version
	}
	return ""
}

type FileDownloadResponse struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Encoded     bool   `protobuf:"varint,1,opt,name=encoded,proto3" json:"encoded,omitempty"`
	FileContent []byte `protobuf:"bytes,2,opt,name=file_content,json=fileContent,proto3" json:"file_content,omitempty"`
	FilePath    string `protobuf:"bytes,3,opt,name=file_path,json=filePath,proto3" json:"file_path,omitempty"`
	InstanceId  string `protobuf:"bytes,4,opt,name=instance_id,json=instanceID,proto3" json:"instance_id,omitempty"`
}

func (x *FileDownloadResponse) Reset() {
	*x = FileDownloadResponse{}
	if protoimpl.UnsafeEnabled {
		mi := &file_api_grpc_instances_file_proto_msgTypes[2]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *FileDownloadResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*FileDownloadResponse) ProtoMessage() {}

func (x *FileDownloadResponse) ProtoReflect() protoreflect.Message {
	mi := &file_api_grpc_instances_file_proto_msgTypes[2]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use FileDownloadResponse.ProtoReflect.Descriptor instead.
func (*FileDownloadResponse) Descriptor() ([]byte, []int) {
	return file_api_grpc_instances_file_proto_rawDescGZIP(), []int{2}
}

func (x *FileDownloadResponse) GetEncoded() bool {
	if x != nil {
		return x.Encoded
	}
	return false
}

func (x *FileDownloadResponse) GetFileContent() []byte {
	if x != nil {
		return x.FileContent
	}
	return nil
}

func (x *FileDownloadResponse) GetFilePath() string {
	if x != nil {
		return x.FilePath
	}
	return ""
}

func (x *FileDownloadResponse) GetInstanceId() string {
	if x != nil {
		return x.InstanceId
	}
	return ""
}

var File_api_grpc_instances_file_proto protoreflect.FileDescriptor

var file_api_grpc_instances_file_proto_rawDesc = []byte{
	0x0a, 0x1d, 0x61, 0x70, 0x69, 0x2f, 0x67, 0x72, 0x70, 0x63, 0x2f, 0x69, 0x6e, 0x73, 0x74, 0x61,
	0x6e, 0x63, 0x65, 0x73, 0x2f, 0x66, 0x69, 0x6c, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x12,
	0x21, 0x66, 0x35, 0x2e, 0x6e, 0x67, 0x69, 0x6e, 0x78, 0x2e, 0x61, 0x67, 0x65, 0x6e, 0x74, 0x2e,
	0x61, 0x70, 0x69, 0x2e, 0x67, 0x72, 0x70, 0x63, 0x2e, 0x69, 0x6e, 0x73, 0x74, 0x61, 0x6e, 0x63,
	0x65, 0x73, 0x1a, 0x1f, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f,
	0x62, 0x75, 0x66, 0x2f, 0x74, 0x69, 0x6d, 0x65, 0x73, 0x74, 0x61, 0x6d, 0x70, 0x2e, 0x70, 0x72,
	0x6f, 0x74, 0x6f, 0x22, 0x46, 0x0a, 0x05, 0x46, 0x69, 0x6c, 0x65, 0x73, 0x12, 0x3d, 0x0a, 0x05,
	0x66, 0x69, 0x6c, 0x65, 0x73, 0x18, 0x01, 0x20, 0x03, 0x28, 0x0b, 0x32, 0x27, 0x2e, 0x66, 0x35,
	0x2e, 0x6e, 0x67, 0x69, 0x6e, 0x78, 0x2e, 0x61, 0x67, 0x65, 0x6e, 0x74, 0x2e, 0x61, 0x70, 0x69,
	0x2e, 0x67, 0x72, 0x70, 0x63, 0x2e, 0x69, 0x6e, 0x73, 0x74, 0x61, 0x6e, 0x63, 0x65, 0x73, 0x2e,
	0x46, 0x69, 0x6c, 0x65, 0x52, 0x05, 0x66, 0x69, 0x6c, 0x65, 0x73, 0x22, 0x75, 0x0a, 0x04, 0x46,
	0x69, 0x6c, 0x65, 0x12, 0x3f, 0x0a, 0x0d, 0x6c, 0x61, 0x73, 0x74, 0x5f, 0x6d, 0x6f, 0x64, 0x69,
	0x66, 0x69, 0x65, 0x64, 0x18, 0x01, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x1a, 0x2e, 0x67, 0x6f, 0x6f,
	0x67, 0x6c, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2e, 0x54, 0x69, 0x6d,
	0x65, 0x73, 0x74, 0x61, 0x6d, 0x70, 0x52, 0x0c, 0x6c, 0x61, 0x73, 0x74, 0x4d, 0x6f, 0x64, 0x69,
	0x66, 0x69, 0x65, 0x64, 0x12, 0x12, 0x0a, 0x04, 0x70, 0x61, 0x74, 0x68, 0x18, 0x02, 0x20, 0x01,
	0x28, 0x09, 0x52, 0x04, 0x70, 0x61, 0x74, 0x68, 0x12, 0x18, 0x0a, 0x07, 0x76, 0x65, 0x72, 0x73,
	0x69, 0x6f, 0x6e, 0x18, 0x03, 0x20, 0x01, 0x28, 0x09, 0x52, 0x07, 0x76, 0x65, 0x72, 0x73, 0x69,
	0x6f, 0x6e, 0x22, 0x91, 0x01, 0x0a, 0x14, 0x46, 0x69, 0x6c, 0x65, 0x44, 0x6f, 0x77, 0x6e, 0x6c,
	0x6f, 0x61, 0x64, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x12, 0x18, 0x0a, 0x07, 0x65,
	0x6e, 0x63, 0x6f, 0x64, 0x65, 0x64, 0x18, 0x01, 0x20, 0x01, 0x28, 0x08, 0x52, 0x07, 0x65, 0x6e,
	0x63, 0x6f, 0x64, 0x65, 0x64, 0x12, 0x21, 0x0a, 0x0c, 0x66, 0x69, 0x6c, 0x65, 0x5f, 0x63, 0x6f,
	0x6e, 0x74, 0x65, 0x6e, 0x74, 0x18, 0x02, 0x20, 0x01, 0x28, 0x0c, 0x52, 0x0b, 0x66, 0x69, 0x6c,
	0x65, 0x43, 0x6f, 0x6e, 0x74, 0x65, 0x6e, 0x74, 0x12, 0x1b, 0x0a, 0x09, 0x66, 0x69, 0x6c, 0x65,
	0x5f, 0x70, 0x61, 0x74, 0x68, 0x18, 0x03, 0x20, 0x01, 0x28, 0x09, 0x52, 0x08, 0x66, 0x69, 0x6c,
	0x65, 0x50, 0x61, 0x74, 0x68, 0x12, 0x1f, 0x0a, 0x0b, 0x69, 0x6e, 0x73, 0x74, 0x61, 0x6e, 0x63,
	0x65, 0x5f, 0x69, 0x64, 0x18, 0x04, 0x20, 0x01, 0x28, 0x09, 0x52, 0x0a, 0x69, 0x6e, 0x73, 0x74,
	0x61, 0x6e, 0x63, 0x65, 0x49, 0x44, 0x42, 0x2e, 0x5a, 0x2c, 0x67, 0x69, 0x74, 0x68, 0x75, 0x62,
	0x2e, 0x63, 0x6f, 0x6d, 0x2f, 0x6e, 0x67, 0x69, 0x6e, 0x78, 0x2f, 0x61, 0x67, 0x65, 0x6e, 0x74,
	0x2f, 0x76, 0x33, 0x2f, 0x61, 0x70, 0x69, 0x2f, 0x67, 0x72, 0x70, 0x63, 0x2f, 0x69, 0x6e, 0x73,
	0x74, 0x61, 0x6e, 0x63, 0x65, 0x73, 0x62, 0x06, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_api_grpc_instances_file_proto_rawDescOnce sync.Once
	file_api_grpc_instances_file_proto_rawDescData = file_api_grpc_instances_file_proto_rawDesc
)

func file_api_grpc_instances_file_proto_rawDescGZIP() []byte {
	file_api_grpc_instances_file_proto_rawDescOnce.Do(func() {
		file_api_grpc_instances_file_proto_rawDescData = protoimpl.X.CompressGZIP(file_api_grpc_instances_file_proto_rawDescData)
	})
	return file_api_grpc_instances_file_proto_rawDescData
}

var file_api_grpc_instances_file_proto_msgTypes = make([]protoimpl.MessageInfo, 3)
var file_api_grpc_instances_file_proto_goTypes = []interface{}{
	(*Files)(nil),                 // 0: f5.nginx.agent.api.grpc.instances.Files
	(*File)(nil),                  // 1: f5.nginx.agent.api.grpc.instances.File
	(*FileDownloadResponse)(nil),  // 2: f5.nginx.agent.api.grpc.instances.FileDownloadResponse
	(*timestamppb.Timestamp)(nil), // 3: google.protobuf.Timestamp
}
var file_api_grpc_instances_file_proto_depIdxs = []int32{
	1, // 0: f5.nginx.agent.api.grpc.instances.Files.files:type_name -> f5.nginx.agent.api.grpc.instances.File
	3, // 1: f5.nginx.agent.api.grpc.instances.File.last_modified:type_name -> google.protobuf.Timestamp
	2, // [2:2] is the sub-list for method output_type
	2, // [2:2] is the sub-list for method input_type
	2, // [2:2] is the sub-list for extension type_name
	2, // [2:2] is the sub-list for extension extendee
	0, // [0:2] is the sub-list for field type_name
}

func init() { file_api_grpc_instances_file_proto_init() }
func file_api_grpc_instances_file_proto_init() {
	if File_api_grpc_instances_file_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_api_grpc_instances_file_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*Files); i {
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
		file_api_grpc_instances_file_proto_msgTypes[1].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*File); i {
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
		file_api_grpc_instances_file_proto_msgTypes[2].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*FileDownloadResponse); i {
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
			RawDescriptor: file_api_grpc_instances_file_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   3,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_api_grpc_instances_file_proto_goTypes,
		DependencyIndexes: file_api_grpc_instances_file_proto_depIdxs,
		MessageInfos:      file_api_grpc_instances_file_proto_msgTypes,
	}.Build()
	File_api_grpc_instances_file_proto = out.File
	file_api_grpc_instances_file_proto_rawDesc = nil
	file_api_grpc_instances_file_proto_goTypes = nil
	file_api_grpc_instances_file_proto_depIdxs = nil
}
