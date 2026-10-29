package coinbase

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
)

func TestGlobalDerivativesGetCancelOnDisconnect(t *testing.T) {
	t.Parallel()
	skipLiveWithoutCredentials(t)
	resp, err := e.GlobalDerivatives().GetCancelOnDisconnect(t.Context())
	require.NoError(t, err, "GetCancelOnDisconnect must not error")
	if mockTests {
		exp := &DerivativesCancelOnDisconnectResponse{Scope: "account"}
		assert.Equal(t, exp, resp, "GetCancelOnDisconnect should decode every field")
	} else {
		assert.Equal(t, DerivativesCancelOnDisconnectScopeAccount, resp.Scope, "GetCancelOnDisconnect should return the account scope")
	}

	_, err = newResponseServerExchange(t, `{"jsonrpc":"2.0","result":null}`).GlobalDerivatives().GetCancelOnDisconnect(t.Context())
	assert.ErrorIs(t, err, common.ErrNoResponse, "GetCancelOnDisconnect should error for a null result")
	_, err = newUnavailableServerExchange(t).GlobalDerivatives().GetCancelOnDisconnect(t.Context())
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetCancelOnDisconnect should return the request failure")
}

func TestValidateDerivativesCancelOnDisconnectScope(t *testing.T) {
	t.Parallel()
	for _, scope := range []string{"", DerivativesCancelOnDisconnectScopeConnection, DerivativesCancelOnDisconnectScopeAccount} {
		assert.NoErrorf(t, validateDerivativesCancelOnDisconnectScope(scope), "validateDerivativesCancelOnDisconnectScope should accept scope %q", scope)
	}
	for _, scope := range []string{"Account", "subaccount"} {
		assert.ErrorIsf(t, validateDerivativesCancelOnDisconnectScope(scope), errDerivativesInvalidCancelOnDisconnectScope, "validateDerivativesCancelOnDisconnectScope should refuse scope %q", scope)
	}
}
