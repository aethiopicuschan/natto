package turn

import (
	"errors"
	"net"
	"sync"
	"time"
)

const defaultTimeout = 5 * time.Second

// Credentials holds long-term auth parameters.
type Credentials struct {
	Username string
	Password string
}

// Client is a TURN client over UDP to a TURN server.
type Client struct {
	server *net.UDPAddr
	conn   *net.UDPConn

	creds Credentials

	mu    sync.Mutex
	realm string
	nonce string

	// allocation state
	relayed  *net.UDPAddr
	lifetime time.Duration
}

// Dial creates a UDP client connected to TURN server address (host:port).
func Dial(serverAddr string, creds Credentials) (*Client, error) {
	srv, err := net.ResolveUDPAddr("udp", serverAddr)
	if err != nil {
		return nil, err
	}
	c, err := net.DialUDP("udp", nil, srv)
	if err != nil {
		return nil, err
	}
	return &Client{
		server: srv,
		conn:   c,
		creds:  creds,
	}, nil
}

// Close closes the underlying UDP connection.
func (c *Client) Close() error {
	return c.conn.Close()
}

// RelayedAddr returns the allocated relayed address (if allocated).
func (c *Client) RelayedAddr() *net.UDPAddr {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.relayed == nil {
		return nil
	}
	cp := *c.relayed
	return &cp
}

// NonceRealm returns last known realm/nonce from server.
func (c *Client) NonceRealm() (realm, nonce string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.realm, c.nonce
}

// setNonceRealm updates realm/nonce from an error response.
func (c *Client) setNonceRealm(realm, nonce string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if realm != "" {
		c.realm = realm
	}
	if nonce != "" {
		c.nonce = nonce
	}
}

// setAllocation updates allocation state.
func (c *Client) setAllocation(relayed *net.UDPAddr, lifetimeSec int) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.relayed = relayed
	c.lifetime = time.Duration(lifetimeSec) * time.Second
}

func isTimeout(err error) bool {
	var ne net.Error
	if errors.As(err, &ne) && ne.Timeout() {
		return true
	}
	return false
}
