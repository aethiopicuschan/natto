package turn

import (
	"encoding/binary"
	"net"
)

// xorAddr encodes XOR-ADDRESS (XOR-MAPPED-ADDRESS / XOR-RELAYED-ADDRESS / XOR-PEER-ADDRESS).
func xorAddr(ip net.IP, port int, tid [12]byte) ([]byte, error) {
	ip4 := ip.To4()
	ip16 := ip.To16()
	if ip4 == nil && ip16 == nil {
		return nil, ErrInvalidAddress
	}

	// Format: 0: reserved, 1: family, 2-3: x-port, 4..: x-address
	// family: 0x01 IPv4, 0x02 IPv6
	if ip4 != nil {
		v := make([]byte, 8)
		v[1] = 0x01
		xport := uint16(port) ^ uint16(stunMagicCookie>>16)
		binary.BigEndian.PutUint16(v[2:4], xport)

		mc := make([]byte, 4)
		binary.BigEndian.PutUint32(mc, stunMagicCookie)
		for i := 0; i < 4; i++ {
			v[4+i] = ip4[i] ^ mc[i]
		}
		return v, nil
	}

	// IPv6
	v := make([]byte, 20)
	v[1] = 0x02
	xport := uint16(port) ^ uint16(stunMagicCookie>>16)
	binary.BigEndian.PutUint16(v[2:4], xport)

	mc := make([]byte, 4)
	binary.BigEndian.PutUint32(mc, stunMagicCookie)

	// XOR with magic cookie + transaction ID (16 bytes)
	xorKey := append(mc, tid[:]...)
	for i := 0; i < 16; i++ {
		v[4+i] = ip16[i] ^ xorKey[i]
	}
	return v, nil
}

// parseXorAddr decodes XOR-ADDRESS into net.UDPAddr.
func parseXorAddr(v []byte, tid [12]byte) (*net.UDPAddr, error) {
	if len(v) < 4 {
		return nil, ErrInvalidAddress
	}
	family := v[1]
	xport := binary.BigEndian.Uint16(v[2:4])
	port := int(xport ^ uint16(stunMagicCookie>>16))

	mc := make([]byte, 4)
	binary.BigEndian.PutUint32(mc, stunMagicCookie)

	switch family {
	case 0x01:
		if len(v) < 8 {
			return nil, ErrInvalidAddress
		}
		ip := make(net.IP, 4)
		for i := 0; i < 4; i++ {
			ip[i] = v[4+i] ^ mc[i]
		}
		return &net.UDPAddr{IP: ip, Port: port}, nil

	case 0x02:
		if len(v) < 20 {
			return nil, ErrInvalidAddress
		}
		ip := make(net.IP, 16)
		xorKey := append(mc, tid[:]...)
		for i := 0; i < 16; i++ {
			ip[i] = v[4+i] ^ xorKey[i]
		}
		return &net.UDPAddr{IP: ip, Port: port}, nil

	default:
		return nil, ErrInvalidAddress
	}
}
