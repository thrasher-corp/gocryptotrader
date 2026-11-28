package coinbase

import (
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
)

const (
	// mockSupportingServerTime is the gateway time the mock responses return, in Unix milliseconds
	mockSupportingServerTime = 1792372193943
	// testSupportingClientName and testSupportingClientVersion introduce the tests to the gateway
	testSupportingClientName    = "GoCryptoTrader"
	testSupportingClientVersion = "test"
	// mockSupportingVersion is the API version the mock responses report
	mockSupportingVersion = "1.2.26"
)

func TestGlobalDerivativesGetTime(t *testing.T) {
	t.Parallel()
	resp, err := e.GlobalDerivatives().GetTime(t.Context())
	require.NoError(t, err, "GetTime must not error")
	if mockTests {
		assert.Equal(t, time.UnixMilli(mockSupportingServerTime), resp, "GetTime should decode the gateway's time")
	} else {
		assert.WithinDuration(t, time.Now(), resp, time.Minute, "GetTime should return the current time")
	}

	_, err = newResponseServerExchange(t, `{"jsonrpc":"2.0","result":null}`).GlobalDerivatives().GetTime(t.Context())
	assert.ErrorIs(t, err, common.ErrNoResponse, "GetTime should error for a null result")
	_, err = newUnavailableServerExchange(t).GlobalDerivatives().GetTime(t.Context())
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetTime should return the request failure")
}

func TestGlobalDerivativesGetStatus(t *testing.T) {
	t.Parallel()
	resp, err := e.GlobalDerivatives().GetStatus(t.Context())
	require.NoError(t, err, "GetStatus must not error")
	if mockTests {
		assert.Equal(t, &DerivativesStatusResponse{Locked: DerivativesLockedFalse}, resp, "GetStatus should decode the lock status")
	} else {
		assert.Contains(t, []string{DerivativesLockedTrue, DerivativesLockedPartial, DerivativesLockedFalse}, resp.Locked, "GetStatus should return a documented lock status")
	}

	// Nothing is locked today, so a partial lock is served as the docs describe it
	resp, err = newResponseServerExchange(t, `{"jsonrpc":"2.0","result":{"locked":"partial","locked_indices":["btc_eurr","eth_eurr"]}}`).GlobalDerivatives().GetStatus(t.Context())
	require.NoError(t, err, "GetStatus must not error for a partial lock")
	assert.Equal(t, &DerivativesStatusResponse{Locked: DerivativesLockedPartial, LockedIndices: []string{"btc_eurr", "eth_eurr"}}, resp, "GetStatus should decode the locked indices of a partial lock")
	_, err = newResponseServerExchange(t, `{"jsonrpc":"2.0","result":null}`).GlobalDerivatives().GetStatus(t.Context())
	assert.ErrorIs(t, err, common.ErrNoResponse, "GetStatus should error for a null result")
	_, err = newUnavailableServerExchange(t).GlobalDerivatives().GetStatus(t.Context())
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetStatus should return the request failure")
}

func TestGlobalDerivativesTestConnection(t *testing.T) {
	t.Parallel()
	_, err := e.GlobalDerivatives().TestConnection(t.Context(), "bogus")
	require.ErrorIs(t, err, errDerivativesInvalidExpectedResult, "TestConnection must refuse an undocumented expected result")

	resp, err := e.GlobalDerivatives().TestConnection(t.Context(), "")
	require.NoError(t, err, "TestConnection must not error")
	if mockTests {
		assert.Equal(t, &DerivativesTestConnectionResponse{Version: mockSupportingVersion}, resp, "TestConnection should decode the API version")
	} else {
		assert.NotEmpty(t, resp.Version, "TestConnection should return the API version")
	}

	_, err = e.GlobalDerivatives().TestConnection(t.Context(), DerivativesTestExpectedResultException)
	derivativesErr, ok := errors.AsType[*DerivativesError](err)
	require.True(t, ok, "TestConnection must return the error the gateway is asked for")
	assert.Equal(t, &DerivativesError{Code: 11094, Message: "internal_server_error"}, derivativesErr, "TestConnection should decode the requested error")

	ex := newTestServerExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		req, ok := readJSONRPC(t, r)
		if ok {
			assert.Empty(t, req.Params, "TestConnection should leave the params out without an expected result")
		}
		writeJSONRPC(t, w, `{"jsonrpc":"2.0","result":{"version":"1.2.26"}}`)
	}))
	_, err = ex.GlobalDerivatives().TestConnection(t.Context(), "")
	require.NoError(t, err, "TestConnection must not error")
	_, err = newResponseServerExchange(t, `{"jsonrpc":"2.0","result":null}`).GlobalDerivatives().TestConnection(t.Context(), "")
	assert.ErrorIs(t, err, common.ErrNoResponse, "TestConnection should error for a null result")
	_, err = newUnavailableServerExchange(t).GlobalDerivatives().TestConnection(t.Context(), "")
	assert.ErrorIs(t, err, request.ErrBadStatus, "TestConnection should return the request failure")
}
