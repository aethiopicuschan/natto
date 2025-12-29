package nat_test

import (
	"net"
	"testing"

	"github.com/aethiopicuschan/natto/nat"
	"github.com/aethiopicuschan/natto/stun"
	"github.com/stretchr/testify/assert"
)

// helper: UDPAddr
func udpAddr(ip string, port int) *net.UDPAddr {
	return &net.UDPAddr{
		IP:   net.ParseIP(ip),
		Port: port,
	}
}

// helper: MappedAddress
func mapped(ip string, port int) stun.MappedAddress {
	return stun.MappedAddress{
		IP:   net.ParseIP(ip),
		Port: port,
	}
}

func TestClassifyNAT(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		local      *net.UDPAddr
		mapped1    stun.MappedAddress
		mapped2    stun.MappedAddress
		wantType   nat.NATType
		wantMap    nat.MappingBehavior
		wantFilter nat.FilteringBehavior
		wantPunch  bool
	}{
		{
			name:       "open internet",
			local:      udpAddr("203.0.113.10", 12345),
			mapped1:    mapped("203.0.113.10", 12345),
			mapped2:    mapped("203.0.113.10", 12345),
			wantType:   nat.NATOpenInternet,
			wantMap:    nat.MappingIndependent,
			wantFilter: nat.FilteringNone,
			wantPunch:  true,
		},
		{
			name:       "port restricted cone nat",
			local:      udpAddr("192.168.1.10", 54321),
			mapped1:    mapped("198.51.100.20", 60000),
			mapped2:    mapped("198.51.100.20", 60000),
			wantType:   nat.NATPortRestricted,
			wantMap:    nat.MappingIndependent,
			wantFilter: nat.FilteringPort,
			wantPunch:  true,
		},
		{
			name:       "symmetric nat",
			local:      udpAddr("192.168.1.10", 54321),
			mapped1:    mapped("198.51.100.20", 60000),
			mapped2:    mapped("198.51.100.20", 60001),
			wantType:   nat.NATSymmetric,
			wantMap:    nat.MappingDependent,
			wantFilter: nat.FilteringAddressPort,
			wantPunch:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			r := &nat.NATResult{
				LocalAddr:   tt.local,
				MappedAddr1: tt.mapped1,
				MappedAddr2: tt.mapped2,
			}

			nat.ExportClassifyNAT(r)

			assert.Equal(t, tt.wantType, r.Type)
			assert.Equal(t, tt.wantMap, r.Mapping)
			assert.Equal(t, tt.wantFilter, r.Filtering)
			assert.Equal(t, tt.wantPunch, r.PunchingOK)
		})
	}
}
