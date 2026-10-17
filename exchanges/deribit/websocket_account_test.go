package deribit

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/fill"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

func TestWebsocketHandleOrders(t *testing.T) {
	t.Parallel()
	ex := newTestExchange(t)
	exp := []order.Detail{{
		Exchange:        ex.Name,
		OrderID:         "5",
		Pair:            currency.NewPairWithDelimiter("BTC", "PERPETUAL", currency.DashDelimiter),
		AssetType:       asset.Futures,
		Side:            order.Buy,
		Type:            order.Limit,
		Status:          order.Open,
		TimeInForce:     order.GoodTillCancel,
		Price:           10502.52,
		Amount:          200,
		RemainingAmount: 200,
		Date:            time.UnixMilli(1581507423789),
		LastUpdated:     time.UnixMilli(1581507423789),
	}}
	require.NoError(t, ex.wsHandleOrders(t.Context(), "user.orders.BTC-PERPETUAL.raw", notification("user.orders.BTC-PERPETUAL.raw", testWebsocketOrder)), "wsHandleOrders must not error for a raw channel's order")
	assert.Equal(t, exp, relayed(t, ex), "wsHandleOrders should relay a raw channel's order")
	require.NoError(t, ex.wsHandleOrders(t.Context(), "user.orders.future.BTC.100ms", notification("user.orders.future.BTC.100ms", "["+testWebsocketOrder+"]")), "wsHandleOrders must not error for an interval channel's orders")
	assert.Equal(t, exp, relayed(t, ex), "wsHandleOrders should relay an interval channel's orders")

	err := ex.wsHandleOrders(t.Context(), "user.orders.future.any.100ms", notification("user.orders.future.any.100ms", `[{"instrument_name":"BTC","direction":"buy"},`+testWebsocketOrder+`]`))
	assert.ErrorIs(t, err, errUnsupportedInstrumentFormat, "wsHandleOrders should report an order it cannot convert")
	assert.Equal(t, exp, relayed(t, ex), "wsHandleOrders should still relay the orders it converts")
	err = ex.wsHandleOrders(t.Context(), "user.orders.future.any.100ms", notification("user.orders.future.any.100ms", `[{"instrument_name":"BTC"}]`))
	assert.ErrorIs(t, err, errUnsupportedInstrumentFormat, "wsHandleOrders should report an order it cannot convert")
	assertNothingRelayed(t, ex)
	assert.ErrorIs(t, ex.wsHandleOrders(t.Context(), "user.orders.BTC-PERPETUAL.raw", notification("user.orders.BTC-PERPETUAL.raw", `[]`)), errMalformedNotification, "wsHandleOrders should reject a raw channel's list")
	assert.ErrorIs(t, ex.wsHandleOrders(t.Context(), "user.orders.future.BTC.100ms", notification("user.orders.future.BTC.100ms", `{}`)), errMalformedNotification, "wsHandleOrders should reject an interval channel's single order")
}

func TestWebsocketHandleUserTrades(t *testing.T) {
	t.Parallel()
	// Following Deribit's documented user.trades example
	const trades = `[{"trade_seq":30289432,"trade_id":"48079254","timestamp":1590484156350,"tick_direction":0,"state":"filled","reduce_only":false,"price":8954,"post_only":false,"order_type":"market","order_id":"4008965646","matching_id":null,"mark_price":8952.86,"liquidity":"T","instrument_name":"BTC-PERPETUAL","index_price":8956.73,"fee_currency":"BTC","fee":0.00000168,"direction":"sell","amount":20}]`
	ex := newTestExchange(t)
	ex.Features.Enabled.FillsFeed = true
	ex.Websocket.Fills.Setup(true, ex.Websocket.DataHandler)
	require.NoError(t, ex.wsHandleUserTrades(t.Context(), notification("user.trades.BTC-PERPETUAL.100ms", trades)), "wsHandleUserTrades must not error")
	exp := []fill.Data{{
		ID:           "48079254",
		TradeID:      "48079254",
		Timestamp:    time.UnixMilli(1590484156350),
		Exchange:     ex.Name,
		AssetType:    asset.Futures,
		CurrencyPair: currency.NewPairWithDelimiter("BTC", "PERPETUAL", currency.DashDelimiter),
		Side:         order.Sell,
		OrderID:      "4008965646",
		Price:        8954,
		Amount:       20,
	}}
	assert.Equal(t, exp, relayed(t, ex), "wsHandleUserTrades should relay the account's trades as fills")

	err := ex.wsHandleUserTrades(t.Context(), notification("user.trades.future.any.100ms", `[{"instrument_name":"BTC","direction":"sell"}]`))
	assert.ErrorIs(t, err, errUnsupportedInstrumentFormat, "wsHandleUserTrades should report a trade it cannot convert")
	assertNothingRelayed(t, ex)
	assert.ErrorIs(t, ex.wsHandleUserTrades(t.Context(), notification("user.trades.future.any.100ms", `{}`)), errMalformedNotification, "wsHandleUserTrades should reject malformed trades")

	disabled := newTestExchange(t)
	disabled.Features.Enabled.FillsFeed = false
	require.NoError(t, disabled.wsHandleUserTrades(t.Context(), notification("user.trades.BTC-PERPETUAL.100ms", trades)), "wsHandleUserTrades must not error with the fills feed off")
	assertNothingRelayed(t, disabled)
}
