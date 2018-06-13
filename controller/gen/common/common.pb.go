// Code generated by protoc-gen-go. DO NOT EDIT.
// source: common.proto

/*
Package common is a generated protocol buffer package.

It is generated from these files:
	common.proto

It has these top-level messages:
	HttpMethod
	Scheme
	IPAddress
	IPv6
	TcpAddress
	Destination
	Eos
	TapEvent
*/
package common

import proto "github.com/golang/protobuf/proto"
import fmt "fmt"
import math "math"
import google_protobuf "github.com/golang/protobuf/ptypes/duration"

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

// This is a compile-time assertion to ensure that this generated file
// is compatible with the proto package it is being compiled against.
// A compilation error at this line likely means your copy of the
// proto package needs to be updated.
const _ = proto.ProtoPackageIsVersion2 // please upgrade the proto package

type Protocol int32

const (
	Protocol_HTTP Protocol = 0
	Protocol_TCP  Protocol = 1
)

var Protocol_name = map[int32]string{
	0: "HTTP",
	1: "TCP",
}
var Protocol_value = map[string]int32{
	"HTTP": 0,
	"TCP":  1,
}

func (x Protocol) String() string {
	return proto.EnumName(Protocol_name, int32(x))
}
func (Protocol) EnumDescriptor() ([]byte, []int) { return fileDescriptor0, []int{0} }

type HttpMethod_Registered int32

const (
	HttpMethod_GET     HttpMethod_Registered = 0
	HttpMethod_POST    HttpMethod_Registered = 1
	HttpMethod_PUT     HttpMethod_Registered = 2
	HttpMethod_DELETE  HttpMethod_Registered = 3
	HttpMethod_PATCH   HttpMethod_Registered = 4
	HttpMethod_OPTIONS HttpMethod_Registered = 5
	HttpMethod_CONNECT HttpMethod_Registered = 6
	HttpMethod_HEAD    HttpMethod_Registered = 7
	HttpMethod_TRACE   HttpMethod_Registered = 8
)

var HttpMethod_Registered_name = map[int32]string{
	0: "GET",
	1: "POST",
	2: "PUT",
	3: "DELETE",
	4: "PATCH",
	5: "OPTIONS",
	6: "CONNECT",
	7: "HEAD",
	8: "TRACE",
}
var HttpMethod_Registered_value = map[string]int32{
	"GET":     0,
	"POST":    1,
	"PUT":     2,
	"DELETE":  3,
	"PATCH":   4,
	"OPTIONS": 5,
	"CONNECT": 6,
	"HEAD":    7,
	"TRACE":   8,
}

func (x HttpMethod_Registered) String() string {
	return proto.EnumName(HttpMethod_Registered_name, int32(x))
}
func (HttpMethod_Registered) EnumDescriptor() ([]byte, []int) { return fileDescriptor0, []int{0, 0} }

type Scheme_Registered int32

const (
	Scheme_HTTP  Scheme_Registered = 0
	Scheme_HTTPS Scheme_Registered = 1
)

var Scheme_Registered_name = map[int32]string{
	0: "HTTP",
	1: "HTTPS",
}
var Scheme_Registered_value = map[string]int32{
	"HTTP":  0,
	"HTTPS": 1,
}

func (x Scheme_Registered) String() string {
	return proto.EnumName(Scheme_Registered_name, int32(x))
}
func (Scheme_Registered) EnumDescriptor() ([]byte, []int) { return fileDescriptor0, []int{1, 0} }

type HttpMethod struct {
	// Types that are valid to be assigned to Type:
	//	*HttpMethod_Registered_
	//	*HttpMethod_Unregistered
	Type isHttpMethod_Type `protobuf_oneof:"type"`
}

func (m *HttpMethod) Reset()                    { *m = HttpMethod{} }
func (m *HttpMethod) String() string            { return proto.CompactTextString(m) }
func (*HttpMethod) ProtoMessage()               {}
func (*HttpMethod) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{0} }

type isHttpMethod_Type interface{ isHttpMethod_Type() }

type HttpMethod_Registered_ struct {
	Registered HttpMethod_Registered `protobuf:"varint,1,opt,name=registered,enum=conduit.common.HttpMethod_Registered,oneof"`
}
type HttpMethod_Unregistered struct {
	Unregistered string `protobuf:"bytes,2,opt,name=unregistered,oneof"`
}

func (*HttpMethod_Registered_) isHttpMethod_Type()  {}
func (*HttpMethod_Unregistered) isHttpMethod_Type() {}

func (m *HttpMethod) GetType() isHttpMethod_Type {
	if m != nil {
		return m.Type
	}
	return nil
}

func (m *HttpMethod) GetRegistered() HttpMethod_Registered {
	if x, ok := m.GetType().(*HttpMethod_Registered_); ok {
		return x.Registered
	}
	return HttpMethod_GET
}

func (m *HttpMethod) GetUnregistered() string {
	if x, ok := m.GetType().(*HttpMethod_Unregistered); ok {
		return x.Unregistered
	}
	return ""
}

// XXX_OneofFuncs is for the internal use of the proto package.
func (*HttpMethod) XXX_OneofFuncs() (func(msg proto.Message, b *proto.Buffer) error, func(msg proto.Message, tag, wire int, b *proto.Buffer) (bool, error), func(msg proto.Message) (n int), []interface{}) {
	return _HttpMethod_OneofMarshaler, _HttpMethod_OneofUnmarshaler, _HttpMethod_OneofSizer, []interface{}{
		(*HttpMethod_Registered_)(nil),
		(*HttpMethod_Unregistered)(nil),
	}
}

func _HttpMethod_OneofMarshaler(msg proto.Message, b *proto.Buffer) error {
	m := msg.(*HttpMethod)
	// type
	switch x := m.Type.(type) {
	case *HttpMethod_Registered_:
		b.EncodeVarint(1<<3 | proto.WireVarint)
		b.EncodeVarint(uint64(x.Registered))
	case *HttpMethod_Unregistered:
		b.EncodeVarint(2<<3 | proto.WireBytes)
		b.EncodeStringBytes(x.Unregistered)
	case nil:
	default:
		return fmt.Errorf("HttpMethod.Type has unexpected type %T", x)
	}
	return nil
}

func _HttpMethod_OneofUnmarshaler(msg proto.Message, tag, wire int, b *proto.Buffer) (bool, error) {
	m := msg.(*HttpMethod)
	switch tag {
	case 1: // type.registered
		if wire != proto.WireVarint {
			return true, proto.ErrInternalBadWireType
		}
		x, err := b.DecodeVarint()
		m.Type = &HttpMethod_Registered_{HttpMethod_Registered(x)}
		return true, err
	case 2: // type.unregistered
		if wire != proto.WireBytes {
			return true, proto.ErrInternalBadWireType
		}
		x, err := b.DecodeStringBytes()
		m.Type = &HttpMethod_Unregistered{x}
		return true, err
	default:
		return false, nil
	}
}

func _HttpMethod_OneofSizer(msg proto.Message) (n int) {
	m := msg.(*HttpMethod)
	// type
	switch x := m.Type.(type) {
	case *HttpMethod_Registered_:
		n += proto.SizeVarint(1<<3 | proto.WireVarint)
		n += proto.SizeVarint(uint64(x.Registered))
	case *HttpMethod_Unregistered:
		n += proto.SizeVarint(2<<3 | proto.WireBytes)
		n += proto.SizeVarint(uint64(len(x.Unregistered)))
		n += len(x.Unregistered)
	case nil:
	default:
		panic(fmt.Sprintf("proto: unexpected type %T in oneof", x))
	}
	return n
}

type Scheme struct {
	// Types that are valid to be assigned to Type:
	//	*Scheme_Registered_
	//	*Scheme_Unregistered
	Type isScheme_Type `protobuf_oneof:"type"`
}

func (m *Scheme) Reset()                    { *m = Scheme{} }
func (m *Scheme) String() string            { return proto.CompactTextString(m) }
func (*Scheme) ProtoMessage()               {}
func (*Scheme) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{1} }

type isScheme_Type interface{ isScheme_Type() }

type Scheme_Registered_ struct {
	Registered Scheme_Registered `protobuf:"varint,1,opt,name=registered,enum=conduit.common.Scheme_Registered,oneof"`
}
type Scheme_Unregistered struct {
	Unregistered string `protobuf:"bytes,2,opt,name=unregistered,oneof"`
}

func (*Scheme_Registered_) isScheme_Type()  {}
func (*Scheme_Unregistered) isScheme_Type() {}

func (m *Scheme) GetType() isScheme_Type {
	if m != nil {
		return m.Type
	}
	return nil
}

func (m *Scheme) GetRegistered() Scheme_Registered {
	if x, ok := m.GetType().(*Scheme_Registered_); ok {
		return x.Registered
	}
	return Scheme_HTTP
}

func (m *Scheme) GetUnregistered() string {
	if x, ok := m.GetType().(*Scheme_Unregistered); ok {
		return x.Unregistered
	}
	return ""
}

// XXX_OneofFuncs is for the internal use of the proto package.
func (*Scheme) XXX_OneofFuncs() (func(msg proto.Message, b *proto.Buffer) error, func(msg proto.Message, tag, wire int, b *proto.Buffer) (bool, error), func(msg proto.Message) (n int), []interface{}) {
	return _Scheme_OneofMarshaler, _Scheme_OneofUnmarshaler, _Scheme_OneofSizer, []interface{}{
		(*Scheme_Registered_)(nil),
		(*Scheme_Unregistered)(nil),
	}
}

func _Scheme_OneofMarshaler(msg proto.Message, b *proto.Buffer) error {
	m := msg.(*Scheme)
	// type
	switch x := m.Type.(type) {
	case *Scheme_Registered_:
		b.EncodeVarint(1<<3 | proto.WireVarint)
		b.EncodeVarint(uint64(x.Registered))
	case *Scheme_Unregistered:
		b.EncodeVarint(2<<3 | proto.WireBytes)
		b.EncodeStringBytes(x.Unregistered)
	case nil:
	default:
		return fmt.Errorf("Scheme.Type has unexpected type %T", x)
	}
	return nil
}

func _Scheme_OneofUnmarshaler(msg proto.Message, tag, wire int, b *proto.Buffer) (bool, error) {
	m := msg.(*Scheme)
	switch tag {
	case 1: // type.registered
		if wire != proto.WireVarint {
			return true, proto.ErrInternalBadWireType
		}
		x, err := b.DecodeVarint()
		m.Type = &Scheme_Registered_{Scheme_Registered(x)}
		return true, err
	case 2: // type.unregistered
		if wire != proto.WireBytes {
			return true, proto.ErrInternalBadWireType
		}
		x, err := b.DecodeStringBytes()
		m.Type = &Scheme_Unregistered{x}
		return true, err
	default:
		return false, nil
	}
}

func _Scheme_OneofSizer(msg proto.Message) (n int) {
	m := msg.(*Scheme)
	// type
	switch x := m.Type.(type) {
	case *Scheme_Registered_:
		n += proto.SizeVarint(1<<3 | proto.WireVarint)
		n += proto.SizeVarint(uint64(x.Registered))
	case *Scheme_Unregistered:
		n += proto.SizeVarint(2<<3 | proto.WireBytes)
		n += proto.SizeVarint(uint64(len(x.Unregistered)))
		n += len(x.Unregistered)
	case nil:
	default:
		panic(fmt.Sprintf("proto: unexpected type %T in oneof", x))
	}
	return n
}

type IPAddress struct {
	// Types that are valid to be assigned to Ip:
	//	*IPAddress_Ipv4
	//	*IPAddress_Ipv6
	Ip isIPAddress_Ip `protobuf_oneof:"ip"`
}

func (m *IPAddress) Reset()                    { *m = IPAddress{} }
func (m *IPAddress) String() string            { return proto.CompactTextString(m) }
func (*IPAddress) ProtoMessage()               {}
func (*IPAddress) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{2} }

type isIPAddress_Ip interface{ isIPAddress_Ip() }

type IPAddress_Ipv4 struct {
	Ipv4 uint32 `protobuf:"fixed32,1,opt,name=ipv4,oneof"`
}
type IPAddress_Ipv6 struct {
	Ipv6 *IPv6 `protobuf:"bytes,2,opt,name=ipv6,oneof"`
}

func (*IPAddress_Ipv4) isIPAddress_Ip() {}
func (*IPAddress_Ipv6) isIPAddress_Ip() {}

func (m *IPAddress) GetIp() isIPAddress_Ip {
	if m != nil {
		return m.Ip
	}
	return nil
}

func (m *IPAddress) GetIpv4() uint32 {
	if x, ok := m.GetIp().(*IPAddress_Ipv4); ok {
		return x.Ipv4
	}
	return 0
}

func (m *IPAddress) GetIpv6() *IPv6 {
	if x, ok := m.GetIp().(*IPAddress_Ipv6); ok {
		return x.Ipv6
	}
	return nil
}

// XXX_OneofFuncs is for the internal use of the proto package.
func (*IPAddress) XXX_OneofFuncs() (func(msg proto.Message, b *proto.Buffer) error, func(msg proto.Message, tag, wire int, b *proto.Buffer) (bool, error), func(msg proto.Message) (n int), []interface{}) {
	return _IPAddress_OneofMarshaler, _IPAddress_OneofUnmarshaler, _IPAddress_OneofSizer, []interface{}{
		(*IPAddress_Ipv4)(nil),
		(*IPAddress_Ipv6)(nil),
	}
}

func _IPAddress_OneofMarshaler(msg proto.Message, b *proto.Buffer) error {
	m := msg.(*IPAddress)
	// ip
	switch x := m.Ip.(type) {
	case *IPAddress_Ipv4:
		b.EncodeVarint(1<<3 | proto.WireFixed32)
		b.EncodeFixed32(uint64(x.Ipv4))
	case *IPAddress_Ipv6:
		b.EncodeVarint(2<<3 | proto.WireBytes)
		if err := b.EncodeMessage(x.Ipv6); err != nil {
			return err
		}
	case nil:
	default:
		return fmt.Errorf("IPAddress.Ip has unexpected type %T", x)
	}
	return nil
}

func _IPAddress_OneofUnmarshaler(msg proto.Message, tag, wire int, b *proto.Buffer) (bool, error) {
	m := msg.(*IPAddress)
	switch tag {
	case 1: // ip.ipv4
		if wire != proto.WireFixed32 {
			return true, proto.ErrInternalBadWireType
		}
		x, err := b.DecodeFixed32()
		m.Ip = &IPAddress_Ipv4{uint32(x)}
		return true, err
	case 2: // ip.ipv6
		if wire != proto.WireBytes {
			return true, proto.ErrInternalBadWireType
		}
		msg := new(IPv6)
		err := b.DecodeMessage(msg)
		m.Ip = &IPAddress_Ipv6{msg}
		return true, err
	default:
		return false, nil
	}
}

func _IPAddress_OneofSizer(msg proto.Message) (n int) {
	m := msg.(*IPAddress)
	// ip
	switch x := m.Ip.(type) {
	case *IPAddress_Ipv4:
		n += proto.SizeVarint(1<<3 | proto.WireFixed32)
		n += 4
	case *IPAddress_Ipv6:
		s := proto.Size(x.Ipv6)
		n += proto.SizeVarint(2<<3 | proto.WireBytes)
		n += proto.SizeVarint(uint64(s))
		n += s
	case nil:
	default:
		panic(fmt.Sprintf("proto: unexpected type %T in oneof", x))
	}
	return n
}

type IPv6 struct {
	First uint64 `protobuf:"fixed64,1,opt,name=first" json:"first,omitempty"`
	Last  uint64 `protobuf:"fixed64,2,opt,name=last" json:"last,omitempty"`
}

func (m *IPv6) Reset()                    { *m = IPv6{} }
func (m *IPv6) String() string            { return proto.CompactTextString(m) }
func (*IPv6) ProtoMessage()               {}
func (*IPv6) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{3} }

func (m *IPv6) GetFirst() uint64 {
	if m != nil {
		return m.First
	}
	return 0
}

func (m *IPv6) GetLast() uint64 {
	if m != nil {
		return m.Last
	}
	return 0
}

type TcpAddress struct {
	Ip   *IPAddress `protobuf:"bytes,1,opt,name=ip" json:"ip,omitempty"`
	Port uint32     `protobuf:"varint,2,opt,name=port" json:"port,omitempty"`
}

func (m *TcpAddress) Reset()                    { *m = TcpAddress{} }
func (m *TcpAddress) String() string            { return proto.CompactTextString(m) }
func (*TcpAddress) ProtoMessage()               {}
func (*TcpAddress) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{4} }

func (m *TcpAddress) GetIp() *IPAddress {
	if m != nil {
		return m.Ip
	}
	return nil
}

func (m *TcpAddress) GetPort() uint32 {
	if m != nil {
		return m.Port
	}
	return 0
}

type Destination struct {
	Scheme string `protobuf:"bytes,1,opt,name=scheme" json:"scheme,omitempty"`
	Path   string `protobuf:"bytes,2,opt,name=path" json:"path,omitempty"`
}

func (m *Destination) Reset()                    { *m = Destination{} }
func (m *Destination) String() string            { return proto.CompactTextString(m) }
func (*Destination) ProtoMessage()               {}
func (*Destination) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{5} }

func (m *Destination) GetScheme() string {
	if m != nil {
		return m.Scheme
	}
	return ""
}

func (m *Destination) GetPath() string {
	if m != nil {
		return m.Path
	}
	return ""
}

type Eos struct {
	// Types that are valid to be assigned to End:
	//	*Eos_GrpcStatusCode
	//	*Eos_ResetErrorCode
	End isEos_End `protobuf_oneof:"end"`
}

func (m *Eos) Reset()                    { *m = Eos{} }
func (m *Eos) String() string            { return proto.CompactTextString(m) }
func (*Eos) ProtoMessage()               {}
func (*Eos) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{6} }

type isEos_End interface{ isEos_End() }

type Eos_GrpcStatusCode struct {
	GrpcStatusCode uint32 `protobuf:"varint,1,opt,name=grpc_status_code,json=grpcStatusCode,oneof"`
}
type Eos_ResetErrorCode struct {
	ResetErrorCode uint32 `protobuf:"varint,2,opt,name=reset_error_code,json=resetErrorCode,oneof"`
}

func (*Eos_GrpcStatusCode) isEos_End() {}
func (*Eos_ResetErrorCode) isEos_End() {}

func (m *Eos) GetEnd() isEos_End {
	if m != nil {
		return m.End
	}
	return nil
}

func (m *Eos) GetGrpcStatusCode() uint32 {
	if x, ok := m.GetEnd().(*Eos_GrpcStatusCode); ok {
		return x.GrpcStatusCode
	}
	return 0
}

func (m *Eos) GetResetErrorCode() uint32 {
	if x, ok := m.GetEnd().(*Eos_ResetErrorCode); ok {
		return x.ResetErrorCode
	}
	return 0
}

// XXX_OneofFuncs is for the internal use of the proto package.
func (*Eos) XXX_OneofFuncs() (func(msg proto.Message, b *proto.Buffer) error, func(msg proto.Message, tag, wire int, b *proto.Buffer) (bool, error), func(msg proto.Message) (n int), []interface{}) {
	return _Eos_OneofMarshaler, _Eos_OneofUnmarshaler, _Eos_OneofSizer, []interface{}{
		(*Eos_GrpcStatusCode)(nil),
		(*Eos_ResetErrorCode)(nil),
	}
}

func _Eos_OneofMarshaler(msg proto.Message, b *proto.Buffer) error {
	m := msg.(*Eos)
	// end
	switch x := m.End.(type) {
	case *Eos_GrpcStatusCode:
		b.EncodeVarint(1<<3 | proto.WireVarint)
		b.EncodeVarint(uint64(x.GrpcStatusCode))
	case *Eos_ResetErrorCode:
		b.EncodeVarint(2<<3 | proto.WireVarint)
		b.EncodeVarint(uint64(x.ResetErrorCode))
	case nil:
	default:
		return fmt.Errorf("Eos.End has unexpected type %T", x)
	}
	return nil
}

func _Eos_OneofUnmarshaler(msg proto.Message, tag, wire int, b *proto.Buffer) (bool, error) {
	m := msg.(*Eos)
	switch tag {
	case 1: // end.grpc_status_code
		if wire != proto.WireVarint {
			return true, proto.ErrInternalBadWireType
		}
		x, err := b.DecodeVarint()
		m.End = &Eos_GrpcStatusCode{uint32(x)}
		return true, err
	case 2: // end.reset_error_code
		if wire != proto.WireVarint {
			return true, proto.ErrInternalBadWireType
		}
		x, err := b.DecodeVarint()
		m.End = &Eos_ResetErrorCode{uint32(x)}
		return true, err
	default:
		return false, nil
	}
}

func _Eos_OneofSizer(msg proto.Message) (n int) {
	m := msg.(*Eos)
	// end
	switch x := m.End.(type) {
	case *Eos_GrpcStatusCode:
		n += proto.SizeVarint(1<<3 | proto.WireVarint)
		n += proto.SizeVarint(uint64(x.GrpcStatusCode))
	case *Eos_ResetErrorCode:
		n += proto.SizeVarint(2<<3 | proto.WireVarint)
		n += proto.SizeVarint(uint64(x.ResetErrorCode))
	case nil:
	default:
		panic(fmt.Sprintf("proto: unexpected type %T in oneof", x))
	}
	return n
}

type TapEvent struct {
	Source          *TcpAddress            `protobuf:"bytes,1,opt,name=source" json:"source,omitempty"`
	Destination     *TcpAddress            `protobuf:"bytes,2,opt,name=destination" json:"destination,omitempty"`
	DestinationMeta *TapEvent_EndpointMeta `protobuf:"bytes,4,opt,name=destination_meta,json=destinationMeta" json:"destination_meta,omitempty"`
	// Types that are valid to be assigned to Event:
	//	*TapEvent_Http_
	Event isTapEvent_Event `protobuf_oneof:"event"`
}

func (m *TapEvent) Reset()                    { *m = TapEvent{} }
func (m *TapEvent) String() string            { return proto.CompactTextString(m) }
func (*TapEvent) ProtoMessage()               {}
func (*TapEvent) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{7} }

type isTapEvent_Event interface{ isTapEvent_Event() }

type TapEvent_Http_ struct {
	Http *TapEvent_Http `protobuf:"bytes,3,opt,name=http,oneof"`
}

func (*TapEvent_Http_) isTapEvent_Event() {}

func (m *TapEvent) GetEvent() isTapEvent_Event {
	if m != nil {
		return m.Event
	}
	return nil
}

func (m *TapEvent) GetSource() *TcpAddress {
	if m != nil {
		return m.Source
	}
	return nil
}

func (m *TapEvent) GetDestination() *TcpAddress {
	if m != nil {
		return m.Destination
	}
	return nil
}

func (m *TapEvent) GetDestinationMeta() *TapEvent_EndpointMeta {
	if m != nil {
		return m.DestinationMeta
	}
	return nil
}

func (m *TapEvent) GetHttp() *TapEvent_Http {
	if x, ok := m.GetEvent().(*TapEvent_Http_); ok {
		return x.Http
	}
	return nil
}

// XXX_OneofFuncs is for the internal use of the proto package.
func (*TapEvent) XXX_OneofFuncs() (func(msg proto.Message, b *proto.Buffer) error, func(msg proto.Message, tag, wire int, b *proto.Buffer) (bool, error), func(msg proto.Message) (n int), []interface{}) {
	return _TapEvent_OneofMarshaler, _TapEvent_OneofUnmarshaler, _TapEvent_OneofSizer, []interface{}{
		(*TapEvent_Http_)(nil),
	}
}

func _TapEvent_OneofMarshaler(msg proto.Message, b *proto.Buffer) error {
	m := msg.(*TapEvent)
	// event
	switch x := m.Event.(type) {
	case *TapEvent_Http_:
		b.EncodeVarint(3<<3 | proto.WireBytes)
		if err := b.EncodeMessage(x.Http); err != nil {
			return err
		}
	case nil:
	default:
		return fmt.Errorf("TapEvent.Event has unexpected type %T", x)
	}
	return nil
}

func _TapEvent_OneofUnmarshaler(msg proto.Message, tag, wire int, b *proto.Buffer) (bool, error) {
	m := msg.(*TapEvent)
	switch tag {
	case 3: // event.http
		if wire != proto.WireBytes {
			return true, proto.ErrInternalBadWireType
		}
		msg := new(TapEvent_Http)
		err := b.DecodeMessage(msg)
		m.Event = &TapEvent_Http_{msg}
		return true, err
	default:
		return false, nil
	}
}

func _TapEvent_OneofSizer(msg proto.Message) (n int) {
	m := msg.(*TapEvent)
	// event
	switch x := m.Event.(type) {
	case *TapEvent_Http_:
		s := proto.Size(x.Http)
		n += proto.SizeVarint(3<<3 | proto.WireBytes)
		n += proto.SizeVarint(uint64(s))
		n += s
	case nil:
	default:
		panic(fmt.Sprintf("proto: unexpected type %T in oneof", x))
	}
	return n
}

type TapEvent_EndpointMeta struct {
	Labels map[string]string `protobuf:"bytes,1,rep,name=labels" json:"labels,omitempty" protobuf_key:"bytes,1,opt,name=key" protobuf_val:"bytes,2,opt,name=value"`
}

func (m *TapEvent_EndpointMeta) Reset()                    { *m = TapEvent_EndpointMeta{} }
func (m *TapEvent_EndpointMeta) String() string            { return proto.CompactTextString(m) }
func (*TapEvent_EndpointMeta) ProtoMessage()               {}
func (*TapEvent_EndpointMeta) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{7, 0} }

func (m *TapEvent_EndpointMeta) GetLabels() map[string]string {
	if m != nil {
		return m.Labels
	}
	return nil
}

type TapEvent_Http struct {
	// Types that are valid to be assigned to Event:
	//	*TapEvent_Http_RequestInit_
	//	*TapEvent_Http_ResponseInit_
	//	*TapEvent_Http_ResponseEnd_
	Event isTapEvent_Http_Event `protobuf_oneof:"event"`
}

func (m *TapEvent_Http) Reset()                    { *m = TapEvent_Http{} }
func (m *TapEvent_Http) String() string            { return proto.CompactTextString(m) }
func (*TapEvent_Http) ProtoMessage()               {}
func (*TapEvent_Http) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{7, 1} }

type isTapEvent_Http_Event interface{ isTapEvent_Http_Event() }

type TapEvent_Http_RequestInit_ struct {
	RequestInit *TapEvent_Http_RequestInit `protobuf:"bytes,1,opt,name=request_init,json=requestInit,oneof"`
}
type TapEvent_Http_ResponseInit_ struct {
	ResponseInit *TapEvent_Http_ResponseInit `protobuf:"bytes,2,opt,name=response_init,json=responseInit,oneof"`
}
type TapEvent_Http_ResponseEnd_ struct {
	ResponseEnd *TapEvent_Http_ResponseEnd `protobuf:"bytes,3,opt,name=response_end,json=responseEnd,oneof"`
}

func (*TapEvent_Http_RequestInit_) isTapEvent_Http_Event()  {}
func (*TapEvent_Http_ResponseInit_) isTapEvent_Http_Event() {}
func (*TapEvent_Http_ResponseEnd_) isTapEvent_Http_Event()  {}

func (m *TapEvent_Http) GetEvent() isTapEvent_Http_Event {
	if m != nil {
		return m.Event
	}
	return nil
}

func (m *TapEvent_Http) GetRequestInit() *TapEvent_Http_RequestInit {
	if x, ok := m.GetEvent().(*TapEvent_Http_RequestInit_); ok {
		return x.RequestInit
	}
	return nil
}

func (m *TapEvent_Http) GetResponseInit() *TapEvent_Http_ResponseInit {
	if x, ok := m.GetEvent().(*TapEvent_Http_ResponseInit_); ok {
		return x.ResponseInit
	}
	return nil
}

func (m *TapEvent_Http) GetResponseEnd() *TapEvent_Http_ResponseEnd {
	if x, ok := m.GetEvent().(*TapEvent_Http_ResponseEnd_); ok {
		return x.ResponseEnd
	}
	return nil
}

// XXX_OneofFuncs is for the internal use of the proto package.
func (*TapEvent_Http) XXX_OneofFuncs() (func(msg proto.Message, b *proto.Buffer) error, func(msg proto.Message, tag, wire int, b *proto.Buffer) (bool, error), func(msg proto.Message) (n int), []interface{}) {
	return _TapEvent_Http_OneofMarshaler, _TapEvent_Http_OneofUnmarshaler, _TapEvent_Http_OneofSizer, []interface{}{
		(*TapEvent_Http_RequestInit_)(nil),
		(*TapEvent_Http_ResponseInit_)(nil),
		(*TapEvent_Http_ResponseEnd_)(nil),
	}
}

func _TapEvent_Http_OneofMarshaler(msg proto.Message, b *proto.Buffer) error {
	m := msg.(*TapEvent_Http)
	// event
	switch x := m.Event.(type) {
	case *TapEvent_Http_RequestInit_:
		b.EncodeVarint(1<<3 | proto.WireBytes)
		if err := b.EncodeMessage(x.RequestInit); err != nil {
			return err
		}
	case *TapEvent_Http_ResponseInit_:
		b.EncodeVarint(2<<3 | proto.WireBytes)
		if err := b.EncodeMessage(x.ResponseInit); err != nil {
			return err
		}
	case *TapEvent_Http_ResponseEnd_:
		b.EncodeVarint(3<<3 | proto.WireBytes)
		if err := b.EncodeMessage(x.ResponseEnd); err != nil {
			return err
		}
	case nil:
	default:
		return fmt.Errorf("TapEvent_Http.Event has unexpected type %T", x)
	}
	return nil
}

func _TapEvent_Http_OneofUnmarshaler(msg proto.Message, tag, wire int, b *proto.Buffer) (bool, error) {
	m := msg.(*TapEvent_Http)
	switch tag {
	case 1: // event.request_init
		if wire != proto.WireBytes {
			return true, proto.ErrInternalBadWireType
		}
		msg := new(TapEvent_Http_RequestInit)
		err := b.DecodeMessage(msg)
		m.Event = &TapEvent_Http_RequestInit_{msg}
		return true, err
	case 2: // event.response_init
		if wire != proto.WireBytes {
			return true, proto.ErrInternalBadWireType
		}
		msg := new(TapEvent_Http_ResponseInit)
		err := b.DecodeMessage(msg)
		m.Event = &TapEvent_Http_ResponseInit_{msg}
		return true, err
	case 3: // event.response_end
		if wire != proto.WireBytes {
			return true, proto.ErrInternalBadWireType
		}
		msg := new(TapEvent_Http_ResponseEnd)
		err := b.DecodeMessage(msg)
		m.Event = &TapEvent_Http_ResponseEnd_{msg}
		return true, err
	default:
		return false, nil
	}
}

func _TapEvent_Http_OneofSizer(msg proto.Message) (n int) {
	m := msg.(*TapEvent_Http)
	// event
	switch x := m.Event.(type) {
	case *TapEvent_Http_RequestInit_:
		s := proto.Size(x.RequestInit)
		n += proto.SizeVarint(1<<3 | proto.WireBytes)
		n += proto.SizeVarint(uint64(s))
		n += s
	case *TapEvent_Http_ResponseInit_:
		s := proto.Size(x.ResponseInit)
		n += proto.SizeVarint(2<<3 | proto.WireBytes)
		n += proto.SizeVarint(uint64(s))
		n += s
	case *TapEvent_Http_ResponseEnd_:
		s := proto.Size(x.ResponseEnd)
		n += proto.SizeVarint(3<<3 | proto.WireBytes)
		n += proto.SizeVarint(uint64(s))
		n += s
	case nil:
	default:
		panic(fmt.Sprintf("proto: unexpected type %T in oneof", x))
	}
	return n
}

type TapEvent_Http_StreamId struct {
	// A randomized base (stable across a process's runtime)
	Base uint32 `protobuf:"varint,1,opt,name=base" json:"base,omitempty"`
	// A stream id unique within the lifetime of `base`.
	Stream uint64 `protobuf:"varint,2,opt,name=stream" json:"stream,omitempty"`
}

func (m *TapEvent_Http_StreamId) Reset()                    { *m = TapEvent_Http_StreamId{} }
func (m *TapEvent_Http_StreamId) String() string            { return proto.CompactTextString(m) }
func (*TapEvent_Http_StreamId) ProtoMessage()               {}
func (*TapEvent_Http_StreamId) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{7, 1, 0} }

func (m *TapEvent_Http_StreamId) GetBase() uint32 {
	if m != nil {
		return m.Base
	}
	return 0
}

func (m *TapEvent_Http_StreamId) GetStream() uint64 {
	if m != nil {
		return m.Stream
	}
	return 0
}

type TapEvent_Http_RequestInit struct {
	Id        *TapEvent_Http_StreamId `protobuf:"bytes,1,opt,name=id" json:"id,omitempty"`
	Method    *HttpMethod             `protobuf:"bytes,2,opt,name=method" json:"method,omitempty"`
	Scheme    *Scheme                 `protobuf:"bytes,3,opt,name=scheme" json:"scheme,omitempty"`
	Authority string                  `protobuf:"bytes,4,opt,name=authority" json:"authority,omitempty"`
	Path      string                  `protobuf:"bytes,5,opt,name=path" json:"path,omitempty"`
}

func (m *TapEvent_Http_RequestInit) Reset()                    { *m = TapEvent_Http_RequestInit{} }
func (m *TapEvent_Http_RequestInit) String() string            { return proto.CompactTextString(m) }
func (*TapEvent_Http_RequestInit) ProtoMessage()               {}
func (*TapEvent_Http_RequestInit) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{7, 1, 1} }

func (m *TapEvent_Http_RequestInit) GetId() *TapEvent_Http_StreamId {
	if m != nil {
		return m.Id
	}
	return nil
}

func (m *TapEvent_Http_RequestInit) GetMethod() *HttpMethod {
	if m != nil {
		return m.Method
	}
	return nil
}

func (m *TapEvent_Http_RequestInit) GetScheme() *Scheme {
	if m != nil {
		return m.Scheme
	}
	return nil
}

func (m *TapEvent_Http_RequestInit) GetAuthority() string {
	if m != nil {
		return m.Authority
	}
	return ""
}

func (m *TapEvent_Http_RequestInit) GetPath() string {
	if m != nil {
		return m.Path
	}
	return ""
}

type TapEvent_Http_ResponseInit struct {
	Id               *TapEvent_Http_StreamId   `protobuf:"bytes,1,opt,name=id" json:"id,omitempty"`
	SinceRequestInit *google_protobuf.Duration `protobuf:"bytes,2,opt,name=since_request_init,json=sinceRequestInit" json:"since_request_init,omitempty"`
	HttpStatus       uint32                    `protobuf:"varint,3,opt,name=http_status,json=httpStatus" json:"http_status,omitempty"`
}

func (m *TapEvent_Http_ResponseInit) Reset()         { *m = TapEvent_Http_ResponseInit{} }
func (m *TapEvent_Http_ResponseInit) String() string { return proto.CompactTextString(m) }
func (*TapEvent_Http_ResponseInit) ProtoMessage()    {}
func (*TapEvent_Http_ResponseInit) Descriptor() ([]byte, []int) {
	return fileDescriptor0, []int{7, 1, 2}
}

func (m *TapEvent_Http_ResponseInit) GetId() *TapEvent_Http_StreamId {
	if m != nil {
		return m.Id
	}
	return nil
}

func (m *TapEvent_Http_ResponseInit) GetSinceRequestInit() *google_protobuf.Duration {
	if m != nil {
		return m.SinceRequestInit
	}
	return nil
}

func (m *TapEvent_Http_ResponseInit) GetHttpStatus() uint32 {
	if m != nil {
		return m.HttpStatus
	}
	return 0
}

type TapEvent_Http_ResponseEnd struct {
	Id                *TapEvent_Http_StreamId   `protobuf:"bytes,1,opt,name=id" json:"id,omitempty"`
	SinceRequestInit  *google_protobuf.Duration `protobuf:"bytes,2,opt,name=since_request_init,json=sinceRequestInit" json:"since_request_init,omitempty"`
	SinceResponseInit *google_protobuf.Duration `protobuf:"bytes,3,opt,name=since_response_init,json=sinceResponseInit" json:"since_response_init,omitempty"`
	ResponseBytes     uint64                    `protobuf:"varint,4,opt,name=response_bytes,json=responseBytes" json:"response_bytes,omitempty"`
	Eos               *Eos                      `protobuf:"bytes,5,opt,name=eos" json:"eos,omitempty"`
}

func (m *TapEvent_Http_ResponseEnd) Reset()                    { *m = TapEvent_Http_ResponseEnd{} }
func (m *TapEvent_Http_ResponseEnd) String() string            { return proto.CompactTextString(m) }
func (*TapEvent_Http_ResponseEnd) ProtoMessage()               {}
func (*TapEvent_Http_ResponseEnd) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{7, 1, 3} }

func (m *TapEvent_Http_ResponseEnd) GetId() *TapEvent_Http_StreamId {
	if m != nil {
		return m.Id
	}
	return nil
}

func (m *TapEvent_Http_ResponseEnd) GetSinceRequestInit() *google_protobuf.Duration {
	if m != nil {
		return m.SinceRequestInit
	}
	return nil
}

func (m *TapEvent_Http_ResponseEnd) GetSinceResponseInit() *google_protobuf.Duration {
	if m != nil {
		return m.SinceResponseInit
	}
	return nil
}

func (m *TapEvent_Http_ResponseEnd) GetResponseBytes() uint64 {
	if m != nil {
		return m.ResponseBytes
	}
	return 0
}

func (m *TapEvent_Http_ResponseEnd) GetEos() *Eos {
	if m != nil {
		return m.Eos
	}
	return nil
}

func init() {
	proto.RegisterType((*HttpMethod)(nil), "conduit.common.HttpMethod")
	proto.RegisterType((*Scheme)(nil), "conduit.common.Scheme")
	proto.RegisterType((*IPAddress)(nil), "conduit.common.IPAddress")
	proto.RegisterType((*IPv6)(nil), "conduit.common.IPv6")
	proto.RegisterType((*TcpAddress)(nil), "conduit.common.TcpAddress")
	proto.RegisterType((*Destination)(nil), "conduit.common.Destination")
	proto.RegisterType((*Eos)(nil), "conduit.common.Eos")
	proto.RegisterType((*TapEvent)(nil), "conduit.common.TapEvent")
	proto.RegisterType((*TapEvent_EndpointMeta)(nil), "conduit.common.TapEvent.EndpointMeta")
	proto.RegisterType((*TapEvent_Http)(nil), "conduit.common.TapEvent.Http")
	proto.RegisterType((*TapEvent_Http_StreamId)(nil), "conduit.common.TapEvent.Http.StreamId")
	proto.RegisterType((*TapEvent_Http_RequestInit)(nil), "conduit.common.TapEvent.Http.RequestInit")
	proto.RegisterType((*TapEvent_Http_ResponseInit)(nil), "conduit.common.TapEvent.Http.ResponseInit")
	proto.RegisterType((*TapEvent_Http_ResponseEnd)(nil), "conduit.common.TapEvent.Http.ResponseEnd")
	proto.RegisterEnum("conduit.common.Protocol", Protocol_name, Protocol_value)
	proto.RegisterEnum("conduit.common.HttpMethod_Registered", HttpMethod_Registered_name, HttpMethod_Registered_value)
	proto.RegisterEnum("conduit.common.Scheme_Registered", Scheme_Registered_name, Scheme_Registered_value)
}

func init() { proto.RegisterFile("common.proto", fileDescriptor0) }

var fileDescriptor0 = []byte{
	// 989 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0xc4, 0x55, 0x41, 0x6f, 0xe3, 0x44,
	0x14, 0x8e, 0x13, 0xc7, 0x69, 0x9e, 0xd3, 0x62, 0x66, 0xab, 0x55, 0x89, 0x58, 0xe8, 0x46, 0x14,
	0xb5, 0x3d, 0x38, 0x90, 0x42, 0xc5, 0x22, 0x2e, 0x4d, 0x6a, 0x35, 0x11, 0xbb, 0xad, 0x99, 0x98,
	0x0b, 0x97, 0xc8, 0x89, 0x67, 0x13, 0x8b, 0xc4, 0x63, 0x66, 0xc6, 0x95, 0xf2, 0x3f, 0xb8, 0x21,
	0x71, 0xe5, 0xca, 0xff, 0xe1, 0x67, 0x20, 0xee, 0x68, 0xc6, 0x13, 0xc7, 0xed, 0xee, 0x76, 0x57,
	0x70, 0xd8, 0x93, 0xe7, 0xbd, 0x79, 0xef, 0xf3, 0xf7, 0xcd, 0x9b, 0xf7, 0x06, 0x5a, 0x33, 0xba,
	0x5a, 0xd1, 0xc4, 0x4d, 0x19, 0x15, 0x14, 0xed, 0xcd, 0x68, 0x12, 0x65, 0xb1, 0x70, 0x73, 0x6f,
	0xfb, 0x93, 0x39, 0xa5, 0xf3, 0x25, 0xe9, 0xaa, 0xdd, 0x69, 0xf6, 0xb2, 0x1b, 0x65, 0x2c, 0x14,
	0xf1, 0x26, 0xbe, 0xf3, 0xb7, 0x01, 0x30, 0x14, 0x22, 0x7d, 0x41, 0xc4, 0x82, 0x46, 0xe8, 0x0a,
	0x80, 0x91, 0x79, 0xcc, 0x05, 0x61, 0x24, 0x3a, 0x30, 0x0e, 0x8d, 0xe3, 0xbd, 0xde, 0x91, 0x7b,
	0x17, 0xd3, 0xdd, 0xc6, 0xbb, 0xb8, 0x08, 0x1e, 0x56, 0x70, 0x29, 0x15, 0x7d, 0x06, 0xad, 0x2c,
	0x29, 0x41, 0x55, 0x0f, 0x8d, 0xe3, 0xe6, 0xb0, 0x82, 0xef, 0x78, 0x3b, 0x09, 0xc0, 0x16, 0x01,
	0x35, 0xa0, 0x76, 0xe5, 0x05, 0x4e, 0x05, 0xed, 0x80, 0xe9, 0xdf, 0x8c, 0x03, 0xc7, 0x90, 0x2e,
	0xff, 0xc7, 0xc0, 0xa9, 0x22, 0x00, 0xeb, 0xd2, 0x7b, 0xee, 0x05, 0x9e, 0x53, 0x43, 0x4d, 0xa8,
	0xfb, 0x17, 0xc1, 0x60, 0xe8, 0x98, 0xc8, 0x86, 0xc6, 0x8d, 0x1f, 0x8c, 0x6e, 0xae, 0xc7, 0x4e,
	0x5d, 0x1a, 0x83, 0x9b, 0xeb, 0x6b, 0x6f, 0x10, 0x38, 0x96, 0xc4, 0x18, 0x7a, 0x17, 0x97, 0x4e,
	0x43, 0x86, 0x07, 0xf8, 0x62, 0xe0, 0x39, 0x3b, 0x7d, 0x0b, 0x4c, 0xb1, 0x4e, 0x49, 0xe7, 0x77,
	0x03, 0xac, 0xf1, 0x6c, 0x41, 0x56, 0x04, 0x0d, 0x5e, 0xa3, 0xf8, 0xe9, 0x7d, 0xc5, 0x79, 0xec,
	0xff, 0x55, 0xfb, 0xf4, 0x8e, 0x5a, 0x49, 0x30, 0x08, 0x7c, 0xa7, 0x22, 0x09, 0xca, 0xd5, 0xd8,
	0x31, 0x0a, 0x82, 0x63, 0x68, 0x8e, 0xfc, 0x8b, 0x28, 0x62, 0x84, 0x73, 0xb4, 0x0f, 0x66, 0x9c,
	0xde, 0x7e, 0xa5, 0xc8, 0x35, 0x86, 0x15, 0xac, 0x2c, 0x74, 0xaa, 0xbc, 0xe7, 0xea, 0x5f, 0x76,
	0x6f, 0xff, 0x3e, 0xe5, 0x91, 0x7f, 0x7b, 0xae, 0x63, 0xcf, 0xfb, 0x26, 0x54, 0xe3, 0xb4, 0xf3,
	0x05, 0x98, 0xd2, 0x8b, 0xf6, 0xa1, 0xfe, 0x32, 0x66, 0x5c, 0x28, 0x40, 0x0b, 0xe7, 0x06, 0x42,
	0x60, 0x2e, 0x43, 0x2e, 0x14, 0x9e, 0x85, 0xd5, 0xba, 0xf3, 0x3d, 0x40, 0x30, 0x4b, 0x37, 0x3c,
	0x4e, 0x24, 0x8a, 0x4a, 0xb2, 0x7b, 0x1f, 0xbd, 0xfa, 0x3f, 0x1d, 0x86, 0xab, 0x71, 0x2a, 0xc1,
	0x52, 0xca, 0x72, 0xb0, 0x5d, 0xac, 0xd6, 0x9d, 0x67, 0x60, 0x5f, 0x12, 0x2e, 0xe2, 0x44, 0xdd,
	0x3f, 0xf4, 0x18, 0x2c, 0xae, 0x8e, 0x55, 0x21, 0x36, 0xb1, 0xb6, 0x54, 0x6a, 0x28, 0x16, 0xf9,
	0x19, 0x62, 0xb5, 0xee, 0x44, 0x50, 0xf3, 0x28, 0x47, 0xa7, 0xe0, 0xcc, 0x59, 0x3a, 0x9b, 0x70,
	0x11, 0x8a, 0x8c, 0x4f, 0x66, 0x34, 0xca, 0x93, 0x77, 0x87, 0x15, 0xbc, 0x27, 0x77, 0xc6, 0x6a,
	0x63, 0x40, 0x23, 0x22, 0x63, 0x19, 0xe1, 0x44, 0x4c, 0x08, 0x63, 0x94, 0xe5, 0xb1, 0xd5, 0x4d,
	0xac, 0xda, 0xf1, 0xe4, 0x86, 0x8c, 0xed, 0xd7, 0xa1, 0x46, 0x92, 0xa8, 0xf3, 0x9b, 0x0d, 0x3b,
	0x41, 0x98, 0x7a, 0xb7, 0x24, 0x11, 0xa8, 0x07, 0x16, 0xa7, 0x19, 0x9b, 0x11, 0x2d, 0xb8, 0x7d,
	0x5f, 0xf0, 0xf6, 0x60, 0xb0, 0x8e, 0x44, 0xdf, 0x81, 0x1d, 0x6d, 0x15, 0xea, 0xca, 0x3c, 0x94,
	0x58, 0x0e, 0x47, 0x3e, 0x38, 0x25, 0x73, 0xb2, 0x22, 0x22, 0x3c, 0x30, 0x15, 0xc4, 0x2b, 0x1d,
	0xb8, 0x61, 0xe9, 0x7a, 0x49, 0x94, 0xd2, 0x38, 0x11, 0x2f, 0x88, 0x08, 0xf1, 0x07, 0xa5, 0x74,
	0xe9, 0x40, 0x67, 0x60, 0x2e, 0x84, 0x48, 0x0f, 0x6a, 0x0a, 0xe5, 0xc9, 0x1b, 0x51, 0x64, 0x43,
	0xcb, 0xbb, 0x22, 0x83, 0xdb, 0xbf, 0x1a, 0xd0, 0x2a, 0xc3, 0xa2, 0x11, 0x58, 0xcb, 0x70, 0x4a,
	0x96, 0xfc, 0xc0, 0x38, 0xac, 0x1d, 0xdb, 0xbd, 0x2f, 0xdf, 0x89, 0x8d, 0xfb, 0x5c, 0xe5, 0x78,
	0x89, 0x60, 0x6b, 0xac, 0x01, 0xda, 0xcf, 0xc0, 0x2e, 0xb9, 0x91, 0x03, 0xb5, 0x9f, 0xc9, 0x5a,
	0xd7, 0x5f, 0x2e, 0xe5, 0xd5, 0xbc, 0x0d, 0x97, 0x19, 0xd1, 0xd5, 0xcf, 0x8d, 0x6f, 0xab, 0xdf,
	0x18, 0xed, 0x7f, 0x1a, 0x60, 0x4a, 0x9e, 0xe8, 0x1a, 0x5a, 0x8c, 0xfc, 0x92, 0x11, 0x2e, 0x26,
	0x71, 0x12, 0x0b, 0x5d, 0x9e, 0x93, 0x07, 0xc5, 0xb9, 0x38, 0xcf, 0x18, 0x25, 0xb1, 0x18, 0x56,
	0xb0, 0xcd, 0xb6, 0x26, 0xfa, 0x01, 0x76, 0x19, 0xe1, 0x29, 0x4d, 0x38, 0xc9, 0x01, 0xf3, 0xb2,
	0x9d, 0xbe, 0x0d, 0x30, 0x4f, 0xd1, 0x88, 0x2d, 0x56, 0xb2, 0x73, 0x8a, 0x1a, 0x92, 0x24, 0x91,
	0x3e, 0xff, 0x93, 0x77, 0x43, 0xf4, 0x92, 0x28, 0xa7, 0x58, 0x98, 0xed, 0x73, 0xd8, 0x19, 0x0b,
	0x46, 0xc2, 0xd5, 0x28, 0x92, 0xed, 0x31, 0x0d, 0xb9, 0xbe, 0xf7, 0x58, 0xad, 0x55, 0x2b, 0xa9,
	0x7d, 0xc5, 0xdd, 0xc4, 0xda, 0x6a, 0xff, 0x65, 0x80, 0x5d, 0x52, 0x8e, 0xce, 0xa1, 0x1a, 0x47,
	0xfa, 0xc0, 0x3e, 0x7f, 0x98, 0xcd, 0xe6, 0x7f, 0xb8, 0x1a, 0x47, 0xb2, 0x17, 0x56, 0x6a, 0xde,
	0xbf, 0xe9, 0x4a, 0x6f, 0x5f, 0x04, 0xac, 0x23, 0x91, 0x5b, 0xb4, 0x77, 0xae, 0xfe, 0xf1, 0xeb,
	0x67, 0x6a, 0xd1, 0xf6, 0x1f, 0x43, 0x33, 0xcc, 0xc4, 0x82, 0xb2, 0x58, 0xac, 0xd5, 0xb5, 0x6f,
	0xe2, 0xad, 0xa3, 0x18, 0x0a, 0xf5, 0xed, 0x50, 0x68, 0xff, 0x69, 0x40, 0xab, 0x5c, 0x86, 0xff,
	0x2c, 0xef, 0x0a, 0x10, 0x8f, 0x93, 0x19, 0x99, 0xdc, 0xb9, 0x57, 0x55, 0x3d, 0xe7, 0xf2, 0x07,
	0xd4, 0xdd, 0x3c, 0xa0, 0xee, 0xa5, 0x7e, 0x40, 0xb1, 0xa3, 0x92, 0xca, 0xe7, 0xfb, 0x29, 0xd8,
	0xb2, 0x85, 0xf4, 0x7c, 0x52, 0xc2, 0x77, 0x31, 0x48, 0x57, 0x3e, 0x98, 0xda, 0x7f, 0x54, 0x65,
	0x41, 0x8a, 0xc2, 0xbe, 0x7f, 0xc6, 0x23, 0x78, 0xb4, 0x01, 0x2a, 0xb7, 0x40, 0xed, 0x6d, 0x48,
	0x1f, 0x6a, 0xa4, 0xd2, 0xe9, 0x1f, 0xc1, 0x5e, 0x01, 0x32, 0x5d, 0x0b, 0xc2, 0x55, 0x15, 0x4d,
	0x5c, 0x74, 0x57, 0x5f, 0x3a, 0xd1, 0x11, 0xd4, 0x08, 0xe5, 0xaa, 0x90, 0x76, 0xef, 0xd1, 0x7d,
	0xcd, 0x1e, 0xe5, 0x58, 0xee, 0xf7, 0x1b, 0x50, 0x27, 0x52, 0x7c, 0xb1, 0x38, 0x7d, 0x02, 0x3b,
	0xbe, 0xe4, 0x31, 0xa3, 0xcb, 0xd2, 0xdb, 0xd9, 0x80, 0x5a, 0x30, 0xf0, 0x1d, 0xa3, 0xff, 0xf5,
	0x4f, 0x67, 0xf3, 0x58, 0x2c, 0xb2, 0xa9, 0x84, 0xea, 0xb2, 0x2c, 0xd1, 0xc8, 0xdd, 0xd2, 0x57,
	0x30, 0xba, 0x5c, 0x12, 0xd6, 0x9d, 0x93, 0xa4, 0x9b, 0xff, 0x70, 0x6a, 0x29, 0x6d, 0x67, 0xff,
	0x06, 0x00, 0x00, 0xff, 0xff, 0x65, 0xd9, 0x8b, 0xf2, 0x46, 0x09, 0x00, 0x00,
}
