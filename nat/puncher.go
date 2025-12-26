package nat

import (
	"context"
	"net"
	"sync"
	"time"
)

type NATBehavior string

const (
	NATUnknown             NATBehavior = "unknown"
	NATEndpointIndependent NATBehavior = "endpoint-independent-like"
	NATEndpointDependent   NATBehavior = "endpoint-dependent-like"
)

// PunchResult represents the outcome of a hole punching attempt.
type PunchResult struct {
	Addr   *net.UDPAddr // observed remote addr that reached us
	PeerID string

	// Behavior is a best-effort heuristic based on observed address changes.
	// Note: with only two peers (no STUN server) this cannot be definitive.
	Behavior NATBehavior
}

type punchState int

const (
	stateInit punchState = iota
	statePeerKnown
	stateDone
)

// Puncher performs message-based UDP hole punching over a shared Mux.
type Puncher struct {
	mux    *Mux
	selfID string

	// initInterval is used before we learn peerID/addr well.
	initInterval time.Duration

	// steadyInterval is used after peer is known (less spammy).
	steadyInterval time.Duration
}

// NewPuncher creates a new Puncher.
// interval is treated as the "steady" interval. init interval becomes smaller.
func NewPuncher(mux *Mux, selfID string, interval time.Duration) *Puncher {
	if interval <= 0 {
		interval = 200 * time.Millisecond
	}
	init := interval / 2
	if init < 25*time.Millisecond {
		init = 25 * time.Millisecond
	}
	return &Puncher{
		mux:            mux,
		selfID:         selfID,
		initInterval:   init,
		steadyInterval: interval,
	}
}

// Punch attempts to establish reachability with the given peer.
//
// Design notes (important):
//   - We ALWAYS listen on both Control() and ControlFor(selfID). Do NOT gate listening by state.
//     Gating causes ACK loss depending on timing (Accept sends ACK with ToPeerID set).
//   - State is used to change send strategy and intervals, not to "route" packets.
func (p *Puncher) Punch(ctx context.Context, peer *Peer) (*PunchResult, error) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// --- shared mutable observed state ---
	var mu sync.Mutex
	state := stateInit

	remoteAddr := (*net.UDPAddr)(nil)
	peerID := ""
	if peer != nil {
		remoteAddr = peer.Addr
		peerID = peer.ID
	}

	// ICE-lite candidates
	var candidates []*net.UDPAddr
	if peer != nil {
		if peer.Addr != nil {
			candidates = append(candidates, peer.Addr)
		}
		for _, c := range peer.Candidates {
			if c == nil {
				continue
			}
			dup := false
			for _, e := range candidates {
				if e.String() == c.String() {
					dup = true
					break
				}
			}
			if !dup {
				candidates = append(candidates, c)
			}
		}
	}

	// behavior heuristic bookkeeping
	var firstObserved *net.UDPAddr
	behavior := NATUnknown

	setObserved := func(addr *net.UDPAddr, id string) {
		mu.Lock()
		defer mu.Unlock()

		if id != "" {
			peerID = id
			if state == stateInit {
				state = statePeerKnown
			}
		}

		if addr != nil {
			if firstObserved == nil {
				firstObserved = addr
			} else if firstObserved.String() != addr.String() {
				// We saw the peer from a different endpoint during handshake.
				// With only two peers, this is merely suggestive of endpoint-dependent behavior.
				behavior = NATEndpointDependent
			}
			// keep remoteAddr current and alias for inbound demux continuity
			if remoteAddr == nil || remoteAddr.String() != addr.String() {
				if remoteAddr != nil {
					p.mux.Alias(remoteAddr, addr)
				}
				remoteAddr = addr
			}
		}

		if behavior == NATUnknown && firstObserved != nil {
			// If we never observed changes, call it "endpoint-independent-like" heuristically.
			behavior = NATEndpointIndependent
		}
	}

	getSnapshot := func() (st punchState, id string, addr *net.UDPAddr, cands []*net.UDPAddr, beh NATBehavior) {
		mu.Lock()
		defer mu.Unlock()

		st = state
		id = peerID
		addr = remoteAddr
		beh = behavior

		// return a shallow copy
		if len(candidates) > 0 {
			cands = append([]*net.UDPAddr(nil), candidates...)
		}
		return
	}

	// --- result signaling (once) ---
	resultCh := make(chan *PunchResult, 1)
	var once sync.Once
	succeed := func(addr *net.UDPAddr, id string) {
		once.Do(func() {
			_, _, _, _, beh := getSnapshot()
			resultCh <- &PunchResult{Addr: addr, PeerID: id, Behavior: beh}
			mu.Lock()
			state = stateDone
			mu.Unlock()
		})
	}

	// --- channels ---
	fallback := p.mux.Control()
	dedicated := p.mux.ControlFor(p.selfID)

	// If we know at least one candidate, also register address-based channels (helps when control demux misses).
	// We register each candidate with its own queue and fan-in in receive loop.
	type addrListen struct {
		addr *net.UDPAddr
		ch   <-chan inbound
	}
	var addrListens []addrListen
	{
		_, _, _, cands, _ := getSnapshot()
		for _, a := range cands {
			addrListens = append(addrListens, addrListen{
				addr: a,
				ch:   p.mux.Register(a, 16),
			})
		}
	}

	handleInbound := func(inb inbound) {
		if inb.pkt.Kind != PacketControl {
			return
		}
		msg, err := DecodeMessage(inb.pkt.Payload)
		if err != nil {
			return
		}
		if msg.ToPeerID != "" && msg.ToPeerID != p.selfID {
			return
		}

		switch msg.Type {
		case MessageHello:
			// learn peer and reply ack
			setObserved(inb.addr, msg.PeerID)

			ack := &Message{
				Type:      MessageAck,
				PeerID:    p.selfID,
				ToPeerID:  msg.PeerID,
				Timestamp: time.Now().UnixNano(),
			}
			if payload, err := EncodeMessage(ack); err == nil {
				_ = p.mux.Send(inb.addr, PacketControl, payload)
			}

			// success on hello-received (prevents half-open)
			succeed(inb.addr, msg.PeerID)

		case MessageAck:
			setObserved(inb.addr, msg.PeerID)
			succeed(inb.addr, msg.PeerID)
		}
	}

	// --- receive loop (ALWAYS listens; never state-gated) ---
	go func() {
		for {
			select {
			case <-ctx.Done():
				return

			case inb := <-fallback:
				handleInbound(inb)
			case inb := <-dedicated:
				handleInbound(inb)

			default:
				// poll addr channels without blocking forever on one
				// (keeps this loop responsive; small overhead but test-stable)
				handled := false
				for _, al := range addrListens {
					select {
					case inb := <-al.ch:
						handleInbound(inb)
						handled = true
					default:
					}
				}
				if handled {
					continue
				}
				// avoid busy loop
				time.Sleep(200 * time.Microsecond)
			}
		}
	}()

	// --- send strategy (state machine decides interval + destinations) ---
	sendHelloTo := func(to *net.UDPAddr, toPeerID string) {
		if to == nil {
			return
		}
		hello := &Message{
			Type:      MessageHello,
			PeerID:    p.selfID,
			ToPeerID:  toPeerID,
			Timestamp: time.Now().UnixNano(),
		}
		if payload, err := EncodeMessage(hello); err == nil {
			_ = p.mux.Send(to, PacketControl, payload)
		}
	}

	// one immediate burst to reduce first-RTT variance
	{
		st, id, addr, cands, _ := getSnapshot()
		if st == stateInit {
			// init: spray to all candidates (ICE-lite)
			for _, c := range cands {
				sendHelloTo(c, id)
			}
			// also send to addr (if set but not in candidates for some reason)
			sendHelloTo(addr, id)
		} else {
			sendHelloTo(addr, id)
		}
	}

	// ticker uses dynamic interval: initInterval until peer known, then steadyInterval
	ticker := time.NewTicker(p.initInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			if ctx.Err() == context.DeadlineExceeded {
				return nil, ErrPunchTimeout
			}
			return nil, ctx.Err()

		case res := <-resultCh:
			return res, nil

		case <-ticker.C:
			st, id, addr, cands, _ := getSnapshot()

			if st == stateDone {
				// should be canceled anyway
				return nil, ErrPunchTimeout
			}

			// switch interval once peer known
			if st == statePeerKnown {
				// move to steady interval if we're still on init ticker
				// (best-effort; no need to check current duration precisely)
				ticker.Stop()
				ticker = time.NewTicker(p.steadyInterval)
			}

			// INIT: send to all candidates (fallback-centric discovery)
			if st == stateInit {
				for _, c := range cands {
					sendHelloTo(c, id)
				}
				sendHelloTo(addr, id)
				continue
			}

			// PEER_KNOWN: send only to the currently best observed addr
			sendHelloTo(addr, id)
		}
	}
}
