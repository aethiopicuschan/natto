package stun_test

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/aethiopicuschan/natto/stun"
	"github.com/stretchr/testify/assert"
)

func startTestSTUNServer(t *testing.T, software string) (*stun.Server, string) {
	t.Helper()

	srv, err := stun.ListenUDP("127.0.0.1:0")
	assert.NoError(t, err)

	srv.Software = software

	go func() {
		_ = srv.Serve()
	}()

	return srv, srv.Conn.LocalAddr().String()
}

func TestServer_BindingRequest_IPv4(t *testing.T) {
	t.Parallel()

	srv, addr := startTestSTUNServer(t, "test-stun")
	defer srv.Close()

	client := stun.NewClient()
	client.Timeout = 500 * time.Millisecond

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	mapped, err := client.BindingRequest(ctx, addr)

	assert.NoError(t, err)
	assert.NotNil(t, mapped.IP)
	assert.True(t, mapped.IP.To4() != nil)
	assert.NotZero(t, mapped.Port)
}

func TestServer_IncludesSoftwareAttribute(t *testing.T) {
	t.Parallel()

	const software = "stun-test-server/1.0"

	srv, addr := startTestSTUNServer(t, software)
	defer srv.Close()

	raddr, err := net.ResolveUDPAddr("udp", addr)
	assert.NoError(t, err)

	conn, err := net.DialUDP("udp", nil, raddr)
	assert.NoError(t, err)
	defer conn.Close()

	tid, err := stun.NewTransactionID()
	assert.NoError(t, err)

	req := stun.NewBindingRequest(tid)
	_, err = conn.Write(req.Marshal())
	assert.NoError(t, err)

	buf := make([]byte, 1500)
	_ = conn.SetReadDeadline(time.Now().Add(time.Second))

	n, err := conn.Read(buf)
	assert.NoError(t, err)

	resp, err := stun.Parse(buf[:n])
	assert.NoError(t, err)

	attr, ok := resp.GetAttribute(stun.AttrSoftware)
	assert.True(t, ok)
	assert.Equal(t, software, string(attr.Value))
}

func TestServer_IgnoresNonBinding(t *testing.T) {
	t.Parallel()

	srv, addr := startTestSTUNServer(t, "")
	defer srv.Close()

	raddr, err := net.ResolveUDPAddr("udp", addr)
	assert.NoError(t, err)

	conn, err := net.DialUDP("udp", nil, raddr)
	assert.NoError(t, err)
	defer conn.Close()

	tid, err := stun.NewTransactionID()
	assert.NoError(t, err)

	// Send Indication instead of Request
	msg := &stun.Message{
		Method:        stun.MethodBinding,
		Class:         stun.ClassIndication,
		Cookie:        stun.MagicCookie,
		TransactionID: tid,
	}

	_, err = conn.Write(msg.Marshal())
	assert.NoError(t, err)

	buf := make([]byte, 1500)
	_ = conn.SetReadDeadline(time.Now().Add(200 * time.Millisecond))
	_, err = conn.Read(buf)

	// No response expected
	assert.Error(t, err)
}

func TestServer_CloseStopsServe(t *testing.T) {
	t.Parallel()

	srv, _ := startTestSTUNServer(t, "")
	err := srv.Close()

	assert.NoError(t, err)
}

func TestBuildXORMappedAddressAttr_IPv4(t *testing.T) {
	t.Parallel()

	ip := net.IPv4(192, 0, 2, 33)
	port := 54321
	tid := stun.TransactionID{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12}

	attr := stun.TestBuildXORMappedAddressAttr(
		&net.UDPAddr{IP: ip, Port: port},
		tid,
	)

	decoded, err := stun.DecodeXORMappedAddress(attr, tid)
	assert.NoError(t, err)

	assert.True(t, ip.Equal(decoded.IP))
	assert.Equal(t, port, decoded.Port)
}

func TestBuildSoftwareAttr(t *testing.T) {
	t.Parallel()

	attr := stun.TestBuildSoftwareAttr("example")

	assert.Equal(t, stun.AttrSoftware, attr.Type)
	assert.Equal(t, "example", string(attr.Value))
}
