package turn

import (
	"context"
	"encoding/binary"
	"net"
	"time"
)

// SendIndication sends data to a peer via TURN using Send Indication.
// Note: This is "best effort" (no response).
func (c *Client) SendIndication(peer *net.UDPAddr, data []byte) error {
	c.mu.Lock()
	hasAlloc := c.relayed != nil
	c.mu.Unlock()
	if !hasAlloc {
		return ErrNoAllocation
	}

	m := NewMessage(stunType(methodSend, classIndication))
	xp, err := xorAddr(peer.IP, peer.Port, m.TransactionID)
	if err != nil {
		return err
	}
	m.addAttr(attrXorPeerAddr, xp)
	m.addAttr(attrData, data)

	_, err = c.conn.Write(m.Encode())
	return err
}

// SendChannelData sends data via ChannelData framing (requires ChannelBind).
func (c *Client) SendChannelData(ch uint16, data []byte) error {
	if ch < channelMin || ch > channelMax {
		return ErrInvalidAddress
	}
	// ChannelData: channel(2), length(2), data...
	b := make([]byte, 4+len(data))
	binary.BigEndian.PutUint16(b[0:2], ch)
	binary.BigEndian.PutUint16(b[2:4], uint16(len(data)))
	copy(b[4:], data)
	_, err := c.conn.Write(b)
	return err
}

// ReadFrom reads either ChannelData or Data Indication and returns peer + payload.
// This is useful for receiving relayed data from TURN.
func (c *Client) ReadFrom(buf []byte) (peer *net.UDPAddr, n int, err error) {
	n, err = c.conn.Read(buf)
	if err != nil {
		return nil, 0, err
	}
	if n < 4 {
		return nil, 0, ErrBadMessage
	}

	// ChannelData frames start with channel number in [0x4000, 0x7FFF].
	ch := binary.BigEndian.Uint16(buf[0:2])
	if ch >= channelMin && ch <= channelMax {
		l := int(binary.BigEndian.Uint16(buf[2:4]))
		if 4+l > n {
			return nil, 0, ErrBadMessage
		}
		// Peer address is not carried in ChannelData; caller must map ch->peer.
		return nil, l, nil
	}

	// Otherwise try parse as STUN message (e.g. Data Indication).
	msg, perr := ParseMessage(buf[:n])
	if perr != nil {
		return nil, 0, perr
	}
	// Data Indication carries XOR-PEER-ADDRESS and DATA
	aPeer, ok := msg.FindAttr(attrXorPeerAddr)
	if !ok {
		return nil, 0, ErrBadMessage
	}
	peer, err = parseXorAddr(aPeer.Value, msg.TransactionID)
	if err != nil {
		return nil, 0, err
	}
	aData, ok := msg.FindAttr(attrData)
	if !ok {
		return peer, 0, nil
	}
	// Copy payload into buf head for convenience
	copy(buf, aData.Value)
	return peer, len(aData.Value), nil
}

// doRequest sends request and waits response (with timeout via ctx).
func (c *Client) doRequest(ctx context.Context, req *Message, withAuth bool) (*Message, error) {
	// If withAuth, add MESSAGE-INTEGRITY + FINGERPRINT.
	if withAuth {
		// MESSAGE-INTEGRITY must be before FINGERPRINT (recommended).
		key := longTermKey(c.creds.Username, c.realm, c.creds.Password)
		addMessageIntegrity(req, key)
		addFingerprint(req)
	}

	raw := req.Encode()

	// deadline from ctx
	if dl, ok := ctx.Deadline(); ok {
		_ = c.conn.SetReadDeadline(dl)
	} else {
		_ = c.conn.SetReadDeadline(time.Now().Add(defaultTimeout))
	}

	if _, err := c.conn.Write(raw); err != nil {
		return nil, err
	}

	// read response
	buf := make([]byte, 2048)
	n, err := c.conn.Read(buf)
	if err != nil {
		if isTimeout(err) {
			return nil, ErrTimeout
		}
		return nil, err
	}

	resp, err := ParseMessage(buf[:n])
	if err != nil {
		return nil, err
	}

	// Transaction ID must match for request/response.
	if resp.TransactionID != req.TransactionID {
		return nil, ErrBadMessage
	}

	// Handle error responses: read REALM/NONCE and map to ErrUnauthorized.
	if isErrorResponse(resp.Type) {
		// Parse ERROR-CODE
		if a, ok := resp.FindAttr(attrErrorCode); ok && len(a.Value) >= 4 {
			// 401 (Unauthorized) or 438 (Stale Nonce)
			code := int(a.Value[2])*100 + int(a.Value[3])
			realm := ""
			nonce := ""

			if r, ok := resp.FindAttr(attrRealm); ok {
				realm = string(r.Value)
			}
			if nn, ok := resp.FindAttr(attrNonce); ok {
				nonce = string(nn.Value)
			}
			c.setNonceRealm(realm, nonce)

			if code == 401 || code == 438 {
				return nil, ErrUnauthorized
			}
		}
		return nil, ErrBadMessage
	}

	if !isSuccessResponse(resp.Type) {
		return nil, ErrBadMessage
	}
	return resp, nil
}
