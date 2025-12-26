package nat

import (
	"encoding/json"
)

// MessageType represents a control message type exchanged between peers.
type MessageType string

const (
	// MessageHello is sent to announce presence and initiate punching.
	MessageHello MessageType = "hello"

	// MessageAck is sent in response to MessageHello to confirm reachability.
	MessageAck MessageType = "ack"
)

// Message is a small control packet exchanged during NAT traversal.
// It is intentionally simple and JSON-encoded for debuggability.
type Message struct {
	// Type indicates the purpose of this message.
	Type MessageType `json:"type"`

	// PeerID identifies the sender of this message.
	PeerID string `json:"peer_id"`

	// ToPeerID identifies the intended receiver.
	// This helps avoid treating unrelated packets as valid handshake messages.
	ToPeerID string `json:"to_peer_id,omitempty"`

	// Timestamp can be used by the receiver to reason about freshness.
	Timestamp int64 `json:"ts"`
}

// EncodeMessage serializes a Message into bytes.
func EncodeMessage(msg *Message) (b []byte, err error) {
	if msg == nil {
		err = ErrMessageIsNil
		return
	}
	b, err = json.Marshal(msg)
	return
}

// DecodeMessage deserializes bytes into a Message.
func DecodeMessage(data []byte) (*Message, error) {
	var msg Message
	if err := json.Unmarshal(data, &msg); err != nil {
		return nil, err
	}
	return &msg, nil
}
