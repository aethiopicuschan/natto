package turn

// stunType builds STUN message type from method and class (RFC 5389 encoding).
func stunType(method uint16, class uint16) uint16 {
	// Method: bits spread into type
	// Class: C0 at bit4, C1 at bit8
	m := method & 0x0FFF
	t := uint16(0)

	t |= (m & 0x000F)
	t |= (m & 0x0070) << 1
	t |= (m & 0x0F80) << 2

	// class bits
	t |= (class & 0x0010) // C0
	t |= (class & 0x0100) // C1
	return t
}

// isSuccessResponse returns true if message class is success.
func isSuccessResponse(msgType uint16) bool {
	return (msgType & 0x0110) == classSuccess
}

// isErrorResponse returns true if message class is error.
func isErrorResponse(msgType uint16) bool {
	return (msgType & 0x0110) == classError
}
