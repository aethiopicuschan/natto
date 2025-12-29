package stun

import (
	"encoding/binary"
)

// HeaderLen is the STUN header size in bytes.
const HeaderLen = 20

// Message represents a STUN message (header + attributes).
type Message struct {
	Method        uint16
	Class         int
	Length        uint16 // computed from Attributes when marshaling
	Cookie        uint32
	TransactionID TransactionID
	Attributes    []Attribute
}

// Attribute represents a single STUN TLV attribute.
type Attribute struct {
	Type  uint16
	Value []byte
}

// NewBindingRequest creates a STUN Binding Request message.
func NewBindingRequest(tid TransactionID) *Message {
	return &Message{
		Method:        MethodBinding,
		Class:         ClassRequest,
		Cookie:        MagicCookie,
		TransactionID: tid,
		Attributes:    nil,
	}
}

// Marshal serializes the message into a byte slice.
func (m *Message) Marshal() []byte {
	// Compute attribute section size with padding (32-bit aligned).
	attrLen := 0
	for _, a := range m.Attributes {
		vlen := len(a.Value)
		padded := (vlen + 3) &^ 3
		attrLen += 4 + padded
	}
	m.Length = uint16(attrLen)

	out := make([]byte, HeaderLen+attrLen)

	// Header
	putU16(out[0:2], stunType(m.Method, m.Class))
	putU16(out[2:4], m.Length)
	putU32(out[4:8], m.Cookie)
	copy(out[8:20], m.TransactionID[:])

	// Attributes
	off := HeaderLen
	for _, a := range m.Attributes {
		putU16(out[off:off+2], a.Type)
		putU16(out[off+2:off+4], uint16(len(a.Value)))
		copy(out[off+4:off+4+len(a.Value)], a.Value)

		// Padding
		vlen := len(a.Value)
		padded := (vlen + 3) &^ 3
		for i := vlen; i < padded; i++ {
			out[off+4+i] = 0
		}
		off += 4 + padded
	}

	return out
}

// Parse parses a raw packet into a STUN message.
func Parse(pkt []byte) (*Message, error) {
	if len(pkt) < HeaderLen {
		return nil, ErrNotSTUN
	}

	// Per RFC 5389: first two bits of type must be 0.
	if (pkt[0] & 0xC0) != 0x00 {
		return nil, ErrNotSTUN
	}

	t := readU16(pkt[0:2])
	length := readU16(pkt[2:4])
	cookie := readU32(pkt[4:8])

	if cookie != MagicCookie {
		return nil, ErrNotSTUN
	}

	if int(HeaderLen+length) > len(pkt) {
		return nil, ErrNotSTUN
	}

	method, class := parseType(t)
	var tid TransactionID
	copy(tid[:], pkt[8:20])

	msg := &Message{
		Method:        method,
		Class:         class,
		Length:        length,
		Cookie:        cookie,
		TransactionID: tid,
	}

	attrs, err := parseAttributes(pkt[HeaderLen : HeaderLen+length])
	if err != nil {
		return nil, err
	}
	msg.Attributes = attrs
	return msg, nil
}

// parseAttributes parses a sequence of STUN attributes.
func parseAttributes(b []byte) ([]Attribute, error) {
	var attrs []Attribute
	off := 0
	for off+4 <= len(b) {
		typ := binary.BigEndian.Uint16(b[off : off+2])
		vlen := int(binary.BigEndian.Uint16(b[off+2 : off+4]))
		off += 4

		if off+vlen > len(b) {
			return nil, ErrNotSTUN
		}
		val := make([]byte, vlen)
		copy(val, b[off:off+vlen])
		attrs = append(attrs, Attribute{Type: typ, Value: val})

		// Skip padding to 32-bit alignment.
		padded := (vlen + 3) &^ 3
		off += padded
	}
	return attrs, nil
}

// GetAttribute returns the first attribute with the given type.
func (m *Message) GetAttribute(typ uint16) (Attribute, bool) {
	for _, a := range m.Attributes {
		if a.Type == typ {
			return a, true
		}
	}
	return Attribute{}, false
}
