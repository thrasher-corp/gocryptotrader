package deribit

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
)

func TestGetCancelOnDisconnect(t *testing.T) {
	t.Parallel()
	skipLiveWithoutCredentials(t)
	result, err := e.GetCancelOnDisconnect(t.Context())
	require.NoError(t, err, "GetCancelOnDisconnect must not error")
	if mockTests {
		assert.Equal(t, &CancelOnDisconnectResponse{Scope: CancelOnDisconnectScopeAccount}, result, "GetCancelOnDisconnect should decode the account's setting")
	} else {
		assert.Equal(t, CancelOnDisconnectScopeAccount, result.Scope, "GetCancelOnDisconnect should return the account scope")
	}

	_, err = newResponseServerExchange(t, `{"jsonrpc":"2.0","result":null}`).GetCancelOnDisconnect(t.Context())
	assert.ErrorIs(t, err, common.ErrNoResponse, "GetCancelOnDisconnect should reject a null result")
	_, err = newUnavailableServerExchange(t).GetCancelOnDisconnect(t.Context())
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetCancelOnDisconnect should return the request failure")
}

func TestValidateCancelOnDisconnectScope(t *testing.T) {
	t.Parallel()
	for _, scope := range []string{"", CancelOnDisconnectScopeConnection, CancelOnDisconnectScopeAccount} {
		assert.NoErrorf(t, validateCancelOnDisconnectScope(scope), "validateCancelOnDisconnectScope should accept scope %q", scope)
	}
	assert.ErrorIs(t, validateCancelOnDisconnectScope("Account"), errInvalidCancelOnDisconnectScope, "validateCancelOnDisconnectScope should reject a scope Deribit does not document")
}
