package nat

import "net"

// Peer represents a remote peer in the P2P connection.
type Peer struct {
	ID string

	// Addr is the peer's externally reachable UDP address (primary).
	Addr *net.UDPAddr

	// Candidates is an optional list of alternate UDP addresses to try (ICE-lite).
	// If empty, Puncher uses Addr only.
	Candidates []*net.UDPAddr

	LocalAddr *net.UDPAddr
}
