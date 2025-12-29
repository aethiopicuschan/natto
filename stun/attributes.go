package stun

import (
	"encoding/binary"
	"net"
)

// MappedAddress is a decoded (XOR-)MAPPED-ADDRESS.
type MappedAddress struct {
	IP   net.IP
	Port int
}

// DecodeMappedAddress decodes MAPPED-ADDRESS (RFC 5389 legacy).
func DecodeMappedAddress(a Attribute) (MappedAddress, error) {
	return decodeAddress(a.Value, false, TransactionID{})
}

// DecodeXORMappedAddress decodes XOR-MAPPED-ADDRESS (RFC 5389).
func DecodeXORMappedAddress(a Attribute, tid TransactionID) (MappedAddress, error) {
	return decodeAddress(a.Value, true, tid)
}

// decodeAddress decodes (XOR-)MAPPED-ADDRESS attribute payload.
func decodeAddress(v []byte, xor bool, tid TransactionID) (MappedAddress, error) {
	// Format:
	// 0: reserved (0)
	// 1: family (0x01 IPv4, 0x02 IPv6)
	// 2-3: port
	// 4.. : address
	if len(v) < 4 {
		return MappedAddress{}, ErrNotSTUN
	}
	fam := v[1]
	port := int(binary.BigEndian.Uint16(v[2:4]))

	switch fam {
	case 0x01: // IPv4
		if len(v) < 8 {
			return MappedAddress{}, ErrNotSTUN
		}
		ip := make(net.IP, net.IPv4len)
		copy(ip, v[4:8])

		if xor {
			// XOR with magic cookie for IPv4.
			c := make([]byte, 4)
			binary.BigEndian.PutUint32(c, MagicCookie)
			for i := 0; i < 4; i++ {
				ip[i] ^= c[i]
			}
			port ^= int(uint16(MagicCookie >> 16))
		}

		return MappedAddress{IP: ip, Port: port}, nil

	case 0x02: // IPv6
		if len(v) < 20 {
			return MappedAddress{}, ErrNotSTUN
		}
		ip := make(net.IP, net.IPv6len)
		copy(ip, v[4:20])

		if xor {
			// XOR with (magic cookie || transaction id) for IPv6.
			key := make([]byte, 16)
			binary.BigEndian.PutUint32(key[0:4], MagicCookie)
			copy(key[4:16], tid[:])
			for i := range 16 {
				ip[i] ^= key[i]
			}
			port ^= int(uint16(MagicCookie >> 16))
		}

		return MappedAddress{IP: ip, Port: port}, nil

	default:
		return MappedAddress{}, ErrNotSTUN
	}
}

// FindMappedAddress tries XOR-MAPPED-ADDRESS first, then MAPPED-ADDRESS.
func FindMappedAddress(msg *Message) (MappedAddress, error) {
	if a, ok := msg.GetAttribute(AttrXORMappedAddress); ok {
		return DecodeXORMappedAddress(a, msg.TransactionID)
	}
	if a, ok := msg.GetAttribute(AttrMappedAddress); ok {
		return DecodeMappedAddress(a)
	}
	return MappedAddress{}, ErrNoMappedAddress
}
