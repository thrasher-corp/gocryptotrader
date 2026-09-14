package bitstamp

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

func TestTradingPairUnmarshalJSON(t *testing.T) {
	t.Parallel()
	const inp = `
{
	"name": "BTC/USD",
	"url_symbol": "btcusd",
	"base_decimals": 8,
	"counter_decimals": 2,
	"instant_order_counter_decimals": 2,
	"minimum_order": "10.00 USD",
	"trading": "Enabled",
	"instant_and_market_orders": "Enabled",
	"description": "Bitcoin / U.S. dollar"
}
`
	var p TradingPair
	require.NoError(t, json.Unmarshal([]byte(inp), &p), "Unmarshal must not error")
	exp := TradingPair{
		Name:            "BTC/USD",
		URLSymbol:       "btcusd",
		BaseDecimals:    8,
		CounterDecimals: 2,
		MinimumOrder:    10,
		Trading:         "Enabled",
		Description:     "Bitcoin / U.S. dollar",
	}
	assert.Equal(t, exp, p, "TradingPair should unmarshal correctly")

	require.NoError(t, json.Unmarshal([]byte(`{"minimum_order":"0.0002"}`), &p), "Unmarshal must not error on a bare number")
	assert.Equal(t, 0.0002, p.MinimumOrder, "MinimumOrder should parse a value with no currency suffix")

	assert.Error(t, json.Unmarshal([]byte(`{"minimum_order":"nope USD"}`), &p), "Unmarshal should error on an unparsable minimum order")
	assert.Error(t, json.Unmarshal([]byte(`{"base_decimals":"five"}`), &p), "Unmarshal should surface a malformed payload")
}

func TestOrderSideUnmarshalJSON(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		data string
		exp  order.Side
		err  error
	}{
		// Bare numbers arrive on the websocket order feed, quoted ones on the REST ticker
		{data: `0`, exp: order.Buy},
		{data: `1`, exp: order.Sell},
		{data: `"0"`, exp: order.Buy},
		{data: `"1"`, exp: order.Sell},

		// Bad data bings
		{data: `null`, err: order.ErrSideIsInvalid},
		{data: `1.2`, err: order.ErrSideIsInvalid},
		{data: `-0`, err: order.ErrSideIsInvalid},
		{data: `-1`, err: order.ErrSideIsInvalid},
		{data: `""`, err: order.ErrSideIsInvalid},
		{data: `"buy"`, err: order.ErrSideIsInvalid},
		{data: `true`, err: order.ErrSideIsInvalid},
		{data: `1e0`, err: order.ErrSideIsInvalid},
		{data: `"-0"`, err: order.ErrSideIsInvalid},
		{data: `"2"`, err: order.ErrSideIsInvalid},
		{data: `"1e0"`, err: order.ErrSideIsInvalid},
	} {
		t.Run(tc.data, func(t *testing.T) {
			t.Parallel()
			var s orderSide
			err := json.Unmarshal([]byte(tc.data), &s)
			if tc.err != nil {
				assert.ErrorIs(t, err, tc.err, "Unmarshal should reject a non-binary side")
				return
			}
			require.NoError(t, err, "Unmarshal must not error")
			assert.Equal(t, tc.exp, s.Side(), "Side should decode")
		})
	}
}
