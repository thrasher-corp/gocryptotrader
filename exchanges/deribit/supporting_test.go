package deribit

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
)

const (
	// mockServerTime is the server time the mock responses return
	mockServerTime = 1792107016663
	// testClientName and testClientVersion introduce the tests to Deribit
	testClientName    = "GoCryptoTrader"
	testClientVersion = "test"
)

func TestGetTime(t *testing.T) {
	t.Parallel()
	result, err := e.GetTime(t.Context())
	require.NoError(t, err, "GetTime must not error")
	if mockTests {
		assert.Equal(t, time.UnixMilli(mockServerTime), result, "GetTime should decode the server time")
	} else {
		assert.WithinDuration(t, time.Now(), result, time.Minute, "GetTime should return the current time")
	}

	_, err = newResponseServerExchange(t, `{"jsonrpc":"2.0","result":null}`).GetTime(t.Context())
	assert.ErrorIs(t, err, common.ErrNoResponse, "GetTime should reject a null result")
	_, err = newUnavailableServerExchange(t).GetTime(t.Context())
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetTime should return the request failure")
}

func TestGetStatus(t *testing.T) {
	t.Parallel()
	result, err := e.GetStatus(t.Context())
	require.NoError(t, err, "GetStatus must not error")
	if mockTests {
		assert.Equal(t, &StatusResponse{Locked: "false"}, result, "GetStatus should decode the lock status")
	} else {
		assert.Contains(t, []string{"true", "partial", "false"}, result.Locked, "GetStatus should return a documented lock status")
	}

	result, err = newResponseServerExchange(t, `{"jsonrpc":"2.0","result":{"locked":"partial","locked_indices":["btc_eurr","eth_eurr","ip_usdc"]},"testnet":true}`).GetStatus(t.Context())
	require.NoError(t, err, "GetStatus must not error for a partial lock")
	assert.Equal(t, &StatusResponse{Locked: "partial", LockedIndices: []string{"btc_eurr", "eth_eurr", "ip_usdc"}}, result, "GetStatus should decode the locked indices of a partial lock, as testnet returned them")

	_, err = newResponseServerExchange(t, `{"jsonrpc":"2.0","result":null}`).GetStatus(t.Context())
	assert.ErrorIs(t, err, common.ErrNoResponse, "GetStatus should reject a null result")
	_, err = newUnavailableServerExchange(t).GetStatus(t.Context())
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetStatus should return the request failure")
}

func TestTestConnection(t *testing.T) {
	t.Parallel()
	_, err := e.TestConnection(t.Context(), "bogus")
	require.ErrorIs(t, err, errInvalidExpectedResult, "TestConnection must reject an undocumented expected result")

	result, err := e.TestConnection(t.Context(), "")
	require.NoError(t, err, "TestConnection must not error")
	if mockTests {
		assert.Equal(t, &TestConnectionResponse{Version: "1.2.26"}, result, "TestConnection should decode the API version")
	} else {
		assert.NotEmpty(t, result.Version, "TestConnection should return the API version")
	}

	_, err = e.TestConnection(t.Context(), testExpectedResultException)
	var apiErr *APIError
	require.ErrorAs(t, err, &apiErr, "TestConnection must return the error Deribit is asked for")
	assert.Equal(t, &APIError{Code: 11094, Message: "internal_server_error"}, apiErr, "TestConnection should decode the requested error")

	_, err = newResponseServerExchange(t, `{"jsonrpc":"2.0","result":null}`).TestConnection(t.Context(), "")
	assert.ErrorIs(t, err, common.ErrNoResponse, "TestConnection should reject a null result")
	_, err = newUnavailableServerExchange(t).TestConnection(t.Context(), "")
	assert.ErrorIs(t, err, request.ErrBadStatus, "TestConnection should return the request failure")
}
