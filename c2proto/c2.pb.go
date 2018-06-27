// Code generated by protoc-gen-go. DO NOT EDIT.
// source: c2.proto

package c2

import proto "github.com/golang/protobuf/proto"
import fmt "fmt"
import math "math"

import (
	context "golang.org/x/net/context"
	grpc "google.golang.org/grpc"
)

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

// This is a compile-time assertion to ensure that this generated file
// is compatible with the proto package it is being compiled against.
// A compilation error at this line likely means your copy of the
// proto package needs to be updated.
const _ = proto.ProtoPackageIsVersion2 // please upgrade the proto package

type C2Request_Command int32

const (
	C2Request_NEW_CLIENT          C2Request_Command = 0
	C2Request_REMOVE_CLIENT       C2Request_Command = 1
	C2Request_NEW_TOPIC_CLIENT    C2Request_Command = 2
	C2Request_REMOVE_TOPIC_CLIENT C2Request_Command = 3
	C2Request_RESET_CLIENT        C2Request_Command = 4
	C2Request_NEW_TOPIC           C2Request_Command = 5
	C2Request_REMOVE_TOPIC        C2Request_Command = 6
	C2Request_NEW_CLIENT_KEY      C2Request_Command = 7
)

var C2Request_Command_name = map[int32]string{
	0: "NEW_CLIENT",
	1: "REMOVE_CLIENT",
	2: "NEW_TOPIC_CLIENT",
	3: "REMOVE_TOPIC_CLIENT",
	4: "RESET_CLIENT",
	5: "NEW_TOPIC",
	6: "REMOVE_TOPIC",
	7: "NEW_CLIENT_KEY",
}
var C2Request_Command_value = map[string]int32{
	"NEW_CLIENT":          0,
	"REMOVE_CLIENT":       1,
	"NEW_TOPIC_CLIENT":    2,
	"REMOVE_TOPIC_CLIENT": 3,
	"RESET_CLIENT":        4,
	"NEW_TOPIC":           5,
	"REMOVE_TOPIC":        6,
	"NEW_CLIENT_KEY":      7,
}

func (x C2Request_Command) String() string {
	return proto.EnumName(C2Request_Command_name, int32(x))
}
func (C2Request_Command) EnumDescriptor() ([]byte, []int) {
	return fileDescriptor_c2_9089753e33bf3b38, []int{0, 0}
}

type C2Request struct {
	Command              C2Request_Command `protobuf:"varint,1,opt,name=command,proto3,enum=c2.C2Request_Command" json:"command,omitempty"`
	Id                   []byte            `protobuf:"bytes,2,opt,name=id,proto3" json:"id,omitempty"`
	Key                  []byte            `protobuf:"bytes,3,opt,name=key,proto3" json:"key,omitempty"`
	Topic                string            `protobuf:"bytes,4,opt,name=topic,proto3" json:"topic,omitempty"`
	XXX_NoUnkeyedLiteral struct{}          `json:"-"`
	XXX_unrecognized     []byte            `json:"-"`
	XXX_sizecache        int32             `json:"-"`
}

func (m *C2Request) Reset()         { *m = C2Request{} }
func (m *C2Request) String() string { return proto.CompactTextString(m) }
func (*C2Request) ProtoMessage()    {}
func (*C2Request) Descriptor() ([]byte, []int) {
	return fileDescriptor_c2_9089753e33bf3b38, []int{0}
}
func (m *C2Request) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_C2Request.Unmarshal(m, b)
}
func (m *C2Request) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_C2Request.Marshal(b, m, deterministic)
}
func (dst *C2Request) XXX_Merge(src proto.Message) {
	xxx_messageInfo_C2Request.Merge(dst, src)
}
func (m *C2Request) XXX_Size() int {
	return xxx_messageInfo_C2Request.Size(m)
}
func (m *C2Request) XXX_DiscardUnknown() {
	xxx_messageInfo_C2Request.DiscardUnknown(m)
}

var xxx_messageInfo_C2Request proto.InternalMessageInfo

func (m *C2Request) GetCommand() C2Request_Command {
	if m != nil {
		return m.Command
	}
	return C2Request_NEW_CLIENT
}

func (m *C2Request) GetId() []byte {
	if m != nil {
		return m.Id
	}
	return nil
}

func (m *C2Request) GetKey() []byte {
	if m != nil {
		return m.Key
	}
	return nil
}

func (m *C2Request) GetTopic() string {
	if m != nil {
		return m.Topic
	}
	return ""
}

type C2Response struct {
	Success              bool     `protobuf:"varint,1,opt,name=success,proto3" json:"success,omitempty"`
	Err                  string   `protobuf:"bytes,2,opt,name=err,proto3" json:"err,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *C2Response) Reset()         { *m = C2Response{} }
func (m *C2Response) String() string { return proto.CompactTextString(m) }
func (*C2Response) ProtoMessage()    {}
func (*C2Response) Descriptor() ([]byte, []int) {
	return fileDescriptor_c2_9089753e33bf3b38, []int{1}
}
func (m *C2Response) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_C2Response.Unmarshal(m, b)
}
func (m *C2Response) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_C2Response.Marshal(b, m, deterministic)
}
func (dst *C2Response) XXX_Merge(src proto.Message) {
	xxx_messageInfo_C2Response.Merge(dst, src)
}
func (m *C2Response) XXX_Size() int {
	return xxx_messageInfo_C2Response.Size(m)
}
func (m *C2Response) XXX_DiscardUnknown() {
	xxx_messageInfo_C2Response.DiscardUnknown(m)
}

var xxx_messageInfo_C2Response proto.InternalMessageInfo

func (m *C2Response) GetSuccess() bool {
	if m != nil {
		return m.Success
	}
	return false
}

func (m *C2Response) GetErr() string {
	if m != nil {
		return m.Err
	}
	return ""
}

func init() {
	proto.RegisterType((*C2Request)(nil), "c2.C2Request")
	proto.RegisterType((*C2Response)(nil), "c2.C2Response")
	proto.RegisterEnum("c2.C2Request_Command", C2Request_Command_name, C2Request_Command_value)
}

// Reference imports to suppress errors if they are not otherwise used.
var _ context.Context
var _ grpc.ClientConn

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
const _ = grpc.SupportPackageIsVersion4

// C2Client is the client API for C2 service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://godoc.org/google.golang.org/grpc#ClientConn.NewStream.
type C2Client interface {
	C2Command(ctx context.Context, in *C2Request, opts ...grpc.CallOption) (*C2Response, error)
}

type c2Client struct {
	cc *grpc.ClientConn
}

func NewC2Client(cc *grpc.ClientConn) C2Client {
	return &c2Client{cc}
}

func (c *c2Client) C2Command(ctx context.Context, in *C2Request, opts ...grpc.CallOption) (*C2Response, error) {
	out := new(C2Response)
	err := c.cc.Invoke(ctx, "/c2.C2/C2Command", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// C2Server is the server API for C2 service.
type C2Server interface {
	C2Command(context.Context, *C2Request) (*C2Response, error)
}

func RegisterC2Server(s *grpc.Server, srv C2Server) {
	s.RegisterService(&_C2_serviceDesc, srv)
}

func _C2_C2Command_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(C2Request)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(C2Server).C2Command(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/c2.C2/C2Command",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(C2Server).C2Command(ctx, req.(*C2Request))
	}
	return interceptor(ctx, in, info, handler)
}

var _C2_serviceDesc = grpc.ServiceDesc{
	ServiceName: "c2.C2",
	HandlerType: (*C2Server)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "C2Command",
			Handler:    _C2_C2Command_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "c2.proto",
}

func init() { proto.RegisterFile("c2.proto", fileDescriptor_c2_9089753e33bf3b38) }

var fileDescriptor_c2_9089753e33bf3b38 = []byte{
	// 283 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0x54, 0x51, 0x3d, 0x4f, 0xc3, 0x30,
	0x14, 0xac, 0xdd, 0x8f, 0x34, 0x4f, 0x6d, 0x64, 0x1e, 0x45, 0x44, 0x4c, 0x55, 0xa6, 0x0e, 0x28,
	0x48, 0x66, 0x61, 0xb7, 0x3c, 0x54, 0x40, 0x8b, 0x4c, 0x05, 0x62, 0xaa, 0xc0, 0xf1, 0x10, 0xa1,
	0x36, 0x21, 0x4e, 0x07, 0x7e, 0x0a, 0x2b, 0xbf, 0x14, 0x39, 0xc1, 0xa9, 0xba, 0xbd, 0xfb, 0xc8,
	0x29, 0x77, 0x86, 0xb1, 0xe6, 0x69, 0x59, 0x15, 0x75, 0x81, 0x54, 0xf3, 0xe4, 0x87, 0x42, 0x28,
	0xb8, 0x32, 0x5f, 0x07, 0x63, 0x6b, 0xbc, 0x81, 0x40, 0x17, 0xbb, 0xdd, 0xfb, 0x3e, 0x8b, 0xc9,
	0x9c, 0x2c, 0x22, 0x7e, 0x91, 0x6a, 0x9e, 0x76, 0x7a, 0x2a, 0x5a, 0x51, 0x79, 0x17, 0x46, 0x40,
	0xf3, 0x2c, 0xa6, 0x73, 0xb2, 0x98, 0x28, 0x9a, 0x67, 0xc8, 0xa0, 0xff, 0x69, 0xbe, 0xe3, 0x7e,
	0x43, 0xb8, 0x13, 0x67, 0x30, 0xac, 0x8b, 0x32, 0xd7, 0xf1, 0x60, 0x4e, 0x16, 0xa1, 0x6a, 0x41,
	0xf2, 0x4b, 0x20, 0x10, 0x5d, 0x06, 0xac, 0xe4, 0xeb, 0x56, 0x3c, 0x2c, 0xe5, 0x6a, 0xc3, 0x7a,
	0x78, 0x06, 0x53, 0x25, 0x1f, 0xd7, 0x2f, 0xd2, 0x53, 0x04, 0x67, 0xc0, 0x9c, 0x65, 0xb3, 0x7e,
	0x5a, 0x0a, 0xcf, 0x52, 0xbc, 0x84, 0xf3, 0x7f, 0xe3, 0x89, 0xd0, 0x47, 0x06, 0x13, 0x25, 0x9f,
	0xe5, 0xc6, 0x33, 0x03, 0x9c, 0x42, 0xd8, 0x05, 0xb0, 0x61, 0x6b, 0x38, 0x7e, 0xc9, 0x46, 0x88,
	0x10, 0x1d, 0x7f, 0x62, 0x7b, 0x2f, 0xdf, 0x58, 0x90, 0xdc, 0x01, 0xb8, 0xea, 0xb6, 0x2c, 0xf6,
	0xd6, 0x60, 0x0c, 0x81, 0x3d, 0x68, 0x6d, 0xac, 0x6d, 0xb6, 0x19, 0x2b, 0x0f, 0x5d, 0x69, 0x53,
	0x55, 0xcd, 0x0a, 0xa1, 0x72, 0x27, 0xe7, 0x40, 0x05, 0xc7, 0x6b, 0x37, 0xad, 0x6f, 0x39, 0x3d,
	0x59, 0xf2, 0x2a, 0xf2, 0xb0, 0x4d, 0x4f, 0x7a, 0x1f, 0xa3, 0xe6, 0x51, 0x6e, 0xff, 0x02, 0x00,
	0x00, 0xff, 0xff, 0x04, 0xad, 0x12, 0x88, 0xa0, 0x01, 0x00, 0x00,
}
