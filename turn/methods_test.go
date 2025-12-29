package turn_test

import (
	"testing"

	"github.com/aethiopicuschan/natto/turn"
	"github.com/stretchr/testify/assert"
)

func TestStunType(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		method uint16
		class  uint16
		want   uint16
	}{
		{"allocate request", 0x0003, 0x0000, 0x0003},
		{"allocate success", 0x0003, 0x0100, 0x0103},
		{"refresh error", 0x0004, 0x0110, 0x0114},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, turn.StunType(tt.method, tt.class))
		})
	}
}
