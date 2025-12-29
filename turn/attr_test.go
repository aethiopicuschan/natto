package turn_test

import (
	"testing"

	"github.com/aethiopicuschan/natto/turn"
	"github.com/stretchr/testify/assert"
)

func TestAttributePadding(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		value []byte
	}{
		{"len=3", []byte{1, 2, 3}},
		{"len=4", []byte{1, 2, 3, 4}},
		{"len=5", []byte{1, 2, 3, 4, 5}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			m := turn.NewMessage(0x0001)
			m.Attrs = append(m.Attrs, turn.Attr{Type: 0x9999, Value: tt.value})
			raw := m.Encode()

			assert.Equal(t, 0, len(raw)%4)
		})
	}
}
