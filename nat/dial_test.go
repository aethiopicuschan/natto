package nat_test

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/aethiopicuschan/natto/nat"
	"github.com/stretchr/testify/assert"
)

func newLocalUDP(t *testing.T) *net.UDPConn {
	t.Helper()

	conn, err := net.ListenUDP("udp", &net.UDPAddr{
		IP:   net.IPv4(127, 0, 0, 1),
		Port: 0,
	})
	if !assert.NoError(t, err) {
		t.FailNow()
	}
	return conn
}

func TestDialAndAccept(t *testing.T) {
	tests := []struct {
		name              string
		interval          time.Duration
		queue             int
		keepaliveInterval time.Duration
	}{
		{"default", 50 * time.Millisecond, 16, 0},
		{"with_keepalive", 50 * time.Millisecond, 16, 100 * time.Millisecond},
		{"small_queue", 30 * time.Millisecond, 1, 0},
	}

	for i := range tests {
		tt := tests[i]

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			// --- Dialer (A) ---
			aConn := newLocalUDP(t)
			defer aConn.Close()

			// --- Acceptor (B) ---
			bConn := newLocalUDP(t)
			defer bConn.Close()

			aMux := nat.NewMux(aConn)
			bMux := nat.NewMux(bConn)

			aMux.Start(ctx)
			bMux.Start(ctx)

			peerA := &nat.Peer{
				ID:   "peer-a",
				Addr: aConn.LocalAddr().(*net.UDPAddr),
			}
			peerB := &nat.Peer{
				ID:   "peer-b",
				Addr: bConn.LocalAddr().(*net.UDPAddr),
			}

			// --- Acceptor ---
			acceptor := nat.NewAcceptor(bMux, peerB.ID, nat.AcceptOptions{
				Queue:             tt.queue,
				KeepaliveInterval: tt.keepaliveInterval,
			})

			type acceptResult struct {
				sess *nat.Session
				res  *nat.PunchResult
				err  error
			}

			acceptCh := make(chan acceptResult, 1)
			go func() {
				sess, res, err := acceptor.Accept(ctx)
				acceptCh <- acceptResult{sess, res, err}
			}()

			// --- Dialer ---
			var (
				dialSess *nat.Session
				dialRes  *nat.PunchResult
				dialErr  error
			)

			done := make(chan struct{})
			go func() {
				dialSess, dialRes, dialErr = nat.Dial(
					ctx,
					aMux,
					peerA.ID,
					peerB,
					nat.DialOptions{
						Interval:          tt.interval,
						Queue:             tt.queue,
						KeepaliveInterval: tt.keepaliveInterval,
					},
				)
				close(done)
			}()

			select {
			case <-done:
			case <-ctx.Done():
				assert.FailNow(t, "timeout waiting for dial")
			}

			assert.NoError(t, dialErr)
			assert.NotNil(t, dialSess)
			assert.NotNil(t, dialRes)

			var acc acceptResult
			select {
			case acc = <-acceptCh:
			case <-ctx.Done():
				assert.FailNow(t, "timeout waiting for accept")
			}

			assert.NoError(t, acc.err)
			assert.NotNil(t, acc.sess)
			assert.NotNil(t, acc.res)

			// --- Verify bidirectional communication ---
			msgA := []byte("hello from dialer")
			msgB := []byte("hello from acceptor")

			assert.NoError(t, dialSess.Send(msgA))
			assert.NoError(t, acc.sess.Send(msgB))

			recvCtx, cancelRecv := context.WithTimeout(ctx, 2*time.Second)
			defer cancelRecv()

			gotB, _, err := acc.sess.Recv(recvCtx)
			assert.NoError(t, err)
			assert.Equal(t, msgA, gotB)

			gotA, _, err := dialSess.Recv(recvCtx)
			assert.NoError(t, err)
			assert.Equal(t, msgB, gotA)
		})
	}
}
