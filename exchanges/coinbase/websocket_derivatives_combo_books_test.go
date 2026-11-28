package coinbase

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket"
)

func TestWebsocketAPICreateCombo(t *testing.T) {
	t.Parallel()
	_, err := e.WebsocketAPI().CreateCombo(t.Context(), nil)
	require.ErrorIs(t, err, errDerivativesComboTradesRequired, "WebsocketAPI.CreateCombo must require trades")

	skipLiveUnlessCanManipulate(t)
	ex, venue := newDerivativesWebsocketTestExchange(t)
	resp, err := ex.WebsocketAPI().CreateCombo(t.Context(), testComboBooksTrades(t))
	require.NoError(t, err, "WebsocketAPI.CreateCombo must not error")
	if mockTests {
		assert.Equal(t, mockComboBooksCreatedCombo(), resp, "WebsocketAPI.CreateCombo should decode every field")
		requests := venue.requestsFor("private/create_combo")
		require.Len(t, requests, 1, "WebsocketAPI.CreateCombo must send one request")
		assert.JSONEq(t, `{"trades":[{"instrument_name":"BTC-29APR22-37500-C","amount":1,"direction":"buy"},{"instrument_name":"BTC-29APR22-37500-P","amount":1,"direction":"sell"}]}`, string(requests[0].Params), "WebsocketAPI.CreateCombo should send the trades")
	} else {
		assert.Len(t, resp.Legs, 2, "WebsocketAPI.CreateCombo should return the combo's legs")
	}

	unauthenticated, _ := newUnauthenticatedDerivativesWebsocketTestExchange(t)
	_, err = unauthenticated.WebsocketAPI().CreateCombo(t.Context(), mockComboBooksTrades())
	assert.ErrorIs(t, err, errDerivativesWebsocketNotAuthenticated, "WebsocketAPI.CreateCombo should refuse an unauthenticated connection")
}

func TestWebsocketAPIGetLegPrices(t *testing.T) {
	t.Parallel()
	_, err := e.WebsocketAPI().GetLegPrices(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "WebsocketAPI.GetLegPrices must reject a nil request")

	skipLiveWithoutCredentials(t)
	ex, _ := newDerivativesWebsocketTestExchange(t)
	resp, err := ex.WebsocketAPI().GetLegPrices(t.Context(), testComboBooksLegPricesRequest(t))
	require.NoError(t, err, "WebsocketAPI.GetLegPrices must not error")
	if mockTests {
		assert.Equal(t, mockComboBooksLegPrices(), resp, "WebsocketAPI.GetLegPrices should decode every field")
	} else {
		assert.Len(t, resp.Legs, 2, "WebsocketAPI.GetLegPrices should return each leg's price")
	}

	unauthenticated, _ := newUnauthenticatedDerivativesWebsocketTestExchange(t)
	_, err = unauthenticated.WebsocketAPI().GetLegPrices(t.Context(), mockComboBooksLegPricesRequest())
	assert.ErrorIs(t, err, errDerivativesWebsocketNotAuthenticated, "WebsocketAPI.GetLegPrices should refuse an unauthenticated connection")
}

func TestWebsocketAPIGetComboDetails(t *testing.T) {
	t.Parallel()
	ex, _ := newDerivativesWebsocketTestExchange(t)
	comboID := marketDataFutureCombo(t)
	resp, err := ex.WebsocketAPI().GetComboDetails(t.Context(), comboID)
	require.NoError(t, err, "WebsocketAPI.GetComboDetails must not error")
	if mockTests {
		assert.Equal(t, mockComboBooksFutureComboDetails(), resp, "WebsocketAPI.GetComboDetails should decode every field")
	} else {
		assert.Equal(t, comboID, resp.ID, "WebsocketAPI.GetComboDetails should return the combo requested")
	}

	_, err = newTestExchange(t).WebsocketAPI().GetComboDetails(t.Context(), comboID)
	assert.ErrorIs(t, err, websocket.ErrNotConnected, "WebsocketAPI.GetComboDetails should error without a connection")
}

func TestWebsocketAPIGetComboIDs(t *testing.T) {
	t.Parallel()
	ex, _ := newDerivativesWebsocketTestExchange(t)
	arg := &DerivativesComboIDsRequest{Currency: currency.BTC, State: DerivativesComboStateActive}
	resp, err := ex.WebsocketAPI().GetComboIDs(t.Context(), arg)
	require.NoError(t, err, "WebsocketAPI.GetComboIDs must not error")
	if mockTests {
		assert.Equal(t, mockComboBooksActiveComboIDs(), resp, "WebsocketAPI.GetComboIDs should decode every ID")
	} else {
		assert.NotEmpty(t, resp, "WebsocketAPI.GetComboIDs should return BTC's active combos")
	}

	_, err = newTestExchange(t).WebsocketAPI().GetComboIDs(t.Context(), arg)
	assert.ErrorIs(t, err, websocket.ErrNotConnected, "WebsocketAPI.GetComboIDs should error without a connection")
}

func TestWebsocketAPIGetCombos(t *testing.T) {
	t.Parallel()
	ex, _ := newDerivativesWebsocketTestExchange(t)
	resp, err := ex.WebsocketAPI().GetCombos(t.Context(), currency.BTC)
	require.NoError(t, err, "WebsocketAPI.GetCombos must not error")
	if mockTests {
		assert.Equal(t, mockComboBooksCombos(), resp, "WebsocketAPI.GetCombos should decode every field")
	} else {
		assert.NotEmpty(t, resp, "WebsocketAPI.GetCombos should return BTC's active combos")
	}

	_, err = newTestExchange(t).WebsocketAPI().GetCombos(t.Context(), currency.BTC)
	assert.ErrorIs(t, err, websocket.ErrNotConnected, "WebsocketAPI.GetCombos should error without a connection")
}
