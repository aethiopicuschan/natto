package turn

import (
	"encoding/binary"
)

// Attr represents a STUN attribute TLV.
type Attr struct {
	Type  uint16
	Value []byte
}

// addAttr appends an attribute with 32-bit padding.
func (m *Message) addAttr(t uint16, v []byte) {
	m.Attrs = append(m.Attrs, Attr{Type: t, Value: v})
}

// encodeAttrs encodes all attributes with padding.
func (m *Message) encodeAttrs() []byte {
	var out []byte
	for _, a := range m.Attrs {
		// type(2) len(2)
		h := make([]byte, 4)
		binary.BigEndian.PutUint16(h[0:2], a.Type)
		binary.BigEndian.PutUint16(h[2:4], uint16(len(a.Value)))
		out = append(out, h...)
		out = append(out, a.Value...)

		// 32-bit padding
		pad := (4 - (len(a.Value) % 4)) % 4
		if pad != 0 {
			out = append(out, make([]byte, pad)...)
		}
	}
	return out
}
