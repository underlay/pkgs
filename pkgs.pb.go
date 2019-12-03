// Code generated by protoc-gen-go. DO NOT EDIT.
// source: pkgs.proto

package main

import (
	fmt "fmt"
	proto "github.com/golang/protobuf/proto"
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
const _ = proto.ProtoPackageIsVersion3 // please upgrade the proto package

type Package struct {
	Resource             string   `protobuf:"bytes,1,opt,name=resource,proto3" json:"resource,omitempty"`
	Id                   []byte   `protobuf:"bytes,2,opt,name=id,proto3" json:"id,omitempty"`
	Subject              string   `protobuf:"bytes,3,opt,name=subject,proto3" json:"subject,omitempty"`
	Value                []byte   `protobuf:"bytes,4,opt,name=value,proto3" json:"value,omitempty"`
	Extent               uint64   `protobuf:"varint,5,opt,name=extent,proto3" json:"extent,omitempty"`
	Created              string   `protobuf:"bytes,6,opt,name=created,proto3" json:"created,omitempty"`
	Modified             string   `protobuf:"bytes,7,opt,name=modified,proto3" json:"modified,omitempty"`
	Member               []string `protobuf:"bytes,8,rep,name=member,proto3" json:"member,omitempty"`
	MemberOf             []string `protobuf:"bytes,9,rep,name=memberOf,proto3" json:"memberOf,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *Package) Reset()         { *m = Package{} }
func (m *Package) String() string { return proto.CompactTextString(m) }
func (*Package) ProtoMessage()    {}
func (*Package) Descriptor() ([]byte, []int) {
	return fileDescriptor_21a791bdd2cceefd, []int{0}
}

func (m *Package) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_Package.Unmarshal(m, b)
}
func (m *Package) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_Package.Marshal(b, m, deterministic)
}
func (m *Package) XXX_Merge(src proto.Message) {
	xxx_messageInfo_Package.Merge(m, src)
}
func (m *Package) XXX_Size() int {
	return xxx_messageInfo_Package.Size(m)
}
func (m *Package) XXX_DiscardUnknown() {
	xxx_messageInfo_Package.DiscardUnknown(m)
}

var xxx_messageInfo_Package proto.InternalMessageInfo

func (m *Package) GetResource() string {
	if m != nil {
		return m.Resource
	}
	return ""
}

func (m *Package) GetId() []byte {
	if m != nil {
		return m.Id
	}
	return nil
}

func (m *Package) GetSubject() string {
	if m != nil {
		return m.Subject
	}
	return ""
}

func (m *Package) GetValue() []byte {
	if m != nil {
		return m.Value
	}
	return nil
}

func (m *Package) GetExtent() uint64 {
	if m != nil {
		return m.Extent
	}
	return 0
}

func (m *Package) GetCreated() string {
	if m != nil {
		return m.Created
	}
	return ""
}

func (m *Package) GetModified() string {
	if m != nil {
		return m.Modified
	}
	return ""
}

func (m *Package) GetMember() []string {
	if m != nil {
		return m.Member
	}
	return nil
}

func (m *Package) GetMemberOf() []string {
	if m != nil {
		return m.MemberOf
	}
	return nil
}

type File struct {
	Value                []byte   `protobuf:"bytes,1,opt,name=value,proto3" json:"value,omitempty"`
	Format               string   `protobuf:"bytes,2,opt,name=format,proto3" json:"format,omitempty"`
	Extent               uint64   `protobuf:"varint,3,opt,name=extent,proto3" json:"extent,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *File) Reset()         { *m = File{} }
func (m *File) String() string { return proto.CompactTextString(m) }
func (*File) ProtoMessage()    {}
func (*File) Descriptor() ([]byte, []int) {
	return fileDescriptor_21a791bdd2cceefd, []int{1}
}

func (m *File) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_File.Unmarshal(m, b)
}
func (m *File) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_File.Marshal(b, m, deterministic)
}
func (m *File) XXX_Merge(src proto.Message) {
	xxx_messageInfo_File.Merge(m, src)
}
func (m *File) XXX_Size() int {
	return xxx_messageInfo_File.Size(m)
}
func (m *File) XXX_DiscardUnknown() {
	xxx_messageInfo_File.DiscardUnknown(m)
}

var xxx_messageInfo_File proto.InternalMessageInfo

func (m *File) GetValue() []byte {
	if m != nil {
		return m.Value
	}
	return nil
}

func (m *File) GetFormat() string {
	if m != nil {
		return m.Format
	}
	return ""
}

func (m *File) GetExtent() uint64 {
	if m != nil {
		return m.Extent
	}
	return 0
}

type Resource struct {
	// Types that are valid to be assigned to Resource:
	//	*Resource_Package
	//	*Resource_Message
	//	*Resource_File
	Resource             isResource_Resource `protobuf_oneof:"resource"`
	XXX_NoUnkeyedLiteral struct{}            `json:"-"`
	XXX_unrecognized     []byte              `json:"-"`
	XXX_sizecache        int32               `json:"-"`
}

func (m *Resource) Reset()         { *m = Resource{} }
func (m *Resource) String() string { return proto.CompactTextString(m) }
func (*Resource) ProtoMessage()    {}
func (*Resource) Descriptor() ([]byte, []int) {
	return fileDescriptor_21a791bdd2cceefd, []int{2}
}

func (m *Resource) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_Resource.Unmarshal(m, b)
}
func (m *Resource) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_Resource.Marshal(b, m, deterministic)
}
func (m *Resource) XXX_Merge(src proto.Message) {
	xxx_messageInfo_Resource.Merge(m, src)
}
func (m *Resource) XXX_Size() int {
	return xxx_messageInfo_Resource.Size(m)
}
func (m *Resource) XXX_DiscardUnknown() {
	xxx_messageInfo_Resource.DiscardUnknown(m)
}

var xxx_messageInfo_Resource proto.InternalMessageInfo

type isResource_Resource interface {
	isResource_Resource()
}

type Resource_Package struct {
	Package *Package `protobuf:"bytes,1,opt,name=package,proto3,oneof"`
}

type Resource_Message struct {
	Message []byte `protobuf:"bytes,2,opt,name=message,proto3,oneof"`
}

type Resource_File struct {
	File *File `protobuf:"bytes,3,opt,name=file,proto3,oneof"`
}

func (*Resource_Package) isResource_Resource() {}

func (*Resource_Message) isResource_Resource() {}

func (*Resource_File) isResource_Resource() {}

func (m *Resource) GetResource() isResource_Resource {
	if m != nil {
		return m.Resource
	}
	return nil
}

func (m *Resource) GetPackage() *Package {
	if x, ok := m.GetResource().(*Resource_Package); ok {
		return x.Package
	}
	return nil
}

func (m *Resource) GetMessage() []byte {
	if x, ok := m.GetResource().(*Resource_Message); ok {
		return x.Message
	}
	return nil
}

func (m *Resource) GetFile() *File {
	if x, ok := m.GetResource().(*Resource_File); ok {
		return x.File
	}
	return nil
}

// XXX_OneofWrappers is for the internal use of the proto package.
func (*Resource) XXX_OneofWrappers() []interface{} {
	return []interface{}{
		(*Resource_Package)(nil),
		(*Resource_Message)(nil),
		(*Resource_File)(nil),
	}
}

func init() {
	proto.RegisterType((*Package)(nil), "main.Package")
	proto.RegisterType((*File)(nil), "main.File")
	proto.RegisterType((*Resource)(nil), "main.Resource")
}

func init() { proto.RegisterFile("pkgs.proto", fileDescriptor_21a791bdd2cceefd) }

var fileDescriptor_21a791bdd2cceefd = []byte{
	// 292 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0x5c, 0x91, 0xbf, 0x6e, 0xb3, 0x30,
	0x14, 0xc5, 0xe3, 0x40, 0xf8, 0x73, 0xf3, 0x7d, 0x1d, 0xac, 0xaa, 0xb2, 0x32, 0x21, 0x26, 0xba,
	0x30, 0xa4, 0x6f, 0xd0, 0xa1, 0xca, 0x50, 0xa9, 0x95, 0xdf, 0xc0, 0xc0, 0x25, 0x72, 0x83, 0x03,
	0x32, 0xa6, 0xea, 0xd6, 0xc7, 0xed, 0x6b, 0x54, 0x36, 0x26, 0x89, 0xba, 0xf1, 0xe3, 0x5c, 0x1f,
	0x1f, 0x9f, 0x0b, 0x30, 0x9c, 0x8e, 0x63, 0x39, 0xe8, 0xde, 0xf4, 0x34, 0x54, 0x42, 0x9e, 0xf3,
	0x1f, 0x02, 0xf1, 0xbb, 0xa8, 0x4f, 0xe2, 0x88, 0x74, 0x07, 0x89, 0xc6, 0xb1, 0x9f, 0x74, 0x8d,
	0x8c, 0x64, 0xa4, 0x48, 0xf9, 0x85, 0xe9, 0x1d, 0xac, 0x65, 0xc3, 0xd6, 0x19, 0x29, 0xfe, 0xf1,
	0xb5, 0x6c, 0x28, 0x83, 0x78, 0x9c, 0xaa, 0x0f, 0xac, 0x0d, 0x0b, 0xdc, 0xe8, 0x82, 0xf4, 0x1e,
	0x36, 0x9f, 0xa2, 0x9b, 0x90, 0x85, 0x6e, 0x78, 0x06, 0xfa, 0x00, 0x11, 0x7e, 0x19, 0x3c, 0x1b,
	0xb6, 0xc9, 0x48, 0x11, 0x72, 0x4f, 0xd6, 0xa7, 0xd6, 0x28, 0x0c, 0x36, 0x2c, 0x9a, 0x7d, 0x3c,
	0xda, 0x34, 0xaa, 0x6f, 0x64, 0x2b, 0xb1, 0x61, 0xf1, 0x9c, 0x66, 0x61, 0xeb, 0xa6, 0x50, 0x55,
	0xa8, 0x59, 0x92, 0x05, 0x45, 0xca, 0x3d, 0xb9, 0x33, 0xee, 0xeb, 0xad, 0x65, 0xa9, 0x53, 0x2e,
	0x9c, 0xbf, 0x42, 0xf8, 0x22, 0x3b, 0xbc, 0xe6, 0x23, 0x7f, 0xf2, 0xb5, 0xbd, 0x56, 0xc2, 0xb8,
	0x37, 0xa6, 0xdc, 0xd3, 0x4d, 0xee, 0xe0, 0x36, 0x77, 0xfe, 0x0d, 0x09, 0x5f, 0xba, 0x79, 0x84,
	0x78, 0x98, 0x2b, 0x74, 0x9e, 0xdb, 0xfd, 0xff, 0xd2, 0x76, 0x5b, 0xfa, 0x5e, 0x0f, 0x2b, 0xbe,
	0xe8, 0x74, 0x07, 0xb1, 0xc2, 0x71, 0xb4, 0xa3, 0xae, 0x4b, 0xab, 0xf9, 0x1f, 0x34, 0x83, 0xb0,
	0x95, 0x1d, 0xba, 0x8b, 0xb6, 0x7b, 0x98, 0x3d, 0x6c, 0xe4, 0xc3, 0x8a, 0x3b, 0xe5, 0x19, 0xae,
	0x0b, 0xaa, 0x22, 0xb7, 0xc5, 0xa7, 0xdf, 0x00, 0x00, 0x00, 0xff, 0xff, 0xa2, 0x5d, 0x5f, 0x36,
	0xd3, 0x01, 0x00, 0x00,
}
