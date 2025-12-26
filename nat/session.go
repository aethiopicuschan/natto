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

func NewSession(mux *Mux, remote *net.UDPAddr, queue int) *Session {
	return &Session{
		mux:        mux,
		remoteAddr: remote,
		in:         mux.Register(remote, queue),
	}
}

func (s *Session) SetKeepalive(interval time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.keepaliveInterval = interval
}

func (s *Session) Send(p []byte) error {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.closed {
		return ErrConnectionClosed
	}
	return s.mux.Send(s.remoteAddr, PacketData, p)
}

func (s *Session) Recv(ctx context.Context) ([]byte, *net.UDPAddr, error) {
	for {
		select {
		case <-ctx.Done():
			return nil, nil, ctx.Err()

		case inb, ok := <-s.in:
			if !ok {
				return nil, nil, ErrConnectionClosed
			}
			if inb.pkt.Kind != PacketData {
				// Session ignores control packets; those are handled by Puncher/Manager.
				continue
			}
			return inb.pkt.Payload, inb.addr, nil
		}
	}
}

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
				_ = s.Send(nil)
			}
		}
	}()
}

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
