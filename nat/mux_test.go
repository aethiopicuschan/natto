package nat_test

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/aethiopicuschan/natto/nat"
	"github.com/stretchr/testify/assert"
)

func TestMuxAliasViaSession(t *testing.T) {
	t.Parallel()

	// Two UDP sockets.
	aConn, err := net.ListenUDP("udp", &net.UDPAddr{
		IP:   net.IPv4(127, 0, 0, 1),
		Port: 0,
	})
	assert.NoError(t, err)
	defer aConn.Close()

	bConn, err := net.ListenUDP("udp", &net.UDPAddr{
		IP:   net.IPv4(127, 0, 0, 1),
		Port: 0,
	})
	assert.NoError(t, err)
	defer bConn.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	aMux := nat.NewMux(aConn)
	bMux := nat.NewMux(bConn)
	aMux.Start(ctx)
	bMux.Start(ctx)

	// Session A expects packets from B.
	sessA := nat.NewSession(aMux, bConn.LocalAddr().(*net.UDPAddr), 4)

	// Simulate "port change" on B side by creating a new socket.
	bConn2, err := net.ListenUDP("udp", &net.UDPAddr{
		IP:   net.IPv4(127, 0, 0, 1),
		Port: 0,
	})
	assert.NoError(t, err)
	defer bConn2.Close()

	// Alias old address to new address.
	aMux.Alias(
		bConn.LocalAddr().(*net.UDPAddr),
		bConn2.LocalAddr().(*net.UDPAddr),
	)

	// Send data from the new port.
	payload := []byte("hello via alias")
	wire, err := nat.EncodePacket(nat.PacketData, payload)
	assert.NoError(t, err)

	_, err = bConn2.WriteToUDP(
		wire,
		aConn.LocalAddr().(*net.UDPAddr),
	)
	assert.NoError(t, err)

	recvCtx, cancelRecv := context.WithTimeout(ctx, time.Second)
	defer cancelRecv()

	got, _, err := sessA.Recv(recvCtx)
	assert.NoError(t, err)
	assert.Equal(t, payload, got)
}
