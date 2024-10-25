package math_utils

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestIGCdex(t *testing.T) {
	t.Parallel()
	possibleValues := []struct {
		a int64
		b int64

		respX int64
		respY int64
		respA int64
	}{
		{2, 3, -1, 1, 1},
		{10, 12, -1, 1, 2},
		{100, 2004, -20, 1, 4},
	}
	var respX, respY, respA *big.Int
	for s := range possibleValues {
		respX, respY, respA = IGCdex(big.NewInt(possibleValues[s].a), big.NewInt(possibleValues[s].b))
		require.Equal(t, big.NewInt(possibleValues[s].respX), respX)
		require.Equal(t, big.NewInt(possibleValues[s].respY), respY)
		require.Equal(t, big.NewInt(possibleValues[s].respA), respA)
	}
}
