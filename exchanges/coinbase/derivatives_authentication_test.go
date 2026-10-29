package coinbase

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
)

func TestDerivativesLogoutParams(t *testing.T) {
	t.Parallel()
	for invalidateToken, exp := range map[bool]string{false: `{"invalidate_token":false}`, true: `{"invalidate_token":true}`} {
		encoded, err := json.Marshal(&DerivativesLogoutWireRequest{InvalidateToken: invalidateToken})
		require.NoError(t, err, "Marshal must not error")
		assert.JSONEqf(t, exp, string(encoded), "derivativesLogoutParams should always send invalidate_token, which the gateway defaults to true, for %t", invalidateToken)
	}
}
