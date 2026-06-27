package apexpro

import (
	"encoding/hex"
	"math/big"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAppendSignatures(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		r        *big.Int
		s        *big.Int
		expected string
	}{
		{
			name:     "both values already 32 bytes",
			r:        new(big.Int).SetBytes(mustDecodeHex(t, "1111111111111111111111111111111111111111111111111111111111111111")),
			s:        new(big.Int).SetBytes(mustDecodeHex(t, "2222222222222222222222222222222222222222222222222222222222222222")),
			expected: "11111111111111111111111111111111111111111111111111111111111111112222222222222222222222222222222222222222222222222222222222222222",
		},
		{
			name:     "small r and s are left-padded to 32 bytes",
			r:        big.NewInt(1),
			s:        big.NewInt(2),
			expected: strings.Repeat("00", 31) + "01" + strings.Repeat("00", 31) + "02",
		},
		{
			name:     "zero values produce 64 zero bytes",
			r:        big.NewInt(0),
			s:        big.NewInt(0),
			expected: strings.Repeat("00", 64),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := appendSignatures(tt.r, tt.s)
			require.Len(t, result, 128, "result must be 64 bytes hex-encoded (128 chars)")
			assert.Equal(t, tt.expected, result, "signature encoding should match expected")

			decoded, err := hex.DecodeString(result)
			require.NoError(t, err, "result must be valid hex")
			assert.Zero(t, new(big.Int).SetBytes(decoded[:32]).Cmp(tt.r), "first 32 bytes should decode back to r")
			assert.Zero(t, new(big.Int).SetBytes(decoded[32:]).Cmp(tt.s), "last 32 bytes should decode back to s")
		})
	}
}

func mustDecodeHex(t *testing.T, s string) []byte {
	t.Helper()
	b, err := hex.DecodeString(s)
	require.NoError(t, err, "test hex literal must be valid")
	return b
}
