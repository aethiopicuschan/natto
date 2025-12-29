// server.go
package stun

import (
	"context"
	"encoding/binary"
	"errors"
	"net"
	"sync"
	"time"
)

// Server is a minimal STUN server (RFC 5389) that supports UDP Binding requests.
//
// It listens on UDP, parses STUN messages, and replies to Binding Requests with
// a Binding Success Response that includes XOR-MAPPED-ADDRESS.
//
// This is intentionally small and designed to be embedded into your NAT/P2P stack.
type Server struct {
	// Conn is the UDP socket the server reads from and writes to.
	Conn *net.UDPConn

	// Software, if non-empty, is included as a SOFTWARE attribute in responses.
	Software string

	// ReadTimeout, if > 0, sets a read deadline each loop iteration.
	// This is mainly useful to make shutdown (Close) more responsive.
	ReadTimeout time.Duration

	// MaxPacketSize is the max UDP datagram size to read into the buffer.
	// If zero, defaults to 1500.
	MaxPacketSize int

	onceClose sync.Once
	closeCh   chan struct{}
	wg        sync.WaitGroup
}

// ListenUDP creates a UDP STUN server bound to addr (e.g. "0.0.0.0:3478").
//
// Call Serve/ServeContext to start handling requests.
func ListenUDP(addr string) (*Server, error) {
	udpAddr, err := net.ResolveUDPAddr("udp", addr)
	if err != nil {
		return nil, err
	}
	conn, err := net.ListenUDP("udp", udpAddr)
	if err != nil {
		return nil, err
	}
	return &Server{
		Conn:          conn,
		ReadTimeout:   1 * time.Second,
		MaxPacketSize: 1500,
		closeCh:       make(chan struct{}),
	}, nil
}

// Close stops the server and closes the underlying UDP socket.
//
// Close is safe to call multiple times.
func (s *Server) Close() error {
	var err error
	s.onceClose.Do(func() {
		close(s.closeCh)
		if s.Conn != nil {
			err = s.Conn.Close()
		}
	})
	s.wg.Wait()
	return err
}

// Serve starts the server loop and blocks until the connection is closed
// or Close() is called.
func (s *Server) Serve() error {
	return s.ServeContext(context.Background())
}

// ServeContext starts the server loop and blocks until ctx is done,
// the connection is closed, or Close() is called.
func (s *Server) ServeContext(ctx context.Context) error {
	if s.Conn == nil {
		return errors.New("stun: server Conn is nil")
	}
	s.wg.Add(1)
	defer s.wg.Done()

	max := s.MaxPacketSize
	if max <= 0 {
		max = 1500
	}
	buf := make([]byte, max)

	for {
		// Allow responsive shutdown.
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-s.closeCh:
			return nil
		default:
		}

		if s.ReadTimeout > 0 {
			_ = s.Conn.SetReadDeadline(time.Now().Add(s.ReadTimeout))
		}

		n, raddr, err := s.Conn.ReadFromUDP(buf)
		if err != nil {
			// If this is a timeout, just continue to allow checking ctx/closeCh.
			if ne, ok := err.(net.Error); ok && ne.Timeout() {
				continue
			}
			// Closed socket -> exit cleanly.
			select {
			case <-s.closeCh:
				return nil
			default:
			}
			return err
		}

		// Handle each packet inline (fast path). If you expect heavy load,
		// you can fork this into a goroutine pool.
		s.handlePacket(buf[:n], raddr)
	}
}

// handlePacket parses a STUN request and replies if it is a supported Binding Request.
func (s *Server) handlePacket(pkt []byte, raddr *net.UDPAddr) {
	req, err := Parse(pkt)
	if err != nil {
		// Ignore non-STUN packets or malformed messages.
		return
	}

	// Only handle Binding Requests.
	if req.Method != MethodBinding || req.Class != ClassRequest {
		// For minimal server, ignore other methods/classes.
		return
	}

	resp := s.makeBindingSuccess(req, raddr)
	_, _ = s.Conn.WriteToUDP(resp.Marshal(), raddr)
}

// makeBindingSuccess builds a Binding Success Response with XOR-MAPPED-ADDRESS.
func (s *Server) makeBindingSuccess(req *Message, raddr *net.UDPAddr) *Message {
	attrs := make([]Attribute, 0, 2)
	attrs = append(attrs, buildXORMappedAddressAttr(raddr, req.TransactionID))

	if s.Software != "" {
		attrs = append(attrs, buildSoftwareAttr(s.Software))
	}

	return &Message{
		Method:        MethodBinding,
		Class:         ClassSuccessResponse,
		Cookie:        MagicCookie,
		TransactionID: req.TransactionID,
		Attributes:    attrs,
	}
}

// buildSoftwareAttr encodes a SOFTWARE attribute (RFC 5389).
func buildSoftwareAttr(software string) Attribute {
	// SOFTWARE is a UTF-8 string. No padding here; Message.Marshal handles padding.
	return Attribute{
		Type:  AttrSoftware,
		Value: []byte(software),
	}
}

// buildXORMappedAddressAttr encodes XOR-MAPPED-ADDRESS for the given remote address.
func buildXORMappedAddressAttr(raddr *net.UDPAddr, tid TransactionID) Attribute {
	ip := raddr.IP
	port := raddr.Port

	// Prefer IPv4 if possible.
	if ip4 := ip.To4(); ip4 != nil {
		v := make([]byte, 8)
		v[0] = 0x00
		v[1] = 0x01 // IPv4

		// Port is XOR'ed with the top 16 bits of the magic cookie.
		xPort := uint16(port) ^ uint16(MagicCookie>>16)
		binary.BigEndian.PutUint16(v[2:4], xPort)

		// Address is XOR'ed with the magic cookie.
		c := make([]byte, 4)
		binary.BigEndian.PutUint32(c, MagicCookie)
		for i := range 4 {
			v[4+i] = ip4[i] ^ c[i]
		}

		return Attribute{Type: AttrXORMappedAddress, Value: v}
	}

	// IPv6
	ip16 := ip.To16()
	if ip16 == nil {
		// If address is not valid IP, send a minimal response without XOR-MAPPED-ADDRESS.
		// (Client will likely fail; this should not happen with net.UDPAddr.)
		return Attribute{Type: AttrXORMappedAddress, Value: nil}
	}

	v := make([]byte, 20)
	v[0] = 0x00
	v[1] = 0x02 // IPv6

	xPort := uint16(port) ^ uint16(MagicCookie>>16)
	binary.BigEndian.PutUint16(v[2:4], xPort)

	// Key = magic cookie (4 bytes) || transaction ID (12 bytes)
	key := make([]byte, 16)
	binary.BigEndian.PutUint32(key[0:4], MagicCookie)
	copy(key[4:16], tid[:])

	for i := range 16 {
		v[4+i] = ip16[i] ^ key[i]
	}

	return Attribute{Type: AttrXORMappedAddress, Value: v}
}
