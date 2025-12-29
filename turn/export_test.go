package turn

import (
	"net"
	"time"
)

// Export internal helpers for tests in turn_test.

var (
	StunType     = stunType
	IsSuccess    = isSuccessResponse
	IsError      = isErrorResponse
	LongTermKey  = longTermKey
	XorAddr      = xorAddr
	ParseXorAddr = parseXorAddr
)

// --- test-only accessors ---

// TestConn exposes the underlying UDP connection for tests.
func (c *Client) TestConn() *net.UDPConn {
	return c.conn
}

// TestSetConn sets UDP connection (used only in tests).
func (c *Client) TestSetConn(conn *net.UDPConn) {
	c.conn = conn
}

// TestLocalAddr returns local UDP address.
func (c *Client) TestLocalAddr() net.Addr {
	if c.conn == nil {
		return nil
	}
	return c.conn.LocalAddr()
}

// TestSetReadDeadline sets read deadline on underlying conn.
func (c *Client) TestSetReadDeadline(t time.Time) error {
	if c.conn == nil {
		return nil
	}
	return c.conn.SetReadDeadline(t)
}
