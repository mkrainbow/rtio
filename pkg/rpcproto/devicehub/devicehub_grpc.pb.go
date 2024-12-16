// Code generated by protoc-gen-go-grpc. DO NOT EDIT.
// versions:
// - protoc-gen-go-grpc v1.2.0
// - protoc             v3.12.4
// source: devicehub/devicehub.proto

package devicehub

import (
	context "context"
	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
)

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
// Requires gRPC-Go v1.32.0 or later.
const _ = grpc.SupportPackageIsVersion7

// AccessServiceClient is the client API for AccessService service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type AccessServiceClient interface {
	CoPost(ctx context.Context, in *CoReq, opts ...grpc.CallOption) (*CoResp, error)
	ObGet(ctx context.Context, in *ObGetReq, opts ...grpc.CallOption) (AccessService_ObGetClient, error)
	DeviceQuery(ctx context.Context, in *DeviceQueryReq, opts ...grpc.CallOption) (*DeviceQueryResp, error)
}

type accessServiceClient struct {
	cc grpc.ClientConnInterface
}

func NewAccessServiceClient(cc grpc.ClientConnInterface) AccessServiceClient {
	return &accessServiceClient{cc}
}

func (c *accessServiceClient) CoPost(ctx context.Context, in *CoReq, opts ...grpc.CallOption) (*CoResp, error) {
	out := new(CoResp)
	err := c.cc.Invoke(ctx, "/devicehub.AccessService/CoPost", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *accessServiceClient) ObGet(ctx context.Context, in *ObGetReq, opts ...grpc.CallOption) (AccessService_ObGetClient, error) {
	stream, err := c.cc.NewStream(ctx, &AccessService_ServiceDesc.Streams[0], "/devicehub.AccessService/ObGet", opts...)
	if err != nil {
		return nil, err
	}
	x := &accessServiceObGetClient{stream}
	if err := x.ClientStream.SendMsg(in); err != nil {
		return nil, err
	}
	if err := x.ClientStream.CloseSend(); err != nil {
		return nil, err
	}
	return x, nil
}

type AccessService_ObGetClient interface {
	Recv() (*ObGetResp, error)
	grpc.ClientStream
}

type accessServiceObGetClient struct {
	grpc.ClientStream
}

func (x *accessServiceObGetClient) Recv() (*ObGetResp, error) {
	m := new(ObGetResp)
	if err := x.ClientStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

func (c *accessServiceClient) DeviceQuery(ctx context.Context, in *DeviceQueryReq, opts ...grpc.CallOption) (*DeviceQueryResp, error) {
	out := new(DeviceQueryResp)
	err := c.cc.Invoke(ctx, "/devicehub.AccessService/DeviceQuery", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// AccessServiceServer is the server API for AccessService service.
// All implementations must embed UnimplementedAccessServiceServer
// for forward compatibility
type AccessServiceServer interface {
	CoPost(context.Context, *CoReq) (*CoResp, error)
	ObGet(*ObGetReq, AccessService_ObGetServer) error
	DeviceQuery(context.Context, *DeviceQueryReq) (*DeviceQueryResp, error)
	mustEmbedUnimplementedAccessServiceServer()
}

// UnimplementedAccessServiceServer must be embedded to have forward compatible implementations.
type UnimplementedAccessServiceServer struct {
}

func (UnimplementedAccessServiceServer) CoPost(context.Context, *CoReq) (*CoResp, error) {
	return nil, status.Errorf(codes.Unimplemented, "method CoPost not implemented")
}
func (UnimplementedAccessServiceServer) ObGet(*ObGetReq, AccessService_ObGetServer) error {
	return status.Errorf(codes.Unimplemented, "method ObGet not implemented")
}
func (UnimplementedAccessServiceServer) DeviceQuery(context.Context, *DeviceQueryReq) (*DeviceQueryResp, error) {
	return nil, status.Errorf(codes.Unimplemented, "method DeviceQuery not implemented")
}
func (UnimplementedAccessServiceServer) mustEmbedUnimplementedAccessServiceServer() {}

// UnsafeAccessServiceServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to AccessServiceServer will
// result in compilation errors.
type UnsafeAccessServiceServer interface {
	mustEmbedUnimplementedAccessServiceServer()
}

func RegisterAccessServiceServer(s grpc.ServiceRegistrar, srv AccessServiceServer) {
	s.RegisterService(&AccessService_ServiceDesc, srv)
}

func _AccessService_CoPost_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(CoReq)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(AccessServiceServer).CoPost(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/devicehub.AccessService/CoPost",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(AccessServiceServer).CoPost(ctx, req.(*CoReq))
	}
	return interceptor(ctx, in, info, handler)
}

func _AccessService_ObGet_Handler(srv interface{}, stream grpc.ServerStream) error {
	m := new(ObGetReq)
	if err := stream.RecvMsg(m); err != nil {
		return err
	}
	return srv.(AccessServiceServer).ObGet(m, &accessServiceObGetServer{stream})
}

type AccessService_ObGetServer interface {
	Send(*ObGetResp) error
	grpc.ServerStream
}

type accessServiceObGetServer struct {
	grpc.ServerStream
}

func (x *accessServiceObGetServer) Send(m *ObGetResp) error {
	return x.ServerStream.SendMsg(m)
}

func _AccessService_DeviceQuery_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(DeviceQueryReq)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(AccessServiceServer).DeviceQuery(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/devicehub.AccessService/DeviceQuery",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(AccessServiceServer).DeviceQuery(ctx, req.(*DeviceQueryReq))
	}
	return interceptor(ctx, in, info, handler)
}

// AccessService_ServiceDesc is the grpc.ServiceDesc for AccessService service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var AccessService_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "devicehub.AccessService",
	HandlerType: (*AccessServiceServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "CoPost",
			Handler:    _AccessService_CoPost_Handler,
		},
		{
			MethodName: "DeviceQuery",
			Handler:    _AccessService_DeviceQuery_Handler,
		},
	},
	Streams: []grpc.StreamDesc{
		{
			StreamName:    "ObGet",
			Handler:       _AccessService_ObGet_Handler,
			ServerStreams: true,
		},
	},
	Metadata: "devicehub/devicehub.proto",
}
