package nat_test

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/aethiopicuschan/natto/nat"
	"github.com/stretchr/testify/assert"
)

func TestSessionSendRecv(t *testing.T) {
	t.Parallel()

	aConn, _ := net.ListenUDP("udp", &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 0})
	bConn, _ := net.ListenUDP("udp", &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 0})
	defer aConn.Close()
	defer bConn.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	aMux := nat.NewMux(aConn)
	bMux := nat.NewMux(bConn)
	aMux.Start(ctx)
	bMux.Start(ctx)

	sA := nat.NewSession(aMux, bConn.LocalAddr().(*net.UDPAddr), 4)
	sB := nat.NewSession(bMux, aConn.LocalAddr().(*net.UDPAddr), 4)

	msg := []byte("ping")
	assert.NoError(t, sA.Send(msg))

	recvCtx, cancelRecv := context.WithTimeout(ctx, time.Second)
	defer cancelRecv()

	got, _, err := sB.Recv(recvCtx)
	assert.NoError(t, err)
	assert.Equal(t, msg, got)
}

func TestSessionClose(t *testing.T) {
	t.Parallel()

	conn, _ := net.ListenUDP("udp", &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 0})
	defer conn.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	mux := nat.NewMux(conn)
	mux.Start(ctx)

	s := nat.NewSession(mux, conn.LocalAddr().(*net.UDPAddr), 1)
	s.Close()

	err := s.Send([]byte("x"))
	assert.Error(t, err)
}
