package turn

import (
	"context"
	"encoding/binary"
	"net"
	"time"
)

// Allocate requests a relayed address on the TURN server (UDP transport).
// This uses long-term credentials and handles 401/438 challenge by retrying once.
func (c *Client) Allocate(ctx context.Context, lifetime time.Duration) (*net.UDPAddr, error) {
	// First try without auth if we don't have nonce/realm yet.
	relayed, lifetimeSec, err := c.allocateOnce(ctx, lifetime, false)
	if err == ErrUnauthorized {
		// Retry with updated nonce/realm.
		relayed, lifetimeSec, err = c.allocateOnce(ctx, lifetime, true)
	}
	if err != nil {
		return nil, err
	}
	c.setAllocation(relayed, lifetimeSec)
	return relayed, nil
}

func (c *Client) allocateOnce(ctx context.Context, lifetime time.Duration, withAuth bool) (*net.UDPAddr, int, error) {
	m := NewMessage(stunType(methodAllocate, classRequest))

	// REQUESTED-TRANSPORT: 17 for UDP (RFC 5766)
	rt := make([]byte, 4)
	rt[0] = 17
	m.addAttr(attrRequestedTransport, rt)

	// LIFETIME in seconds
	sec := uint32(lifetime / time.Second)
	lt := make([]byte, 4)
	binary.BigEndian.PutUint32(lt, sec)
	m.addAttr(attrLifetime, lt)

	if withAuth {
		c.addAuthAttrs(m)
	}

	resp, err := c.doRequest(ctx, m, withAuth)
	if err != nil {
		return nil, 0, err
	}

	// Success: XOR-RELAYED-ADDRESS, LIFETIME
	a, ok := resp.FindAttr(attrXorRelayedAddr)
	if !ok {
		return nil, 0, ErrBadMessage
	}
	relayed, err := parseXorAddr(a.Value, resp.TransactionID)
	if err != nil {
		return nil, 0, err
	}

	lifetimeSec := int(sec)
	if ltAttr, ok := resp.FindAttr(attrLifetime); ok && len(ltAttr.Value) == 4 {
		lifetimeSec = int(binary.BigEndian.Uint32(ltAttr.Value))
	}
	return relayed, lifetimeSec, nil
}

// Refresh refreshes allocation lifetime.
func (c *Client) Refresh(ctx context.Context, lifetime time.Duration) error {
	c.mu.Lock()
	hasAlloc := c.relayed != nil
	c.mu.Unlock()
	if !hasAlloc {
		return ErrNoAllocation
	}

	// Always require auth for refresh.
	m := NewMessage(stunType(methodRefresh, classRequest))
	sec := uint32(lifetime / time.Second)
	lt := make([]byte, 4)
	binary.BigEndian.PutUint32(lt, sec)
	m.addAttr(attrLifetime, lt)

	c.addAuthAttrs(m)
	resp, err := c.doRequest(ctx, m, true)
	if err == ErrUnauthorized {
		// Retry once on stale nonce.
		c.addAuthAttrs(m)
		resp, err = c.doRequest(ctx, m, true)
	}
	if err != nil {
		return err
	}

	// Update lifetime if provided
	if ltAttr, ok := resp.FindAttr(attrLifetime); ok && len(ltAttr.Value) == 4 {
		c.mu.Lock()
		c.lifetime = time.Duration(binary.BigEndian.Uint32(ltAttr.Value)) * time.Second
		c.mu.Unlock()
	}
	return nil
}
