package stun_test

import (
	"testing"

	"github.com/aethiopicuschan/natto/stun"
	"github.com/stretchr/testify/assert"
)

func TestNewTransactionID(t *testing.T) {
	t.Parallel()

	id1, err1 := stun.NewTransactionID()
	id2, err2 := stun.NewTransactionID()

	assert.NoError(t, err1)
	assert.NoError(t, err2)

	// Must not be all-zero
	assert.NotEqual(t, stun.TransactionID{}, id1)

	// Extremely high probability of being different
	assert.NotEqual(t, id1, id2)
}

func TestStunType_ParseType_RoundTrip(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		method uint16
		class  int
	}{
		{
			name:   "binding request",
			method: stun.MethodBinding,
			class:  stun.ClassRequest,
		},
		{
			name:   "binding success response",
			method: stun.MethodBinding,
			class:  stun.ClassSuccessResponse,
		},
		{
			name:   "binding error response",
			method: stun.MethodBinding,
			class:  stun.ClassErrorResponse,
		},
		{
			name:   "indication",
			method: 0x0002,
			class:  stun.ClassIndication,
		},
		{
			name:   "custom method",
			method: 0x03EF,
			class:  stun.ClassRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			typ := stun.TestStunType(tt.method, tt.class)
			method, class := stun.TestParseType(typ)

			assert.Equal(t, tt.method&0x0FFF, method)
			assert.Equal(t, tt.class&0x03, class)
		})
	}
}

func TestStunType_BitConstraints(t *testing.T) {
	t.Parallel()

	typ := stun.TestStunType(stun.MethodBinding, stun.ClassRequest)

	// RFC 5389: first two bits must be zero
	assert.Equal(t, uint16(0), typ&0xC000)
}

func TestEndianHelpers(t *testing.T) {
	t.Parallel()

	buf16 := make([]byte, 2)
	buf32 := make([]byte, 4)

	stun.TestPutU16(buf16, 0xABCD)
	stun.TestPutU32(buf32, 0xDEADBEEF)

	assert.Equal(t, uint16(0xABCD), stun.TestReadU16(buf16))
	assert.Equal(t, uint32(0xDEADBEEF), stun.TestReadU32(buf32))
}

func TestConstants_Sanity(t *testing.T) {
	t.Parallel()

	// Magic cookie is fixed by RFC 5389
	assert.Equal(t, uint32(0x2112A442), stun.MagicCookie)

	// Method / class sanity
	assert.Equal(t, uint16(0x0001), stun.MethodBinding)
	assert.Equal(t, 0x00, stun.ClassRequest)
	assert.Equal(t, 0x02, stun.ClassSuccessResponse)

	// Attribute types sanity
	assert.Equal(t, uint16(0x0001), stun.AttrMappedAddress)
	assert.Equal(t, uint16(0x0020), stun.AttrXORMappedAddress)
}
