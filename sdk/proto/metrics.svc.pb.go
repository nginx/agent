// Code generated by protoc-gen-gogo. DO NOT EDIT.
// source: metrics.svc.proto

package proto

import (
	context "context"
	fmt "fmt"
	proto "github.com/gogo/protobuf/proto"
	types "github.com/gogo/protobuf/types"
	events "github.com/nginx/agent/sdk/v2/proto/events"
	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
	math "math"
)

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

// This is a compile-time assertion to ensure that this generated file
// is compatible with the proto package it is being compiled against.
// A compilation error at this line likely means your copy of the
// proto package needs to be updated.
const _ = proto.GoGoProtoPackageIsVersion3 // please upgrade the proto package

func init() { proto.RegisterFile("metrics.svc.proto", fileDescriptor_ece8a4321458910f) }

var fileDescriptor_ece8a4321458910f = []byte{
	// 229 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0xe2, 0x12, 0xcc, 0x4d, 0x2d, 0x29,
	0xca, 0x4c, 0x2e, 0xd6, 0x2b, 0x2e, 0x4b, 0xd6, 0x2b, 0x28, 0xca, 0x2f, 0xc9, 0x17, 0x12, 0x4a,
	0x33, 0xd5, 0xcb, 0x4b, 0xcf, 0xcc, 0xab, 0xd0, 0x4b, 0x4c, 0x4f, 0xcd, 0x2b, 0xd1, 0x2b, 0x4e,
	0xc9, 0x96, 0x92, 0x4e, 0xcf, 0xcf, 0x4f, 0xcf, 0x49, 0xd5, 0x07, 0xab, 0x48, 0x2a, 0x4d, 0xd3,
	0x4f, 0xcd, 0x2d, 0x28, 0xa9, 0x84, 0x68, 0x90, 0x12, 0x4a, 0x2d, 0x4b, 0xcd, 0x2b, 0x29, 0xd6,
	0x07, 0x53, 0x50, 0x31, 0x5e, 0x98, 0xb9, 0x60, 0xae, 0xd1, 0x5a, 0x46, 0x2e, 0x3e, 0x5f, 0x88,
	0x48, 0x70, 0x6a, 0x51, 0x59, 0x66, 0x72, 0xaa, 0x90, 0x3b, 0x17, 0x5b, 0x70, 0x49, 0x51, 0x6a,
	0x62, 0xae, 0x90, 0xa2, 0x1e, 0xa6, 0x8d, 0x7a, 0x50, 0xd5, 0x41, 0xa9, 0x05, 0xf9, 0x45, 0x25,
	0x52, 0x62, 0x7a, 0x10, 0x07, 0xe8, 0xc1, 0x1c, 0xa0, 0xe7, 0x0a, 0x72, 0x80, 0x12, 0x83, 0x06,
	0xa3, 0x50, 0x10, 0x17, 0x0f, 0xc4, 0x20, 0x57, 0xb0, 0x33, 0x84, 0xd4, 0xb0, 0x19, 0x07, 0x71,
	0xa2, 0x1e, 0x58, 0x09, 0x61, 0x33, 0x9d, 0xcc, 0x4f, 0x3c, 0x92, 0x63, 0xbc, 0xf0, 0x48, 0x8e,
	0xf1, 0xc1, 0x23, 0x39, 0xc6, 0x28, 0xcd, 0xf4, 0xcc, 0x92, 0x8c, 0xd2, 0x24, 0xbd, 0xe4, 0xfc,
	0x5c, 0x7d, 0xb0, 0xc1, 0xfa, 0x60, 0x83, 0xf5, 0x8b, 0x53, 0xb2, 0xf5, 0xcb, 0x8c, 0x20, 0x81,
	0x62, 0x0d, 0x31, 0x85, 0x0d, 0x4c, 0x19, 0x03, 0x02, 0x00, 0x00, 0xff, 0xff, 0x9b, 0xc4, 0x8d,
	0xf5, 0x58, 0x01, 0x00, 0x00,
}

// Reference imports to suppress errors if they are not otherwise used.
var _ context.Context
var _ grpc.ClientConn

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
const _ = grpc.SupportPackageIsVersion4

// MetricsServiceClient is the client API for MetricsService service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://godoc.org/google.golang.org/grpc#ClientConn.NewStream.
type MetricsServiceClient interface {
	// A client-to-server streaming RPC to deliver high volume metrics reports.
	Stream(ctx context.Context, opts ...grpc.CallOption) (MetricsService_StreamClient, error)
	// A client-to-server streaming RPC to deliver high volume event reports.
	StreamEvents(ctx context.Context, opts ...grpc.CallOption) (MetricsService_StreamEventsClient, error)
}

type metricsServiceClient struct {
	cc *grpc.ClientConn
}

func NewMetricsServiceClient(cc *grpc.ClientConn) MetricsServiceClient {
	return &metricsServiceClient{cc}
}

func (c *metricsServiceClient) Stream(ctx context.Context, opts ...grpc.CallOption) (MetricsService_StreamClient, error) {
	stream, err := c.cc.NewStream(ctx, &_MetricsService_serviceDesc.Streams[0], "/f5.nginx.agent.sdk.MetricsService/Stream", opts...)
	if err != nil {
		return nil, err
	}
	x := &metricsServiceStreamClient{stream}
	return x, nil
}

type MetricsService_StreamClient interface {
	Send(*MetricsReport) error
	CloseAndRecv() (*types.Empty, error)
	grpc.ClientStream
}

type metricsServiceStreamClient struct {
	grpc.ClientStream
}

func (x *metricsServiceStreamClient) Send(m *MetricsReport) error {
	return x.ClientStream.SendMsg(m)
}

func (x *metricsServiceStreamClient) CloseAndRecv() (*types.Empty, error) {
	if err := x.ClientStream.CloseSend(); err != nil {
		return nil, err
	}
	m := new(types.Empty)
	if err := x.ClientStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

func (c *metricsServiceClient) StreamEvents(ctx context.Context, opts ...grpc.CallOption) (MetricsService_StreamEventsClient, error) {
	stream, err := c.cc.NewStream(ctx, &_MetricsService_serviceDesc.Streams[1], "/f5.nginx.agent.sdk.MetricsService/StreamEvents", opts...)
	if err != nil {
		return nil, err
	}
	x := &metricsServiceStreamEventsClient{stream}
	return x, nil
}

type MetricsService_StreamEventsClient interface {
	Send(*events.EventReport) error
	CloseAndRecv() (*types.Empty, error)
	grpc.ClientStream
}

type metricsServiceStreamEventsClient struct {
	grpc.ClientStream
}

func (x *metricsServiceStreamEventsClient) Send(m *events.EventReport) error {
	return x.ClientStream.SendMsg(m)
}

func (x *metricsServiceStreamEventsClient) CloseAndRecv() (*types.Empty, error) {
	if err := x.ClientStream.CloseSend(); err != nil {
		return nil, err
	}
	m := new(types.Empty)
	if err := x.ClientStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

// MetricsServiceServer is the server API for MetricsService service.
type MetricsServiceServer interface {
	// A client-to-server streaming RPC to deliver high volume metrics reports.
	Stream(MetricsService_StreamServer) error
	// A client-to-server streaming RPC to deliver high volume event reports.
	StreamEvents(MetricsService_StreamEventsServer) error
}

// UnimplementedMetricsServiceServer can be embedded to have forward compatible implementations.
type UnimplementedMetricsServiceServer struct {
}

func (*UnimplementedMetricsServiceServer) Stream(srv MetricsService_StreamServer) error {
	return status.Errorf(codes.Unimplemented, "method Stream not implemented")
}
func (*UnimplementedMetricsServiceServer) StreamEvents(srv MetricsService_StreamEventsServer) error {
	return status.Errorf(codes.Unimplemented, "method StreamEvents not implemented")
}

func RegisterMetricsServiceServer(s *grpc.Server, srv MetricsServiceServer) {
	s.RegisterService(&_MetricsService_serviceDesc, srv)
}

func _MetricsService_Stream_Handler(srv interface{}, stream grpc.ServerStream) error {
	return srv.(MetricsServiceServer).Stream(&metricsServiceStreamServer{stream})
}

type MetricsService_StreamServer interface {
	SendAndClose(*types.Empty) error
	Recv() (*MetricsReport, error)
	grpc.ServerStream
}

type metricsServiceStreamServer struct {
	grpc.ServerStream
}

func (x *metricsServiceStreamServer) SendAndClose(m *types.Empty) error {
	return x.ServerStream.SendMsg(m)
}

func (x *metricsServiceStreamServer) Recv() (*MetricsReport, error) {
	m := new(MetricsReport)
	if err := x.ServerStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

func _MetricsService_StreamEvents_Handler(srv interface{}, stream grpc.ServerStream) error {
	return srv.(MetricsServiceServer).StreamEvents(&metricsServiceStreamEventsServer{stream})
}

type MetricsService_StreamEventsServer interface {
	SendAndClose(*types.Empty) error
	Recv() (*events.EventReport, error)
	grpc.ServerStream
}

type metricsServiceStreamEventsServer struct {
	grpc.ServerStream
}

func (x *metricsServiceStreamEventsServer) SendAndClose(m *types.Empty) error {
	return x.ServerStream.SendMsg(m)
}

func (x *metricsServiceStreamEventsServer) Recv() (*events.EventReport, error) {
	m := new(events.EventReport)
	if err := x.ServerStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

var _MetricsService_serviceDesc = grpc.ServiceDesc{
	ServiceName: "f5.nginx.agent.sdk.MetricsService",
	HandlerType: (*MetricsServiceServer)(nil),
	Methods:     []grpc.MethodDesc{},
	Streams: []grpc.StreamDesc{
		{
			StreamName:    "Stream",
			Handler:       _MetricsService_Stream_Handler,
			ClientStreams: true,
		},
		{
			StreamName:    "StreamEvents",
			Handler:       _MetricsService_StreamEvents_Handler,
			ClientStreams: true,
		},
	},
	Metadata: "metrics.svc.proto",
}
