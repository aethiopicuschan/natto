package turn_test

import (
	"testing"

	"github.com/aethiopicuschan/natto/turn"
	"github.com/stretchr/testify/assert"
)

func TestErrorsExist(t *testing.T) {
	t.Parallel()

	assert.Error(t, turn.ErrUnauthorized)
	assert.Error(t, turn.ErrTimeout)
	assert.Error(t, turn.ErrBadMessage)
	assert.Error(t, turn.ErrNoAllocation)
}
