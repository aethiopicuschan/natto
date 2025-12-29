package turn

import (
	"crypto/hmac"
	"crypto/md5"
	"crypto/sha1"
	"encoding/binary"
	"hash/crc32"
)

// longTermKey derives the long-term credential key:
// key = MD5(username ":" realm ":" password) (RFC 5389)
func longTermKey(username, realm, password string) []byte {
	h := md5.Sum([]byte(username + ":" + realm + ":" + password))
	return h[:]
}

// addMessageIntegrity appends MESSAGE-INTEGRITY attribute computed over the message
// up to (and including) the attribute header of MESSAGE-INTEGRITY, with length set accordingly.
func addMessageIntegrity(m *Message, key []byte) {
	// Placeholder 20 bytes for HMAC-SHA1
	m.addAttr(attrMessageIntegrity, make([]byte, 20))

	raw := m.Encode()

	// Compute HMAC-SHA1 over raw with MESSAGE-INTEGRITY value present (per RFC rules).
	mac := hmac.New(sha1.New, key)
	mac.Write(raw)
	sum := mac.Sum(nil)

	// Replace last MESSAGE-INTEGRITY value.
	for i := range m.Attrs {
		if m.Attrs[i].Type == attrMessageIntegrity && len(m.Attrs[i].Value) == 20 {
			copy(m.Attrs[i].Value, sum)
			return
		}
	}
}

// addFingerprint appends FINGERPRINT attribute (CRC32 XOR 0x5354554e).
func addFingerprint(m *Message) {
	// Append placeholder first
	m.addAttr(attrFingerprint, make([]byte, 4))

	raw := m.Encode()
	c := crc32.ChecksumIEEE(raw)
	c ^= 0x5354554e

	// Replace value
	for i := range m.Attrs {
		if m.Attrs[i].Type == attrFingerprint && len(m.Attrs[i].Value) == 4 {
			binary.BigEndian.PutUint32(m.Attrs[i].Value, c)
			return
		}
	}
}
