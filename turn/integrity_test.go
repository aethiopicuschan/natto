package turn_test

import (
	"encoding/hex"
	"testing"

	"github.com/aethiopicuschan/natto/turn"
	"github.com/stretchr/testify/assert"
)

func TestLongTermKey(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		user     string
		realm    string
		pass     string
		expected string
	}{
		{
			name:     "rfc example (user:example.org:pass)",
			user:     "user",
			realm:    "example.org",
			pass:     "pass",
			expected: "abca35356f4b00fbc33e2d8c2c43b9d6",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			key := turn.LongTermKey(tt.user, tt.realm, tt.pass)
			assert.Equal(t, tt.expected, hex.EncodeToString(key))
		})
	}
}
