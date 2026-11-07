package coinbase

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket"
)

func TestWebsocketAPIGetTime(t *testing.T) {
	t.Parallel()
	ex, _ := newDerivativesWebsocketTestExchange(t)
	resp, err := ex.WebsocketAPI().GetTime(t.Context())
	require.NoError(t, err, "WebsocketAPI.GetTime must not error")
	if mockTests {
		assert.Equal(t, time.UnixMilli(mockSupportingServerTime), resp, "WebsocketAPI.GetTime should decode the gateway's time")
	} else {
		assert.WithinDuration(t, time.Now(), resp, time.Minute, "WebsocketAPI.GetTime should return the current time")
	}

	_, err = newTestExchange(t).WebsocketAPI().GetTime(t.Context())
	assert.ErrorIs(t, err, websocket.ErrNotConnected, "WebsocketAPI.GetTime should error without a connection")
}

func TestWebsocketAPIHello(t *testing.T) {
	t.Parallel()
	ex, venue := newDerivativesWebsocketTestExchange(t)
	_, err := ex.WebsocketAPI().Hello(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "WebsocketAPI.Hello must reject a nil request")
	_, err = ex.WebsocketAPI().Hello(t.Context(), &DerivativesHelloRequest{ClientVersion: testSupportingClientVersion})
	require.ErrorIs(t, err, errDerivativesClientNameRequired, "WebsocketAPI.Hello must require a client name")
	_, err = ex.WebsocketAPI().Hello(t.Context(), &DerivativesHelloRequest{ClientName: testSupportingClientName})
	require.ErrorIs(t, err, errDerivativesClientVersionRequired, "WebsocketAPI.Hello must require a client version")

	resp, err := ex.WebsocketAPI().Hello(t.Context(), &DerivativesHelloRequest{ClientName: testSupportingClientName, ClientVersion: testSupportingClientVersion})
	require.NoError(t, err, "WebsocketAPI.Hello must not error")
	if mockTests {
		assert.Equal(t, &DerivativesHelloResponse{Version: mockSupportingVersion}, resp, "WebsocketAPI.Hello should decode the API version")
		requests := venue.requestsFor("public/hello")
		require.Len(t, requests, 1, "WebsocketAPI.Hello must send one request")
		assert.JSONEq(t, `{"client_name":"GoCryptoTrader","client_version":"test"}`, string(requests[0].Params), "WebsocketAPI.Hello should introduce the client")
	} else {
		assert.NotEmpty(t, resp.Version, "WebsocketAPI.Hello should return the API version")
	}

	_, err = newTestExchange(t).WebsocketAPI().Hello(t.Context(), &DerivativesHelloRequest{ClientName: testSupportingClientName, ClientVersion: testSupportingClientVersion})
	assert.ErrorIs(t, err, websocket.ErrNotConnected, "WebsocketAPI.Hello should error without a connection")
}

func TestWebsocketAPIGetStatus(t *testing.T) {
	t.Parallel()
	ex, _ := newDerivativesWebsocketTestExchange(t)
	resp, err := ex.WebsocketAPI().GetStatus(t.Context())
	require.NoError(t, err, "WebsocketAPI.GetStatus must not error")
	if mockTests {
		assert.Equal(t, &DerivativesStatusResponse{Locked: DerivativesLockedFalse}, resp, "WebsocketAPI.GetStatus should decode the lock status")
	} else {
		assert.Contains(t, []string{DerivativesLockedTrue, DerivativesLockedPartial, DerivativesLockedFalse}, resp.Locked, "WebsocketAPI.GetStatus should return a documented lock status")
	}

	_, err = newTestExchange(t).WebsocketAPI().GetStatus(t.Context())
	assert.ErrorIs(t, err, websocket.ErrNotConnected, "WebsocketAPI.GetStatus should error without a connection")
}

func TestWebsocketAPITestConnection(t *testing.T) {
	t.Parallel()
	ex, venue := newDerivativesWebsocketTestExchange(t)
	resp, err := ex.WebsocketAPI().TestConnection(t.Context(), "")
	require.NoError(t, err, "WebsocketAPI.TestConnection must not error")
	if mockTests {
		assert.Equal(t, &DerivativesTestConnectionResponse{Version: mockSupportingVersion}, resp, "WebsocketAPI.TestConnection should decode the API version")
		requests := venue.requestsFor("public/test")
		require.NotEmpty(t, requests, "WebsocketAPI.TestConnection must send a public/test request")
		assert.Empty(t, requests[len(requests)-1].Params, "WebsocketAPI.TestConnection should leave the params out without an expected result")
	} else {
		assert.NotEmpty(t, resp.Version, "WebsocketAPI.TestConnection should return the API version")
	}

	_, err = ex.WebsocketAPI().TestConnection(t.Context(), DerivativesTestExpectedResultException)
	derivativesErr, ok := errors.AsType[*DerivativesError](err)
	require.True(t, ok, "WebsocketAPI.TestConnection must return the error the gateway is asked for")
	assert.Equal(t, int64(11094), derivativesErr.Code, "WebsocketAPI.TestConnection should decode the requested error")

	_, err = newTestExchange(t).WebsocketAPI().TestConnection(t.Context(), "")
	assert.ErrorIs(t, err, websocket.ErrNotConnected, "WebsocketAPI.TestConnection should error without a connection")
}
