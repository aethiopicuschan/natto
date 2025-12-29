package turn

import "errors"

var (
	// ErrUnauthorized indicates TURN server requires authentication or nonce is stale.
	ErrUnauthorized = errors.New("turn: unauthorized (need auth or stale nonce)")

	// ErrBadMessage indicates received STUN/TURN message is malformed.
	ErrBadMessage = errors.New("turn: bad message")

	// ErrTimeout indicates a request timed out.
	ErrTimeout = errors.New("turn: timeout")

	// ErrNoAllocation indicates operation requires an active allocation.
	ErrNoAllocation = errors.New("turn: no active allocation")

	// ErrInvalidAddress indicates address parsing/encoding failure.
	ErrInvalidAddress = errors.New("turn: invalid address")
)
