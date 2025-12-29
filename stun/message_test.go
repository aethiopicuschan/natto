package stun_test

import (
	"encoding/binary"
	"testing"

	"github.com/aethiopicuschan/natto/stun"
	"github.com/stretchr/testify/assert"
)

func TestNewBindingRequest(t *testing.T) {
	t.Parallel()

	tid := stun.TransactionID{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12}
	msg := stun.NewBindingRequest(tid)

	assert.Equal(t, stun.MethodBinding, msg.Method)
	assert.Equal(t, stun.ClassRequest, msg.Class)
	assert.Equal(t, stun.MagicCookie, msg.Cookie)
	assert.Equal(t, tid, msg.TransactionID)
	assert.Empty(t, msg.Attributes)
}

func TestMessage_MarshalParse_RoundTrip(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		attributes []stun.Attribute
	}{
		{
			name:       "no attributes",
			attributes: nil,
		},
		{
			name: "single attribute no padding",
			attributes: []stun.Attribute{
				{Type: 0x0001, Value: []byte{1, 2, 3, 4}},
			},
		},
		{
			name: "single attribute with padding",
			attributes: []stun.Attribute{
				{Type: 0x0002, Value: []byte{1, 2, 3}},
			},
		},
		{
			name: "multiple attributes mixed padding",
			attributes: []stun.Attribute{
				{Type: 0x0003, Value: []byte{1}},
				{Type: 0x0004, Value: []byte{1, 2, 3, 4}},
				{Type: 0x0005, Value: []byte{1, 2}},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			tid := stun.TransactionID{9, 8, 7, 6, 5, 4, 3, 2, 1, 0, 1, 2}

			msg := &stun.Message{
				Method:        stun.MethodBinding,
				Class:         stun.ClassSuccessResponse,
				Cookie:        stun.MagicCookie,
				TransactionID: tid,
				Attributes:    tt.attributes,
			}

			raw := msg.Marshal()
			parsed, err := stun.Parse(raw)

			assert.NoError(t, err)
			assert.Equal(t, msg.Method, parsed.Method)
			assert.Equal(t, msg.Class, parsed.Class)
			assert.Equal(t, msg.Cookie, parsed.Cookie)
			assert.Equal(t, msg.TransactionID, parsed.TransactionID)
			assert.Len(t, parsed.Attributes, len(tt.attributes))

			for i := range tt.attributes {
				assert.Equal(t, tt.attributes[i].Type, parsed.Attributes[i].Type)
				assert.Equal(t, tt.attributes[i].Value, parsed.Attributes[i].Value)
			}
		})
	}
}

func TestParse_Errors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		pkt  []byte
	}{
		{
			name: "too short",
			pkt:  []byte{0x00, 0x01},
		},
		{
			name: "invalid type bits",
			pkt: func() []byte {
				b := make([]byte, stun.HeaderLen)
				b[0] = 0xC0
				return b
			}(),
		},
		{
			name: "invalid magic cookie",
			pkt: func() []byte {
				b := make([]byte, stun.HeaderLen)
				binary.BigEndian.PutUint32(b[4:8], 0xdeadbeef)
				return b
			}(),
		},
		{
			name: "length exceeds packet",
			pkt: func() []byte {
				b := make([]byte, stun.HeaderLen)
				binary.BigEndian.PutUint16(b[2:4], 100)
				binary.BigEndian.PutUint32(b[4:8], stun.MagicCookie)
				return b
			}(),
		},
		{
			name: "attribute length overflow",
			pkt: func() []byte {
				b := make([]byte, stun.HeaderLen+4)
				binary.BigEndian.PutUint32(b[4:8], stun.MagicCookie)
				binary.BigEndian.PutUint16(b[2:4], 4)
				binary.BigEndian.PutUint16(b[stun.HeaderLen+2:], 10)
				return b
			}(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, err := stun.Parse(tt.pkt)
			assert.ErrorIs(t, err, stun.ErrNotSTUN)
		})
	}
}

func TestParseAttributes_PaddingHandledCorrectly(t *testing.T) {
	t.Parallel()

	// Attribute with length=3, padded to 4.
	attr := []byte{
		0x00, 0x01, // type
		0x00, 0x03, // length
		0xAA, 0xBB, 0xCC, // value
		0x00, // padding
	}

	header := make([]byte, stun.HeaderLen)
	binary.BigEndian.PutUint32(header[4:8], stun.MagicCookie)
	binary.BigEndian.PutUint16(header[2:4], uint16(len(attr)))

	msg, err := stun.Parse(append(header, attr...))

	assert.NoError(t, err)
	assert.Len(t, msg.Attributes, 1)
	assert.Equal(t, uint16(0x0001), msg.Attributes[0].Type)
	assert.Equal(t, []byte{0xAA, 0xBB, 0xCC}, msg.Attributes[0].Value)
}

func TestMessage_GetAttribute(t *testing.T) {
	t.Parallel()

	msg := &stun.Message{
		Attributes: []stun.Attribute{
			{Type: 1, Value: []byte{1}},
			{Type: 2, Value: []byte{2}},
		},
	}

	attr, ok := msg.GetAttribute(2)
	assert.True(t, ok)
	assert.Equal(t, []byte{2}, attr.Value)

	_, ok = msg.GetAttribute(3)
	assert.False(t, ok)
}
