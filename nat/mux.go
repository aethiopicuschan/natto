package nat

import (
	"context"
	"net"
	"sync"
)

// inbound represents a received packet with its source address.
type inbound struct {
	pkt  *Packet
	addr *net.UDPAddr
}

// Mux multiplexes incoming UDP packets by address and control semantics.
type Mux struct {
	conn *net.UDPConn

	// Address-based demux
	addrMu sync.RWMutex
	byAddr map[string]chan inbound

	// Control fallback channel
	controlCh chan inbound

	// Control demux by peer ID
	controlMu     sync.RWMutex
	controlByPeer map[string]chan inbound

	startOnce sync.Once
}

// NewMux creates a new Mux for the given UDP connection.
func NewMux(conn *net.UDPConn) *Mux {
	return &Mux{
		conn:          conn,
		byAddr:        make(map[string]chan inbound),
		controlCh:     make(chan inbound, 32),
		controlByPeer: make(map[string]chan inbound),
	}
}

// Start begins the receive loop.
// It must be called exactly once.
func (m *Mux) Start(ctx context.Context) {
	m.startOnce.Do(func() {
		go m.recvLoop(ctx)
	})
}

// Control returns the fallback control channel.
// Packets not addressed to a specific peer are delivered here.
func (m *Mux) Control() <-chan inbound {
	return m.controlCh
}

// ControlFor returns a dedicated control channel for the given peer ID.
// Only control packets addressed to this peer will be delivered.
func (m *Mux) ControlFor(peerID string) <-chan inbound {
	m.controlMu.Lock()
	defer m.controlMu.Unlock()

	ch, ok := m.controlByPeer[peerID]
	if !ok {
		ch = make(chan inbound, 32)
		m.controlByPeer[peerID] = ch
	}
	return ch
}

// Register registers a channel for packets from the given address.
func (m *Mux) Register(addr *net.UDPAddr, queue int) <-chan inbound {
	key := addr.String()

	if queue <= 0 {
		queue = 32
	}

	m.addrMu.Lock()
	defer m.addrMu.Unlock()

	ch, ok := m.byAddr[key]
	if !ok {
		ch = make(chan inbound, queue)
		m.byAddr[key] = ch
	}
	return ch
}

// Alias aliases packets from oldAddr to newAddr.
func (m *Mux) Alias(oldAddr, newAddr *net.UDPAddr) {
	if oldAddr == nil || newAddr == nil {
		return
	}

	oldKey := oldAddr.String()
	newKey := newAddr.String()

	m.addrMu.Lock()
	defer m.addrMu.Unlock()

	ch, ok := m.byAddr[oldKey]
	if !ok {
		return
	}

	delete(m.byAddr, oldKey)
	m.byAddr[newKey] = ch
}

// Send sends a packet to the given address.
func (m *Mux) Send(addr *net.UDPAddr, kind PacketKind, payload []byte) error {
	wire, err := EncodePacket(kind, payload)
	if err != nil {
		return err
	}
	_, err = m.conn.WriteToUDP(wire, addr)
	return err
}

// recvLoop reads packets from the UDP connection and dispatches them.
func (m *Mux) recvLoop(ctx context.Context) {
	buf := make([]byte, 64*1024)

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		n, addr, err := m.conn.ReadFromUDP(buf)
		if err != nil {
			continue
		}

		frame := make([]byte, n)
		copy(frame, buf[:n])

		pkt, err := DecodePacket(frame)
		if err != nil {
			continue
		}

		inb := inbound{
			pkt:  pkt,
			addr: addr,
		}

		// First try address-based dispatch.
		if m.dispatchByAddr(inb) {
			continue
		}

		// Otherwise, handle control demux.
		if pkt.Kind == PacketControl {
			m.dispatchControl(inb)
		}
	}
}

// dispatchByAddr dispatches packets by source address.
// Returns true if dispatched.
func (m *Mux) dispatchByAddr(inb inbound) bool {
	key := inb.addr.String()

	m.addrMu.RLock()
	ch, ok := m.byAddr[key]
	m.addrMu.RUnlock()

	if ok {
		select {
		case ch <- inb:
		default:
		}
		return true
	}
	return false
}

// dispatchControl dispatches control packets by ToPeerID.
func (m *Mux) dispatchControl(inb inbound) {
	msg, err := DecodeMessage(inb.pkt.Payload)
	if err != nil || msg.ToPeerID == "" {
		m.controlCh <- inb
		return
	}

	m.controlMu.RLock()
	ch, ok := m.controlByPeer[msg.ToPeerID]
	m.controlMu.RUnlock()

	if ok {
		select {
		case ch <- inb:
		default:
		}
	} else {
		m.controlCh <- inb
	}
}
