package stun_test

import (
	"encoding/binary"
	"net"
	"testing"

	"github.com/aethiopicuschan/natto/stun"
	"github.com/stretchr/testify/assert"
)

func TestDecodeMappedAddress_IPv4(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		ip     net.IP
		port   int
		expect stun.MappedAddress
	}{
		{
			name: "basic IPv4 mapped address",
			ip:   net.IPv4(192, 0, 2, 1),
			port: 54321,
			expect: stun.MappedAddress{
				IP:   net.IPv4(192, 0, 2, 1),
				Port: 54321,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			v := make([]byte, 8)
			v[1] = 0x01 // IPv4
			binary.BigEndian.PutUint16(v[2:4], uint16(tt.port))
			copy(v[4:8], tt.ip.To4())

			attr := stun.Attribute{Value: v}
			got, err := stun.DecodeMappedAddress(attr)

			assert.NoError(t, err)
			assert.Equal(t, tt.expect.Port, got.Port)
			assert.True(t, tt.expect.IP.Equal(got.IP))
		})
	}
}

func TestDecodeXORMappedAddress_IPv4(t *testing.T) {
	t.Parallel()

	tid := stun.TransactionID{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11}
	ip := net.IPv4(203, 0, 113, 10)
	port := 40000

	v := make([]byte, 8)
	v[1] = 0x01 // IPv4

	xPort := port ^ int(uint16(stun.MagicCookie>>16))
	binary.BigEndian.PutUint16(v[2:4], uint16(xPort))

	cookie := make([]byte, 4)
	binary.BigEndian.PutUint32(cookie, stun.MagicCookie)
	ip4 := ip.To4()
	for i := range 4 {
		v[4+i] = ip4[i] ^ cookie[i]
	}

	attr := stun.Attribute{Value: v}
	got, err := stun.DecodeXORMappedAddress(attr, tid)

	assert.NoError(t, err)
	assert.Equal(t, port, got.Port)
	assert.True(t, ip.Equal(got.IP))
}

func TestDecodeMappedAddress_IPv6(t *testing.T) {
	t.Parallel()

	ip := net.ParseIP("2001:db8::1")
	port := 12345

	v := make([]byte, 20)
	v[1] = 0x02 // IPv6
	binary.BigEndian.PutUint16(v[2:4], uint16(port))
	copy(v[4:20], ip.To16())

	attr := stun.Attribute{Value: v}
	got, err := stun.DecodeMappedAddress(attr)

	assert.NoError(t, err)
	assert.Equal(t, port, got.Port)
	assert.True(t, ip.Equal(got.IP))
}

func TestDecodeXORMappedAddress_IPv6(t *testing.T) {
	t.Parallel()

	ip := net.ParseIP("2001:db8::dead:beef")
	port := 60000
	tid := stun.TransactionID{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12}

	v := make([]byte, 20)
	v[1] = 0x02 // IPv6

	xPort := port ^ int(uint16(stun.MagicCookie>>16))
	binary.BigEndian.PutUint16(v[2:4], uint16(xPort))

	key := make([]byte, 16)
	binary.BigEndian.PutUint32(key[0:4], stun.MagicCookie)
	copy(key[4:16], tid[:])

	for i := range 16 {
		v[4+i] = ip[i] ^ key[i]
	}

	attr := stun.Attribute{Value: v}
	got, err := stun.DecodeXORMappedAddress(attr, tid)

	assert.NoError(t, err)
	assert.Equal(t, port, got.Port)
	assert.True(t, ip.Equal(got.IP))
}

func TestDecodeMappedAddress_Errors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		value []byte
	}{
		{
			name:  "too short",
			value: []byte{0x00, 0x01},
		},
		{
			name:  "unsupported family",
			value: []byte{0x00, 0xFF, 0x00, 0x01},
		},
		{
			name:  "ipv4 too short",
			value: []byte{0x00, 0x01, 0x00, 0x01, 1, 2, 3},
		},
		{
			name:  "ipv6 too short",
			value: []byte{0x00, 0x02, 0x00, 0x01, 1, 2},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, err := stun.DecodeMappedAddress(stun.Attribute{Value: tt.value})
			assert.ErrorIs(t, err, stun.ErrNotSTUN)
		})
	}
}

func TestFindMappedAddress(t *testing.T) {
	t.Parallel()

	tid := stun.TransactionID{9, 9, 9, 9, 9, 9, 9, 9, 9, 9, 9, 9}
	ip := net.IPv4(198, 51, 100, 42)
	port := 50000

	v := make([]byte, 8)
	v[1] = 0x01
	binary.BigEndian.PutUint16(v[2:4], uint16(port))
	copy(v[4:8], ip.To4())

	msg := &stun.Message{
		TransactionID: tid,
		Attributes: []stun.Attribute{
			{Type: stun.AttrMappedAddress, Value: v},
		},
	}

	got, err := stun.FindMappedAddress(msg)
	assert.NoError(t, err)
	assert.Equal(t, port, got.Port)
	assert.True(t, ip.Equal(got.IP))
}

func TestFindMappedAddress_NoAttributes(t *testing.T) {
	t.Parallel()

	msg := &stun.Message{}
	_, err := stun.FindMappedAddress(msg)

	assert.ErrorIs(t, err, stun.ErrNoMappedAddress)
}
