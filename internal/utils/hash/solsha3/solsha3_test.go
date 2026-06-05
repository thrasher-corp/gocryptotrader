package solsha3

import (
	"encoding/hex"
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSoliditySHA3(t *testing.T) {
	result, err := SoliditySHA3([]string{"string"}, []string{"testing"})
	require.NoError(t, err)
	assert.Equal(t, "5f16f4c7f149ac4f9510d9cf8cf384038ad348b3bcdc01915f95de12df9d1b02", hex.EncodeToString(result))

	result, err = SoliditySHA3(
		[]string{"address", "uint256", "address", "uint256"},
		[]any{"0x1234567890123456789012345678901234567890", "123456", "0xaAaAaAaaAaAaAaaAaAAAAAAAAaaaAaAaAaaAaaAa", "1311768467294899695"},
	)
	require.NoError(t, err)
	assert.Equal(t, "34052387b5efb6132a42b244cff52a85a507ab319c414564d7a89207d4473672", hex.EncodeToString(result))
}

func TestLeftPadBytes(t *testing.T) {
	t.Parallel()
	b := []byte{0x01, 0x02}
	newVal := LeftPadBytes(b, 32)
	assert.Len(t, newVal, 32)
	assert.Equal(t, big.NewInt(0).SetBytes(newVal), big.NewInt(0).SetBytes(b))
	assert.Equal(t, b, newVal[30:], "trailing bytes should match original")
	assert.Equal(t, make([]byte, 30), newVal[:30], "leading bytes should be zero")
	assert.Equal(t, b, LeftPadBytes(b, 1), "should return original slice when length <= len(slice)")
}

func TestRightPadBytes(t *testing.T) {
	t.Parallel()
	b := []byte{0x01, 0x02}
	newVal := rightPadBytes(b, 32)
	require.Len(t, newVal, 32, "result must have length 32")
	assert.Equal(t, b, newVal[:2], "leading bytes should match original")
	assert.Equal(t, make([]byte, 30), newVal[2:], "trailing bytes should be zero")
	assert.Equal(t, b, rightPadBytes(b, 1), "should return original slice when length <= len(slice)")
}
