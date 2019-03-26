// Code generated by protoc-gen-go. DO NOT EDIT.
// source: config/config.proto

package config // import "github.com/linkerd/linkerd2/controller/gen/config"

import proto "github.com/golang/protobuf/proto"
import fmt "fmt"
import math "math"
import duration "github.com/golang/protobuf/ptypes/duration"

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

// This is a compile-time assertion to ensure that this generated file
// is compatible with the proto package it is being compiled against.
// A compilation error at this line likely means your copy of the
// proto package needs to be updated.
const _ = proto.ProtoPackageIsVersion2 // please upgrade the proto package

type All struct {
	Global               *Global  `protobuf:"bytes,1,opt,name=global,proto3" json:"global,omitempty"`
	Proxy                *Proxy   `protobuf:"bytes,2,opt,name=proxy,proto3" json:"proxy,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *All) Reset()         { *m = All{} }
func (m *All) String() string { return proto.CompactTextString(m) }
func (*All) ProtoMessage()    {}
func (*All) Descriptor() ([]byte, []int) {
	return fileDescriptor_config_fd709b401e3b0efb, []int{0}
}
func (m *All) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_All.Unmarshal(m, b)
}
func (m *All) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_All.Marshal(b, m, deterministic)
}
func (dst *All) XXX_Merge(src proto.Message) {
	xxx_messageInfo_All.Merge(dst, src)
}
func (m *All) XXX_Size() int {
	return xxx_messageInfo_All.Size(m)
}
func (m *All) XXX_DiscardUnknown() {
	xxx_messageInfo_All.DiscardUnknown(m)
}

var xxx_messageInfo_All proto.InternalMessageInfo

func (m *All) GetGlobal() *Global {
	if m != nil {
		return m.Global
	}
	return nil
}

func (m *All) GetProxy() *Proxy {
	if m != nil {
		return m.Proxy
	}
	return nil
}

type Global struct {
	LinkerdNamespace string `protobuf:"bytes,1,opt,name=linkerd_namespace,json=linkerdNamespace,proto3" json:"linkerd_namespace,omitempty"`
	CniEnabled       bool   `protobuf:"varint,2,opt,name=cni_enabled,json=cniEnabled,proto3" json:"cni_enabled,omitempty"`
	Version          string `protobuf:"bytes,3,opt,name=version,proto3" json:"version,omitempty"`
	// If present, configures identity.
	IdentityContext *IdentityContext `protobuf:"bytes,4,opt,name=identity_context,json=identityContext,proto3" json:"identity_context,omitempty"`
	// If present, indicates that the Mutating Webhook Admission Controller should
	// be configured to automatically inject proxies.
	AutoInjectContext    *AutoInjectContext `protobuf:"bytes,6,opt,name=auto_inject_context,json=autoInjectContext,proto3" json:"auto_inject_context,omitempty"`
	InstallationUuid     string             `protobuf:"bytes,5,opt,name=installation_uuid,json=installationUuid,proto3" json:"installation_uuid,omitempty"`
	XXX_NoUnkeyedLiteral struct{}           `json:"-"`
	XXX_unrecognized     []byte             `json:"-"`
	XXX_sizecache        int32              `json:"-"`
}

func (m *Global) Reset()         { *m = Global{} }
func (m *Global) String() string { return proto.CompactTextString(m) }
func (*Global) ProtoMessage()    {}
func (*Global) Descriptor() ([]byte, []int) {
	return fileDescriptor_config_fd709b401e3b0efb, []int{1}
}
func (m *Global) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_Global.Unmarshal(m, b)
}
func (m *Global) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_Global.Marshal(b, m, deterministic)
}
func (dst *Global) XXX_Merge(src proto.Message) {
	xxx_messageInfo_Global.Merge(dst, src)
}
func (m *Global) XXX_Size() int {
	return xxx_messageInfo_Global.Size(m)
}
func (m *Global) XXX_DiscardUnknown() {
	xxx_messageInfo_Global.DiscardUnknown(m)
}

var xxx_messageInfo_Global proto.InternalMessageInfo

func (m *Global) GetLinkerdNamespace() string {
	if m != nil {
		return m.LinkerdNamespace
	}
	return ""
}

func (m *Global) GetCniEnabled() bool {
	if m != nil {
		return m.CniEnabled
	}
	return false
}

func (m *Global) GetVersion() string {
	if m != nil {
		return m.Version
	}
	return ""
}

func (m *Global) GetIdentityContext() *IdentityContext {
	if m != nil {
		return m.IdentityContext
	}
	return nil
}

func (m *Global) GetAutoInjectContext() *AutoInjectContext {
	if m != nil {
		return m.AutoInjectContext
	}
	return nil
}

func (m *Global) GetInstallationUuid() string {
	if m != nil {
		return m.InstallationUuid
	}
	return ""
}

type Proxy struct {
	ProxyImage              *Image                `protobuf:"bytes,1,opt,name=proxy_image,json=proxyImage,proto3" json:"proxy_image,omitempty"`
	ProxyInitImage          *Image                `protobuf:"bytes,2,opt,name=proxy_init_image,json=proxyInitImage,proto3" json:"proxy_init_image,omitempty"`
	ControlPort             *Port                 `protobuf:"bytes,3,opt,name=control_port,json=controlPort,proto3" json:"control_port,omitempty"`
	IgnoreInboundPorts      []*Port               `protobuf:"bytes,4,rep,name=ignore_inbound_ports,json=ignoreInboundPorts,proto3" json:"ignore_inbound_ports,omitempty"`
	IgnoreOutboundPorts     []*Port               `protobuf:"bytes,5,rep,name=ignore_outbound_ports,json=ignoreOutboundPorts,proto3" json:"ignore_outbound_ports,omitempty"`
	InboundPort             *Port                 `protobuf:"bytes,6,opt,name=inbound_port,json=inboundPort,proto3" json:"inbound_port,omitempty"`
	AdminPort               *Port                 `protobuf:"bytes,7,opt,name=admin_port,json=adminPort,proto3" json:"admin_port,omitempty"`
	OutboundPort            *Port                 `protobuf:"bytes,8,opt,name=outbound_port,json=outboundPort,proto3" json:"outbound_port,omitempty"`
	Resource                *ResourceRequirements `protobuf:"bytes,9,opt,name=resource,proto3" json:"resource,omitempty"`
	ProxyUid                int64                 `protobuf:"varint,10,opt,name=proxy_uid,json=proxyUid,proto3" json:"proxy_uid,omitempty"`
	LogLevel                *LogLevel             `protobuf:"bytes,11,opt,name=log_level,json=logLevel,proto3" json:"log_level,omitempty"`
	DisableExternalProfiles bool                  `protobuf:"varint,12,opt,name=disable_external_profiles,json=disableExternalProfiles,proto3" json:"disable_external_profiles,omitempty"`
	XXX_NoUnkeyedLiteral    struct{}              `json:"-"`
	XXX_unrecognized        []byte                `json:"-"`
	XXX_sizecache           int32                 `json:"-"`
}

func (m *Proxy) Reset()         { *m = Proxy{} }
func (m *Proxy) String() string { return proto.CompactTextString(m) }
func (*Proxy) ProtoMessage()    {}
func (*Proxy) Descriptor() ([]byte, []int) {
	return fileDescriptor_config_fd709b401e3b0efb, []int{2}
}
func (m *Proxy) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_Proxy.Unmarshal(m, b)
}
func (m *Proxy) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_Proxy.Marshal(b, m, deterministic)
}
func (dst *Proxy) XXX_Merge(src proto.Message) {
	xxx_messageInfo_Proxy.Merge(dst, src)
}
func (m *Proxy) XXX_Size() int {
	return xxx_messageInfo_Proxy.Size(m)
}
func (m *Proxy) XXX_DiscardUnknown() {
	xxx_messageInfo_Proxy.DiscardUnknown(m)
}

var xxx_messageInfo_Proxy proto.InternalMessageInfo

func (m *Proxy) GetProxyImage() *Image {
	if m != nil {
		return m.ProxyImage
	}
	return nil
}

func (m *Proxy) GetProxyInitImage() *Image {
	if m != nil {
		return m.ProxyInitImage
	}
	return nil
}

func (m *Proxy) GetControlPort() *Port {
	if m != nil {
		return m.ControlPort
	}
	return nil
}

func (m *Proxy) GetIgnoreInboundPorts() []*Port {
	if m != nil {
		return m.IgnoreInboundPorts
	}
	return nil
}

func (m *Proxy) GetIgnoreOutboundPorts() []*Port {
	if m != nil {
		return m.IgnoreOutboundPorts
	}
	return nil
}

func (m *Proxy) GetInboundPort() *Port {
	if m != nil {
		return m.InboundPort
	}
	return nil
}

func (m *Proxy) GetAdminPort() *Port {
	if m != nil {
		return m.AdminPort
	}
	return nil
}

func (m *Proxy) GetOutboundPort() *Port {
	if m != nil {
		return m.OutboundPort
	}
	return nil
}

func (m *Proxy) GetResource() *ResourceRequirements {
	if m != nil {
		return m.Resource
	}
	return nil
}

func (m *Proxy) GetProxyUid() int64 {
	if m != nil {
		return m.ProxyUid
	}
	return 0
}

func (m *Proxy) GetLogLevel() *LogLevel {
	if m != nil {
		return m.LogLevel
	}
	return nil
}

func (m *Proxy) GetDisableExternalProfiles() bool {
	if m != nil {
		return m.DisableExternalProfiles
	}
	return false
}

type Image struct {
	ImageName            string   `protobuf:"bytes,1,opt,name=image_name,json=imageName,proto3" json:"image_name,omitempty"`
	PullPolicy           string   `protobuf:"bytes,2,opt,name=pull_policy,json=pullPolicy,proto3" json:"pull_policy,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *Image) Reset()         { *m = Image{} }
func (m *Image) String() string { return proto.CompactTextString(m) }
func (*Image) ProtoMessage()    {}
func (*Image) Descriptor() ([]byte, []int) {
	return fileDescriptor_config_fd709b401e3b0efb, []int{3}
}
func (m *Image) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_Image.Unmarshal(m, b)
}
func (m *Image) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_Image.Marshal(b, m, deterministic)
}
func (dst *Image) XXX_Merge(src proto.Message) {
	xxx_messageInfo_Image.Merge(dst, src)
}
func (m *Image) XXX_Size() int {
	return xxx_messageInfo_Image.Size(m)
}
func (m *Image) XXX_DiscardUnknown() {
	xxx_messageInfo_Image.DiscardUnknown(m)
}

var xxx_messageInfo_Image proto.InternalMessageInfo

func (m *Image) GetImageName() string {
	if m != nil {
		return m.ImageName
	}
	return ""
}

func (m *Image) GetPullPolicy() string {
	if m != nil {
		return m.PullPolicy
	}
	return ""
}

type Port struct {
	Port                 uint32   `protobuf:"varint,1,opt,name=port,proto3" json:"port,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *Port) Reset()         { *m = Port{} }
func (m *Port) String() string { return proto.CompactTextString(m) }
func (*Port) ProtoMessage()    {}
func (*Port) Descriptor() ([]byte, []int) {
	return fileDescriptor_config_fd709b401e3b0efb, []int{4}
}
func (m *Port) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_Port.Unmarshal(m, b)
}
func (m *Port) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_Port.Marshal(b, m, deterministic)
}
func (dst *Port) XXX_Merge(src proto.Message) {
	xxx_messageInfo_Port.Merge(dst, src)
}
func (m *Port) XXX_Size() int {
	return xxx_messageInfo_Port.Size(m)
}
func (m *Port) XXX_DiscardUnknown() {
	xxx_messageInfo_Port.DiscardUnknown(m)
}

var xxx_messageInfo_Port proto.InternalMessageInfo

func (m *Port) GetPort() uint32 {
	if m != nil {
		return m.Port
	}
	return 0
}

type ResourceRequirements struct {
	RequestCpu           string   `protobuf:"bytes,1,opt,name=request_cpu,json=requestCpu,proto3" json:"request_cpu,omitempty"`
	RequestMemory        string   `protobuf:"bytes,2,opt,name=request_memory,json=requestMemory,proto3" json:"request_memory,omitempty"`
	LimitCpu             string   `protobuf:"bytes,3,opt,name=limit_cpu,json=limitCpu,proto3" json:"limit_cpu,omitempty"`
	LimitMemory          string   `protobuf:"bytes,4,opt,name=limit_memory,json=limitMemory,proto3" json:"limit_memory,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *ResourceRequirements) Reset()         { *m = ResourceRequirements{} }
func (m *ResourceRequirements) String() string { return proto.CompactTextString(m) }
func (*ResourceRequirements) ProtoMessage()    {}
func (*ResourceRequirements) Descriptor() ([]byte, []int) {
	return fileDescriptor_config_fd709b401e3b0efb, []int{5}
}
func (m *ResourceRequirements) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_ResourceRequirements.Unmarshal(m, b)
}
func (m *ResourceRequirements) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_ResourceRequirements.Marshal(b, m, deterministic)
}
func (dst *ResourceRequirements) XXX_Merge(src proto.Message) {
	xxx_messageInfo_ResourceRequirements.Merge(dst, src)
}
func (m *ResourceRequirements) XXX_Size() int {
	return xxx_messageInfo_ResourceRequirements.Size(m)
}
func (m *ResourceRequirements) XXX_DiscardUnknown() {
	xxx_messageInfo_ResourceRequirements.DiscardUnknown(m)
}

var xxx_messageInfo_ResourceRequirements proto.InternalMessageInfo

func (m *ResourceRequirements) GetRequestCpu() string {
	if m != nil {
		return m.RequestCpu
	}
	return ""
}

func (m *ResourceRequirements) GetRequestMemory() string {
	if m != nil {
		return m.RequestMemory
	}
	return ""
}

func (m *ResourceRequirements) GetLimitCpu() string {
	if m != nil {
		return m.LimitCpu
	}
	return ""
}

func (m *ResourceRequirements) GetLimitMemory() string {
	if m != nil {
		return m.LimitMemory
	}
	return ""
}

// Currently, this is basically a boolean.
type AutoInjectContext struct {
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *AutoInjectContext) Reset()         { *m = AutoInjectContext{} }
func (m *AutoInjectContext) String() string { return proto.CompactTextString(m) }
func (*AutoInjectContext) ProtoMessage()    {}
func (*AutoInjectContext) Descriptor() ([]byte, []int) {
	return fileDescriptor_config_fd709b401e3b0efb, []int{6}
}
func (m *AutoInjectContext) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_AutoInjectContext.Unmarshal(m, b)
}
func (m *AutoInjectContext) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_AutoInjectContext.Marshal(b, m, deterministic)
}
func (dst *AutoInjectContext) XXX_Merge(src proto.Message) {
	xxx_messageInfo_AutoInjectContext.Merge(dst, src)
}
func (m *AutoInjectContext) XXX_Size() int {
	return xxx_messageInfo_AutoInjectContext.Size(m)
}
func (m *AutoInjectContext) XXX_DiscardUnknown() {
	xxx_messageInfo_AutoInjectContext.DiscardUnknown(m)
}

var xxx_messageInfo_AutoInjectContext proto.InternalMessageInfo

type IdentityContext struct {
	TrustDomain          string             `protobuf:"bytes,1,opt,name=trust_domain,json=trustDomain,proto3" json:"trust_domain,omitempty"`
	TrustAnchorsPem      string             `protobuf:"bytes,2,opt,name=trust_anchors_pem,json=trustAnchorsPem,proto3" json:"trust_anchors_pem,omitempty"`
	IssuanceLifetime     *duration.Duration `protobuf:"bytes,3,opt,name=issuance_lifetime,json=issuanceLifetime,proto3" json:"issuance_lifetime,omitempty"`
	ClockSkewAllowance   *duration.Duration `protobuf:"bytes,4,opt,name=clock_skew_allowance,json=clockSkewAllowance,proto3" json:"clock_skew_allowance,omitempty"`
	XXX_NoUnkeyedLiteral struct{}           `json:"-"`
	XXX_unrecognized     []byte             `json:"-"`
	XXX_sizecache        int32              `json:"-"`
}

func (m *IdentityContext) Reset()         { *m = IdentityContext{} }
func (m *IdentityContext) String() string { return proto.CompactTextString(m) }
func (*IdentityContext) ProtoMessage()    {}
func (*IdentityContext) Descriptor() ([]byte, []int) {
	return fileDescriptor_config_fd709b401e3b0efb, []int{7}
}
func (m *IdentityContext) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_IdentityContext.Unmarshal(m, b)
}
func (m *IdentityContext) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_IdentityContext.Marshal(b, m, deterministic)
}
func (dst *IdentityContext) XXX_Merge(src proto.Message) {
	xxx_messageInfo_IdentityContext.Merge(dst, src)
}
func (m *IdentityContext) XXX_Size() int {
	return xxx_messageInfo_IdentityContext.Size(m)
}
func (m *IdentityContext) XXX_DiscardUnknown() {
	xxx_messageInfo_IdentityContext.DiscardUnknown(m)
}

var xxx_messageInfo_IdentityContext proto.InternalMessageInfo

func (m *IdentityContext) GetTrustDomain() string {
	if m != nil {
		return m.TrustDomain
	}
	return ""
}

func (m *IdentityContext) GetTrustAnchorsPem() string {
	if m != nil {
		return m.TrustAnchorsPem
	}
	return ""
}

func (m *IdentityContext) GetIssuanceLifetime() *duration.Duration {
	if m != nil {
		return m.IssuanceLifetime
	}
	return nil
}

func (m *IdentityContext) GetClockSkewAllowance() *duration.Duration {
	if m != nil {
		return m.ClockSkewAllowance
	}
	return nil
}

type LogLevel struct {
	Level                string   `protobuf:"bytes,1,opt,name=level,proto3" json:"level,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *LogLevel) Reset()         { *m = LogLevel{} }
func (m *LogLevel) String() string { return proto.CompactTextString(m) }
func (*LogLevel) ProtoMessage()    {}
func (*LogLevel) Descriptor() ([]byte, []int) {
	return fileDescriptor_config_fd709b401e3b0efb, []int{8}
}
func (m *LogLevel) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_LogLevel.Unmarshal(m, b)
}
func (m *LogLevel) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_LogLevel.Marshal(b, m, deterministic)
}
func (dst *LogLevel) XXX_Merge(src proto.Message) {
	xxx_messageInfo_LogLevel.Merge(dst, src)
}
func (m *LogLevel) XXX_Size() int {
	return xxx_messageInfo_LogLevel.Size(m)
}
func (m *LogLevel) XXX_DiscardUnknown() {
	xxx_messageInfo_LogLevel.DiscardUnknown(m)
}

var xxx_messageInfo_LogLevel proto.InternalMessageInfo

func (m *LogLevel) GetLevel() string {
	if m != nil {
		return m.Level
	}
	return ""
}

func init() {
	proto.RegisterType((*All)(nil), "linkerd2.config.All")
	proto.RegisterType((*Global)(nil), "linkerd2.config.Global")
	proto.RegisterType((*Proxy)(nil), "linkerd2.config.Proxy")
	proto.RegisterType((*Image)(nil), "linkerd2.config.Image")
	proto.RegisterType((*Port)(nil), "linkerd2.config.Port")
	proto.RegisterType((*ResourceRequirements)(nil), "linkerd2.config.ResourceRequirements")
	proto.RegisterType((*AutoInjectContext)(nil), "linkerd2.config.AutoInjectContext")
	proto.RegisterType((*IdentityContext)(nil), "linkerd2.config.IdentityContext")
	proto.RegisterType((*LogLevel)(nil), "linkerd2.config.LogLevel")
}

func init() { proto.RegisterFile("config/config.proto", fileDescriptor_config_fd709b401e3b0efb) }

var fileDescriptor_config_fd709b401e3b0efb = []byte{
	// 850 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0x7c, 0x55, 0x51, 0x6f, 0xdb, 0x36,
	0x10, 0x86, 0x13, 0x3b, 0xb5, 0xcf, 0x4e, 0x13, 0x33, 0xe9, 0xaa, 0x74, 0xd8, 0xe6, 0x09, 0x28,
	0x50, 0x6c, 0x83, 0x8d, 0xa5, 0xc3, 0x56, 0xf4, 0x69, 0x5e, 0xdb, 0x05, 0x46, 0xb3, 0x2d, 0xd0,
	0xd0, 0x97, 0xbd, 0x10, 0xb2, 0x74, 0x51, 0xb9, 0x50, 0xa4, 0x4b, 0x91, 0x4d, 0xfa, 0x67, 0xf6,
	0x53, 0xf6, 0x1b, 0xf6, 0x67, 0xf6, 0x3e, 0xf0, 0x48, 0x75, 0x69, 0xdc, 0xf8, 0x49, 0xd2, 0x77,
	0xdf, 0xf7, 0xdd, 0xe9, 0x78, 0x3a, 0xc1, 0x41, 0xa1, 0xd5, 0xb9, 0xa8, 0x66, 0xe1, 0x32, 0x5d,
	0x19, 0x6d, 0x35, 0xdb, 0x93, 0x42, 0x5d, 0xa0, 0x29, 0x8f, 0xa7, 0x01, 0x7e, 0xf0, 0x79, 0xa5,
	0x75, 0x25, 0x71, 0x46, 0xe1, 0xa5, 0x3b, 0x9f, 0x95, 0xce, 0xe4, 0x56, 0x68, 0x15, 0x04, 0x69,
	0x09, 0xdb, 0x73, 0x29, 0xd9, 0x0c, 0x76, 0x2a, 0xa9, 0x97, 0xb9, 0x4c, 0x3a, 0x93, 0xce, 0xa3,
	0xe1, 0xf1, 0xfd, 0xe9, 0x0d, 0xa3, 0xe9, 0x09, 0x85, 0xb3, 0x48, 0x63, 0xdf, 0x40, 0x6f, 0x65,
	0xf4, 0xd5, 0xbb, 0x64, 0x8b, 0xf8, 0x9f, 0xac, 0xf1, 0xcf, 0x7c, 0x34, 0x0b, 0xa4, 0xf4, 0xef,
	0x2d, 0xd8, 0x09, 0x06, 0xec, 0x6b, 0x18, 0x47, 0x2a, 0x57, 0x79, 0x8d, 0xcd, 0x2a, 0x2f, 0x90,
	0x92, 0x0e, 0xb2, 0xfd, 0x18, 0xf8, 0xb5, 0xc5, 0xd9, 0x17, 0x30, 0x2c, 0x94, 0xe0, 0xa8, 0xf2,
	0xa5, 0xc4, 0x92, 0x72, 0xf5, 0x33, 0x28, 0x94, 0x78, 0x11, 0x10, 0x96, 0xc0, 0x9d, 0xb7, 0x68,
	0x1a, 0xa1, 0x55, 0xb2, 0x4d, 0x1e, 0xed, 0x23, 0x7b, 0x09, 0xfb, 0xa2, 0x44, 0x65, 0x85, 0x7d,
	0xc7, 0x0b, 0xad, 0x2c, 0x5e, 0xd9, 0xa4, 0x4b, 0xb5, 0x4e, 0xd6, 0x6a, 0x5d, 0x44, 0xe2, 0xb3,
	0xc0, 0xcb, 0xf6, 0xc4, 0x87, 0x00, 0xcb, 0xe0, 0x20, 0x77, 0x56, 0x73, 0xa1, 0xfe, 0xc4, 0xc2,
	0xbe, 0xf7, 0xdb, 0x21, 0xbf, 0x74, 0xcd, 0x6f, 0xee, 0xac, 0x5e, 0x10, 0xb5, 0x75, 0x1c, 0xe7,
	0x37, 0x21, 0xdf, 0x08, 0xa1, 0x1a, 0x9b, 0x4b, 0x49, 0xe7, 0xc1, 0x9d, 0x13, 0x65, 0xd2, 0x0b,
	0x8d, 0xb8, 0x1e, 0x78, 0xe5, 0x44, 0x99, 0xfe, 0xd3, 0x83, 0x1e, 0x75, 0x94, 0xfd, 0x00, 0x43,
	0xea, 0x29, 0x17, 0x75, 0x5e, 0x61, 0x3c, 0xae, 0xf5, 0xf6, 0x2f, 0x7c, 0x34, 0x03, 0xa2, 0xd2,
	0x3d, 0xfb, 0x11, 0xf6, 0xa3, 0x50, 0x09, 0x1b, 0xd5, 0x5b, 0x1b, 0xd5, 0x77, 0x83, 0x5a, 0x09,
	0x1b, 0x1c, 0x9e, 0xc0, 0xc8, 0xbf, 0xb9, 0xd1, 0x92, 0xaf, 0xb4, 0xb1, 0xd4, 0xf1, 0xe1, 0xf1,
	0xbd, 0xf5, 0xa3, 0xd7, 0xc6, 0x66, 0xc3, 0x48, 0xf5, 0x0f, 0xec, 0x04, 0x0e, 0x45, 0xa5, 0xb4,
	0x41, 0x2e, 0xd4, 0x52, 0x3b, 0x55, 0x92, 0x41, 0x93, 0x74, 0x27, 0xdb, 0xb7, 0x3b, 0xb0, 0x20,
	0x59, 0x04, 0x85, 0x87, 0x1a, 0xb6, 0x80, 0x7b, 0xd1, 0x48, 0x3b, 0x7b, 0xdd, 0xa9, 0xb7, 0xc9,
	0xe9, 0x20, 0x68, 0x7e, 0x8b, 0x92, 0x60, 0xf5, 0x04, 0x46, 0xd7, 0x8b, 0x89, 0x87, 0x79, 0xdb,
	0xdb, 0x88, 0xff, 0xab, 0x60, 0xdf, 0x01, 0xe4, 0x65, 0x2d, 0x54, 0xd0, 0xdd, 0xd9, 0xa4, 0x1b,
	0x10, 0x91, 0x54, 0x4f, 0x61, 0xf7, 0x83, 0x9a, 0x93, 0xfe, 0x26, 0xe1, 0x48, 0x5f, 0x2b, 0x96,
	0xcd, 0xa1, 0x6f, 0xb0, 0xd1, 0xce, 0x14, 0x98, 0x0c, 0x48, 0xf6, 0x70, 0x4d, 0x96, 0x45, 0x42,
	0x86, 0x6f, 0x9c, 0x30, 0x58, 0xa3, 0xb2, 0x4d, 0xf6, 0x5e, 0xc6, 0x3e, 0x85, 0x41, 0x38, 0x7e,
	0x3f, 0x66, 0x30, 0xe9, 0x3c, 0xda, 0xce, 0xfa, 0x04, 0xbc, 0x12, 0x25, 0xfb, 0x1e, 0x06, 0x52,
	0x57, 0x5c, 0xe2, 0x5b, 0x94, 0xc9, 0x90, 0x12, 0x1c, 0xad, 0x25, 0x38, 0xd5, 0xd5, 0xa9, 0x27,
	0x64, 0x7d, 0x19, 0xef, 0xd8, 0x53, 0x38, 0x2a, 0x45, 0xe3, 0x3f, 0x45, 0x8e, 0x57, 0x16, 0x8d,
	0xca, 0x25, 0x5f, 0x19, 0x7d, 0x2e, 0x24, 0x36, 0xc9, 0x88, 0xbe, 0xd6, 0xfb, 0x91, 0xf0, 0x22,
	0xc6, 0xcf, 0x62, 0x38, 0x3d, 0x81, 0x5e, 0x18, 0xab, 0xcf, 0x00, 0x68, 0x1a, 0x69, 0x1f, 0xc4,
	0x55, 0x30, 0x20, 0xc4, 0x2f, 0x02, 0xbf, 0x03, 0x56, 0x4e, 0xfa, 0x91, 0x93, 0xa2, 0x08, 0xfb,
	0x66, 0x90, 0x81, 0x87, 0xce, 0x08, 0x49, 0x1f, 0x40, 0x97, 0x9a, 0xc4, 0xa0, 0x4b, 0x7d, 0xf5,
	0x0e, 0xbb, 0x19, 0xdd, 0xa7, 0x7f, 0x75, 0xe0, 0xf0, 0x63, 0x8d, 0xf1, 0xae, 0x06, 0xdf, 0x38,
	0x6c, 0x2c, 0x2f, 0x56, 0x2e, 0x66, 0x85, 0x08, 0x3d, 0x5b, 0x39, 0xf6, 0x10, 0xee, 0xb6, 0x84,
	0x1a, 0x6b, 0x6d, 0xda, 0xcc, 0xbb, 0x11, 0xfd, 0x85, 0x40, 0xdf, 0x56, 0x29, 0x6a, 0x11, 0x5c,
	0xc2, 0x0a, 0xea, 0x13, 0xe0, 0x3d, 0xbe, 0x84, 0x51, 0x08, 0x46, 0x87, 0x2e, 0xc5, 0x87, 0x84,
	0x05, 0x7d, 0x7a, 0x00, 0xe3, 0xb5, 0x6d, 0x91, 0xfe, 0xdb, 0x81, 0xbd, 0x1b, 0x3b, 0xc9, 0x7b,
	0x59, 0xe3, 0x1a, 0xcb, 0x4b, 0x5d, 0xe7, 0x42, 0xc5, 0x8a, 0x87, 0x84, 0x3d, 0x27, 0x88, 0x7d,
	0x05, 0xe3, 0x40, 0xc9, 0x55, 0xf1, 0x5a, 0x9b, 0x86, 0xaf, 0xb0, 0x8e, 0x55, 0xef, 0x51, 0x60,
	0x1e, 0xf0, 0x33, 0xac, 0xd9, 0xcf, 0x30, 0x16, 0x4d, 0xe3, 0x72, 0x55, 0x20, 0x97, 0xe2, 0x1c,
	0xad, 0xa8, 0x31, 0x7e, 0xd0, 0x47, 0xd3, 0xf0, 0xcf, 0x98, 0xb6, 0xff, 0x8c, 0xe9, 0xf3, 0xf8,
	0xcf, 0xc8, 0xf6, 0x5b, 0xcd, 0x69, 0x94, 0xb0, 0x97, 0x70, 0x58, 0x48, 0x5d, 0x5c, 0xf0, 0xe6,
	0x02, 0x2f, 0x79, 0x2e, 0xa5, 0xbe, 0xf4, 0xf1, 0xb8, 0x6a, 0x37, 0x58, 0x31, 0x92, 0xfd, 0x7e,
	0x81, 0x97, 0xf3, 0x56, 0x94, 0x4e, 0xa0, 0xdf, 0x0e, 0x19, 0x3b, 0x84, 0x5e, 0x18, 0xc7, 0xf0,
	0xa2, 0xe1, 0xe1, 0xa7, 0xc7, 0x7f, 0x7c, 0x5b, 0x09, 0xfb, 0xda, 0x2d, 0xa7, 0x85, 0xae, 0x67,
	0x71, 0x42, 0xdb, 0xeb, 0xf1, 0x2c, 0xee, 0x1c, 0x89, 0x66, 0x56, 0xa1, 0x8a, 0xbf, 0xc6, 0xe5,
	0x0e, 0x65, 0x7f, 0xfc, 0x5f, 0x00, 0x00, 0x00, 0xff, 0xff, 0x43, 0x53, 0xc5, 0x64, 0x32, 0x07,
	0x00, 0x00,
}
