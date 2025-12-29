package turn

import (
	"context"
	"encoding/binary"
	"net"
)

// CreatePermission creates a permission for a peer address (required for Send/ChannelData).
func (c *Client) CreatePermission(ctx context.Context, peer *net.UDPAddr) error {
	c.mu.Lock()
	hasAlloc := c.relayed != nil
	c.mu.Unlock()
	if !hasAlloc {
		return ErrNoAllocation
	}

	m := NewMessage(stunType(methodCreatePerm, classRequest))
	xp, err := xorAddr(peer.IP, peer.Port, m.TransactionID)
	if err != nil {
		return err
	}
	m.addAttr(attrXorPeerAddr, xp)

	c.addAuthAttrs(m)
	_, err = c.doRequest(ctx, m, true)
	if err == ErrUnauthorized {
		// Retry once on stale nonce.
		c.addAuthAttrs(m)
		_, err = c.doRequest(ctx, m, true)
	}
	return err
}

// ChannelBind binds a channel number to a peer for faster data (ChannelData frames).
func (c *Client) ChannelBind(ctx context.Context, peer *net.UDPAddr, ch uint16) error {
	if ch < channelMin || ch > channelMax {
		return ErrInvalidAddress
	}

	c.mu.Lock()
	hasAlloc := c.relayed != nil
	c.mu.Unlock()
	if !hasAlloc {
		return ErrNoAllocation
	}

	m := NewMessage(stunType(methodChannelBind, classRequest))

	// CHANNEL-NUMBER is 4 bytes: channel(2) + RFFU(2)
	cn := make([]byte, 4)
	binary.BigEndian.PutUint16(cn[0:2], ch)
	m.addAttr(attrChannelNumber, cn)

	xp, err := xorAddr(peer.IP, peer.Port, m.TransactionID)
	if err != nil {
		return err
	}
	m.addAttr(attrXorPeerAddr, xp)

	c.addAuthAttrs(m)
	_, err = c.doRequest(ctx, m, true)
	if err == ErrUnauthorized {
		c.addAuthAttrs(m)
		_, err = c.doRequest(ctx, m, true)
	}
	return err
}
