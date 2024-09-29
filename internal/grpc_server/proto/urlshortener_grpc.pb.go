// Code generated by protoc-gen-go-grpc. DO NOT EDIT.
// versions:
// - protoc-gen-go-grpc v1.5.1
// - protoc             v3.12.4
// source: internal/grpc_server/proto/urlshortener.proto

package proto

import (
	context "context"
	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
)

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
// Requires gRPC-Go v1.64.0 or later.
const _ = grpc.SupportPackageIsVersion9

const (
	URLShortener_ShortenURL_FullMethodName      = "/urlshortener.URLShortener/ShortenURL"
	URLShortener_ShortenBatchURL_FullMethodName = "/urlshortener.URLShortener/ShortenBatchURL"
	URLShortener_GetURL_FullMethodName          = "/urlshortener.URLShortener/GetURL"
	URLShortener_GetUserURLs_FullMethodName     = "/urlshortener.URLShortener/GetUserURLs"
	URLShortener_DeleteUserURLs_FullMethodName  = "/urlshortener.URLShortener/DeleteUserURLs"
	URLShortener_GetStats_FullMethodName        = "/urlshortener.URLShortener/GetStats"
	URLShortener_Ping_FullMethodName            = "/urlshortener.URLShortener/Ping"
)

// URLShortenerClient is the client API for URLShortener service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type URLShortenerClient interface {
	ShortenURL(ctx context.Context, in *ShortenURLReq, opts ...grpc.CallOption) (*ShortenURLRes, error)
	ShortenBatchURL(ctx context.Context, in *ShortenBatchURLReq, opts ...grpc.CallOption) (*ShortenBatchURLRes, error)
	GetURL(ctx context.Context, in *GetURLReq, opts ...grpc.CallOption) (*GetURLRes, error)
	GetUserURLs(ctx context.Context, in *GetUsersURLsReq, opts ...grpc.CallOption) (*GetUsersURLsRes, error)
	DeleteUserURLs(ctx context.Context, in *DeleteUserURLsReq, opts ...grpc.CallOption) (*DeleteUserURLsRes, error)
	GetStats(ctx context.Context, in *GetStatsReq, opts ...grpc.CallOption) (*GetStatsRes, error)
	Ping(ctx context.Context, in *PingReq, opts ...grpc.CallOption) (*PingRes, error)
}

type uRLShortenerClient struct {
	cc grpc.ClientConnInterface
}

func NewURLShortenerClient(cc grpc.ClientConnInterface) URLShortenerClient {
	return &uRLShortenerClient{cc}
}

func (c *uRLShortenerClient) ShortenURL(ctx context.Context, in *ShortenURLReq, opts ...grpc.CallOption) (*ShortenURLRes, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(ShortenURLRes)
	err := c.cc.Invoke(ctx, URLShortener_ShortenURL_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *uRLShortenerClient) ShortenBatchURL(ctx context.Context, in *ShortenBatchURLReq, opts ...grpc.CallOption) (*ShortenBatchURLRes, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(ShortenBatchURLRes)
	err := c.cc.Invoke(ctx, URLShortener_ShortenBatchURL_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *uRLShortenerClient) GetURL(ctx context.Context, in *GetURLReq, opts ...grpc.CallOption) (*GetURLRes, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(GetURLRes)
	err := c.cc.Invoke(ctx, URLShortener_GetURL_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *uRLShortenerClient) GetUserURLs(ctx context.Context, in *GetUsersURLsReq, opts ...grpc.CallOption) (*GetUsersURLsRes, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(GetUsersURLsRes)
	err := c.cc.Invoke(ctx, URLShortener_GetUserURLs_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *uRLShortenerClient) DeleteUserURLs(ctx context.Context, in *DeleteUserURLsReq, opts ...grpc.CallOption) (*DeleteUserURLsRes, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(DeleteUserURLsRes)
	err := c.cc.Invoke(ctx, URLShortener_DeleteUserURLs_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *uRLShortenerClient) GetStats(ctx context.Context, in *GetStatsReq, opts ...grpc.CallOption) (*GetStatsRes, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(GetStatsRes)
	err := c.cc.Invoke(ctx, URLShortener_GetStats_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *uRLShortenerClient) Ping(ctx context.Context, in *PingReq, opts ...grpc.CallOption) (*PingRes, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(PingRes)
	err := c.cc.Invoke(ctx, URLShortener_Ping_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// URLShortenerServer is the server API for URLShortener service.
// All implementations must embed UnimplementedURLShortenerServer
// for forward compatibility.
type URLShortenerServer interface {
	ShortenURL(context.Context, *ShortenURLReq) (*ShortenURLRes, error)
	ShortenBatchURL(context.Context, *ShortenBatchURLReq) (*ShortenBatchURLRes, error)
	GetURL(context.Context, *GetURLReq) (*GetURLRes, error)
	GetUserURLs(context.Context, *GetUsersURLsReq) (*GetUsersURLsRes, error)
	DeleteUserURLs(context.Context, *DeleteUserURLsReq) (*DeleteUserURLsRes, error)
	GetStats(context.Context, *GetStatsReq) (*GetStatsRes, error)
	Ping(context.Context, *PingReq) (*PingRes, error)
	mustEmbedUnimplementedURLShortenerServer()
}

// UnimplementedURLShortenerServer must be embedded to have
// forward compatible implementations.
//
// NOTE: this should be embedded by value instead of pointer to avoid a nil
// pointer dereference when methods are called.
type UnimplementedURLShortenerServer struct{}

func (UnimplementedURLShortenerServer) ShortenURL(context.Context, *ShortenURLReq) (*ShortenURLRes, error) {
	return nil, status.Errorf(codes.Unimplemented, "method ShortenURL not implemented")
}
func (UnimplementedURLShortenerServer) ShortenBatchURL(context.Context, *ShortenBatchURLReq) (*ShortenBatchURLRes, error) {
	return nil, status.Errorf(codes.Unimplemented, "method ShortenBatchURL not implemented")
}
func (UnimplementedURLShortenerServer) GetURL(context.Context, *GetURLReq) (*GetURLRes, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetURL not implemented")
}
func (UnimplementedURLShortenerServer) GetUserURLs(context.Context, *GetUsersURLsReq) (*GetUsersURLsRes, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetUserURLs not implemented")
}
func (UnimplementedURLShortenerServer) DeleteUserURLs(context.Context, *DeleteUserURLsReq) (*DeleteUserURLsRes, error) {
	return nil, status.Errorf(codes.Unimplemented, "method DeleteUserURLs not implemented")
}
func (UnimplementedURLShortenerServer) GetStats(context.Context, *GetStatsReq) (*GetStatsRes, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetStats not implemented")
}
func (UnimplementedURLShortenerServer) Ping(context.Context, *PingReq) (*PingRes, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Ping not implemented")
}
func (UnimplementedURLShortenerServer) mustEmbedUnimplementedURLShortenerServer() {}
func (UnimplementedURLShortenerServer) testEmbeddedByValue()                      {}

// UnsafeURLShortenerServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to URLShortenerServer will
// result in compilation errors.
type UnsafeURLShortenerServer interface {
	mustEmbedUnimplementedURLShortenerServer()
}

func RegisterURLShortenerServer(s grpc.ServiceRegistrar, srv URLShortenerServer) {
	// If the following call pancis, it indicates UnimplementedURLShortenerServer was
	// embedded by pointer and is nil.  This will cause panics if an
	// unimplemented method is ever invoked, so we test this at initialization
	// time to prevent it from happening at runtime later due to I/O.
	if t, ok := srv.(interface{ testEmbeddedByValue() }); ok {
		t.testEmbeddedByValue()
	}
	s.RegisterService(&URLShortener_ServiceDesc, srv)
}

func _URLShortener_ShortenURL_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(ShortenURLReq)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(URLShortenerServer).ShortenURL(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: URLShortener_ShortenURL_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(URLShortenerServer).ShortenURL(ctx, req.(*ShortenURLReq))
	}
	return interceptor(ctx, in, info, handler)
}

func _URLShortener_ShortenBatchURL_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(ShortenBatchURLReq)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(URLShortenerServer).ShortenBatchURL(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: URLShortener_ShortenBatchURL_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(URLShortenerServer).ShortenBatchURL(ctx, req.(*ShortenBatchURLReq))
	}
	return interceptor(ctx, in, info, handler)
}

func _URLShortener_GetURL_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(GetURLReq)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(URLShortenerServer).GetURL(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: URLShortener_GetURL_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(URLShortenerServer).GetURL(ctx, req.(*GetURLReq))
	}
	return interceptor(ctx, in, info, handler)
}

func _URLShortener_GetUserURLs_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(GetUsersURLsReq)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(URLShortenerServer).GetUserURLs(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: URLShortener_GetUserURLs_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(URLShortenerServer).GetUserURLs(ctx, req.(*GetUsersURLsReq))
	}
	return interceptor(ctx, in, info, handler)
}

func _URLShortener_DeleteUserURLs_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(DeleteUserURLsReq)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(URLShortenerServer).DeleteUserURLs(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: URLShortener_DeleteUserURLs_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(URLShortenerServer).DeleteUserURLs(ctx, req.(*DeleteUserURLsReq))
	}
	return interceptor(ctx, in, info, handler)
}

func _URLShortener_GetStats_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(GetStatsReq)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(URLShortenerServer).GetStats(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: URLShortener_GetStats_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(URLShortenerServer).GetStats(ctx, req.(*GetStatsReq))
	}
	return interceptor(ctx, in, info, handler)
}

func _URLShortener_Ping_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(PingReq)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(URLShortenerServer).Ping(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: URLShortener_Ping_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(URLShortenerServer).Ping(ctx, req.(*PingReq))
	}
	return interceptor(ctx, in, info, handler)
}

// URLShortener_ServiceDesc is the grpc.ServiceDesc for URLShortener service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var URLShortener_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "urlshortener.URLShortener",
	HandlerType: (*URLShortenerServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "ShortenURL",
			Handler:    _URLShortener_ShortenURL_Handler,
		},
		{
			MethodName: "ShortenBatchURL",
			Handler:    _URLShortener_ShortenBatchURL_Handler,
		},
		{
			MethodName: "GetURL",
			Handler:    _URLShortener_GetURL_Handler,
		},
		{
			MethodName: "GetUserURLs",
			Handler:    _URLShortener_GetUserURLs_Handler,
		},
		{
			MethodName: "DeleteUserURLs",
			Handler:    _URLShortener_DeleteUserURLs_Handler,
		},
		{
			MethodName: "GetStats",
			Handler:    _URLShortener_GetStats_Handler,
		},
		{
			MethodName: "Ping",
			Handler:    _URLShortener_Ping_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "internal/grpc_server/proto/urlshortener.proto",
}