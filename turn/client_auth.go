package turn

// addAuthAttrs adds USERNAME/REALM/NONCE for long-term credentials auth.
// MESSAGE-INTEGRITY/FINGERPRINT are added later in doRequest() because they
// must be computed over the final message bytes.
func (c *Client) addAuthAttrs(m *Message) {
	c.mu.Lock()
	realm := c.realm
	nonce := c.nonce
	c.mu.Unlock()

	if c.creds.Username != "" {
		m.addAttr(attrUsername, []byte(c.creds.Username))
	}
	if realm != "" {
		m.addAttr(attrRealm, []byte(realm))
	}
	if nonce != "" {
		m.addAttr(attrNonce, []byte(nonce))
	}
}
