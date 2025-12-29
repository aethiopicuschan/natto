package turn_test

import (
	"net"
	"testing"

	"github.com/aethiopicuschan/natto/turn"
	"github.com/stretchr/testify/assert"
)

func TestXorAddrIPv4(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		ip   string
		port int
	}{
		{"private", "192.168.0.1", 12345},
		{"public", "8.8.8.8", 53},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ip := net.ParseIP(tt.ip)
			var tid [12]byte

			raw, err := turn.XorAddr(ip, tt.port, tid)
			assert.NoError(t, err)

			addr, err := turn.ParseXorAddr(raw, tid)
			assert.NoError(t, err)
			assert.Equal(t, tt.port, addr.Port)
			assert.True(t, addr.IP.Equal(ip))
		})
	}
}
