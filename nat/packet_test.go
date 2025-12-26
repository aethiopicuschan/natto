package nat_test

import (
	"testing"

	"github.com/aethiopicuschan/natto/nat"
	"github.com/stretchr/testify/assert"
)

func TestPacketRoundTrip(t *testing.T) {
	t.Parallel()

	payload := []byte("hello")
	wire, err := nat.EncodePacket(nat.PacketControl, payload)
	assert.NoError(t, err)

	pkt, err := nat.DecodePacket(wire)
	assert.NoError(t, err)

	assert.Equal(t, nat.PacketControl, pkt.Kind)
	assert.Equal(t, payload, pkt.Payload)
}

func TestPacketRejectsForeignData(t *testing.T) {
	t.Parallel()

	_, err := nat.DecodePacket([]byte("foreign payload"))
	assert.ErrorIs(t, err, nat.ErrNotOurPacket)
}
