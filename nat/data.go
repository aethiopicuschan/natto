package nat

// DataPacket is a simple wrapper for application payloads.
// This allows us to distinguish data packets from control messages.
type DataPacket struct {
	// Payload is the raw application data.
	Payload []byte
}

// EncodeDataPacket wraps raw bytes for sending.
// Currently this is a pass-through, but exists for future extensibility
// (e.g., versioning, flags, or encryption markers).
func EncodeDataPacket(p []byte) []byte {
	return p
}

// DecodeDataPacket unwraps received bytes.
// The caller must ensure that control messages are filtered out beforehand.
func DecodeDataPacket(p []byte) *DataPacket {
	return &DataPacket{Payload: p}
}
