package stun

import (
	"crypto/rand"
	"encoding/binary"
)

// RFC 5389 magic cookie.
const MagicCookie uint32 = 0x2112A442

// STUN message classes/methods (RFC 5389).
const (
	MethodBinding uint16 = 0x0001
)

// STUN message types we use (Binding Request / Success / Error).
const (
	ClassRequest         = 0x00
	ClassIndication      = 0x01
	ClassSuccessResponse = 0x02
	ClassErrorResponse   = 0x03
)

// Common attribute types (RFC 5389 / RFC 5780 etc).
const (
	AttrMappedAddress     uint16 = 0x0001
	AttrXORMappedAddress  uint16 = 0x0020
	AttrErrorCode         uint16 = 0x0009
	AttrUnknownAttributes uint16 = 0x000A
	AttrSoftware          uint16 = 0x8022
)

// TransactionID is a 96-bit (12 bytes) ID used to match requests and responses.
type TransactionID [12]byte

// NewTransactionID generates a new cryptographically random transaction ID.
func NewTransactionID() (TransactionID, error) {
	var id TransactionID
	_, err := rand.Read(id[:])
	return id, err
}

// stunType encodes method/class into the 16-bit STUN message type field.
func stunType(method uint16, class int) uint16 {
	// RFC 5389: Type = M11..M0 + C1,C0 bits interleaved
	// This encoder supports typical cases and matches the spec bit layout.
	m := method & 0x0FFF
	c := uint16(class & 0x03)

	// Bits:
	// 0-3  : M0-3
	// 4    : C0
	// 5-7  : M4-6
	// 8    : C1
	// 9-12 : M7-10
	// 13-15: M11 (and unused)
	t := uint16(0)
	t |= (m & 0x000F)
	t |= (c & 0x01) << 4
	t |= (m & 0x0070) << 1
	t |= (c & 0x02) << 7
	t |= (m & 0x0F80) << 2
	return t
}

// parseType decodes a STUN message type into method/class.
func parseType(t uint16) (method uint16, class int) {
	// Reverse of stunType()
	m := uint16(0)
	c0 := (t >> 4) & 0x1
	c1 := (t >> 8) & 0x1

	m |= (t & 0x000F)
	m |= (t >> 1) & 0x0070
	m |= (t >> 2) & 0x0F80

	return m, int(c0 | (c1 << 1))
}

// readU16 reads a big-endian uint16.
func readU16(b []byte) uint16 { return binary.BigEndian.Uint16(b) }

// readU32 reads a big-endian uint32.
func readU32(b []byte) uint32 { return binary.BigEndian.Uint32(b) }

// putU16 writes a big-endian uint16.
func putU16(b []byte, v uint16) { binary.BigEndian.PutUint16(b, v) }

// putU32 writes a big-endian uint32.
func putU32(b []byte, v uint32) { binary.BigEndian.PutUint32(b, v) }
