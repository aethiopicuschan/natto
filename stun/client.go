package stun

import (
	"context"
	"errors"
	"net"
	"time"
)

// Client is a simple STUN UDP client.
type Client struct {
	// Timeout is the per-transaction deadline used if ctx has no deadline.
	Timeout time.Duration

	// Retries controls how many times to retransmit the same request on timeout.
	Retries int

	// RTO is the initial retransmission timeout.
	RTO time.Duration
}

// NewClient returns a Client with sensible defaults.
func NewClient() *Client {
	return &Client{
		Timeout: 3 * time.Second,
		Retries: 6,
		RTO:     250 * time.Millisecond,
	}
}

// BindingRequest sends a STUN Binding Request to serverAddr and returns the public mapped address.
// serverAddr should be like "stun.l.google.com:19302".
func (c *Client) BindingRequest(ctx context.Context, serverAddr string) (MappedAddress, error) {
	raddr, err := net.ResolveUDPAddr("udp", serverAddr)
	if err != nil {
		return MappedAddress{}, err
	}

	conn, err := net.DialUDP("udp", nil, raddr)
	if err != nil {
		return MappedAddress{}, err
	}
	defer conn.Close()

	return c.BindingRequestConn(ctx, conn)
}

// BindingRequestConn performs a STUN Binding Request using an existing UDP connection.
// The connection must be connected to the STUN server (DialUDP), not a raw ListenUDP socket.
func (c *Client) BindingRequestConn(ctx context.Context, conn *net.UDPConn) (MappedAddress, error) {
	tid, err := NewTransactionID()
	if err != nil {
		return MappedAddress{}, err
	}

	req := NewBindingRequest(tid)
	reqBytes := req.Marshal()

	// Determine overall deadline.
	deadline, hasDeadline := ctx.Deadline()
	if !hasDeadline {
		deadline = time.Now().Add(c.Timeout)
	}

	rto := c.RTO
	buf := make([]byte, 1500)

	for attempt := 0; attempt <= c.Retries; attempt++ {
		// Respect context cancellation.
		select {
		case <-ctx.Done():
			return MappedAddress{}, ctx.Err()
		default:
		}

		// Send request.
		if _, err := conn.Write(reqBytes); err != nil {
			return MappedAddress{}, err
		}

		// Wait for response until min(deadline, now+rto).
		waitUntil := time.Now().Add(rto)
		if waitUntil.After(deadline) {
			waitUntil = deadline
		}
		_ = conn.SetReadDeadline(waitUntil)

		n, err := conn.Read(buf)
		if err != nil {
			// Timeout -> retransmit with backoff.
			if ne, ok := err.(net.Error); ok && ne.Timeout() {
				if time.Now().After(deadline) {
					return MappedAddress{}, ErrTimeout
				}
				rto *= 2
				continue
			}
			return MappedAddress{}, err
		}

		resp, err := Parse(buf[:n])
		if err != nil {
			// Ignore non-STUN packets and keep trying within this attempt window.
			continue
		}

		// Match transaction ID.
		if resp.TransactionID != tid {
			continue
		}

		// Only accept Binding Success Response.
		if resp.Method != MethodBinding || resp.Class != ClassSuccessResponse {
			// If it's an error response, surface a helpful error.
			if resp.Class == ClassErrorResponse {
				return MappedAddress{}, errors.New("stun: received error response")
			}
			return MappedAddress{}, ErrNotSTUN
		}

		return FindMappedAddress(resp)
	}

	return MappedAddress{}, ErrTimeout
}
