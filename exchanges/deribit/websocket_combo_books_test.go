package deribit

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
	skipLiveUnlessCanManipulate(t)
	ex, venue := newWebsocketTestExchange(t)
	_, err := ex.WebsocketAPI().CreateCombo(t.Context(), nil)
	require.ErrorIs(t, err, errComboTradesRequired, "WebsocketAPI.CreateCombo must reject an empty trade list")

	result, err := ex.WebsocketAPI().CreateCombo(t.Context(), testComboTrades(t))
	require.NoError(t, err, "WebsocketAPI.CreateCombo must not error")
	if mockTests {
		assert.Equal(t, testCreatedCombo(), result, "WebsocketAPI.CreateCombo should decode every field")
		requests := venue.received("private/create_combo")
		require.Len(t, requests, 1, "WebsocketAPI.CreateCombo must send one request")
		assert.JSONEq(t, `{"trades":[{"instrument_name":"BTC-29APR22-37500-C","amount":1,"direction":"buy"},{"instrument_name":"BTC-29APR22-37500-P","amount":1,"direction":"sell"}]}`, string(requests[0].Params), "WebsocketAPI.CreateCombo should send the trades")
	} else {
		assert.Len(t, result.Legs, 2, "WebsocketAPI.CreateCombo should return the combo's legs")
	}

	_, err = newTestExchange(t).WebsocketAPI().CreateCombo(t.Context(), testComboTrades(t))
	assert.ErrorIs(t, err, errWebsocketNotAuthenticated, "WebsocketAPI.CreateCombo should error without an authenticated connection")
}

func TestWebsocketAPIGetLegPrices(t *testing.T) {
	t.Parallel()
	skipLiveWithoutCredentials(t)
	ex, _ := newWebsocketTestExchange(t)
	_, err := ex.WebsocketAPI().GetLegPrices(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "WebsocketAPI.GetLegPrices must reject a nil request")

	result, err := ex.WebsocketAPI().GetLegPrices(t.Context(), testLegPricesRequest(t))
	require.NoError(t, err, "WebsocketAPI.GetLegPrices must not error")
	if mockTests {
		assert.Equal(t, testLegPrices(), result, "WebsocketAPI.GetLegPrices should decode every field")
	} else {
		assert.Len(t, result.Legs, 2, "WebsocketAPI.GetLegPrices should return each leg's price")
	}

	_, err = newTestExchange(t).WebsocketAPI().GetLegPrices(t.Context(), testLegPricesRequest(t))
	assert.ErrorIs(t, err, errWebsocketNotAuthenticated, "WebsocketAPI.GetLegPrices should error without an authenticated connection")
}

func TestWebsocketAPIGetComboDetails(t *testing.T) {
	t.Parallel()
	ex, _ := newWebsocketTestExchange(t)
	comboID := marketDataFutureCombo(t)
	result, err := ex.WebsocketAPI().GetComboDetails(t.Context(), comboID)
	require.NoError(t, err, "WebsocketAPI.GetComboDetails must not error")
	if mockTests {
		assert.Equal(t, testFutureComboDetails(), result, "WebsocketAPI.GetComboDetails should decode every field")
	} else {
		assert.Equal(t, comboID, result.ID, "WebsocketAPI.GetComboDetails should return the combo requested")
	}

	_, err = newTestExchange(t).WebsocketAPI().GetComboDetails(t.Context(), comboID)
	assert.ErrorIs(t, err, websocket.ErrNotConnected, "WebsocketAPI.GetComboDetails should error without a websocket connection")
}

func TestWebsocketAPIGetComboIDs(t *testing.T) {
	t.Parallel()
	ex, _ := newWebsocketTestExchange(t)
	arg := &ComboIDsRequest{Currency: currency.BTC, State: ComboStateActive}
	result, err := ex.WebsocketAPI().GetComboIDs(t.Context(), arg)
	require.NoError(t, err, "WebsocketAPI.GetComboIDs must not error")
	if mockTests {
		assert.Equal(t, testActiveComboIDs(), result, "WebsocketAPI.GetComboIDs should decode every ID")
	} else {
		assert.NotEmpty(t, result, "WebsocketAPI.GetComboIDs should return BTC's active combos")
	}

	_, err = newTestExchange(t).WebsocketAPI().GetComboIDs(t.Context(), arg)
	assert.ErrorIs(t, err, websocket.ErrNotConnected, "WebsocketAPI.GetComboIDs should error without a websocket connection")
}

func TestWebsocketAPIGetCombos(t *testing.T) {
	t.Parallel()
	ex, _ := newWebsocketTestExchange(t)
	result, err := ex.WebsocketAPI().GetCombos(t.Context(), currency.BTC)
	require.NoError(t, err, "WebsocketAPI.GetCombos must not error")
	if mockTests {
		assert.Equal(t, testCombos(), result, "WebsocketAPI.GetCombos should decode every field")
	} else {
		assert.NotEmpty(t, result, "WebsocketAPI.GetCombos should return BTC's active combos")
	}

	_, err = newTestExchange(t).WebsocketAPI().GetCombos(t.Context(), currency.BTC)
	assert.ErrorIs(t, err, websocket.ErrNotConnected, "WebsocketAPI.GetCombos should error without a websocket connection")
}
