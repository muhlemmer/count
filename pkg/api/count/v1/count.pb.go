// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.28.1
// 	protoc        (unknown)
// source: count/v1/count.proto

package countv1

import (
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
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

// Method defines all HTTP standard method types.
// GRPC value can be used when a request is received
// over a gRPC connection.
type Method int32

const (
	Method_UNSPECIFIED Method = 0
	// HTTP request methods
	Method_CONNECT Method = 1
	Method_DELETE  Method = 2
	Method_GET     Method = 3
	Method_HEAD    Method = 4
	Method_OPTIONS Method = 5
	Method_POST    Method = 6
	Method_PUT     Method = 7
	Method_TRACE   Method = 8
	// gRPC requests
	Method_GRPC Method = 100
)

// Enum value maps for Method.
var (
	Method_name = map[int32]string{
		0:   "UNSPECIFIED",
		1:   "CONNECT",
		2:   "DELETE",
		3:   "GET",
		4:   "HEAD",
		5:   "OPTIONS",
		6:   "POST",
		7:   "PUT",
		8:   "TRACE",
		100: "GRPC",
	}
	Method_value = map[string]int32{
		"UNSPECIFIED": 0,
		"CONNECT":     1,
		"DELETE":      2,
		"GET":         3,
		"HEAD":        4,
		"OPTIONS":     5,
		"POST":        6,
		"PUT":         7,
		"TRACE":       8,
		"GRPC":        100,
	}
)

func (x Method) Enum() *Method {
	p := new(Method)
	*p = x
	return p
}

func (x Method) String() string {
	return protoimpl.X.EnumStringOf(x.Descriptor(), protoreflect.EnumNumber(x))
}

func (Method) Descriptor() protoreflect.EnumDescriptor {
	return file_count_v1_count_proto_enumTypes[0].Descriptor()
}

func (Method) Type() protoreflect.EnumType {
	return &file_count_v1_count_proto_enumTypes[0]
}

func (x Method) Number() protoreflect.EnumNumber {
	return protoreflect.EnumNumber(x)
}

// Deprecated: Use Method.Descriptor instead.
func (Method) EnumDescriptor() ([]byte, []int) {
	return file_count_v1_count_proto_rawDescGZIP(), []int{0}
}

// AddRequest is a datapoint for request counting.
type AddRequest struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// Method of the request can be a HTTP method or GRPC.
	Method Method `protobuf:"varint,1,opt,name=method,proto3,enum=count.v1.Method" json:"method,omitempty"`
	// Path of the request, or name of the gRPC method.
	// This value is required.
	Path string `protobuf:"bytes,2,opt,name=path,proto3" json:"path,omitempty"`
	// Timestamp of the request, using the server's wall clock.
	// This value is required.
	RequestTimestamp *timestamppb.Timestamp `protobuf:"bytes,3,opt,name=request_timestamp,json=requestTimestamp,proto3" json:"request_timestamp,omitempty"`
}

func (x *AddRequest) Reset() {
	*x = AddRequest{}
	if protoimpl.UnsafeEnabled {
		mi := &file_count_v1_count_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *AddRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*AddRequest) ProtoMessage() {}

func (x *AddRequest) ProtoReflect() protoreflect.Message {
	mi := &file_count_v1_count_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use AddRequest.ProtoReflect.Descriptor instead.
func (*AddRequest) Descriptor() ([]byte, []int) {
	return file_count_v1_count_proto_rawDescGZIP(), []int{0}
}

func (x *AddRequest) GetMethod() Method {
	if x != nil {
		return x.Method
	}
	return Method_UNSPECIFIED
}

func (x *AddRequest) GetPath() string {
	if x != nil {
		return x.Path
	}
	return ""
}

func (x *AddRequest) GetRequestTimestamp() *timestamppb.Timestamp {
	if x != nil {
		return x.RequestTimestamp
	}
	return nil
}

type AddResponse struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields
}

func (x *AddResponse) Reset() {
	*x = AddResponse{}
	if protoimpl.UnsafeEnabled {
		mi := &file_count_v1_count_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *AddResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*AddResponse) ProtoMessage() {}

func (x *AddResponse) ProtoReflect() protoreflect.Message {
	mi := &file_count_v1_count_proto_msgTypes[1]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use AddResponse.ProtoReflect.Descriptor instead.
func (*AddResponse) Descriptor() ([]byte, []int) {
	return file_count_v1_count_proto_rawDescGZIP(), []int{1}
}

// CountDailyTotalsRequest determines data points
// to be counted.
type CountDailyTotalsRequest struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// date where the timestamp falls in.
	// The timestamp is rounded down to whole days.
	// So hours, minutes, seconds etc are discarded.
	Date *timestamppb.Timestamp `protobuf:"bytes,1,opt,name=date,proto3" json:"date,omitempty"`
}

func (x *CountDailyTotalsRequest) Reset() {
	*x = CountDailyTotalsRequest{}
	if protoimpl.UnsafeEnabled {
		mi := &file_count_v1_count_proto_msgTypes[2]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *CountDailyTotalsRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*CountDailyTotalsRequest) ProtoMessage() {}

func (x *CountDailyTotalsRequest) ProtoReflect() protoreflect.Message {
	mi := &file_count_v1_count_proto_msgTypes[2]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use CountDailyTotalsRequest.ProtoReflect.Descriptor instead.
func (*CountDailyTotalsRequest) Descriptor() ([]byte, []int) {
	return file_count_v1_count_proto_rawDescGZIP(), []int{2}
}

func (x *CountDailyTotalsRequest) GetDate() *timestamppb.Timestamp {
	if x != nil {
		return x.Date
	}
	return nil
}

type MethodCount struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// Method of the request can be a HTTP method or GRPC.
	Method Method `protobuf:"varint,1,opt,name=method,proto3,enum=count.v1.Method" json:"method,omitempty"`
	// Path of the request, or name of the gRPC method.
	Path string `protobuf:"bytes,2,opt,name=path,proto3" json:"path,omitempty"`
	// Amuont of times each method and path pair was requested.
	Count int32 `protobuf:"varint,3,opt,name=count,proto3" json:"count,omitempty"`
}

func (x *MethodCount) Reset() {
	*x = MethodCount{}
	if protoimpl.UnsafeEnabled {
		mi := &file_count_v1_count_proto_msgTypes[3]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *MethodCount) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*MethodCount) ProtoMessage() {}

func (x *MethodCount) ProtoReflect() protoreflect.Message {
	mi := &file_count_v1_count_proto_msgTypes[3]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use MethodCount.ProtoReflect.Descriptor instead.
func (*MethodCount) Descriptor() ([]byte, []int) {
	return file_count_v1_count_proto_rawDescGZIP(), []int{3}
}

func (x *MethodCount) GetMethod() Method {
	if x != nil {
		return x.Method
	}
	return Method_UNSPECIFIED
}

func (x *MethodCount) GetPath() string {
	if x != nil {
		return x.Path
	}
	return ""
}

func (x *MethodCount) GetCount() int32 {
	if x != nil {
		return x.Count
	}
	return 0
}

type CountDailyTotalsResponse struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	MethodCounts []*MethodCount `protobuf:"bytes,1,rep,name=method_counts,json=methodCounts,proto3" json:"method_counts,omitempty"`
}

func (x *CountDailyTotalsResponse) Reset() {
	*x = CountDailyTotalsResponse{}
	if protoimpl.UnsafeEnabled {
		mi := &file_count_v1_count_proto_msgTypes[4]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *CountDailyTotalsResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*CountDailyTotalsResponse) ProtoMessage() {}

func (x *CountDailyTotalsResponse) ProtoReflect() protoreflect.Message {
	mi := &file_count_v1_count_proto_msgTypes[4]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use CountDailyTotalsResponse.ProtoReflect.Descriptor instead.
func (*CountDailyTotalsResponse) Descriptor() ([]byte, []int) {
	return file_count_v1_count_proto_rawDescGZIP(), []int{4}
}

func (x *CountDailyTotalsResponse) GetMethodCounts() []*MethodCount {
	if x != nil {
		return x.MethodCounts
	}
	return nil
}

var File_count_v1_count_proto protoreflect.FileDescriptor

var file_count_v1_count_proto_rawDesc = []byte{
	0x0a, 0x14, 0x63, 0x6f, 0x75, 0x6e, 0x74, 0x2f, 0x76, 0x31, 0x2f, 0x63, 0x6f, 0x75, 0x6e, 0x74,
	0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x12, 0x08, 0x63, 0x6f, 0x75, 0x6e, 0x74, 0x2e, 0x76, 0x31,
	0x1a, 0x1f, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75,
	0x66, 0x2f, 0x74, 0x69, 0x6d, 0x65, 0x73, 0x74, 0x61, 0x6d, 0x70, 0x2e, 0x70, 0x72, 0x6f, 0x74,
	0x6f, 0x22, 0x93, 0x01, 0x0a, 0x0a, 0x41, 0x64, 0x64, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74,
	0x12, 0x28, 0x0a, 0x06, 0x6d, 0x65, 0x74, 0x68, 0x6f, 0x64, 0x18, 0x01, 0x20, 0x01, 0x28, 0x0e,
	0x32, 0x10, 0x2e, 0x63, 0x6f, 0x75, 0x6e, 0x74, 0x2e, 0x76, 0x31, 0x2e, 0x4d, 0x65, 0x74, 0x68,
	0x6f, 0x64, 0x52, 0x06, 0x6d, 0x65, 0x74, 0x68, 0x6f, 0x64, 0x12, 0x12, 0x0a, 0x04, 0x70, 0x61,
	0x74, 0x68, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x04, 0x70, 0x61, 0x74, 0x68, 0x12, 0x47,
	0x0a, 0x11, 0x72, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x5f, 0x74, 0x69, 0x6d, 0x65, 0x73, 0x74,
	0x61, 0x6d, 0x70, 0x18, 0x03, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x1a, 0x2e, 0x67, 0x6f, 0x6f, 0x67,
	0x6c, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2e, 0x54, 0x69, 0x6d, 0x65,
	0x73, 0x74, 0x61, 0x6d, 0x70, 0x52, 0x10, 0x72, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x54, 0x69,
	0x6d, 0x65, 0x73, 0x74, 0x61, 0x6d, 0x70, 0x22, 0x0d, 0x0a, 0x0b, 0x41, 0x64, 0x64, 0x52, 0x65,
	0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x22, 0x49, 0x0a, 0x17, 0x43, 0x6f, 0x75, 0x6e, 0x74, 0x44,
	0x61, 0x69, 0x6c, 0x79, 0x54, 0x6f, 0x74, 0x61, 0x6c, 0x73, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73,
	0x74, 0x12, 0x2e, 0x0a, 0x04, 0x64, 0x61, 0x74, 0x65, 0x18, 0x01, 0x20, 0x01, 0x28, 0x0b, 0x32,
	0x1a, 0x2e, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75,
	0x66, 0x2e, 0x54, 0x69, 0x6d, 0x65, 0x73, 0x74, 0x61, 0x6d, 0x70, 0x52, 0x04, 0x64, 0x61, 0x74,
	0x65, 0x22, 0x61, 0x0a, 0x0b, 0x4d, 0x65, 0x74, 0x68, 0x6f, 0x64, 0x43, 0x6f, 0x75, 0x6e, 0x74,
	0x12, 0x28, 0x0a, 0x06, 0x6d, 0x65, 0x74, 0x68, 0x6f, 0x64, 0x18, 0x01, 0x20, 0x01, 0x28, 0x0e,
	0x32, 0x10, 0x2e, 0x63, 0x6f, 0x75, 0x6e, 0x74, 0x2e, 0x76, 0x31, 0x2e, 0x4d, 0x65, 0x74, 0x68,
	0x6f, 0x64, 0x52, 0x06, 0x6d, 0x65, 0x74, 0x68, 0x6f, 0x64, 0x12, 0x12, 0x0a, 0x04, 0x70, 0x61,
	0x74, 0x68, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x04, 0x70, 0x61, 0x74, 0x68, 0x12, 0x14,
	0x0a, 0x05, 0x63, 0x6f, 0x75, 0x6e, 0x74, 0x18, 0x03, 0x20, 0x01, 0x28, 0x05, 0x52, 0x05, 0x63,
	0x6f, 0x75, 0x6e, 0x74, 0x22, 0x56, 0x0a, 0x18, 0x43, 0x6f, 0x75, 0x6e, 0x74, 0x44, 0x61, 0x69,
	0x6c, 0x79, 0x54, 0x6f, 0x74, 0x61, 0x6c, 0x73, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65,
	0x12, 0x3a, 0x0a, 0x0d, 0x6d, 0x65, 0x74, 0x68, 0x6f, 0x64, 0x5f, 0x63, 0x6f, 0x75, 0x6e, 0x74,
	0x73, 0x18, 0x01, 0x20, 0x03, 0x28, 0x0b, 0x32, 0x15, 0x2e, 0x63, 0x6f, 0x75, 0x6e, 0x74, 0x2e,
	0x76, 0x31, 0x2e, 0x4d, 0x65, 0x74, 0x68, 0x6f, 0x64, 0x43, 0x6f, 0x75, 0x6e, 0x74, 0x52, 0x0c,
	0x6d, 0x65, 0x74, 0x68, 0x6f, 0x64, 0x43, 0x6f, 0x75, 0x6e, 0x74, 0x73, 0x2a, 0x7a, 0x0a, 0x06,
	0x4d, 0x65, 0x74, 0x68, 0x6f, 0x64, 0x12, 0x0f, 0x0a, 0x0b, 0x55, 0x4e, 0x53, 0x50, 0x45, 0x43,
	0x49, 0x46, 0x49, 0x45, 0x44, 0x10, 0x00, 0x12, 0x0b, 0x0a, 0x07, 0x43, 0x4f, 0x4e, 0x4e, 0x45,
	0x43, 0x54, 0x10, 0x01, 0x12, 0x0a, 0x0a, 0x06, 0x44, 0x45, 0x4c, 0x45, 0x54, 0x45, 0x10, 0x02,
	0x12, 0x07, 0x0a, 0x03, 0x47, 0x45, 0x54, 0x10, 0x03, 0x12, 0x08, 0x0a, 0x04, 0x48, 0x45, 0x41,
	0x44, 0x10, 0x04, 0x12, 0x0b, 0x0a, 0x07, 0x4f, 0x50, 0x54, 0x49, 0x4f, 0x4e, 0x53, 0x10, 0x05,
	0x12, 0x08, 0x0a, 0x04, 0x50, 0x4f, 0x53, 0x54, 0x10, 0x06, 0x12, 0x07, 0x0a, 0x03, 0x50, 0x55,
	0x54, 0x10, 0x07, 0x12, 0x09, 0x0a, 0x05, 0x54, 0x52, 0x41, 0x43, 0x45, 0x10, 0x08, 0x12, 0x08,
	0x0a, 0x04, 0x47, 0x52, 0x50, 0x43, 0x10, 0x64, 0x32, 0xa3, 0x01, 0x0a, 0x0c, 0x43, 0x6f, 0x75,
	0x6e, 0x74, 0x53, 0x65, 0x72, 0x76, 0x69, 0x63, 0x65, 0x12, 0x36, 0x0a, 0x03, 0x41, 0x64, 0x64,
	0x12, 0x14, 0x2e, 0x63, 0x6f, 0x75, 0x6e, 0x74, 0x2e, 0x76, 0x31, 0x2e, 0x41, 0x64, 0x64, 0x52,
	0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x1a, 0x15, 0x2e, 0x63, 0x6f, 0x75, 0x6e, 0x74, 0x2e, 0x76,
	0x31, 0x2e, 0x41, 0x64, 0x64, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x22, 0x00, 0x28,
	0x01, 0x12, 0x5b, 0x0a, 0x10, 0x43, 0x6f, 0x75, 0x6e, 0x74, 0x44, 0x61, 0x69, 0x6c, 0x79, 0x54,
	0x6f, 0x74, 0x61, 0x6c, 0x73, 0x12, 0x21, 0x2e, 0x63, 0x6f, 0x75, 0x6e, 0x74, 0x2e, 0x76, 0x31,
	0x2e, 0x43, 0x6f, 0x75, 0x6e, 0x74, 0x44, 0x61, 0x69, 0x6c, 0x79, 0x54, 0x6f, 0x74, 0x61, 0x6c,
	0x73, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x1a, 0x22, 0x2e, 0x63, 0x6f, 0x75, 0x6e, 0x74,
	0x2e, 0x76, 0x31, 0x2e, 0x43, 0x6f, 0x75, 0x6e, 0x74, 0x44, 0x61, 0x69, 0x6c, 0x79, 0x54, 0x6f,
	0x74, 0x61, 0x6c, 0x73, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x22, 0x00, 0x42, 0x90,
	0x01, 0x0a, 0x0c, 0x63, 0x6f, 0x6d, 0x2e, 0x63, 0x6f, 0x75, 0x6e, 0x74, 0x2e, 0x76, 0x31, 0x42,
	0x0a, 0x43, 0x6f, 0x75, 0x6e, 0x74, 0x50, 0x72, 0x6f, 0x74, 0x6f, 0x50, 0x01, 0x5a, 0x33, 0x67,
	0x69, 0x74, 0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f, 0x6d, 0x2f, 0x6d, 0x75, 0x68, 0x6c, 0x65, 0x6d,
	0x6d, 0x65, 0x72, 0x2f, 0x63, 0x6f, 0x75, 0x6e, 0x74, 0x2f, 0x70, 0x6b, 0x67, 0x2f, 0x61, 0x70,
	0x69, 0x2f, 0x63, 0x6f, 0x75, 0x6e, 0x74, 0x2f, 0x76, 0x31, 0x3b, 0x63, 0x6f, 0x75, 0x6e, 0x74,
	0x76, 0x31, 0xa2, 0x02, 0x03, 0x43, 0x58, 0x58, 0xaa, 0x02, 0x08, 0x43, 0x6f, 0x75, 0x6e, 0x74,
	0x2e, 0x56, 0x31, 0xca, 0x02, 0x08, 0x43, 0x6f, 0x75, 0x6e, 0x74, 0x5c, 0x56, 0x31, 0xe2, 0x02,
	0x14, 0x43, 0x6f, 0x75, 0x6e, 0x74, 0x5c, 0x56, 0x31, 0x5c, 0x47, 0x50, 0x42, 0x4d, 0x65, 0x74,
	0x61, 0x64, 0x61, 0x74, 0x61, 0xea, 0x02, 0x09, 0x43, 0x6f, 0x75, 0x6e, 0x74, 0x3a, 0x3a, 0x56,
	0x31, 0x62, 0x06, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_count_v1_count_proto_rawDescOnce sync.Once
	file_count_v1_count_proto_rawDescData = file_count_v1_count_proto_rawDesc
)

func file_count_v1_count_proto_rawDescGZIP() []byte {
	file_count_v1_count_proto_rawDescOnce.Do(func() {
		file_count_v1_count_proto_rawDescData = protoimpl.X.CompressGZIP(file_count_v1_count_proto_rawDescData)
	})
	return file_count_v1_count_proto_rawDescData
}

var file_count_v1_count_proto_enumTypes = make([]protoimpl.EnumInfo, 1)
var file_count_v1_count_proto_msgTypes = make([]protoimpl.MessageInfo, 5)
var file_count_v1_count_proto_goTypes = []interface{}{
	(Method)(0),                      // 0: count.v1.Method
	(*AddRequest)(nil),               // 1: count.v1.AddRequest
	(*AddResponse)(nil),              // 2: count.v1.AddResponse
	(*CountDailyTotalsRequest)(nil),  // 3: count.v1.CountDailyTotalsRequest
	(*MethodCount)(nil),              // 4: count.v1.MethodCount
	(*CountDailyTotalsResponse)(nil), // 5: count.v1.CountDailyTotalsResponse
	(*timestamppb.Timestamp)(nil),    // 6: google.protobuf.Timestamp
}
var file_count_v1_count_proto_depIdxs = []int32{
	0, // 0: count.v1.AddRequest.method:type_name -> count.v1.Method
	6, // 1: count.v1.AddRequest.request_timestamp:type_name -> google.protobuf.Timestamp
	6, // 2: count.v1.CountDailyTotalsRequest.date:type_name -> google.protobuf.Timestamp
	0, // 3: count.v1.MethodCount.method:type_name -> count.v1.Method
	4, // 4: count.v1.CountDailyTotalsResponse.method_counts:type_name -> count.v1.MethodCount
	1, // 5: count.v1.CountService.Add:input_type -> count.v1.AddRequest
	3, // 6: count.v1.CountService.CountDailyTotals:input_type -> count.v1.CountDailyTotalsRequest
	2, // 7: count.v1.CountService.Add:output_type -> count.v1.AddResponse
	5, // 8: count.v1.CountService.CountDailyTotals:output_type -> count.v1.CountDailyTotalsResponse
	7, // [7:9] is the sub-list for method output_type
	5, // [5:7] is the sub-list for method input_type
	5, // [5:5] is the sub-list for extension type_name
	5, // [5:5] is the sub-list for extension extendee
	0, // [0:5] is the sub-list for field type_name
}

func init() { file_count_v1_count_proto_init() }
func file_count_v1_count_proto_init() {
	if File_count_v1_count_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_count_v1_count_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*AddRequest); i {
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
		file_count_v1_count_proto_msgTypes[1].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*AddResponse); i {
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
		file_count_v1_count_proto_msgTypes[2].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*CountDailyTotalsRequest); i {
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
		file_count_v1_count_proto_msgTypes[3].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*MethodCount); i {
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
		file_count_v1_count_proto_msgTypes[4].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*CountDailyTotalsResponse); i {
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
			RawDescriptor: file_count_v1_count_proto_rawDesc,
			NumEnums:      1,
			NumMessages:   5,
			NumExtensions: 0,
			NumServices:   1,
		},
		GoTypes:           file_count_v1_count_proto_goTypes,
		DependencyIndexes: file_count_v1_count_proto_depIdxs,
		EnumInfos:         file_count_v1_count_proto_enumTypes,
		MessageInfos:      file_count_v1_count_proto_msgTypes,
	}.Build()
	File_count_v1_count_proto = out.File
	file_count_v1_count_proto_rawDesc = nil
	file_count_v1_count_proto_goTypes = nil
	file_count_v1_count_proto_depIdxs = nil
}
