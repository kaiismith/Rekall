// Hand-maintained gRPC client/server stubs for asr/proto/asr.proto.
// See asr.pb.go for the regeneration procedure.

package pb

import (
	context "context"

	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"

	emptypb "google.golang.org/protobuf/types/known/emptypb"
)

// Verify gRPC version compatibility at build time.
const _ = grpc.SupportPackageIsVersion7

const (
	ASR_StartSession_FullMethodName  = "/rekall.asr.v1.ASR/StartSession"
	ASR_EndSession_FullMethodName    = "/rekall.asr.v1.ASR/EndSession"
	ASR_GetSession_FullMethodName    = "/rekall.asr.v1.ASR/GetSession"
	ASR_Health_FullMethodName        = "/rekall.asr.v1.ASR/Health"
	ASR_ReloadModels_FullMethodName  = "/rekall.asr.v1.ASR/ReloadModels"
	ASR_StreamSession_FullMethodName = "/rekall.asr.v1.ASR/StreamSession"
)

// ─── Client interface ───────────────────────────────────────────────────────

// ASRClient is the client API for ASR service.
type ASRClient interface {
	StartSession(ctx context.Context, in *StartSessionRequest, opts ...grpc.CallOption) (*StartSessionResponse, error)
	EndSession(ctx context.Context, in *EndSessionRequest, opts ...grpc.CallOption) (*EndSessionResponse, error)
	GetSession(ctx context.Context, in *GetSessionRequest, opts ...grpc.CallOption) (*SessionInfo, error)
	Health(ctx context.Context, in *emptypb.Empty, opts ...grpc.CallOption) (*HealthResponse, error)
	ReloadModels(ctx context.Context, in *ReloadModelsRequest, opts ...grpc.CallOption) (*ReloadModelsResponse, error)
	StreamSession(ctx context.Context, opts ...grpc.CallOption) (ASR_StreamSessionClient, error)
}

type aSRClient struct {
	cc grpc.ClientConnInterface
}

func NewASRClient(cc grpc.ClientConnInterface) ASRClient {
	return &aSRClient{cc: cc}
}

func (c *aSRClient) StartSession(ctx context.Context, in *StartSessionRequest, opts ...grpc.CallOption) (*StartSessionResponse, error) {
	out := new(StartSessionResponse)
	err := c.cc.Invoke(ctx, ASR_StartSession_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *aSRClient) EndSession(ctx context.Context, in *EndSessionRequest, opts ...grpc.CallOption) (*EndSessionResponse, error) {
	out := new(EndSessionResponse)
	err := c.cc.Invoke(ctx, ASR_EndSession_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *aSRClient) GetSession(ctx context.Context, in *GetSessionRequest, opts ...grpc.CallOption) (*SessionInfo, error) {
	out := new(SessionInfo)
	err := c.cc.Invoke(ctx, ASR_GetSession_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *aSRClient) Health(ctx context.Context, in *emptypb.Empty, opts ...grpc.CallOption) (*HealthResponse, error) {
	out := new(HealthResponse)
	err := c.cc.Invoke(ctx, ASR_Health_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *aSRClient) ReloadModels(ctx context.Context, in *ReloadModelsRequest, opts ...grpc.CallOption) (*ReloadModelsResponse, error) {
	out := new(ReloadModelsResponse)
	err := c.cc.Invoke(ctx, ASR_ReloadModels_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// ─── Bidirectional streaming (StreamSession) ────────────────────────────────

type ASR_StreamSessionClient interface {
	Send(*StreamChunk) error
	Recv() (*TranscriptEvent, error)
	grpc.ClientStream
}

type aSRStreamSessionClient struct {
	grpc.ClientStream
}

func (x *aSRStreamSessionClient) Send(m *StreamChunk) error { return x.ClientStream.SendMsg(m) }
func (x *aSRStreamSessionClient) Recv() (*TranscriptEvent, error) {
	m := new(TranscriptEvent)
	if err := x.ClientStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

func (c *aSRClient) StreamSession(ctx context.Context, opts ...grpc.CallOption) (ASR_StreamSessionClient, error) {
	stream, err := c.cc.NewStream(ctx, &ASR_ServiceDesc.Streams[0], ASR_StreamSession_FullMethodName, opts...)
	if err != nil {
		return nil, err
	}
	return &aSRStreamSessionClient{ClientStream: stream}, nil
}

// ─── Server interface ───────────────────────────────────────────────────────

// ASRServer is the server API for ASR service.
type ASRServer interface {
	StartSession(context.Context, *StartSessionRequest) (*StartSessionResponse, error)
	EndSession(context.Context, *EndSessionRequest) (*EndSessionResponse, error)
	GetSession(context.Context, *GetSessionRequest) (*SessionInfo, error)
	Health(context.Context, *emptypb.Empty) (*HealthResponse, error)
	ReloadModels(context.Context, *ReloadModelsRequest) (*ReloadModelsResponse, error)
	StreamSession(ASR_StreamSessionServer) error
	mustEmbedUnimplementedASRServer()
}

// UnimplementedASRServer is provided so server impls don't have to satisfy
// every method; embedding it gives forward-compatibility when new RPCs land.
type UnimplementedASRServer struct{}

func (UnimplementedASRServer) StartSession(context.Context, *StartSessionRequest) (*StartSessionResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method StartSession not implemented")
}
func (UnimplementedASRServer) EndSession(context.Context, *EndSessionRequest) (*EndSessionResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method EndSession not implemented")
}
func (UnimplementedASRServer) GetSession(context.Context, *GetSessionRequest) (*SessionInfo, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetSession not implemented")
}
func (UnimplementedASRServer) Health(context.Context, *emptypb.Empty) (*HealthResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Health not implemented")
}
func (UnimplementedASRServer) ReloadModels(context.Context, *ReloadModelsRequest) (*ReloadModelsResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method ReloadModels not implemented")
}
func (UnimplementedASRServer) StreamSession(ASR_StreamSessionServer) error {
	return status.Errorf(codes.Unimplemented, "method StreamSession not implemented")
}
func (UnimplementedASRServer) mustEmbedUnimplementedASRServer() {}

// UnsafeASRServer is a marker interface that disables forward-compatibility
// embedding; not used by Rekall but emitted by protoc-gen-go-grpc for parity.
type UnsafeASRServer interface{ mustEmbedUnimplementedASRServer() }

func RegisterASRServer(s grpc.ServiceRegistrar, srv ASRServer) {
	s.RegisterService(&ASR_ServiceDesc, srv)
}

// ─── Server-side dispatch ───────────────────────────────────────────────────

func _ASR_StartSession_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(StartSessionRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ASRServer).StartSession(ctx, in)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: ASR_StartSession_FullMethodName}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ASRServer).StartSession(ctx, req.(*StartSessionRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _ASR_EndSession_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(EndSessionRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ASRServer).EndSession(ctx, in)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: ASR_EndSession_FullMethodName}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ASRServer).EndSession(ctx, req.(*EndSessionRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _ASR_GetSession_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(GetSessionRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ASRServer).GetSession(ctx, in)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: ASR_GetSession_FullMethodName}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ASRServer).GetSession(ctx, req.(*GetSessionRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _ASR_Health_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(emptypb.Empty)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ASRServer).Health(ctx, in)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: ASR_Health_FullMethodName}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ASRServer).Health(ctx, req.(*emptypb.Empty))
	}
	return interceptor(ctx, in, info, handler)
}

func _ASR_ReloadModels_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(ReloadModelsRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ASRServer).ReloadModels(ctx, in)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: ASR_ReloadModels_FullMethodName}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ASRServer).ReloadModels(ctx, req.(*ReloadModelsRequest))
	}
	return interceptor(ctx, in, info, handler)
}

type ASR_StreamSessionServer interface {
	Send(*TranscriptEvent) error
	Recv() (*StreamChunk, error)
	grpc.ServerStream
}

type aSRStreamSessionServer struct {
	grpc.ServerStream
}

func (x *aSRStreamSessionServer) Send(m *TranscriptEvent) error { return x.ServerStream.SendMsg(m) }
func (x *aSRStreamSessionServer) Recv() (*StreamChunk, error) {
	m := new(StreamChunk)
	if err := x.ServerStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

func _ASR_StreamSession_Handler(srv interface{}, stream grpc.ServerStream) error {
	return srv.(ASRServer).StreamSession(&aSRStreamSessionServer{ServerStream: stream})
}

// ASR_ServiceDesc is the descriptor used by grpc.Server.RegisterService.
var ASR_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "rekall.asr.v1.ASR",
	HandlerType: (*ASRServer)(nil),
	Methods: []grpc.MethodDesc{
		{MethodName: "StartSession", Handler: _ASR_StartSession_Handler},
		{MethodName: "EndSession", Handler: _ASR_EndSession_Handler},
		{MethodName: "GetSession", Handler: _ASR_GetSession_Handler},
		{MethodName: "Health", Handler: _ASR_Health_Handler},
		{MethodName: "ReloadModels", Handler: _ASR_ReloadModels_Handler},
	},
	Streams: []grpc.StreamDesc{
		{
			StreamName:    "StreamSession",
			Handler:       _ASR_StreamSession_Handler,
			ServerStreams: true,
			ClientStreams: true,
		},
	},
	Metadata: "asr.proto",
}
