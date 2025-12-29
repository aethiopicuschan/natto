package turn

import (
	"crypto/rand"
	"encoding/binary"
)

type Message struct {
	Type          uint16
	TransactionID [12]byte
	Attrs         []Attr
}

// NewMessage creates a new STUN/TURN message with a random transaction ID.
func NewMessage(msgType uint16) *Message {
	var tid [12]byte
	_, _ = rand.Read(tid[:])
	return &Message{Type: msgType, TransactionID: tid}
}

// Encode encodes message header and attributes.
// Note: MESSAGE-INTEGRITY/FINGERPRINT are handled by higher-level helpers.
func (m *Message) Encode() []byte {
	attrs := m.encodeAttrs()
	b := make([]byte, 20+len(attrs))

	// STUN header:
	// type(2), length(2), magic cookie(4), transaction id(12)
	binary.BigEndian.PutUint16(b[0:2], m.Type)
	binary.BigEndian.PutUint16(b[2:4], uint16(len(attrs)))
	binary.BigEndian.PutUint32(b[4:8], stunMagicCookie)
	copy(b[8:20], m.TransactionID[:])
	copy(b[20:], attrs)
	return b
}

// ParseMessage parses a STUN/TURN message bytes into Message + attributes.
func ParseMessage(p []byte) (*Message, error) {
	if len(p) < 20 {
		return nil, ErrBadMessage
	}
	msgLen := int(binary.BigEndian.Uint16(p[2:4]))
	if len(p) < 20+msgLen {
		return nil, ErrBadMessage
	}
	m := &Message{}
	m.Type = binary.BigEndian.Uint16(p[0:2])
	copy(m.TransactionID[:], p[8:20])

	// Attributes parsing
	i := 20
	end := 20 + msgLen
	for i+4 <= end {
		t := binary.BigEndian.Uint16(p[i : i+2])
		l := int(binary.BigEndian.Uint16(p[i+2 : i+4]))
		i += 4
		if i+l > end {
			return nil, ErrBadMessage
		}
		v := make([]byte, l)
		copy(v, p[i:i+l])
		m.Attrs = append(m.Attrs, Attr{Type: t, Value: v})
		i += l

		// skip padding
		pad := (4 - (l % 4)) % 4
		i += pad
	}
	return m, nil
}

// FindAttr finds the first attribute with given type.
func (m *Message) FindAttr(t uint16) (Attr, bool) {
	for _, a := range m.Attrs {
		if a.Type == t {
			return a, true
		}
	}
	return Attr{}, false
}
