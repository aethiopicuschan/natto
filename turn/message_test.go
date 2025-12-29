package turn_test

import (
	"testing"

	"github.com/aethiopicuschan/natto/turn"
	"github.com/stretchr/testify/assert"
)

func TestMessageEncodeParse(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		attr turn.Attr
	}{
		{"username attr", turn.Attr{Type: 0x0006, Value: []byte("user")}},
		{"realm attr", turn.Attr{Type: 0x0014, Value: []byte("example")}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			m := turn.NewMessage(0x0001)
			m.Attrs = append(m.Attrs, tt.attr)

			raw := m.Encode()
			parsed, err := turn.ParseMessage(raw)

			assert.NoError(t, err)
			assert.Equal(t, m.Type, parsed.Type)
			assert.Equal(t, m.TransactionID, parsed.TransactionID)

			a, ok := parsed.FindAttr(tt.attr.Type)
			assert.True(t, ok)
			assert.Equal(t, tt.attr.Value, a.Value)
		})
	}
}
