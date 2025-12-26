package nat

import (
	"context"
	"net"
	"sync"
	"time"
)

// Session represents an established path to a peer over a shared Mux.
type Session struct {
	mux        *Mux
	remoteAddr *net.UDPAddr
	in         <-chan inbound

	mu                sync.RWMutex
	closed            bool
	keepaliveInterval time.Duration
}

// NewSession creates a new Session to the given remote address over the Mux.
func NewSession(mux *Mux, remote *net.UDPAddr, queue int) *Session {
	return &Session{
		mux:        mux,
		remoteAddr: remote,
		in:         mux.Register(remote, queue),
	}
}

// -----------------------------------------------------------------------------
// Data plane (application payload)
// -----------------------------------------------------------------------------

// SendData sends application data to the remote peer.
func (s *Session) SendData(p []byte) error {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.closed {
		return ErrConnectionClosed
	}
	return s.mux.Send(s.remoteAddr, PacketData, p)
}

// RecvData receives application data from the remote peer.
func (s *Session) RecvData(ctx context.Context) ([]byte, *net.UDPAddr, error) {
	for {
		select {
		case <-ctx.Done():
			return nil, nil, ctx.Err()

		case inb, ok := <-s.in:
			if !ok {
				return nil, nil, ErrConnectionClosed
			}
			if inb.pkt.Kind != PacketData {
				continue
			}
			return inb.pkt.Payload, inb.addr, nil
		}
	}
}

// -----------------------------------------------------------------------------
// Control plane (meta / coordination)
// -----------------------------------------------------------------------------

// SendControl sends a control packet to the peer.
func (s *Session) SendControl(p []byte) error {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.closed {
		return ErrConnectionClosed
	}
	return s.mux.Send(s.remoteAddr, PacketControl, p)
}

// RecvControl receives a control packet from the peer.
func (s *Session) RecvControl(ctx context.Context) ([]byte, *net.UDPAddr, error) {
	for {
		select {
		case <-ctx.Done():
			return nil, nil, ctx.Err()

		case inb, ok := <-s.in:
			if !ok {
				return nil, nil, ErrConnectionClosed
			}
			if inb.pkt.Kind != PacketControl {
				continue
			}
			return inb.pkt.Payload, inb.addr, nil
		}
	}
}

// -----------------------------------------------------------------------------
// Backward compatibility (defaults to data plane)
// -----------------------------------------------------------------------------

// Send sends application data to the remote peer.
func (s *Session) Send(p []byte) error {
	return s.SendData(p)
}

// Recv receives application data from the remote peer.
func (s *Session) Recv(ctx context.Context) ([]byte, *net.UDPAddr, error) {
	return s.RecvData(ctx)
}

// -----------------------------------------------------------------------------
// Keepalive (control plane)
// -----------------------------------------------------------------------------

// SetKeepalive sets the keepalive interval for the session.
func (s *Session) SetKeepalive(interval time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.keepaliveInterval = interval
}

// StartKeepalive starts the keepalive goroutine.
func (s *Session) StartKeepalive(ctx context.Context) {
	s.mu.RLock()
	interval := s.keepaliveInterval
	s.mu.RUnlock()

	if interval <= 0 {
		return
	}

	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				// keepalive is a control packet with empty payload
				_ = s.SendControl(nil)
			}
		}
	}()
}

// Close closes the session.
func (s *Session) Close() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.closed = true
}

// UpdateRemote updates the remote address used for sending,
// and aliases the new address to the existing inbound channel.
func (s *Session) UpdateRemote(newRemote *net.UDPAddr) {
	if newRemote == nil {
		return
	}

	// Alias new remote address to the existing inbound channel.
	s.mux.Alias(s.remoteAddr, newRemote)

	s.mu.Lock()
	s.remoteAddr = newRemote
	s.mu.Unlock()
}
