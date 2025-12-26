package nat

import (
	"context"
	"time"
)

// AcceptOptions configures Accept behavior.
type AcceptOptions struct {
	Queue             int
	KeepaliveInterval time.Duration
}

// Acceptor waits for incoming hole-punching attempts.
type Acceptor struct {
	mux    *Mux
	selfID string
	opts   AcceptOptions

	closed chan struct{}
}

func NewAcceptor(mux *Mux, selfID string, opts AcceptOptions) *Acceptor {
	return &Acceptor{
		mux:    mux,
		selfID: selfID,
		opts:   opts,
		closed: make(chan struct{}),
	}
}

// Accept waits for a peer to initiate hole punching and establishes a Session.
// Only a single peer is accepted per Acceptor.
func (a *Acceptor) Accept(ctx context.Context) (*Session, *PunchResult, error) {
	control := a.mux.Control()

	for {
		select {
		case <-ctx.Done():
			return nil, nil, ctx.Err()
		case <-a.closed:
			return nil, nil, ErrConnectionClosed
		case inb := <-control:
			if inb.pkt.Kind != PacketControl {
				continue
			}

			msg, err := DecodeMessage(inb.pkt.Payload)
			if err != nil {
				continue
			}
			if msg.Type != MessageHello {
				continue
			}

			// If the initiator specified the destination, ensure it's for us.
			if msg.ToPeerID != "" && msg.ToPeerID != a.selfID {
				continue
			}

			// Immediately ACK the first HELLO so the dialer can progress without waiting
			// for a second HELLO tick.
			ack := &Message{
				Type:      MessageAck,
				PeerID:    a.selfID,
				ToPeerID:  msg.PeerID,
				Timestamp: time.Now().UnixNano(),
			}
			if payload, err := EncodeMessage(ack); err == nil {
				_ = a.mux.Send(inb.addr, PacketControl, payload)
			}

			res := &PunchResult{
				Addr:   inb.addr,
				PeerID: msg.PeerID,
			}

			queue := a.opts.Queue
			if queue <= 0 {
				queue = 32
			}

			sess := NewSession(a.mux, res.Addr, queue)

			if a.opts.KeepaliveInterval > 0 {
				sess.SetKeepalive(a.opts.KeepaliveInterval)
				sess.StartKeepalive(ctx)
			}

			return sess, res, nil
		}
	}
}

func (a *Acceptor) Close() {
	close(a.closed)
}
