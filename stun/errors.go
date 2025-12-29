package stun

import "errors"

var (
	// ErrNotSTUN indicates that the packet is not a valid STUN message.
	ErrNotSTUN = errors.New("stun: not a stun message")

	// ErrUnsupported indicates that the message/attribute is not supported by this implementation.
	ErrUnsupported = errors.New("stun: unsupported feature")

	// ErrNoMappedAddress indicates that the response did not contain any mapped address attribute.
	ErrNoMappedAddress = errors.New("stun: no mapped address in response")

	// ErrTimeout indicates that the STUN transaction timed out.
	ErrTimeout = errors.New("stun: timeout")
)
