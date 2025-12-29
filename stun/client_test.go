package stun_test

import (
	"context"
	"encoding/binary"
	"net"
	"testing"
	"time"

	"github.com/aethiopicuschan/natto/stun"
	"github.com/stretchr/testify/assert"
)

func startMockSTUNServer(
	t *testing.T,
	handler func(req *stun.Message, raddr *net.UDPAddr) *stun.Message,
) (addr string, closeFn func()) {
	t.Helper()

	conn, err := net.ListenUDP("udp", &net.UDPAddr{
		IP:   net.IPv4(127, 0, 0, 1),
		Port: 0,
	})
	assert.NoError(t, err)

	done := make(chan struct{})

	go func() {
		defer close(done)

		buf := make([]byte, 1500)
		n, raddr, err := conn.ReadFromUDP(buf)
		if err != nil {
			return
		}

		req, err := stun.Parse(buf[:n])
		if err != nil {
			return
		}

		resp := handler(req, raddr)
		if resp == nil {
			return
		}

		_, _ = conn.WriteToUDP(resp.Marshal(), raddr)
	}()

	return conn.LocalAddr().String(), func() {
		_ = conn.Close()
		<-done
	}
}

func makeSuccessResponse(
	req *stun.Message,
	ip net.IP,
	port int,
) *stun.Message {
	v := make([]byte, 8)
	v[1] = 0x01
	binary.BigEndian.PutUint16(v[2:4], uint16(port))
	copy(v[4:8], ip.To4())

	return &stun.Message{
		Method:        stun.MethodBinding,
		Class:         stun.ClassSuccessResponse,
		Cookie:        stun.MagicCookie,
		TransactionID: req.TransactionID,
		Attributes: []stun.Attribute{
			{
				Type:  stun.AttrMappedAddress,
				Value: v,
			},
		},
	}
}

func TestClient_BindingRequest_Success(t *testing.T) {
	t.Parallel()

	publicIP := net.IPv4(203, 0, 113, 9)
	publicPort := 54321

	serverAddr, closeFn := startMockSTUNServer(t, func(req *stun.Message, _ *net.UDPAddr) *stun.Message {
		assert.Equal(t, stun.MethodBinding, req.Method)
		assert.Equal(t, stun.ClassRequest, req.Class)
		return makeSuccessResponse(req, publicIP, publicPort)
	})
	defer closeFn()

	client := stun.NewClient()
	client.Timeout = 500 * time.Millisecond

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	got, err := client.BindingRequest(ctx, serverAddr)

	assert.NoError(t, err)
	assert.Equal(t, publicPort, got.Port)
	assert.True(t, publicIP.Equal(got.IP))
}

func TestClient_BindingRequest_TransactionIDMismatch(t *testing.T) {
	t.Parallel()

	serverAddr, closeFn := startMockSTUNServer(t, func(req *stun.Message, _ *net.UDPAddr) *stun.Message {
		resp := makeSuccessResponse(req, net.IPv4(1, 2, 3, 4), 1111)
		resp.TransactionID[0] ^= 0xFF // break TID
		return resp
	})
	defer closeFn()

	client := stun.NewClient()
	client.Timeout = 200 * time.Millisecond
	client.Retries = 0

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	_, err := client.BindingRequest(ctx, serverAddr)

	assert.ErrorIs(t, err, stun.ErrTimeout)
}

func TestClient_BindingRequest_ErrorResponse(t *testing.T) {
	t.Parallel()

	serverAddr, closeFn := startMockSTUNServer(t, func(req *stun.Message, _ *net.UDPAddr) *stun.Message {
		return &stun.Message{
			Method:        stun.MethodBinding,
			Class:         stun.ClassErrorResponse,
			Cookie:        stun.MagicCookie,
			TransactionID: req.TransactionID,
		}
	})
	defer closeFn()

	client := stun.NewClient()
	client.Timeout = 300 * time.Millisecond

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	_, err := client.BindingRequest(ctx, serverAddr)

	assert.Error(t, err)
}

func TestClient_BindingRequest_ContextCanceled(t *testing.T) {
	t.Parallel()

	// Server never replies.
	serverAddr, closeFn := startMockSTUNServer(t, func(_ *stun.Message, _ *net.UDPAddr) *stun.Message {
		return nil
	})
	defer closeFn()

	client := stun.NewClient()
	client.Timeout = time.Second

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // immediately cancel

	_, err := client.BindingRequest(ctx, serverAddr)

	assert.ErrorIs(t, err, context.Canceled)
}

func TestClient_BindingRequest_Timeout(t *testing.T) {
	t.Parallel()

	// Server never replies.
	serverAddr, closeFn := startMockSTUNServer(t, func(_ *stun.Message, _ *net.UDPAddr) *stun.Message {
		return nil
	})
	defer closeFn()

	client := stun.NewClient()
	client.Timeout = 200 * time.Millisecond
	client.Retries = 1
	client.RTO = 50 * time.Millisecond

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	_, err := client.BindingRequest(ctx, serverAddr)

	assert.ErrorIs(t, err, stun.ErrTimeout)
}
