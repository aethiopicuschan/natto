package nat_test

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/aethiopicuschan/natto/nat"
	"github.com/stretchr/testify/assert"
)

func TestPuncherHandshake(t *testing.T) {
	t.Parallel()

	aConn, _ := net.ListenUDP("udp", &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 0})
	bConn, _ := net.ListenUDP("udp", &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 0})
	defer aConn.Close()
	defer bConn.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	aMux := nat.NewMux(aConn)
	bMux := nat.NewMux(bConn)
	aMux.Start(ctx)
	bMux.Start(ctx)

	peerA := &nat.Peer{ID: "A", Addr: aConn.LocalAddr().(*net.UDPAddr)}
	peerB := &nat.Peer{ID: "B", Addr: bConn.LocalAddr().(*net.UDPAddr)}

	pA := nat.NewPuncher(aMux, peerA.ID, 50*time.Millisecond)
	pB := nat.NewPuncher(bMux, peerB.ID, 50*time.Millisecond)

	chA := make(chan *nat.PunchResult, 1)
	chB := make(chan *nat.PunchResult, 1)

	go func() {
		res, err := pA.Punch(ctx, peerB)
		assert.NoError(t, err)
		chA <- res
	}()
	go func() {
		res, err := pB.Punch(ctx, peerA)
		assert.NoError(t, err)
		chB <- res
	}()

	assert.NotNil(t, <-chA)
	assert.NotNil(t, <-chB)
}
