package turn_test

import (
	"net"
	"testing"
	"time"

	"github.com/aethiopicuschan/natto/turn"
	"github.com/stretchr/testify/assert"
)

func TestSendChannelData_InvalidChannel(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		channel uint16
	}{
		{"too small", 0x3FFF},
		{"too large", 0x8000},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Client without Dial is fine here: SendChannelData
			// returns before touching conn.
			c := &turn.Client{}
			err := c.SendChannelData(tt.channel, []byte("data"))
			assert.ErrorIs(t, err, turn.ErrInvalidAddress)
		})
	}
}

func TestSendIndication_NoAllocation(t *testing.T) {
	t.Parallel()

	// Dial is required because SendIndication touches conn.
	server, err := net.ListenUDP("udp", &net.UDPAddr{
		IP:   net.ParseIP("127.0.0.1"),
		Port: 0,
	})
	assert.NoError(t, err)
	defer server.Close()

	c, err := turn.Dial(server.LocalAddr().String(), turn.Credentials{})
	assert.NoError(t, err)
	defer c.Close()

	peer := &net.UDPAddr{
		IP:   net.ParseIP("127.0.0.1"),
		Port: 9999,
	}

	err = c.SendIndication(peer, []byte("hello"))
	assert.ErrorIs(t, err, turn.ErrNoAllocation)
}

func TestReadFrom_ChannelData(t *testing.T) {
	t.Parallel()

	// Fake TURN server
	server, err := net.ListenUDP("udp", &net.UDPAddr{
		IP:   net.ParseIP("127.0.0.1"),
		Port: 0,
	})
	assert.NoError(t, err)
	defer server.Close()

	// Client must be created via Dial
	c, err := turn.Dial(server.LocalAddr().String(), turn.Credentials{})
	assert.NoError(t, err)
	defer c.Close()

	// Send ChannelData from server to client
	channel := uint16(0x4000)
	payload := []byte("hello")

	frame := make([]byte, 4+len(payload))
	frame[0] = byte(channel >> 8)
	frame[1] = byte(channel)
	frame[2] = 0
	frame[3] = byte(len(payload))
	copy(frame[4:], payload)

	_, err = server.WriteToUDP(frame, c.RelayedAddr())
	assert.Error(t, err)
	// RelayedAddr is nil → instead send to client local addr
	_, err = server.WriteToUDP(frame, c.TestLocalAddr().(*net.UDPAddr))
	assert.NoError(t, err)

	buf := make([]byte, 64)
	_ = c.TestSetReadDeadline(time.Now().Add(1 * time.Second))

	peer, n, err := c.ReadFrom(buf)

	assert.NoError(t, err)
	assert.Nil(t, peer) // ChannelData does not carry peer address
	assert.Equal(t, len(payload), n)
}

func TestReadFrom_ShortPacket(t *testing.T) {
	t.Parallel()

	server, err := net.ListenUDP("udp", &net.UDPAddr{
		IP:   net.ParseIP("127.0.0.1"),
		Port: 0,
	})
	assert.NoError(t, err)
	defer server.Close()

	c, err := turn.Dial(server.LocalAddr().String(), turn.Credentials{})
	assert.NoError(t, err)
	defer c.Close()

	// Send invalid short packet (<4 bytes)
	_, err = server.WriteToUDP([]byte{0x01, 0x02}, c.TestLocalAddr().(*net.UDPAddr))
	assert.NoError(t, err)

	buf := make([]byte, 64)
	_, _, err = c.ReadFrom(buf)
	assert.ErrorIs(t, err, turn.ErrBadMessage)
}
