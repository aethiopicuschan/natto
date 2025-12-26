package nat

import "errors"

var (
	// ErrPunchTimeout is returned when hole punching does not succeed
	// within the provided context deadline.
	ErrPunchTimeout = errors.New("nat traversal timed out")

	// ErrInvalidMessage indicates that a received control message
	// is syntactically or semantically invalid.
	ErrInvalidMessage = errors.New("invalid control message")

	// ErrConnectionClosed is returned when the underlying UDP connection
	// is unexpectedly closed.
	ErrConnectionClosed = errors.New("connection closed")

	// ErrMessageIsNil is returned when trying to encode a nil message.
	ErrMessageIsNil = errors.New("message is nil")

	// ErrNotOurPacket indicates the packet does not have the expected magic header.
	ErrNotOurPacket = errors.New("not a nat packet")

	// ErrMalformedPacket indicates the packet is too short or invalid.
	ErrMalformedPacket = errors.New("malformed nat packet")
)
