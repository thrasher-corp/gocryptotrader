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
	var p TradingPair
	err := json.Unmarshal([]byte(`{"name":"EUR/USD","url_symbol":"eurusd","base_decimals":5,"counter_decimals":5,"minimum_order":"10.00000 USD","trading":"Enabled","description":"Euro / U.S. dollar"}`), &p)
	require.NoError(t, err, "Unmarshal must not error")
	assert.Equal(t, 10.0, p.MinimumOrder, "MinimumOrder should drop the currency suffix")
	assert.Equal(t, "EUR/USD", p.Name, "Name should decode")
	assert.Equal(t, "eurusd", p.URLSymbol, "URLSymbol should decode")
	assert.Equal(t, 5, p.BaseDecimals, "BaseDecimals should decode")
	assert.Equal(t, 5, p.CounterDecimals, "CounterDecimals should decode")
	assert.Equal(t, "Enabled", p.Trading, "Trading should decode")
	assert.Equal(t, "Euro / U.S. dollar", p.Description, "Description should decode")

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
