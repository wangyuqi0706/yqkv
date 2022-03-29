// Code generated by protoc-gen-gogo. DO NOT EDIT.
// source: session/sessionpb/session.proto

package sessionpb

import (
	fmt "fmt"
	_ "github.com/gogo/protobuf/gogoproto"
	types "github.com/gogo/protobuf/types"
	proto "github.com/golang/protobuf/proto"
	io "io"
	math "math"
	math_bits "math/bits"
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

type Err int32

const (
	OK            Err = 0
	ERR_RETURNED  Err = 1
	ERR_DUPLICATE Err = 2
)

var Err_name = map[int32]string{
	0: "OK",
	1: "ERR_RETURNED",
	2: "ERR_DUPLICATE",
}

var Err_value = map[string]int32{
	"OK":            0,
	"ERR_RETURNED":  1,
	"ERR_DUPLICATE": 2,
}

func (x Err) String() string {
	return proto.EnumName(Err_name, int32(x))
}

func (Err) EnumDescriptor() ([]byte, []int) {
	return fileDescriptor_6898b9fca01a5700, []int{0}
}

type SessionHeader struct {
	ID  int64  `protobuf:"varint,1,opt,name=ID,proto3" json:"ID,omitempty"`
	Seq uint64 `protobuf:"varint,2,opt,name=Seq,proto3" json:"Seq,omitempty"`
}

func (m *SessionHeader) Reset()         { *m = SessionHeader{} }
func (m *SessionHeader) String() string { return proto.CompactTextString(m) }
func (*SessionHeader) ProtoMessage()    {}
func (*SessionHeader) Descriptor() ([]byte, []int) {
	return fileDescriptor_6898b9fca01a5700, []int{0}
}
func (m *SessionHeader) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *SessionHeader) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_SessionHeader.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalToSizedBuffer(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (m *SessionHeader) XXX_Merge(src proto.Message) {
	xxx_messageInfo_SessionHeader.Merge(m, src)
}
func (m *SessionHeader) XXX_Size() int {
	return m.Size()
}
func (m *SessionHeader) XXX_DiscardUnknown() {
	xxx_messageInfo_SessionHeader.DiscardUnknown(m)
}

var xxx_messageInfo_SessionHeader proto.InternalMessageInfo

type ServerSession struct {
	ID           int64      `protobuf:"varint,1,opt,name=ID,proto3" json:"ID,omitempty"`
	ProcessedSeq uint64     `protobuf:"varint,2,opt,name=ProcessedSeq,proto3" json:"ProcessedSeq,omitempty"`
	LastResponse *types.Any `protobuf:"bytes,3,opt,name=LastResponse,proto3" json:"LastResponse,omitempty"`
}

func (m *ServerSession) Reset()         { *m = ServerSession{} }
func (m *ServerSession) String() string { return proto.CompactTextString(m) }
func (*ServerSession) ProtoMessage()    {}
func (*ServerSession) Descriptor() ([]byte, []int) {
	return fileDescriptor_6898b9fca01a5700, []int{1}
}
func (m *ServerSession) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *ServerSession) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_ServerSession.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalToSizedBuffer(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (m *ServerSession) XXX_Merge(src proto.Message) {
	xxx_messageInfo_ServerSession.Merge(m, src)
}
func (m *ServerSession) XXX_Size() int {
	return m.Size()
}
func (m *ServerSession) XXX_DiscardUnknown() {
	xxx_messageInfo_ServerSession.DiscardUnknown(m)
}

var xxx_messageInfo_ServerSession proto.InternalMessageInfo

func init() {
	proto.RegisterEnum("sessionpb.Err", Err_name, Err_value)
	proto.RegisterType((*SessionHeader)(nil), "sessionpb.SessionHeader")
	proto.RegisterType((*ServerSession)(nil), "sessionpb.ServerSession")
}

func init() { proto.RegisterFile("session/sessionpb/session.proto", fileDescriptor_6898b9fca01a5700) }

var fileDescriptor_6898b9fca01a5700 = []byte{
	// 325 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0xe2, 0x92, 0x2f, 0x4e, 0x2d, 0x2e,
	0xce, 0xcc, 0xcf, 0xd3, 0x87, 0xd2, 0x05, 0x49, 0x30, 0x96, 0x5e, 0x41, 0x51, 0x7e, 0x49, 0xbe,
	0x10, 0x27, 0x5c, 0x42, 0x4a, 0x37, 0x3d, 0xb3, 0x24, 0xa3, 0x34, 0x49, 0x2f, 0x39, 0x3f, 0x57,
	0x3f, 0x3d, 0x3f, 0x3d, 0x5f, 0x1f, 0xac, 0x22, 0xa9, 0x34, 0x0d, 0xcc, 0x03, 0x73, 0xc0, 0x2c,
	0x88, 0x4e, 0x29, 0xc9, 0xf4, 0xfc, 0xfc, 0xf4, 0x9c, 0x54, 0x84, 0xaa, 0xc4, 0xbc, 0x4a, 0x88,
	0x94, 0x92, 0x21, 0x17, 0x6f, 0x30, 0xc4, 0x58, 0x8f, 0xd4, 0xc4, 0x94, 0xd4, 0x22, 0x21, 0x3e,
	0x2e, 0x26, 0x4f, 0x17, 0x09, 0x46, 0x05, 0x46, 0x0d, 0xe6, 0x20, 0x26, 0x4f, 0x17, 0x21, 0x01,
	0x2e, 0xe6, 0xe0, 0xd4, 0x42, 0x09, 0x26, 0x05, 0x46, 0x0d, 0x96, 0x20, 0x10, 0x53, 0xa9, 0x16,
	0xa4, 0xa5, 0xa8, 0x2c, 0xb5, 0x08, 0xaa, 0x11, 0x43, 0x8b, 0x12, 0x17, 0x4f, 0x40, 0x51, 0x7e,
	0x72, 0x6a, 0x71, 0x71, 0x6a, 0x0a, 0x42, 0x2f, 0x8a, 0x98, 0x90, 0x05, 0x17, 0x8f, 0x4f, 0x62,
	0x71, 0x49, 0x50, 0x6a, 0x71, 0x41, 0x7e, 0x5e, 0x71, 0xaa, 0x04, 0xb3, 0x02, 0xa3, 0x06, 0xb7,
	0x91, 0x88, 0x1e, 0xc4, 0xa5, 0x7a, 0x30, 0x97, 0xea, 0x39, 0xe6, 0x55, 0x06, 0xa1, 0xa8, 0xd4,
	0x32, 0xe2, 0x62, 0x76, 0x2d, 0x2a, 0x12, 0x62, 0xe3, 0x62, 0xf2, 0xf7, 0x16, 0x60, 0x10, 0x12,
	0xe0, 0xe2, 0x71, 0x0d, 0x0a, 0x8a, 0x0f, 0x72, 0x0d, 0x09, 0x0d, 0xf2, 0x73, 0x75, 0x11, 0x60,
	0x14, 0x12, 0xe4, 0xe2, 0x05, 0x89, 0xb8, 0x84, 0x06, 0xf8, 0x78, 0x3a, 0x3b, 0x86, 0xb8, 0x0a,
	0x30, 0x39, 0x85, 0x9c, 0x78, 0x28, 0xc7, 0x70, 0xe1, 0xa1, 0x1c, 0xc3, 0x89, 0x47, 0x72, 0x8c,
	0x17, 0x1e, 0xc9, 0x31, 0x3e, 0x78, 0x24, 0xc7, 0x38, 0xe1, 0xb1, 0x1c, 0xc3, 0x8c, 0xc7, 0x72,
	0x0c, 0x17, 0x1e, 0xcb, 0x31, 0xdc, 0x78, 0x2c, 0xc7, 0x10, 0xa5, 0x87, 0x14, 0xaa, 0xe5, 0x89,
	0x79, 0xe9, 0x95, 0xa5, 0x85, 0x99, 0x06, 0xe6, 0x06, 0x66, 0xfa, 0x95, 0x85, 0xd9, 0x65, 0xfa,
	0x18, 0xd1, 0x93, 0xc4, 0x06, 0x76, 0xa5, 0x31, 0x20, 0x00, 0x00, 0xff, 0xff, 0xe4, 0x99, 0xa1,
	0x05, 0xba, 0x01, 0x00, 0x00,
}

func (m *SessionHeader) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *SessionHeader) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *SessionHeader) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	if m.Seq != 0 {
		i = encodeVarintSession(dAtA, i, uint64(m.Seq))
		i--
		dAtA[i] = 0x10
	}
	if m.ID != 0 {
		i = encodeVarintSession(dAtA, i, uint64(m.ID))
		i--
		dAtA[i] = 0x8
	}
	return len(dAtA) - i, nil
}

func (m *ServerSession) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *ServerSession) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *ServerSession) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	if m.LastResponse != nil {
		{
			size, err := m.LastResponse.MarshalToSizedBuffer(dAtA[:i])
			if err != nil {
				return 0, err
			}
			i -= size
			i = encodeVarintSession(dAtA, i, uint64(size))
		}
		i--
		dAtA[i] = 0x1a
	}
	if m.ProcessedSeq != 0 {
		i = encodeVarintSession(dAtA, i, uint64(m.ProcessedSeq))
		i--
		dAtA[i] = 0x10
	}
	if m.ID != 0 {
		i = encodeVarintSession(dAtA, i, uint64(m.ID))
		i--
		dAtA[i] = 0x8
	}
	return len(dAtA) - i, nil
}

func encodeVarintSession(dAtA []byte, offset int, v uint64) int {
	offset -= sovSession(v)
	base := offset
	for v >= 1<<7 {
		dAtA[offset] = uint8(v&0x7f | 0x80)
		v >>= 7
		offset++
	}
	dAtA[offset] = uint8(v)
	return base
}
func (m *SessionHeader) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	if m.ID != 0 {
		n += 1 + sovSession(uint64(m.ID))
	}
	if m.Seq != 0 {
		n += 1 + sovSession(uint64(m.Seq))
	}
	return n
}

func (m *ServerSession) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	if m.ID != 0 {
		n += 1 + sovSession(uint64(m.ID))
	}
	if m.ProcessedSeq != 0 {
		n += 1 + sovSession(uint64(m.ProcessedSeq))
	}
	if m.LastResponse != nil {
		l = m.LastResponse.Size()
		n += 1 + l + sovSession(uint64(l))
	}
	return n
}

func sovSession(x uint64) (n int) {
	return (math_bits.Len64(x|1) + 6) / 7
}
func sozSession(x uint64) (n int) {
	return sovSession(uint64((x << 1) ^ uint64((int64(x) >> 63))))
}
func (m *SessionHeader) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowSession
			}
			if iNdEx >= l {
				return io.ErrUnexpectedEOF
			}
			b := dAtA[iNdEx]
			iNdEx++
			wire |= uint64(b&0x7F) << shift
			if b < 0x80 {
				break
			}
		}
		fieldNum := int32(wire >> 3)
		wireType := int(wire & 0x7)
		if wireType == 4 {
			return fmt.Errorf("proto: SessionHeader: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: SessionHeader: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field ID", wireType)
			}
			m.ID = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowSession
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				m.ID |= int64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		case 2:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field Seq", wireType)
			}
			m.Seq = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowSession
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				m.Seq |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		default:
			iNdEx = preIndex
			skippy, err := skipSession(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if (skippy < 0) || (iNdEx+skippy) < 0 {
				return ErrInvalidLengthSession
			}
			if (iNdEx + skippy) > l {
				return io.ErrUnexpectedEOF
			}
			iNdEx += skippy
		}
	}

	if iNdEx > l {
		return io.ErrUnexpectedEOF
	}
	return nil
}
func (m *ServerSession) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowSession
			}
			if iNdEx >= l {
				return io.ErrUnexpectedEOF
			}
			b := dAtA[iNdEx]
			iNdEx++
			wire |= uint64(b&0x7F) << shift
			if b < 0x80 {
				break
			}
		}
		fieldNum := int32(wire >> 3)
		wireType := int(wire & 0x7)
		if wireType == 4 {
			return fmt.Errorf("proto: ServerSession: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: ServerSession: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field ID", wireType)
			}
			m.ID = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowSession
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				m.ID |= int64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		case 2:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field ProcessedSeq", wireType)
			}
			m.ProcessedSeq = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowSession
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				m.ProcessedSeq |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		case 3:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field LastResponse", wireType)
			}
			var msglen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowSession
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				msglen |= int(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			if msglen < 0 {
				return ErrInvalidLengthSession
			}
			postIndex := iNdEx + msglen
			if postIndex < 0 {
				return ErrInvalidLengthSession
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			if m.LastResponse == nil {
				m.LastResponse = &types.Any{}
			}
			if err := m.LastResponse.Unmarshal(dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		default:
			iNdEx = preIndex
			skippy, err := skipSession(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if (skippy < 0) || (iNdEx+skippy) < 0 {
				return ErrInvalidLengthSession
			}
			if (iNdEx + skippy) > l {
				return io.ErrUnexpectedEOF
			}
			iNdEx += skippy
		}
	}

	if iNdEx > l {
		return io.ErrUnexpectedEOF
	}
	return nil
}
func skipSession(dAtA []byte) (n int, err error) {
	l := len(dAtA)
	iNdEx := 0
	depth := 0
	for iNdEx < l {
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return 0, ErrIntOverflowSession
			}
			if iNdEx >= l {
				return 0, io.ErrUnexpectedEOF
			}
			b := dAtA[iNdEx]
			iNdEx++
			wire |= (uint64(b) & 0x7F) << shift
			if b < 0x80 {
				break
			}
		}
		wireType := int(wire & 0x7)
		switch wireType {
		case 0:
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return 0, ErrIntOverflowSession
				}
				if iNdEx >= l {
					return 0, io.ErrUnexpectedEOF
				}
				iNdEx++
				if dAtA[iNdEx-1] < 0x80 {
					break
				}
			}
		case 1:
			iNdEx += 8
		case 2:
			var length int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return 0, ErrIntOverflowSession
				}
				if iNdEx >= l {
					return 0, io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				length |= (int(b) & 0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			if length < 0 {
				return 0, ErrInvalidLengthSession
			}
			iNdEx += length
		case 3:
			depth++
		case 4:
			if depth == 0 {
				return 0, ErrUnexpectedEndOfGroupSession
			}
			depth--
		case 5:
			iNdEx += 4
		default:
			return 0, fmt.Errorf("proto: illegal wireType %d", wireType)
		}
		if iNdEx < 0 {
			return 0, ErrInvalidLengthSession
		}
		if depth == 0 {
			return iNdEx, nil
		}
	}
	return 0, io.ErrUnexpectedEOF
}

var (
	ErrInvalidLengthSession        = fmt.Errorf("proto: negative length found during unmarshaling")
	ErrIntOverflowSession          = fmt.Errorf("proto: integer overflow")
	ErrUnexpectedEndOfGroupSession = fmt.Errorf("proto: unexpected end of group")
)
