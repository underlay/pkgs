// Code generated by protoc-gen-go. DO NOT EDIT.
// source: types/types.proto

package types

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
	RevisionOf           []byte   `protobuf:"bytes,8,opt,name=revisionOf,proto3" json:"revisionOf,omitempty"`
	RevisionOfSubject    string   `protobuf:"bytes,9,opt,name=revisionOfSubject,proto3" json:"revisionOfSubject,omitempty"`
	Member               []string `protobuf:"bytes,10,rep,name=member,proto3" json:"member,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *Package) Reset()         { *m = Package{} }
func (m *Package) String() string { return proto.CompactTextString(m) }
func (*Package) ProtoMessage()    {}
func (*Package) Descriptor() ([]byte, []int) {
	return fileDescriptor_2c0f90c600ad7e2e, []int{0}
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

func (m *Package) GetRevisionOf() []byte {
	if m != nil {
		return m.RevisionOf
	}
	return nil
}

func (m *Package) GetRevisionOfSubject() string {
	if m != nil {
		return m.RevisionOfSubject
	}
	return ""
}

func (m *Package) GetMember() []string {
	if m != nil {
		return m.Member
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
	return fileDescriptor_2c0f90c600ad7e2e, []int{1}
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

func init() {
	proto.RegisterType((*Package)(nil), "types.Package")
	proto.RegisterType((*File)(nil), "types.File")
}

func init() { proto.RegisterFile("types/types.proto", fileDescriptor_2c0f90c600ad7e2e) }

var fileDescriptor_2c0f90c600ad7e2e = []byte{
	// 249 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0x64, 0x90, 0xcf, 0x4a, 0xc4, 0x30,
	0x10, 0xc6, 0x49, 0xff, 0x6e, 0x07, 0x11, 0x36, 0x88, 0x0c, 0x1e, 0xa4, 0xec, 0xa9, 0x07, 0xd1,
	0x83, 0xef, 0xe0, 0x49, 0x50, 0xe2, 0x13, 0xa4, 0xcd, 0x54, 0xa2, 0xdb, 0xcd, 0x92, 0xa4, 0x8b,
	0x3e, 0x8a, 0x6f, 0x2b, 0x4d, 0xb3, 0x6e, 0x71, 0x2f, 0x81, 0x5f, 0x32, 0x7c, 0xf9, 0x7e, 0x03,
	0x6b, 0xff, 0xbd, 0x27, 0xf7, 0x10, 0xce, 0xfb, 0xbd, 0x35, 0xde, 0xf0, 0x3c, 0xc0, 0xe6, 0x27,
	0x81, 0xf2, 0x55, 0x76, 0x9f, 0xf2, 0x9d, 0xf8, 0x0d, 0xac, 0x2c, 0x39, 0x33, 0xda, 0x8e, 0x90,
	0xd5, 0xac, 0xa9, 0xc4, 0x1f, 0xf3, 0x4b, 0x48, 0xb4, 0xc2, 0xa4, 0x66, 0xcd, 0x85, 0x48, 0xb4,
	0xe2, 0x08, 0xa5, 0x1b, 0xdb, 0x0f, 0xea, 0x3c, 0xa6, 0x61, 0xf4, 0x88, 0xfc, 0x0a, 0xf2, 0x83,
	0xdc, 0x8e, 0x84, 0x59, 0x18, 0x9e, 0x81, 0x5f, 0x43, 0x41, 0x5f, 0x9e, 0x76, 0x1e, 0xf3, 0x9a,
	0x35, 0x99, 0x88, 0x34, 0xe5, 0x74, 0x96, 0xa4, 0x27, 0x85, 0xc5, 0x9c, 0x13, 0x71, 0x6a, 0x33,
	0x18, 0xa5, 0x7b, 0x4d, 0x0a, 0xcb, 0xb9, 0xcd, 0x91, 0xf9, 0x2d, 0x80, 0xa5, 0x83, 0x76, 0xda,
	0xec, 0x5e, 0x7a, 0x5c, 0x85, 0x8f, 0x16, 0x37, 0xfc, 0x0e, 0xd6, 0x27, 0x7a, 0x8b, 0x3d, 0xab,
	0x10, 0x72, 0xfe, 0x30, 0x75, 0x1b, 0x68, 0x68, 0xc9, 0x22, 0xd4, 0x69, 0x53, 0x89, 0x48, 0x9b,
	0x67, 0xc8, 0x9e, 0xf4, 0x96, 0x4e, 0x46, 0xec, 0x9f, 0x51, 0x6f, 0xec, 0x20, 0x7d, 0xd8, 0x4a,
	0x25, 0x22, 0x2d, 0x4c, 0xd3, 0xa5, 0x69, 0x5b, 0x84, 0xbd, 0x3f, 0xfe, 0x06, 0x00, 0x00, 0xff,
	0xff, 0x80, 0x10, 0x3d, 0x7c, 0x8c, 0x01, 0x00, 0x00,
}
