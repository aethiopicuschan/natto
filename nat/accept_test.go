package nat_test

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/aethiopicuschan/natto/nat"
	"github.com/stretchr/testify/assert"
)

func TestAcceptorAccept(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name              string
		queue             int
		keepaliveInterval time.Duration
	}{
		{
			name:  "default",
			queue: 0,
		},
		{
			name:              "with_keepalive",
			queue:             8,
			keepaliveInterval: 50 * time.Millisecond,
		},
		{
			name:  "small_queue",
			queue: 1,
		},
	}

	for i := range tests {
		t.Run(tests[i].name, func(t *testing.T) {
			t.Parallel()

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			// --- UDP sockets ---
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

			// --- Mux ---
			aMux := nat.NewMux(aConn)
			bMux := nat.NewMux(bConn)
			aMux.Start(ctx)
			bMux.Start(ctx)

			peerA := &nat.Peer{
				ID:   "peer-A",
				Addr: aConn.LocalAddr().(*net.UDPAddr),
			}
			peerB := &nat.Peer{
				ID:   "peer-B",
				Addr: bConn.LocalAddr().(*net.UDPAddr),
			}

			// --- Acceptor ---
			acceptor := nat.NewAcceptor(
				bMux,
				peerB.ID,
				nat.AcceptOptions{
					Queue:             tests[i].queue,
					KeepaliveInterval: tests[i].keepaliveInterval,
				},
			)

			type acceptResult struct {
				sess *nat.Session
				res  *nat.PunchResult
				err  error
			}

			accCh := make(chan acceptResult, 1)
			go func() {
				sess, res, err := acceptor.Accept(ctx)
				accCh <- acceptResult{sess, res, err}
			}()

			// --- Dial ---
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
						Interval:          30 * time.Millisecond,
						Queue:             tests[i].queue,
						KeepaliveInterval: tests[i].keepaliveInterval,
					},
				)
				close(done)
			}()

			select {
			case <-done:
			case <-ctx.Done():
				assert.FailNow(t, "dial timeout")
			}

			assert.NoError(t, dialErr)
			assert.NotNil(t, dialSess)
			assert.NotNil(t, dialRes)

			var acc acceptResult
			select {
			case acc = <-accCh:
			case <-ctx.Done():
				assert.FailNow(t, "accept timeout")
			}

			assert.NoError(t, acc.err)
			assert.NotNil(t, acc.sess)
			assert.NotNil(t, acc.res)

			// --- Verify basic data path ---
			msg := []byte("hello accept")

			assert.NoError(t, dialSess.Send(msg))

			recvCtx, cancelRecv := context.WithTimeout(ctx, time.Second)
			defer cancelRecv()

			got, _, err := acc.sess.Recv(recvCtx)
			assert.NoError(t, err)
			assert.Equal(t, msg, got)
		})
	}
}
