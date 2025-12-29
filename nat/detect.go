package nat

import (
	"context"
	"net"
	"time"

	"github.com/aethiopicuschan/natto/stun"
)

// NATType represents a coarse NAT classification.
type NATType string

const (
	NATOpenInternet   NATType = "Open Internet"
	NATFullCone       NATType = "Full Cone NAT"
	NATRestricted     NATType = "Restricted Cone NAT"
	NATPortRestricted NATType = "Port Restricted Cone NAT"
	NATSymmetric      NATType = "Symmetric NAT"
)

// MappingBehavior describes how external ports are mapped.
type MappingBehavior string

const (
	MappingIndependent MappingBehavior = "Endpoint Independent"
	MappingDependent   MappingBehavior = "Address/Port Dependent"
)

// FilteringBehavior describes inbound filtering rules.
type FilteringBehavior string

const (
	FilteringNone        FilteringBehavior = "None"
	FilteringAddress     FilteringBehavior = "Address Restricted"
	FilteringPort        FilteringBehavior = "Port Restricted"
	FilteringAddressPort FilteringBehavior = "Address and Port Restricted"
)

// NATResult is the final detection result.
type NATResult struct {
	LocalAddr *net.UDPAddr

	MappedAddr1 stun.MappedAddress
	MappedAddr2 stun.MappedAddress

	Type       NATType
	Mapping    MappingBehavior
	Filtering  FilteringBehavior
	PunchingOK bool
}

// DetectNAT performs a best-effort NAT type detection using STUN.
func DetectNAT(
	ctx context.Context,
	conn *net.UDPConn,
	opts ...DetectOption,
) (*NATResult, error) {

	cfg := detectConfig{
		STUNServers: []string{
			"stun.l.google.com:19302",
			"stun1.l.google.com:19302",
		},
		Timeout: 2 * time.Second,
	}
	for _, opt := range opts {
		opt(&cfg)
	}

	result := &NATResult{
		LocalAddr: conn.LocalAddr().(*net.UDPAddr),
	}

	client := stun.NewClient()
	client.Timeout = cfg.Timeout

	// ---- STUN #1 ----
	m1, err := stunBind(ctx, client, cfg.STUNServers[0])
	if err != nil {
		return nil, err
	}
	result.MappedAddr1 = m1

	// ---- STUN #2 ----
	m2, err := stunBind(ctx, client, cfg.STUNServers[1])
	if err != nil {
		return nil, err
	}
	result.MappedAddr2 = m2

	classifyNAT(result)
	return result, nil
}

// ---- internal helpers ----

func stunBind(
	ctx context.Context,
	client *stun.Client,
	addr string,
) (stun.MappedAddress, error) {

	raddr, err := net.ResolveUDPAddr("udp", addr)
	if err != nil {
		return stun.MappedAddress{}, err
	}

	conn, err := net.DialUDP("udp", nil, raddr)
	if err != nil {
		return stun.MappedAddress{}, err
	}
	defer conn.Close()

	return client.BindingRequestConn(ctx, conn)
}

func classifyNAT(r *NATResult) {
	local := r.LocalAddr
	m1 := r.MappedAddr1
	m2 := r.MappedAddr2

	// Open Internet
	if m1.IP.Equal(local.IP) && m1.Port == local.Port {
		r.Type = NATOpenInternet
		r.Mapping = MappingIndependent
		r.Filtering = FilteringNone
		r.PunchingOK = true
		return
	}

	// Mapping behavior
	if m1.IP.Equal(m2.IP) && m1.Port == m2.Port {
		r.Mapping = MappingIndependent
	} else {
		r.Mapping = MappingDependent
	}

	// Final classification
	if r.Mapping == MappingIndependent {
		r.Type = NATPortRestricted
		r.Filtering = FilteringPort
		r.PunchingOK = true
	} else {
		r.Type = NATSymmetric
		r.Filtering = FilteringAddressPort
		r.PunchingOK = false
	}
}

type detectConfig struct {
	STUNServers []string
	Timeout     time.Duration
}

type DetectOption func(*detectConfig)

func WithSTUNServers(servers ...string) DetectOption {
	return func(c *detectConfig) {
		c.STUNServers = servers
	}
}

func WithDetectTimeout(d time.Duration) DetectOption {
	return func(c *detectConfig) {
		c.Timeout = d
	}
}
