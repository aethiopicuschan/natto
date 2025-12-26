package nat_test

import (
	"testing"

	"github.com/aethiopicuschan/natto/nat"
	"github.com/stretchr/testify/assert"
)

func TestMessageEncodeDecode(t *testing.T) {
	t.Parallel()

	in := &nat.Message{
		Type:      nat.MessageHello,
		PeerID:    "peer-a",
		ToPeerID:  "peer-b",
		Timestamp: 123456,
	}

	b, err := nat.EncodeMessage(in)
	assert.NoError(t, err)

	out, err := nat.DecodeMessage(b)
	assert.NoError(t, err)

	assert.Equal(t, in.Type, out.Type)
	assert.Equal(t, in.PeerID, out.PeerID)
	assert.Equal(t, in.ToPeerID, out.ToPeerID)
	assert.Equal(t, in.Timestamp, out.Timestamp)
}
