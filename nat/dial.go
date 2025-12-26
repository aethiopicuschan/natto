package nat

import (
	"context"
	"time"
)

// DialOptions configures dialing behavior.
type DialOptions struct {
	// Interval is how often HELLO is sent during punching.
	Interval time.Duration

	// Queue is the inbound queue size for the created Session.
	Queue int

	// KeepaliveInterval enables session keepalive if > 0.
	KeepaliveInterval time.Duration
}

// Dial performs NAT traversal with the peer and returns a Session on success.
// The Mux must already be started.
func Dial(ctx context.Context, mux *Mux, selfID string, peer *Peer, opt DialOptions) (sess *Session, pr *PunchResult, err error) {
	interval := opt.Interval
	if interval <= 0 {
		interval = 200 * time.Millisecond
	}

	queue := opt.Queue
	if queue <= 0 {
		queue = 32
	}

	p := NewPuncher(mux, selfID, interval)

	pr, err = p.Punch(ctx, peer)
	if err != nil {
		return
	}

	sess = NewSession(mux, pr.Addr, queue)
	sess.UpdateRemote(pr.Addr)

	if opt.KeepaliveInterval > 0 {
		sess.SetKeepalive(opt.KeepaliveInterval)
		sess.StartKeepalive(ctx)
	}

	return
}
