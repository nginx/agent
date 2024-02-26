// Copyright (c) F5, Inc.
//
// This source code is licensed under the Apache License, Version 2.0 license found in the
// LICENSE file in the root directory of this source tree.

// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.32.0
// 	protoc        v4.25.2
// source: file.proto

package v1

import (
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
	emptypb "google.golang.org/protobuf/types/known/emptypb"
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

// Action enumeration
type File_FileAction int32

const (
	// Default value, no action
	File_FILE_ACTION_UNSPECIFIED File_FileAction = 0
	// No changes to the file
	File_FILE_ACTION_UNCHANGED File_FileAction = 1
	// New file
	File_FILE_ACTION_ADD File_FileAction = 2
	// Updated file
	File_FILE_ACTION_UPDATE File_FileAction = 3
	// File deleted
	File_FILE_ACTION_DELETE File_FileAction = 4
)

// Enum value maps for File_FileAction.
var (
	File_FileAction_name = map[int32]string{
		0: "FILE_ACTION_UNSPECIFIED",
		1: "FILE_ACTION_UNCHANGED",
		2: "FILE_ACTION_ADD",
		3: "FILE_ACTION_UPDATE",
		4: "FILE_ACTION_DELETE",
	}
	File_FileAction_value = map[string]int32{
		"FILE_ACTION_UNSPECIFIED": 0,
		"FILE_ACTION_UNCHANGED":   1,
		"FILE_ACTION_ADD":         2,
		"FILE_ACTION_UPDATE":      3,
		"FILE_ACTION_DELETE":      4,
	}
)

func (x File_FileAction) Enum() *File_FileAction {
	p := new(File_FileAction)
	*p = x
	return p
}

func (x File_FileAction) String() string {
	return protoimpl.X.EnumStringOf(x.Descriptor(), protoreflect.EnumNumber(x))
}

func (File_FileAction) Descriptor() protoreflect.EnumDescriptor {
	return file_file_proto_enumTypes[0].Descriptor()
}

func (File_FileAction) Type() protoreflect.EnumType {
	return &file_file_proto_enumTypes[0]
}

func (x File_FileAction) Number() protoreflect.EnumNumber {
	return protoreflect.EnumNumber(x)
}

// Deprecated: Use File_FileAction.Descriptor instead.
func (File_FileAction) EnumDescriptor() ([]byte, []int) {
	return file_file_proto_rawDescGZIP(), []int{2, 0}
}

// Represents a specific configuration version associated with an instance
type ConfigVersion struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// the instance identifier
	InstanceId string `protobuf:"bytes,1,opt,name=instance_id,json=instanceId,proto3" json:"instance_id,omitempty"`
	// the version of the configuration
	Version string `protobuf:"bytes,2,opt,name=version,proto3" json:"version,omitempty"`
}

func (x *ConfigVersion) Reset() {
	*x = ConfigVersion{}
	if protoimpl.UnsafeEnabled {
		mi := &file_file_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *ConfigVersion) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ConfigVersion) ProtoMessage() {}

func (x *ConfigVersion) ProtoReflect() protoreflect.Message {
	mi := &file_file_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ConfigVersion.ProtoReflect.Descriptor instead.
func (*ConfigVersion) Descriptor() ([]byte, []int) {
	return file_file_proto_rawDescGZIP(), []int{0}
}

func (x *ConfigVersion) GetInstanceId() string {
	if x != nil {
		return x.InstanceId
	}
	return ""
}

func (x *ConfigVersion) GetVersion() string {
	if x != nil {
		return x.Version
	}
	return ""
}

// Represents a collection of files
type FileOverview struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// A list of files
	Files []*File `protobuf:"bytes,1,rep,name=files,proto3" json:"files,omitempty"`
	// the configuration version of the current set of files
	Version *ConfigVersion `protobuf:"bytes,2,opt,name=version,proto3" json:"version,omitempty"`
}

func (x *FileOverview) Reset() {
	*x = FileOverview{}
	if protoimpl.UnsafeEnabled {
		mi := &file_file_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *FileOverview) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*FileOverview) ProtoMessage() {}

func (x *FileOverview) ProtoReflect() protoreflect.Message {
	mi := &file_file_proto_msgTypes[1]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use FileOverview.ProtoReflect.Descriptor instead.
func (*FileOverview) Descriptor() ([]byte, []int) {
	return file_file_proto_rawDescGZIP(), []int{1}
}

func (x *FileOverview) GetFiles() []*File {
	if x != nil {
		return x.Files
	}
	return nil
}

func (x *FileOverview) GetVersion() *ConfigVersion {
	if x != nil {
		return x.Version
	}
	return nil
}

// Represents meta data about a file
type File struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// Meta information about the file, the name (including path) and hash
	FileMeta *FileMeta `protobuf:"bytes,1,opt,name=file_meta,json=fileMeta,proto3" json:"file_meta,omitempty"`
	// last modified time of the file (created time if never modified)
	ModifiedTime *timestamppb.Timestamp `protobuf:"bytes,2,opt,name=modified_time,json=modifiedTime,proto3" json:"modified_time,omitempty"`
	// the permission set associated with a particular file
	Permissions string `protobuf:"bytes,3,opt,name=permissions,proto3" json:"permissions,omitempty"`
	// Size of the file in bytes
	Size int64 `protobuf:"varint,4,opt,name=size,proto3" json:"size,omitempty"`
	// optional action
	Action File_FileAction `protobuf:"varint,5,opt,name=action,proto3,enum=f5.nginx.agent.api.grpc.mpi.v1.file.File_FileAction" json:"action,omitempty"`
	// optional contents
	Contents *FileContents `protobuf:"bytes,6,opt,name=contents,proto3" json:"contents,omitempty"`
}

func (x *File) Reset() {
	*x = File{}
	if protoimpl.UnsafeEnabled {
		mi := &file_file_proto_msgTypes[2]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *File) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*File) ProtoMessage() {}

func (x *File) ProtoReflect() protoreflect.Message {
	mi := &file_file_proto_msgTypes[2]
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
	return file_file_proto_rawDescGZIP(), []int{2}
}

func (x *File) GetFileMeta() *FileMeta {
	if x != nil {
		return x.FileMeta
	}
	return nil
}

func (x *File) GetModifiedTime() *timestamppb.Timestamp {
	if x != nil {
		return x.ModifiedTime
	}
	return nil
}

func (x *File) GetPermissions() string {
	if x != nil {
		return x.Permissions
	}
	return ""
}

func (x *File) GetSize() int64 {
	if x != nil {
		return x.Size
	}
	return 0
}

func (x *File) GetAction() File_FileAction {
	if x != nil {
		return x.Action
	}
	return File_FILE_ACTION_UNSPECIFIED
}

func (x *File) GetContents() *FileContents {
	if x != nil {
		return x.Contents
	}
	return nil
}

type FileRequest struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// Meta-information associated with a message
	MessageMeta *MessageMeta `protobuf:"bytes,1,opt,name=message_meta,json=messageMeta,proto3" json:"message_meta,omitempty"`
	// Meta-information associated with the file
	FileMeta *FileMeta `protobuf:"bytes,2,opt,name=file_meta,json=fileMeta,proto3" json:"file_meta,omitempty"`
}

func (x *FileRequest) Reset() {
	*x = FileRequest{}
	if protoimpl.UnsafeEnabled {
		mi := &file_file_proto_msgTypes[3]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *FileRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*FileRequest) ProtoMessage() {}

func (x *FileRequest) ProtoReflect() protoreflect.Message {
	mi := &file_file_proto_msgTypes[3]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use FileRequest.ProtoReflect.Descriptor instead.
func (*FileRequest) Descriptor() ([]byte, []int) {
	return file_file_proto_rawDescGZIP(), []int{3}
}

func (x *FileRequest) GetMessageMeta() *MessageMeta {
	if x != nil {
		return x.MessageMeta
	}
	return nil
}

func (x *FileRequest) GetFileMeta() *FileMeta {
	if x != nil {
		return x.FileMeta
	}
	return nil
}

// Represents the bytes contents of the file https://protobuf.dev/programming-guides/api/#dont-encode-data-in-a-string
type FileContents struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// byte representation of a file without encoding
	Contents []byte `protobuf:"bytes,1,opt,name=contents,proto3" json:"contents,omitempty"`
}

func (x *FileContents) Reset() {
	*x = FileContents{}
	if protoimpl.UnsafeEnabled {
		mi := &file_file_proto_msgTypes[4]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *FileContents) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*FileContents) ProtoMessage() {}

func (x *FileContents) ProtoReflect() protoreflect.Message {
	mi := &file_file_proto_msgTypes[4]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use FileContents.ProtoReflect.Descriptor instead.
func (*FileContents) Descriptor() ([]byte, []int) {
	return file_file_proto_rawDescGZIP(), []int{4}
}

func (x *FileContents) GetContents() []byte {
	if x != nil {
		return x.Contents
	}
	return nil
}

// Meta information about the file, the name (including path) and hash
type FileMeta struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// the name of the file
	Name string `protobuf:"bytes,1,opt,name=name,proto3" json:"name,omitempty"`
	// the hash of the file contents
	Hash string `protobuf:"bytes,2,opt,name=hash,proto3" json:"hash,omitempty"`
}

func (x *FileMeta) Reset() {
	*x = FileMeta{}
	if protoimpl.UnsafeEnabled {
		mi := &file_file_proto_msgTypes[5]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *FileMeta) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*FileMeta) ProtoMessage() {}

func (x *FileMeta) ProtoReflect() protoreflect.Message {
	mi := &file_file_proto_msgTypes[5]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use FileMeta.ProtoReflect.Descriptor instead.
func (*FileMeta) Descriptor() ([]byte, []int) {
	return file_file_proto_rawDescGZIP(), []int{5}
}

func (x *FileMeta) GetName() string {
	if x != nil {
		return x.Name
	}
	return ""
}

func (x *FileMeta) GetHash() string {
	if x != nil {
		return x.Hash
	}
	return ""
}

var File_file_proto protoreflect.FileDescriptor

var file_file_proto_rawDesc = []byte{
	0x0a, 0x0a, 0x66, 0x69, 0x6c, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x12, 0x23, 0x66, 0x35,
	0x2e, 0x6e, 0x67, 0x69, 0x6e, 0x78, 0x2e, 0x61, 0x67, 0x65, 0x6e, 0x74, 0x2e, 0x61, 0x70, 0x69,
	0x2e, 0x67, 0x72, 0x70, 0x63, 0x2e, 0x6d, 0x70, 0x69, 0x2e, 0x76, 0x31, 0x2e, 0x66, 0x69, 0x6c,
	0x65, 0x1a, 0x0c, 0x63, 0x6f, 0x6d, 0x6d, 0x6f, 0x6e, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x1a,
	0x1b, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66,
	0x2f, 0x65, 0x6d, 0x70, 0x74, 0x79, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x1a, 0x1f, 0x67, 0x6f,
	0x6f, 0x67, 0x6c, 0x65, 0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2f, 0x74, 0x69,
	0x6d, 0x65, 0x73, 0x74, 0x61, 0x6d, 0x70, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x22, 0x4a, 0x0a,
	0x0d, 0x43, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x56, 0x65, 0x72, 0x73, 0x69, 0x6f, 0x6e, 0x12, 0x1f,
	0x0a, 0x0b, 0x69, 0x6e, 0x73, 0x74, 0x61, 0x6e, 0x63, 0x65, 0x5f, 0x69, 0x64, 0x18, 0x01, 0x20,
	0x01, 0x28, 0x09, 0x52, 0x0a, 0x69, 0x6e, 0x73, 0x74, 0x61, 0x6e, 0x63, 0x65, 0x49, 0x64, 0x12,
	0x18, 0x0a, 0x07, 0x76, 0x65, 0x72, 0x73, 0x69, 0x6f, 0x6e, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09,
	0x52, 0x07, 0x76, 0x65, 0x72, 0x73, 0x69, 0x6f, 0x6e, 0x22, 0x9d, 0x01, 0x0a, 0x0c, 0x46, 0x69,
	0x6c, 0x65, 0x4f, 0x76, 0x65, 0x72, 0x76, 0x69, 0x65, 0x77, 0x12, 0x3f, 0x0a, 0x05, 0x66, 0x69,
	0x6c, 0x65, 0x73, 0x18, 0x01, 0x20, 0x03, 0x28, 0x0b, 0x32, 0x29, 0x2e, 0x66, 0x35, 0x2e, 0x6e,
	0x67, 0x69, 0x6e, 0x78, 0x2e, 0x61, 0x67, 0x65, 0x6e, 0x74, 0x2e, 0x61, 0x70, 0x69, 0x2e, 0x67,
	0x72, 0x70, 0x63, 0x2e, 0x6d, 0x70, 0x69, 0x2e, 0x76, 0x31, 0x2e, 0x66, 0x69, 0x6c, 0x65, 0x2e,
	0x46, 0x69, 0x6c, 0x65, 0x52, 0x05, 0x66, 0x69, 0x6c, 0x65, 0x73, 0x12, 0x4c, 0x0a, 0x07, 0x76,
	0x65, 0x72, 0x73, 0x69, 0x6f, 0x6e, 0x18, 0x02, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x32, 0x2e, 0x66,
	0x35, 0x2e, 0x6e, 0x67, 0x69, 0x6e, 0x78, 0x2e, 0x61, 0x67, 0x65, 0x6e, 0x74, 0x2e, 0x61, 0x70,
	0x69, 0x2e, 0x67, 0x72, 0x70, 0x63, 0x2e, 0x6d, 0x70, 0x69, 0x2e, 0x76, 0x31, 0x2e, 0x66, 0x69,
	0x6c, 0x65, 0x2e, 0x43, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x56, 0x65, 0x72, 0x73, 0x69, 0x6f, 0x6e,
	0x52, 0x07, 0x76, 0x65, 0x72, 0x73, 0x69, 0x6f, 0x6e, 0x22, 0xf2, 0x03, 0x0a, 0x04, 0x46, 0x69,
	0x6c, 0x65, 0x12, 0x4a, 0x0a, 0x09, 0x66, 0x69, 0x6c, 0x65, 0x5f, 0x6d, 0x65, 0x74, 0x61, 0x18,
	0x01, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x2d, 0x2e, 0x66, 0x35, 0x2e, 0x6e, 0x67, 0x69, 0x6e, 0x78,
	0x2e, 0x61, 0x67, 0x65, 0x6e, 0x74, 0x2e, 0x61, 0x70, 0x69, 0x2e, 0x67, 0x72, 0x70, 0x63, 0x2e,
	0x6d, 0x70, 0x69, 0x2e, 0x76, 0x31, 0x2e, 0x66, 0x69, 0x6c, 0x65, 0x2e, 0x46, 0x69, 0x6c, 0x65,
	0x4d, 0x65, 0x74, 0x61, 0x52, 0x08, 0x66, 0x69, 0x6c, 0x65, 0x4d, 0x65, 0x74, 0x61, 0x12, 0x3f,
	0x0a, 0x0d, 0x6d, 0x6f, 0x64, 0x69, 0x66, 0x69, 0x65, 0x64, 0x5f, 0x74, 0x69, 0x6d, 0x65, 0x18,
	0x02, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x1a, 0x2e, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2e, 0x70,
	0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2e, 0x54, 0x69, 0x6d, 0x65, 0x73, 0x74, 0x61, 0x6d,
	0x70, 0x52, 0x0c, 0x6d, 0x6f, 0x64, 0x69, 0x66, 0x69, 0x65, 0x64, 0x54, 0x69, 0x6d, 0x65, 0x12,
	0x20, 0x0a, 0x0b, 0x70, 0x65, 0x72, 0x6d, 0x69, 0x73, 0x73, 0x69, 0x6f, 0x6e, 0x73, 0x18, 0x03,
	0x20, 0x01, 0x28, 0x09, 0x52, 0x0b, 0x70, 0x65, 0x72, 0x6d, 0x69, 0x73, 0x73, 0x69, 0x6f, 0x6e,
	0x73, 0x12, 0x12, 0x0a, 0x04, 0x73, 0x69, 0x7a, 0x65, 0x18, 0x04, 0x20, 0x01, 0x28, 0x03, 0x52,
	0x04, 0x73, 0x69, 0x7a, 0x65, 0x12, 0x4c, 0x0a, 0x06, 0x61, 0x63, 0x74, 0x69, 0x6f, 0x6e, 0x18,
	0x05, 0x20, 0x01, 0x28, 0x0e, 0x32, 0x34, 0x2e, 0x66, 0x35, 0x2e, 0x6e, 0x67, 0x69, 0x6e, 0x78,
	0x2e, 0x61, 0x67, 0x65, 0x6e, 0x74, 0x2e, 0x61, 0x70, 0x69, 0x2e, 0x67, 0x72, 0x70, 0x63, 0x2e,
	0x6d, 0x70, 0x69, 0x2e, 0x76, 0x31, 0x2e, 0x66, 0x69, 0x6c, 0x65, 0x2e, 0x46, 0x69, 0x6c, 0x65,
	0x2e, 0x46, 0x69, 0x6c, 0x65, 0x41, 0x63, 0x74, 0x69, 0x6f, 0x6e, 0x52, 0x06, 0x61, 0x63, 0x74,
	0x69, 0x6f, 0x6e, 0x12, 0x4d, 0x0a, 0x08, 0x63, 0x6f, 0x6e, 0x74, 0x65, 0x6e, 0x74, 0x73, 0x18,
	0x06, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x31, 0x2e, 0x66, 0x35, 0x2e, 0x6e, 0x67, 0x69, 0x6e, 0x78,
	0x2e, 0x61, 0x67, 0x65, 0x6e, 0x74, 0x2e, 0x61, 0x70, 0x69, 0x2e, 0x67, 0x72, 0x70, 0x63, 0x2e,
	0x6d, 0x70, 0x69, 0x2e, 0x76, 0x31, 0x2e, 0x66, 0x69, 0x6c, 0x65, 0x2e, 0x46, 0x69, 0x6c, 0x65,
	0x43, 0x6f, 0x6e, 0x74, 0x65, 0x6e, 0x74, 0x73, 0x52, 0x08, 0x63, 0x6f, 0x6e, 0x74, 0x65, 0x6e,
	0x74, 0x73, 0x22, 0x89, 0x01, 0x0a, 0x0a, 0x46, 0x69, 0x6c, 0x65, 0x41, 0x63, 0x74, 0x69, 0x6f,
	0x6e, 0x12, 0x1b, 0x0a, 0x17, 0x46, 0x49, 0x4c, 0x45, 0x5f, 0x41, 0x43, 0x54, 0x49, 0x4f, 0x4e,
	0x5f, 0x55, 0x4e, 0x53, 0x50, 0x45, 0x43, 0x49, 0x46, 0x49, 0x45, 0x44, 0x10, 0x00, 0x12, 0x19,
	0x0a, 0x15, 0x46, 0x49, 0x4c, 0x45, 0x5f, 0x41, 0x43, 0x54, 0x49, 0x4f, 0x4e, 0x5f, 0x55, 0x4e,
	0x43, 0x48, 0x41, 0x4e, 0x47, 0x45, 0x44, 0x10, 0x01, 0x12, 0x13, 0x0a, 0x0f, 0x46, 0x49, 0x4c,
	0x45, 0x5f, 0x41, 0x43, 0x54, 0x49, 0x4f, 0x4e, 0x5f, 0x41, 0x44, 0x44, 0x10, 0x02, 0x12, 0x16,
	0x0a, 0x12, 0x46, 0x49, 0x4c, 0x45, 0x5f, 0x41, 0x43, 0x54, 0x49, 0x4f, 0x4e, 0x5f, 0x55, 0x50,
	0x44, 0x41, 0x54, 0x45, 0x10, 0x03, 0x12, 0x16, 0x0a, 0x12, 0x46, 0x49, 0x4c, 0x45, 0x5f, 0x41,
	0x43, 0x54, 0x49, 0x4f, 0x4e, 0x5f, 0x44, 0x45, 0x4c, 0x45, 0x54, 0x45, 0x10, 0x04, 0x22, 0xb0,
	0x01, 0x0a, 0x0b, 0x46, 0x69, 0x6c, 0x65, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x12, 0x55,
	0x0a, 0x0c, 0x6d, 0x65, 0x73, 0x73, 0x61, 0x67, 0x65, 0x5f, 0x6d, 0x65, 0x74, 0x61, 0x18, 0x01,
	0x20, 0x01, 0x28, 0x0b, 0x32, 0x32, 0x2e, 0x66, 0x35, 0x2e, 0x6e, 0x67, 0x69, 0x6e, 0x78, 0x2e,
	0x61, 0x67, 0x65, 0x6e, 0x74, 0x2e, 0x61, 0x70, 0x69, 0x2e, 0x67, 0x72, 0x70, 0x63, 0x2e, 0x6d,
	0x70, 0x69, 0x2e, 0x76, 0x31, 0x2e, 0x63, 0x6f, 0x6d, 0x6d, 0x6f, 0x6e, 0x2e, 0x4d, 0x65, 0x73,
	0x73, 0x61, 0x67, 0x65, 0x4d, 0x65, 0x74, 0x61, 0x52, 0x0b, 0x6d, 0x65, 0x73, 0x73, 0x61, 0x67,
	0x65, 0x4d, 0x65, 0x74, 0x61, 0x12, 0x4a, 0x0a, 0x09, 0x66, 0x69, 0x6c, 0x65, 0x5f, 0x6d, 0x65,
	0x74, 0x61, 0x18, 0x02, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x2d, 0x2e, 0x66, 0x35, 0x2e, 0x6e, 0x67,
	0x69, 0x6e, 0x78, 0x2e, 0x61, 0x67, 0x65, 0x6e, 0x74, 0x2e, 0x61, 0x70, 0x69, 0x2e, 0x67, 0x72,
	0x70, 0x63, 0x2e, 0x6d, 0x70, 0x69, 0x2e, 0x76, 0x31, 0x2e, 0x66, 0x69, 0x6c, 0x65, 0x2e, 0x46,
	0x69, 0x6c, 0x65, 0x4d, 0x65, 0x74, 0x61, 0x52, 0x08, 0x66, 0x69, 0x6c, 0x65, 0x4d, 0x65, 0x74,
	0x61, 0x22, 0x2a, 0x0a, 0x0c, 0x46, 0x69, 0x6c, 0x65, 0x43, 0x6f, 0x6e, 0x74, 0x65, 0x6e, 0x74,
	0x73, 0x12, 0x1a, 0x0a, 0x08, 0x63, 0x6f, 0x6e, 0x74, 0x65, 0x6e, 0x74, 0x73, 0x18, 0x01, 0x20,
	0x01, 0x28, 0x0c, 0x52, 0x08, 0x63, 0x6f, 0x6e, 0x74, 0x65, 0x6e, 0x74, 0x73, 0x22, 0x32, 0x0a,
	0x08, 0x46, 0x69, 0x6c, 0x65, 0x4d, 0x65, 0x74, 0x61, 0x12, 0x12, 0x0a, 0x04, 0x6e, 0x61, 0x6d,
	0x65, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x04, 0x6e, 0x61, 0x6d, 0x65, 0x12, 0x12, 0x0a,
	0x04, 0x68, 0x61, 0x73, 0x68, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x04, 0x68, 0x61, 0x73,
	0x68, 0x32, 0xc0, 0x03, 0x0a, 0x0b, 0x46, 0x69, 0x6c, 0x65, 0x53, 0x65, 0x72, 0x76, 0x69, 0x63,
	0x65, 0x12, 0x76, 0x0a, 0x0b, 0x47, 0x65, 0x74, 0x4f, 0x76, 0x65, 0x72, 0x76, 0x69, 0x65, 0x77,
	0x12, 0x32, 0x2e, 0x66, 0x35, 0x2e, 0x6e, 0x67, 0x69, 0x6e, 0x78, 0x2e, 0x61, 0x67, 0x65, 0x6e,
	0x74, 0x2e, 0x61, 0x70, 0x69, 0x2e, 0x67, 0x72, 0x70, 0x63, 0x2e, 0x6d, 0x70, 0x69, 0x2e, 0x76,
	0x31, 0x2e, 0x66, 0x69, 0x6c, 0x65, 0x2e, 0x43, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x56, 0x65, 0x72,
	0x73, 0x69, 0x6f, 0x6e, 0x1a, 0x31, 0x2e, 0x66, 0x35, 0x2e, 0x6e, 0x67, 0x69, 0x6e, 0x78, 0x2e,
	0x61, 0x67, 0x65, 0x6e, 0x74, 0x2e, 0x61, 0x70, 0x69, 0x2e, 0x67, 0x72, 0x70, 0x63, 0x2e, 0x6d,
	0x70, 0x69, 0x2e, 0x76, 0x31, 0x2e, 0x66, 0x69, 0x6c, 0x65, 0x2e, 0x46, 0x69, 0x6c, 0x65, 0x4f,
	0x76, 0x65, 0x72, 0x76, 0x69, 0x65, 0x77, 0x22, 0x00, 0x12, 0x5d, 0x0a, 0x0e, 0x55, 0x70, 0x64,
	0x61, 0x74, 0x65, 0x4f, 0x76, 0x65, 0x72, 0x76, 0x69, 0x65, 0x77, 0x12, 0x31, 0x2e, 0x66, 0x35,
	0x2e, 0x6e, 0x67, 0x69, 0x6e, 0x78, 0x2e, 0x61, 0x67, 0x65, 0x6e, 0x74, 0x2e, 0x61, 0x70, 0x69,
	0x2e, 0x67, 0x72, 0x70, 0x63, 0x2e, 0x6d, 0x70, 0x69, 0x2e, 0x76, 0x31, 0x2e, 0x66, 0x69, 0x6c,
	0x65, 0x2e, 0x46, 0x69, 0x6c, 0x65, 0x4f, 0x76, 0x65, 0x72, 0x76, 0x69, 0x65, 0x77, 0x1a, 0x16,
	0x2e, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66,
	0x2e, 0x45, 0x6d, 0x70, 0x74, 0x79, 0x22, 0x00, 0x12, 0x70, 0x0a, 0x07, 0x47, 0x65, 0x74, 0x46,
	0x69, 0x6c, 0x65, 0x12, 0x30, 0x2e, 0x66, 0x35, 0x2e, 0x6e, 0x67, 0x69, 0x6e, 0x78, 0x2e, 0x61,
	0x67, 0x65, 0x6e, 0x74, 0x2e, 0x61, 0x70, 0x69, 0x2e, 0x67, 0x72, 0x70, 0x63, 0x2e, 0x6d, 0x70,
	0x69, 0x2e, 0x76, 0x31, 0x2e, 0x66, 0x69, 0x6c, 0x65, 0x2e, 0x46, 0x69, 0x6c, 0x65, 0x52, 0x65,
	0x71, 0x75, 0x65, 0x73, 0x74, 0x1a, 0x31, 0x2e, 0x66, 0x35, 0x2e, 0x6e, 0x67, 0x69, 0x6e, 0x78,
	0x2e, 0x61, 0x67, 0x65, 0x6e, 0x74, 0x2e, 0x61, 0x70, 0x69, 0x2e, 0x67, 0x72, 0x70, 0x63, 0x2e,
	0x6d, 0x70, 0x69, 0x2e, 0x76, 0x31, 0x2e, 0x66, 0x69, 0x6c, 0x65, 0x2e, 0x46, 0x69, 0x6c, 0x65,
	0x43, 0x6f, 0x6e, 0x74, 0x65, 0x6e, 0x74, 0x73, 0x22, 0x00, 0x12, 0x68, 0x0a, 0x0a, 0x55, 0x70,
	0x64, 0x61, 0x74, 0x65, 0x46, 0x69, 0x6c, 0x65, 0x12, 0x29, 0x2e, 0x66, 0x35, 0x2e, 0x6e, 0x67,
	0x69, 0x6e, 0x78, 0x2e, 0x61, 0x67, 0x65, 0x6e, 0x74, 0x2e, 0x61, 0x70, 0x69, 0x2e, 0x67, 0x72,
	0x70, 0x63, 0x2e, 0x6d, 0x70, 0x69, 0x2e, 0x76, 0x31, 0x2e, 0x66, 0x69, 0x6c, 0x65, 0x2e, 0x46,
	0x69, 0x6c, 0x65, 0x1a, 0x2d, 0x2e, 0x66, 0x35, 0x2e, 0x6e, 0x67, 0x69, 0x6e, 0x78, 0x2e, 0x61,
	0x67, 0x65, 0x6e, 0x74, 0x2e, 0x61, 0x70, 0x69, 0x2e, 0x67, 0x72, 0x70, 0x63, 0x2e, 0x6d, 0x70,
	0x69, 0x2e, 0x76, 0x31, 0x2e, 0x66, 0x69, 0x6c, 0x65, 0x2e, 0x46, 0x69, 0x6c, 0x65, 0x4d, 0x65,
	0x74, 0x61, 0x22, 0x00, 0x42, 0x2b, 0x5a, 0x29, 0x67, 0x69, 0x74, 0x68, 0x75, 0x62, 0x2e, 0x63,
	0x6f, 0x6d, 0x2f, 0x6e, 0x67, 0x69, 0x6e, 0x78, 0x2f, 0x61, 0x67, 0x65, 0x6e, 0x74, 0x2f, 0x76,
	0x33, 0x2f, 0x61, 0x70, 0x69, 0x2f, 0x67, 0x72, 0x70, 0x63, 0x2f, 0x6d, 0x70, 0x69, 0x2f, 0x76,
	0x31, 0x62, 0x06, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_file_proto_rawDescOnce sync.Once
	file_file_proto_rawDescData = file_file_proto_rawDesc
)

func file_file_proto_rawDescGZIP() []byte {
	file_file_proto_rawDescOnce.Do(func() {
		file_file_proto_rawDescData = protoimpl.X.CompressGZIP(file_file_proto_rawDescData)
	})
	return file_file_proto_rawDescData
}

var file_file_proto_enumTypes = make([]protoimpl.EnumInfo, 1)
var file_file_proto_msgTypes = make([]protoimpl.MessageInfo, 6)
var file_file_proto_goTypes = []interface{}{
	(File_FileAction)(0),          // 0: f5.nginx.agent.api.grpc.mpi.v1.file.File.FileAction
	(*ConfigVersion)(nil),         // 1: f5.nginx.agent.api.grpc.mpi.v1.file.ConfigVersion
	(*FileOverview)(nil),          // 2: f5.nginx.agent.api.grpc.mpi.v1.file.FileOverview
	(*File)(nil),                  // 3: f5.nginx.agent.api.grpc.mpi.v1.file.File
	(*FileRequest)(nil),           // 4: f5.nginx.agent.api.grpc.mpi.v1.file.FileRequest
	(*FileContents)(nil),          // 5: f5.nginx.agent.api.grpc.mpi.v1.file.FileContents
	(*FileMeta)(nil),              // 6: f5.nginx.agent.api.grpc.mpi.v1.file.FileMeta
	(*timestamppb.Timestamp)(nil), // 7: google.protobuf.Timestamp
	(*MessageMeta)(nil),           // 8: f5.nginx.agent.api.grpc.mpi.v1.common.MessageMeta
	(*emptypb.Empty)(nil),         // 9: google.protobuf.Empty
}
var file_file_proto_depIdxs = []int32{
	3,  // 0: f5.nginx.agent.api.grpc.mpi.v1.file.FileOverview.files:type_name -> f5.nginx.agent.api.grpc.mpi.v1.file.File
	1,  // 1: f5.nginx.agent.api.grpc.mpi.v1.file.FileOverview.version:type_name -> f5.nginx.agent.api.grpc.mpi.v1.file.ConfigVersion
	6,  // 2: f5.nginx.agent.api.grpc.mpi.v1.file.File.file_meta:type_name -> f5.nginx.agent.api.grpc.mpi.v1.file.FileMeta
	7,  // 3: f5.nginx.agent.api.grpc.mpi.v1.file.File.modified_time:type_name -> google.protobuf.Timestamp
	0,  // 4: f5.nginx.agent.api.grpc.mpi.v1.file.File.action:type_name -> f5.nginx.agent.api.grpc.mpi.v1.file.File.FileAction
	5,  // 5: f5.nginx.agent.api.grpc.mpi.v1.file.File.contents:type_name -> f5.nginx.agent.api.grpc.mpi.v1.file.FileContents
	8,  // 6: f5.nginx.agent.api.grpc.mpi.v1.file.FileRequest.message_meta:type_name -> f5.nginx.agent.api.grpc.mpi.v1.common.MessageMeta
	6,  // 7: f5.nginx.agent.api.grpc.mpi.v1.file.FileRequest.file_meta:type_name -> f5.nginx.agent.api.grpc.mpi.v1.file.FileMeta
	1,  // 8: f5.nginx.agent.api.grpc.mpi.v1.file.FileService.GetOverview:input_type -> f5.nginx.agent.api.grpc.mpi.v1.file.ConfigVersion
	2,  // 9: f5.nginx.agent.api.grpc.mpi.v1.file.FileService.UpdateOverview:input_type -> f5.nginx.agent.api.grpc.mpi.v1.file.FileOverview
	4,  // 10: f5.nginx.agent.api.grpc.mpi.v1.file.FileService.GetFile:input_type -> f5.nginx.agent.api.grpc.mpi.v1.file.FileRequest
	3,  // 11: f5.nginx.agent.api.grpc.mpi.v1.file.FileService.UpdateFile:input_type -> f5.nginx.agent.api.grpc.mpi.v1.file.File
	2,  // 12: f5.nginx.agent.api.grpc.mpi.v1.file.FileService.GetOverview:output_type -> f5.nginx.agent.api.grpc.mpi.v1.file.FileOverview
	9,  // 13: f5.nginx.agent.api.grpc.mpi.v1.file.FileService.UpdateOverview:output_type -> google.protobuf.Empty
	5,  // 14: f5.nginx.agent.api.grpc.mpi.v1.file.FileService.GetFile:output_type -> f5.nginx.agent.api.grpc.mpi.v1.file.FileContents
	6,  // 15: f5.nginx.agent.api.grpc.mpi.v1.file.FileService.UpdateFile:output_type -> f5.nginx.agent.api.grpc.mpi.v1.file.FileMeta
	12, // [12:16] is the sub-list for method output_type
	8,  // [8:12] is the sub-list for method input_type
	8,  // [8:8] is the sub-list for extension type_name
	8,  // [8:8] is the sub-list for extension extendee
	0,  // [0:8] is the sub-list for field type_name
}

func init() { file_file_proto_init() }
func file_file_proto_init() {
	if File_file_proto != nil {
		return
	}
	file_common_proto_init()
	if !protoimpl.UnsafeEnabled {
		file_file_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*ConfigVersion); i {
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
		file_file_proto_msgTypes[1].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*FileOverview); i {
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
		file_file_proto_msgTypes[2].Exporter = func(v interface{}, i int) interface{} {
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
		file_file_proto_msgTypes[3].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*FileRequest); i {
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
		file_file_proto_msgTypes[4].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*FileContents); i {
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
		file_file_proto_msgTypes[5].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*FileMeta); i {
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
			RawDescriptor: file_file_proto_rawDesc,
			NumEnums:      1,
			NumMessages:   6,
			NumExtensions: 0,
			NumServices:   1,
		},
		GoTypes:           file_file_proto_goTypes,
		DependencyIndexes: file_file_proto_depIdxs,
		EnumInfos:         file_file_proto_enumTypes,
		MessageInfos:      file_file_proto_msgTypes,
	}.Build()
	File_file_proto = out.File
	file_file_proto_rawDesc = nil
	file_file_proto_goTypes = nil
	file_file_proto_depIdxs = nil
}
